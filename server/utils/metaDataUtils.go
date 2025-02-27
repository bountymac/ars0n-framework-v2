package utils

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
)

type MetaDataStatus struct {
	ID            string         `json:"id"`
	ScanID        string         `json:"scan_id"`
	Domain        string         `json:"domain"`
	Status        string         `json:"status"`
	Result        sql.NullString `json:"result,omitempty"`
	Error         sql.NullString `json:"error,omitempty"`
	StdOut        sql.NullString `json:"stdout,omitempty"`
	StdErr        sql.NullString `json:"stderr,omitempty"`
	Command       sql.NullString `json:"command,omitempty"`
	ExecTime      sql.NullString `json:"execution_time,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	ScopeTargetID string         `json:"scope_target_id"`
}

type DNSResults struct {
	ARecords     []string
	AAAARecords  []string
	CNAMERecords []string
	MXRecords    []string
	TXTRecords   []string
	NSRecords    []string
	PTRRecords   []string
	SRVRecords   []string
}

func NormalizeURL(url string) string {
	// Fix double colon issue
	url = strings.ReplaceAll(url, "https:://", "https://")
	url = strings.ReplaceAll(url, "http:://", "http://")

	// Ensure URL has proper scheme
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	return url
}

func SanitizeResponse(input []byte) string {
	// Remove null bytes
	sanitized := bytes.ReplaceAll(input, []byte{0}, []byte{})

	// Convert to string and handle any invalid UTF-8
	str := string(sanitized)

	// Replace any other problematic characters
	str = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1 // Drop the character
		}
		return r
	}, str)

	return str
}

func RunMetaDataScan(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ScopeTargetID string `json:"scope_target_id" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.ScopeTargetID == "" {
		http.Error(w, "Invalid request body. `scope_target_id` is required.", http.StatusBadRequest)
		return
	}

	// Get domain from scope target
	var domain string
	err := dbPool.QueryRow(context.Background(),
		`SELECT TRIM(LEADING '*.' FROM scope_target) FROM scope_targets WHERE id = $1`,
		payload.ScopeTargetID).Scan(&domain)
	if err != nil {
		log.Printf("[ERROR] Failed to get domain: %v", err)
		http.Error(w, "Failed to get domain", http.StatusInternalServerError)
		return
	}

	scanID := uuid.New().String()
	insertQuery := `INSERT INTO metadata_scans (scan_id, domain, status, scope_target_id) VALUES ($1, $2, $3, $4)`
	_, err = dbPool.Exec(context.Background(), insertQuery, scanID, domain, "pending", payload.ScopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to create scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}

	go ExecuteAndParseMetaDataScan(scanID, domain)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func ExecuteAndParseMetaDataScan(scanID, domain string) {
	log.Printf("[INFO] Starting Nuclei SSL scan for domain %s (scan ID: %s)", domain, scanID)
	startTime := time.Now()

	// Get scope target ID and latest httpx results
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(),
		`SELECT scope_target_id FROM metadata_scans WHERE scan_id = $1`,
		scanID).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to get scope target ID: %v", err)
		UpdateMetaDataScanStatus(scanID, "error", "", fmt.Sprintf("Failed to get scope target ID: %v", err), "", time.Since(startTime).String())
		return
	}

	// Get latest httpx results
	var httpxResults string
	err = dbPool.QueryRow(context.Background(), `
		SELECT result 
		FROM httpx_scans 
		WHERE scope_target_id = $1 
		AND status = 'success' 
		ORDER BY created_at DESC 
		LIMIT 1`, scopeTargetID).Scan(&httpxResults)
	if err != nil {
		log.Printf("[ERROR] Failed to get httpx results: %v", err)
		UpdateMetaDataScanStatus(scanID, "error", "", fmt.Sprintf("Failed to get httpx results: %v", err), "", time.Since(startTime).String())
		return
	}

	// Create a temporary file for URLs
	tempFile, err := os.CreateTemp("", "urls-*.txt")
	if err != nil {
		log.Printf("[ERROR] Failed to create temp file for scan ID %s: %v", scanID, err)
		UpdateMetaDataScanStatus(scanID, "error", "", fmt.Sprintf("Failed to create temp file: %v", err), "", time.Since(startTime).String())
		return
	}
	defer os.Remove(tempFile.Name())
	log.Printf("[INFO] Created temporary file for URLs: %s", tempFile.Name())

	// Process httpx results and write URLs to temp file
	var urls []string
	for _, line := range strings.Split(httpxResults, "\n") {
		if line == "" {
			continue
		}
		var result struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			log.Printf("[WARN] Failed to parse httpx result line for scan ID %s: %v", scanID, err)
			continue
		}
		if result.URL != "" && strings.HasPrefix(result.URL, "https://") {
			urls = append(urls, result.URL)
		}
	}

	if len(urls) == 0 {
		log.Printf("[ERROR] No valid HTTPS URLs found in httpx results for scan ID: %s", scanID)
		UpdateMetaDataScanStatus(scanID, "error", "", "No valid HTTPS URLs found in httpx results", "", time.Since(startTime).String())
		return
	}

	// Write URLs to temp file
	if err := os.WriteFile(tempFile.Name(), []byte(strings.Join(urls, "\n")), 0644); err != nil {
		log.Printf("[ERROR] Failed to write URLs to temp file for scan ID %s: %v", scanID, err)
		UpdateMetaDataScanStatus(scanID, "error", "", fmt.Sprintf("Failed to write URLs to temp file: %v", err), "", time.Since(startTime).String())
		return
	}
	log.Printf("[INFO] Successfully wrote %d URLs to temp file for scan ID: %s", len(urls), scanID)

	// Run Katana scan first
	log.Printf("[INFO] Starting Katana scan for scan ID: %s", scanID)
	katanaResults := make(map[string][]string)
	for _, url := range urls {
		log.Printf("[DEBUG] Running Katana scan for URL: %s", url)
		katanaCmd := exec.Command(
			"docker", "exec", "ars0n-framework-v2-katana-1",
			"katana",
			"-u", url,
			"-jc",
			"-d", "3",
			"-j",
			"-v",
		)

		var stdout, stderr bytes.Buffer
		katanaCmd.Stdout = &stdout
		katanaCmd.Stderr = &stderr

		log.Printf("[DEBUG] Executing Katana command: %s", katanaCmd.String())
		if err := katanaCmd.Run(); err != nil {
			log.Printf("[WARN] Katana scan failed for URL %s: %v\nStderr: %s", url, err, stderr.String())
			continue
		}
		log.Printf("[DEBUG] Katana scan completed for URL: %s", url)

		var crawledURLs []string
		seenURLs := make(map[string]bool)

		for _, line := range strings.Split(stdout.String(), "\n") {
			if line == "" {
				continue
			}

			var result struct {
				Timestamp string `json:"timestamp"`
				Request   struct {
					Method   string `json:"method"`
					Endpoint string `json:"endpoint"`
					Tag      string `json:"tag"`
					Source   string `json:"source"`
					Raw      string `json:"raw"`
				} `json:"request"`
				Response struct {
					StatusCode    int                    `json:"status_code"`
					Headers       map[string]interface{} `json:"headers"`
					Body          string                 `json:"body"`
					ContentLength int                    `json:"content_length"`
				} `json:"response"`
			}

			if err := json.Unmarshal([]byte(line), &result); err != nil {
				log.Printf("[WARN] Failed to parse Katana output line: %v", err)
				continue
			}

			// Add unique URLs from various sources
			addUniqueURL := func(urlStr string) {
				if urlStr != "" && !seenURLs[urlStr] {
					seenURLs[urlStr] = true
					crawledURLs = append(crawledURLs, urlStr)
				}
			}

			// Process endpoint URL
			addUniqueURL(result.Request.Endpoint)

			// Process source URL
			addUniqueURL(result.Request.Source)

			// Look for URLs in response headers
			for _, headerVals := range result.Response.Headers {
				switch v := headerVals.(type) {
				case string:
					if strings.Contains(v, "http://") || strings.Contains(v, "https://") {
						addUniqueURL(v)
					}
				case []interface{}:
					for _, val := range v {
						if str, ok := val.(string); ok {
							if strings.Contains(str, "http://") || strings.Contains(str, "https://") {
								addUniqueURL(str)
							}
						}
					}
				}
			}
		}

		// Remove any invalid URLs and normalize the rest
		var validURLs []string
		for _, urlStr := range crawledURLs {
			if strings.HasPrefix(urlStr, "http://") || strings.HasPrefix(urlStr, "https://") {
				validURLs = append(validURLs, NormalizeURL(urlStr))
			}
		}
		crawledURLs = validURLs

		log.Printf("[INFO] Katana found %d unique URLs for %s", len(crawledURLs), url)
		if len(crawledURLs) > 0 {
			log.Printf("[DEBUG] First 5 URLs found by Katana for %s:", url)
			for i, crawledURL := range crawledURLs {
				if i >= 5 {
					break
				}
				log.Printf("[DEBUG]   - %s", crawledURL)
			}
			if len(crawledURLs) > 5 {
				log.Printf("[DEBUG]   ... and %d more URLs", len(crawledURLs)-5)
			}
		}
		katanaResults[url] = crawledURLs
	}

	log.Printf("[INFO] Katana scan completed for all URLs. Total results: %d URLs across %d targets",
		func() int {
			total := 0
			for _, urls := range katanaResults {
				total += len(urls)
			}
			return total
		}(),
		len(katanaResults))

	// Update the target_urls table to include Katana results
	for baseURL, crawledURLs := range katanaResults {
		// First check if the target_url exists
		var exists bool
		err = dbPool.QueryRow(context.Background(),
			`SELECT EXISTS(SELECT 1 FROM target_urls WHERE url = $1 AND scope_target_id = $2)`,
			baseURL, scopeTargetID).Scan(&exists)
		if err != nil {
			log.Printf("[ERROR] Failed to check if target URL exists %s: %v", baseURL, err)
			continue
		}

		// If it doesn't exist, insert it
		if !exists {
			_, err = dbPool.Exec(context.Background(),
				`INSERT INTO target_urls (url, scope_target_id) VALUES ($1, $2)`,
				baseURL, scopeTargetID)
			if err != nil {
				log.Printf("[ERROR] Failed to insert target URL %s: %v", baseURL, err)
				continue
			}
			log.Printf("[DEBUG] Inserted new target URL: %s", baseURL)
		} else {
			log.Printf("[DEBUG] Target URL already exists: %s", baseURL)
		}

		// Then update with Katana results
		katanaResultsJSON, err := json.Marshal(crawledURLs)
		if err != nil {
			log.Printf("[ERROR] Failed to marshal Katana results for URL %s: %v", baseURL, err)
			continue
		}

		_, err = dbPool.Exec(context.Background(),
			`UPDATE target_urls 
			 SET katana_results = $1::jsonb 
			 WHERE url = $2 AND scope_target_id = $3`,
			string(katanaResultsJSON), baseURL, scopeTargetID)
		if err != nil {
			log.Printf("[ERROR] Failed to update Katana results for URL %s: %v", baseURL, err)
		} else {
			log.Printf("[INFO] Successfully stored %d Katana results for URL %s", len(crawledURLs), baseURL)
		}
	}

	// Copy the URLs file into the container for SSL scan
	copyCmd := exec.Command(
		"docker", "cp",
		tempFile.Name(),
		"ars0n-framework-v2-nuclei-1:/urls.txt",
	)
	if err := copyCmd.Run(); err != nil {
		log.Printf("[ERROR] Failed to copy URLs file to container: %v", err)
		UpdateMetaDataScanStatus(scanID, "error", "", fmt.Sprintf("Failed to copy URLs file: %v", err), "", time.Since(startTime).String())
		return
	}

	// Run all templates in one scan with JSON output
	cmd := exec.Command(
		"docker", "exec", "ars0n-framework-v2-nuclei-1",
		"nuclei",
		"-t", "/root/nuclei-templates/ssl/",
		"-list", "/urls.txt",
		"-j",
		"-o", "/output.json",
	)
	log.Printf("[INFO] Executing command: %s", cmd.String())

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		log.Printf("[ERROR] Nuclei scan failed: %v", err)
		UpdateMetaDataScanStatus(scanID, "error", "", stderr.String(), cmd.String(), time.Since(startTime).String())
		return
	}

	// Read the JSON output file
	outputCmd := exec.Command(
		"docker", "exec", "ars0n-framework-v2-nuclei-1",
		"cat", "/output.json",
	)
	output, err := outputCmd.Output()
	if err != nil {
		log.Printf("[ERROR] Failed to read output file: %v", err)
		UpdateMetaDataScanStatus(scanID, "error", "", fmt.Sprintf("Failed to read output file: %v", err), cmd.String(), time.Since(startTime).String())
		return
	}

	// Process each finding and update the database
	findings := strings.Split(string(output), "\n")
	for _, finding := range findings {
		if finding == "" {
			continue
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(finding), &result); err != nil {
			log.Printf("[ERROR] Failed to parse JSON finding: %v", err)
			continue
		}

		templateID, ok := result["template-id"].(string)
		if !ok {
			continue
		}

		matchedURL, ok := result["matched-at"].(string)
		if !ok {
			continue
		}

		// Convert matched-at (host:port) to URL
		url := "https://" + strings.TrimSuffix(matchedURL, ":443")

		// Update the target_urls table based on the template
		var updateField string
		switch templateID {
		case "deprecated-tls":
			updateField = "has_deprecated_tls"
		case "expired-ssl":
			updateField = "has_expired_ssl"
		case "mismatched-ssl-certificate":
			updateField = "has_mismatched_ssl"
		case "revoked-ssl-certificate":
			updateField = "has_revoked_ssl"
		case "self-signed-ssl":
			updateField = "has_self_signed_ssl"
		case "untrusted-root-certificate":
			updateField = "has_untrusted_root_ssl"
		default:
			continue
		}

		query := fmt.Sprintf("UPDATE target_urls SET %s = true WHERE url = $1 AND scope_target_id = $2", updateField)
		commandTag, err := dbPool.Exec(context.Background(), query, url, scopeTargetID)
		if err != nil {
			log.Printf("[ERROR] Failed to update target URL %s for template %s: %v", url, templateID, err)
		} else {
			rowsAffected := commandTag.RowsAffected()
			log.Printf("[INFO] Successfully updated target URL %s with %s = true (Rows affected: %d)", url, updateField, rowsAffected)
		}
	}

	// Update scan status to indicate SSL scan is complete but tech scan is pending
	UpdateMetaDataScanStatus(
		scanID,
		"running",
		string(output),
		stderr.String(),
		cmd.String(),
		time.Since(startTime).String(),
	)

	// Clean up the output file
	exec.Command("docker", "exec", "ars0n-framework-v2-nuclei-1", "rm", "/output.json").Run()

	log.Printf("[INFO] SSL scan completed for scan ID: %s, starting tech scan", scanID)

	// Run the HTTP/technologies scan
	if err := ExecuteAndParseNucleiTechScan(urls, scopeTargetID); err != nil {
		log.Printf("[ERROR] Failed to run HTTP/technologies scan: %v", err)
		UpdateMetaDataScanStatus(scanID, "error", string(output), fmt.Sprintf("Tech scan failed: %v", err), cmd.String(), time.Since(startTime).String())
		return
	}

	// Update final scan status after both scans complete successfully
	UpdateMetaDataScanStatus(
		scanID,
		"success",
		string(output),
		stderr.String(),
		cmd.String(),
		time.Since(startTime).String(),
	)

	log.Printf("[INFO] Both SSL and tech scans completed successfully for scan ID: %s", scanID)
}

func ExecuteAndParseNucleiTechScan(urls []string, scopeTargetID string) error {
	log.Printf("[INFO] Starting Nuclei HTTP/technologies scan")
	startTime := time.Now()

	// Create an HTTP client with reasonable timeouts and TLS config
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
		},
	}

	// Create a temporary file for URLs
	tempFile, err := os.CreateTemp("", "urls-*.txt")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write URLs to temp file
	if err := os.WriteFile(tempFile.Name(), []byte(strings.Join(urls, "\n")), 0644); err != nil {
		return fmt.Errorf("failed to write URLs to temp file: %v", err)
	}

	// Copy the URLs file into the container
	copyCmd := exec.Command(
		"docker", "cp",
		tempFile.Name(),
		"ars0n-framework-v2-nuclei-1:/urls.txt",
	)
	if err := copyCmd.Run(); err != nil {
		return fmt.Errorf("failed to copy URLs file to container: %v", err)
	}

	// Run HTTP/technologies templates
	cmd := exec.Command(
		"docker", "exec", "ars0n-framework-v2-nuclei-1",
		"nuclei",
		"-t", "/root/nuclei-templates/http/technologies/",
		"-list", "/urls.txt",
		"-j",
		"-o", "/tech-output.json",
	)
	log.Printf("[INFO] Executing command: %s", cmd.String())

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("nuclei tech scan failed: %v\nstderr: %s", err, stderr.String())
	}

	// Read the JSON output file
	outputCmd := exec.Command(
		"docker", "exec", "ars0n-framework-v2-nuclei-1",
		"cat", "/tech-output.json",
	)
	output, err := outputCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to read output file: %v", err)
	}

	// Process findings and update the database
	findings := strings.Split(string(output), "\n")
	urlFindings := make(map[string][]interface{})

	for _, finding := range findings {
		if finding == "" {
			continue
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(finding), &result); err != nil {
			log.Printf("[ERROR] Failed to parse JSON finding: %v", err)
			continue
		}

		matchedURL, ok := result["matched-at"].(string)
		if !ok {
			continue
		}

		// Convert matched-at to proper URL
		if strings.Contains(matchedURL, "://") {
			// Already a full URL
			matchedURL = NormalizeURL(matchedURL)
		} else if strings.Contains(matchedURL, ":") {
			// hostname:port format
			host := strings.Split(matchedURL, ":")[0]
			matchedURL = NormalizeURL("https://" + host)
		} else {
			// Just a hostname
			matchedURL = NormalizeURL("https://" + matchedURL)
		}

		// Add finding to the URL's findings array
		urlFindings[matchedURL] = append(urlFindings[matchedURL], result)
	}

	// Make HTTP requests and update each URL with its findings and response data
	for urlStr, findings := range urlFindings {
		// Parse URL to get hostname for DNS lookups
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			log.Printf("[ERROR] Failed to parse URL %s: %v", urlStr, err)
			continue
		}

		// Perform DNS lookups
		dnsResults := PerformDNSLookups(parsedURL.Hostname())

		// Make HTTP request
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			log.Printf("[ERROR] Failed to create request for URL %s: %v", urlStr, err)
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[ERROR] Failed to make request to URL %s: %v", urlStr, err)
			continue
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("[ERROR] Failed to read response body from URL %s: %v", urlStr, err)
			continue
		}

		// Sanitize the response body
		sanitizedBody := SanitizeResponse(body)

		// Convert headers to map for JSON storage
		headers := make(map[string]interface{})
		for k, v := range resp.Header {
			// Convert header values to a consistent format
			if len(v) == 1 {
				headers[k] = v[0] // Store single value directly
			} else {
				headers[k] = v // Store multiple values as string slice
			}
		}

		// Convert headers to JSON before storing
		headersJSON, err := json.Marshal(headers)
		if err != nil {
			log.Printf("[ERROR] Failed to marshal headers for URL %s: %v", urlStr, err)
			continue
		}

		// Update database with findings, response data, and DNS records
		query := `
			UPDATE target_urls 
			SET 
				findings_json = $1::jsonb,
				http_response = $2,
				http_response_headers = $3::jsonb,
				dns_a_records = $4,
				dns_aaaa_records = $5,
				dns_cname_records = $6,
				dns_mx_records = $7,
				dns_txt_records = $8,
				dns_ns_records = $9,
				dns_ptr_records = $10,
				dns_srv_records = $11
			WHERE url = $12 AND scope_target_id = $13`

		// Convert findings to proper JSON
		findingsJSON, err := json.Marshal(findings)
		if err != nil {
			log.Printf("[ERROR] Failed to marshal findings for URL %s: %v", urlStr, err)
			continue
		}

		commandTag, err := dbPool.Exec(context.Background(),
			query,
			findingsJSON,
			sanitizedBody,
			string(headersJSON), // Store headers as JSONB
			dnsResults.ARecords,
			dnsResults.AAAARecords,
			dnsResults.CNAMERecords,
			dnsResults.MXRecords,
			dnsResults.TXTRecords,
			dnsResults.NSRecords,
			dnsResults.PTRRecords,
			dnsResults.SRVRecords,
			urlStr,
			scopeTargetID,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to update findings and response data for URL %s: %v", urlStr, err)
			continue
		}
		rowsAffected := commandTag.RowsAffected()
		log.Printf("[INFO] Updated findings and response data for URL %s (Rows affected: %d)", urlStr, rowsAffected)
	}

	// Clean up the output file
	exec.Command("docker", "exec", "ars0n-framework-v2-nuclei-1", "rm", "/tech-output.json").Run()

	log.Printf("[INFO] HTTP/technologies scan completed in %s", time.Since(startTime))
	return nil
}

func PerformDNSLookups(hostname string) DNSResults {
	log.Printf("[DEBUG] Starting DNS lookups for hostname: %s", hostname)
	var results DNSResults
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// First, try to get DNS records from latest Amass scan
	var amassResult string
	err := dbPool.QueryRow(context.Background(), `
		SELECT result
		FROM amass_scans 
		WHERE status = 'success' 
		AND result::jsonb ? 'dns_records'
		ORDER BY created_at DESC 
		LIMIT 1`).Scan(&amassResult)

	if err == nil && amassResult != "" {
		var amassData struct {
			DNSRecords struct {
				A     []string `json:"a"`
				AAAA  []string `json:"aaaa"`
				CNAME []string `json:"cname"`
				MX    []string `json:"mx"`
				TXT   []string `json:"txt"`
				NS    []string `json:"ns"`
				PTR   []string `json:"ptr"`
				SRV   []string `json:"srv"`
			} `json:"dns_records"`
		}

		if err := json.Unmarshal([]byte(amassResult), &amassData); err == nil {
			log.Printf("[DEBUG] Found Amass DNS records for %s:", hostname)
			log.Printf("[DEBUG]   A Records: %d", len(amassData.DNSRecords.A))
			log.Printf("[DEBUG]   AAAA Records: %d", len(amassData.DNSRecords.AAAA))
			log.Printf("[DEBUG]   CNAME Records: %d", len(amassData.DNSRecords.CNAME))
			log.Printf("[DEBUG]   MX Records: %d", len(amassData.DNSRecords.MX))
			log.Printf("[DEBUG]   TXT Records: %d", len(amassData.DNSRecords.TXT))
			log.Printf("[DEBUG]   NS Records: %d", len(amassData.DNSRecords.NS))
			log.Printf("[DEBUG]   PTR Records: %d", len(amassData.DNSRecords.PTR))
			log.Printf("[DEBUG]   SRV Records: %d", len(amassData.DNSRecords.SRV))

			results.ARecords = append(results.ARecords, amassData.DNSRecords.A...)
			results.AAAARecords = append(results.AAAARecords, amassData.DNSRecords.AAAA...)
			results.CNAMERecords = append(results.CNAMERecords, amassData.DNSRecords.CNAME...)
			results.MXRecords = append(results.MXRecords, amassData.DNSRecords.MX...)
			results.TXTRecords = append(results.TXTRecords, amassData.DNSRecords.TXT...)
			results.NSRecords = append(results.NSRecords, amassData.DNSRecords.NS...)
			results.PTRRecords = append(results.PTRRecords, amassData.DNSRecords.PTR...)
			results.SRVRecords = append(results.SRVRecords, amassData.DNSRecords.SRV...)
		}
	} else if err != pgx.ErrNoRows {
		log.Printf("[DEBUG] Error fetching Amass DNS records: %v", err)
	}

	// Create a custom resolver with shorter timeout
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 5 * time.Second,
			}
			return d.DialContext(ctx, network, "8.8.8.8:53")
		},
	}

	// A and AAAA records with error logging
	log.Printf("[DEBUG] Looking up A/AAAA records for %s", hostname)
	if ips, err := resolver.LookupIPAddr(ctx, hostname); err == nil {
		for _, ip := range ips {
			if ipv4 := ip.IP.To4(); ipv4 != nil {
				results.ARecords = append(results.ARecords, ipv4.String())
				log.Printf("[DEBUG] Found A record: %s", ipv4.String())
			} else {
				results.AAAARecords = append(results.AAAARecords, ip.IP.String())
				log.Printf("[DEBUG] Found AAAA record: %s", ip.IP.String())
			}
		}
	} else {
		log.Printf("[DEBUG] A/AAAA lookup failed for %s: %v", hostname, err)
	}

	// CNAME lookup with better error handling
	log.Printf("[DEBUG] Looking up CNAME records for %s", hostname)
	if cname, err := resolver.LookupCNAME(ctx, hostname); err == nil && cname != "" {
		cname = strings.TrimSuffix(cname, ".")
		if cname != hostname {
			record := fmt.Sprintf("%s -> %s", hostname, cname)
			results.CNAMERecords = append(results.CNAMERecords, record)
			log.Printf("[DEBUG] Found CNAME record: %s", record)
		}
	} else if err != nil && !strings.Contains(err.Error(), "no such host") {
		log.Printf("[DEBUG] CNAME lookup failed for %s: %v", hostname, err)
	}

	// MX lookup with improved formatting
	log.Printf("[DEBUG] Looking up MX records for %s", hostname)
	if mxRecords, err := resolver.LookupMX(ctx, hostname); err == nil {
		for _, mx := range mxRecords {
			record := fmt.Sprintf("Priority: %d | Server: %s", mx.Pref, strings.TrimSuffix(mx.Host, "."))
			results.MXRecords = append(results.MXRecords, record)
			log.Printf("[DEBUG] Found MX record: %s", record)
		}
	} else if err != nil && !strings.Contains(err.Error(), "no such host") {
		log.Printf("[DEBUG] MX lookup failed for %s: %v", hostname, err)
	}

	// TXT lookup with error handling
	log.Printf("[DEBUG] Looking up TXT records for %s", hostname)
	if txtRecords, err := resolver.LookupTXT(ctx, hostname); err == nil {
		for _, txt := range txtRecords {
			results.TXTRecords = append(results.TXTRecords, txt)
			log.Printf("[DEBUG] Found TXT record: %s", txt)
		}
	} else if err != nil && !strings.Contains(err.Error(), "no such host") {
		log.Printf("[DEBUG] TXT lookup failed for %s: %v", hostname, err)
	}

	// NS lookup with error handling
	log.Printf("[DEBUG] Looking up NS records for %s", hostname)
	if nsRecords, err := resolver.LookupNS(ctx, hostname); err == nil {
		for _, ns := range nsRecords {
			record := strings.TrimSuffix(ns.Host, ".")
			results.NSRecords = append(results.NSRecords, record)
			log.Printf("[DEBUG] Found NS record: %s", record)
		}
	} else if err != nil && !strings.Contains(err.Error(), "no such host") {
		log.Printf("[DEBUG] NS lookup failed for %s: %v", hostname, err)
	}

	// PTR lookup for both IPv4 and IPv6
	lookupPTR := func(ip string) {
		log.Printf("[DEBUG] Looking up PTR records for IP %s", ip)
		if names, err := resolver.LookupAddr(ctx, ip); err == nil {
			for _, name := range names {
				record := fmt.Sprintf("%s -> %s", ip, strings.TrimSuffix(name, "."))
				results.PTRRecords = append(results.PTRRecords, record)
				log.Printf("[DEBUG] Found PTR record: %s", record)
			}
		} else if err != nil && !strings.Contains(err.Error(), "no such host") {
			log.Printf("[DEBUG] PTR lookup failed for %s: %v", ip, err)
		}
	}

	// Perform PTR lookups for both A and AAAA records
	for _, ip := range results.ARecords {
		lookupPTR(ip)
	}
	for _, ip := range results.AAAARecords {
		lookupPTR(ip)
	}

	// SRV lookup with improved formatting and error handling
	services := []string{"_http._tcp", "_https._tcp", "_ldap._tcp", "_kerberos._tcp"}
	for _, service := range services {
		fullService := service + "." + hostname
		log.Printf("[DEBUG] Looking up SRV records for %s", fullService)
		if _, addrs, err := resolver.LookupSRV(ctx, "", "", fullService); err == nil {
			for _, addr := range addrs {
				record := fmt.Sprintf("Service: %s | Target: %s | Port: %d | Priority: %d | Weight: %d",
					service,
					strings.TrimSuffix(addr.Target, "."),
					addr.Port,
					addr.Priority,
					addr.Weight)
				results.SRVRecords = append(results.SRVRecords, record)
				log.Printf("[DEBUG] Found SRV record: %s", record)
			}
		} else if err != nil && !strings.Contains(err.Error(), "no such host") {
			log.Printf("[DEBUG] SRV lookup failed for %s: %v", fullService, err)
		}
	}

	// Deduplicate all record types
	dedup := func(slice []string) []string {
		seen := make(map[string]bool)
		result := []string{}
		for _, item := range slice {
			if !seen[item] {
				seen[item] = true
				result = append(result, item)
			}
		}
		return result
	}

	results.ARecords = dedup(results.ARecords)
	results.AAAARecords = dedup(results.AAAARecords)
	results.CNAMERecords = dedup(results.CNAMERecords)
	results.MXRecords = dedup(results.MXRecords)
	results.TXTRecords = dedup(results.TXTRecords)
	results.NSRecords = dedup(results.NSRecords)
	results.PTRRecords = dedup(results.PTRRecords)
	results.SRVRecords = dedup(results.SRVRecords)

	log.Printf("[DEBUG] Final DNS records for %s:", hostname)
	log.Printf("[DEBUG]   A Records: %d", len(results.ARecords))
	log.Printf("[DEBUG]   AAAA Records: %d", len(results.AAAARecords))
	log.Printf("[DEBUG]   CNAME Records: %d", len(results.CNAMERecords))
	log.Printf("[DEBUG]   MX Records: %d", len(results.MXRecords))
	log.Printf("[DEBUG]   TXT Records: %d", len(results.TXTRecords))
	log.Printf("[DEBUG]   NS Records: %d", len(results.NSRecords))
	log.Printf("[DEBUG]   PTR Records: %d", len(results.PTRRecords))
	log.Printf("[DEBUG]   SRV Records: %d", len(results.SRVRecords))

	return results
}

func UpdateMetaDataScanStatus(scanID, status, result, stderr, command, execTime string) {
	log.Printf("[INFO] Updating Nuclei SSL scan status for %s to %s", scanID, status)
	query := `UPDATE metadata_scans SET status = $1, result = $2, stderr = $3, command = $4, execution_time = $5 WHERE scan_id = $6`
	_, err := dbPool.Exec(context.Background(), query, status, result, stderr, command, execTime, scanID)
	if err != nil {
		log.Printf("[ERROR] Failed to update Nuclei SSL scan status for %s: %v", scanID, err)
	} else {
		log.Printf("[INFO] Successfully updated Nuclei SSL scan status for %s", scanID)
	}
}

func GetMetaDataScanStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scanID := vars["scan_id"]

	var scan MetaDataStatus
	query := `SELECT * FROM metadata_scans WHERE scan_id = $1`
	err := dbPool.QueryRow(context.Background(), query, scanID).Scan(
		&scan.ID,
		&scan.ScanID,
		&scan.Domain,
		&scan.Status,
		&scan.Result,
		&scan.Error,
		&scan.StdOut,
		&scan.StdErr,
		&scan.Command,
		&scan.ExecTime,
		&scan.CreatedAt,
		&scan.ScopeTargetID,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "Scan not found", http.StatusNotFound)
		} else {
			log.Printf("[ERROR] Failed to get scan status: %v", err)
			http.Error(w, "Failed to get scan status", http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"id":              scan.ID,
		"scan_id":         scan.ScanID,
		"domain":          scan.Domain,
		"status":          scan.Status,
		"result":          nullStringToString(scan.Result),
		"error":           nullStringToString(scan.Error),
		"stdout":          nullStringToString(scan.StdOut),
		"stderr":          nullStringToString(scan.StdErr),
		"command":         nullStringToString(scan.Command),
		"execution_time":  nullStringToString(scan.ExecTime),
		"created_at":      scan.CreatedAt.Format(time.RFC3339),
		"scope_target_id": scan.ScopeTargetID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func GetMetaDataScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]

	query := `SELECT * FROM metadata_scans WHERE scope_target_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to get scans: %v", err)
		http.Error(w, "Failed to get scans", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []map[string]interface{}
	for rows.Next() {
		var scan MetaDataStatus
		err := rows.Scan(
			&scan.ID,
			&scan.ScanID,
			&scan.Domain,
			&scan.Status,
			&scan.Result,
			&scan.Error,
			&scan.StdOut,
			&scan.StdErr,
			&scan.Command,
			&scan.ExecTime,
			&scan.CreatedAt,
			&scan.ScopeTargetID,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			continue
		}

		scans = append(scans, map[string]interface{}{
			"id":              scan.ID,
			"scan_id":         scan.ScanID,
			"domain":          scan.Domain,
			"status":          scan.Status,
			"result":          nullStringToString(scan.Result),
			"error":           nullStringToString(scan.Error),
			"stdout":          nullStringToString(scan.StdOut),
			"stderr":          nullStringToString(scan.StdErr),
			"command":         nullStringToString(scan.Command),
			"execution_time":  nullStringToString(scan.ExecTime),
			"created_at":      scan.CreatedAt.Format(time.RFC3339),
			"scope_target_id": scan.ScopeTargetID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scans)
}

func GanitizeResponse(input []byte) string {
	// Remove null bytes
	sanitized := bytes.ReplaceAll(input, []byte{0}, []byte{})

	// Convert to string and handle any invalid UTF-8
	str := string(sanitized)

	// Replace any other problematic characters
	str = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1 // Drop the character
		}
		return r
	}, str)

	return str
}
