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
	"path/filepath"
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

type FfufResult struct {
	Input struct {
		FUZZ string `json:"FUZZ"`
	} `json:"input"`
	Position         int    `json:"position"`
	Status           int    `json:"status"`
	Length           int    `json:"length"`
	Words            int    `json:"words"`
	Lines            int    `json:"lines"`
	ContentType      string `json:"content-type"`
	RedirectLocation string `json:"redirectlocation"`
	Url              string `json:"url"`
	Duration         int64  `json:"duration"`
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
	log.Printf("[INFO] Starting Katana scan for scan ID: %s - Total URLs to scan: %d", scanID, len(urls))
	katanaResults := make(map[string][]string)
	completedKatana := 0
	for _, url := range urls {
		completedKatana++
		log.Printf("[INFO] Running Katana scan for URL: %s (%d/%d)", url, completedKatana, len(urls))
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		katanaCmd := exec.CommandContext(ctx,
			"docker", "exec", "ars0n-framework-v2-katana-1",
			"katana",
			"-u", url,
			"-jc",
			"-d", "2",
			"-j",
			"-v",
			"-timeout", "30",
			"-c", "15",
			"p", "15",
		)

		katanaCmd.WaitDelay = 30 * time.Second

		var stdout, stderr bytes.Buffer
		katanaCmd.Stdout = &stdout
		katanaCmd.Stderr = &stderr

		log.Printf("[DEBUG] Executing Katana command: %s", katanaCmd.String())
		if err := katanaCmd.Run(); err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				log.Printf("[WARN] Katana scan timed out for URL %s (%d/%d)", url, completedKatana, len(urls))
				continue
			}
			log.Printf("[WARN] Katana scan failed for URL %s (%d/%d): %v\nStderr: %s", url, completedKatana, len(urls), err, stderr.String())
			continue
		}
		log.Printf("[INFO] Completed Katana scan for URL: %s (%d/%d)", url, completedKatana, len(urls))

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
				`INSERT INTO target_urls (url, scope_target_id, roi_score) VALUES ($1, $2, 50)`,
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

	// Run ffuf scan for each URL
	log.Printf("[INFO] Starting ffuf scans for all URLs")
	for baseURL := range katanaResults {
		if err := ExecuteFfufScan(baseURL, scopeTargetID); err != nil {
			log.Printf("[ERROR] Failed to run ffuf scan for URL %s: %v", baseURL, err)
			continue
		}
	}

	// Update final scan status after all scans complete successfully
	UpdateMetaDataScanStatus(
		scanID,
		"success",
		string(output),
		stderr.String(),
		cmd.String(),
		time.Since(startTime).String(),
	)

	log.Printf("[INFO] All scans completed successfully for scan ID: %s", scanID)
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

	// Process each URL first to get response headers and body
	for _, urlStr := range urls {
		log.Printf("[DEBUG] Processing URL for headers: %s", urlStr)

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

		log.Printf("[DEBUG] Got response for URL %s - Status: %d, Number of headers: %d", urlStr, resp.StatusCode, len(resp.Header))

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
		log.Printf("[DEBUG] Processing headers for URL %s:", urlStr)
		for k, v := range resp.Header {
			log.Printf("[DEBUG] Header: %s = %v", k, v)
			// Convert header values to a consistent format
			if len(v) == 1 {
				headers[k] = v[0] // Store single value directly
				log.Printf("[DEBUG] Stored single value header: %s = %s", k, v[0])
			} else {
				headers[k] = v // Store multiple values as string slice
				log.Printf("[DEBUG] Stored multi-value header: %s = %v", k, v)
			}
		}

		// Convert headers to JSON before storing
		headersJSON, err := json.Marshal(headers)
		if err != nil {
			log.Printf("[ERROR] Failed to marshal headers for URL %s: %v", urlStr, err)
			continue
		}
		log.Printf("[DEBUG] Marshaled headers JSON (length: %d): %s", len(headersJSON), string(headersJSON))

		// Store response data in database
		_, err = dbPool.Exec(context.Background(),
			`UPDATE target_urls 
			 SET http_response = $1,
			     http_response_headers = $2::jsonb
			 WHERE url = $3 AND scope_target_id = $4`,
			sanitizedBody,
			string(headersJSON),
			urlStr,
			scopeTargetID)
		if err != nil {
			log.Printf("[ERROR] Failed to store response data for URL %s: %v", urlStr, err)
			continue
		}
		log.Printf("[INFO] Successfully stored response data for URL %s with %d headers", urlStr, len(headers))
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

	// Update findings and DNS records for each URL
	for urlStr, findings := range urlFindings {
		// Parse URL to get hostname for DNS lookups
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			log.Printf("[ERROR] Failed to parse URL %s: %v", urlStr, err)
			continue
		}

		// Perform DNS lookups
		dnsResults := PerformDNSLookups(parsedURL.Hostname())

		// Convert findings to proper JSON
		findingsJSON, err := json.Marshal(findings)
		if err != nil {
			log.Printf("[ERROR] Failed to marshal findings for URL %s: %v", urlStr, err)
			continue
		}

		// Update database with findings and DNS records
		_, err = dbPool.Exec(context.Background(),
			`UPDATE target_urls 
			 SET findings_json = $1::jsonb,
			     dns_a_records = $2,
			     dns_aaaa_records = $3,
			     dns_cname_records = $4,
			     dns_mx_records = $5,
			     dns_txt_records = $6,
			     dns_ns_records = $7,
			     dns_ptr_records = $8,
			     dns_srv_records = $9
			 WHERE url = $10 AND scope_target_id = $11`,
			findingsJSON,
			dnsResults.ARecords,
			dnsResults.AAAARecords,
			dnsResults.CNAMERecords,
			dnsResults.MXRecords,
			dnsResults.TXTRecords,
			dnsResults.NSRecords,
			dnsResults.PTRRecords,
			dnsResults.SRVRecords,
			urlStr,
			scopeTargetID)
		if err != nil {
			log.Printf("[ERROR] Failed to update findings and DNS records for URL %s: %v", urlStr, err)
			continue
		}
		log.Printf("[INFO] Updated findings and DNS records for URL %s", urlStr)
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

	// Create a custom resolver with shorter timeout
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 500 * time.Millisecond,
			}
			// Try Docker's internal DNS first
			conn, err := d.DialContext(ctx, network, "127.0.0.11:53")
			if err == nil {
				return conn, nil
			}
			// Fall back to Google DNS only
			return d.DialContext(ctx, network, "8.8.8.8:53")
		},
	}

	// A and AAAA records
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
	}

	// CNAME lookup
	log.Printf("[DEBUG] Looking up CNAME records for %s", hostname)
	if cname, err := resolver.LookupCNAME(ctx, hostname); err == nil && cname != "" {
		cname = strings.TrimSuffix(cname, ".")
		if cname != hostname {
			record := fmt.Sprintf("%s -> %s", hostname, cname)
			results.CNAMERecords = append(results.CNAMERecords, record)
			log.Printf("[DEBUG] Found CNAME record: %s", record)
		}
	}

	// MX lookup
	log.Printf("[DEBUG] Looking up MX records for %s", hostname)
	if mxRecords, err := resolver.LookupMX(ctx, hostname); err == nil {
		for _, mx := range mxRecords {
			record := fmt.Sprintf("Priority: %d | Server: %s", mx.Pref, strings.TrimSuffix(mx.Host, "."))
			results.MXRecords = append(results.MXRecords, record)
			log.Printf("[DEBUG] Found MX record: %s", record)
		}
	}

	// TXT lookup
	log.Printf("[DEBUG] Looking up TXT records for %s", hostname)
	if txtRecords, err := resolver.LookupTXT(ctx, hostname); err == nil {
		for _, txt := range txtRecords {
			results.TXTRecords = append(results.TXTRecords, txt)
			log.Printf("[DEBUG] Found TXT record: %s", txt)
		}
	}

	// NS lookup
	log.Printf("[DEBUG] Looking up NS records for %s", hostname)
	if nsRecords, err := resolver.LookupNS(ctx, hostname); err == nil {
		for _, ns := range nsRecords {
			record := strings.TrimSuffix(ns.Host, ".")
			results.NSRecords = append(results.NSRecords, record)
			log.Printf("[DEBUG] Found NS record: %s", record)
		}
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
		}
	}

	// Perform PTR lookups for both A and AAAA records
	for _, ip := range results.ARecords {
		lookupPTR(ip)
	}
	for _, ip := range results.AAAARecords {
		lookupPTR(ip)
	}

	// SRV lookup
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

func extractDNSRecord(line string) string {
	parts := strings.Fields(line)
	for _, part := range parts {
		if strings.Contains(part, ".") && !strings.Contains(part, "(") && !strings.Contains(part, ")") {
			return part
		}
	}
	return ""
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

func ExecuteFfufScan(url string, scopeTargetID string) error {
	log.Printf("[INFO] Starting ffuf scan for URL: %s", url)
	startTime := time.Now()

	// Parse the base URL to get the list of URLs to scan
	var urlsToScan []string
	err := dbPool.QueryRow(context.Background(),
		`SELECT katana_results FROM target_urls WHERE url = $1 AND scope_target_id = $2`,
		url, scopeTargetID).Scan(&urlsToScan)

	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to get katana results: %v", err)
	}

	// If no Katana results, just scan the base URL
	if len(urlsToScan) == 0 {
		urlsToScan = []string{url}
	}

	log.Printf("[INFO] Starting ffuf scans - Total URLs to scan: %d", len(urlsToScan))
	completedFfuf := 0

	// Create a temporary directory for output
	tempDir := filepath.Join("/tmp", fmt.Sprintf("ffuf-%s", uuid.New().String()))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Copy wordlist to container
	copyCmd := exec.Command(
		"docker", "exec",
		"ars0n-framework-v2-ffuf-1",
		"cp",
		"/wordlists/ffuf-wordlist-5000.txt",
		"/wordlist.txt",
	)

	var copyStderr bytes.Buffer
	copyCmd.Stderr = &copyStderr
	if err := copyCmd.Run(); err != nil {
		log.Printf("[ERROR] Failed to copy wordlist in container. Command: %s, Error: %v, Stderr: %s",
			copyCmd.String(), err, copyStderr.String())
		return fmt.Errorf("failed to copy wordlist in container: %v (stderr: %s)", err, copyStderr.String())
	}
	log.Printf("[DEBUG] Successfully copied wordlist in container")

	// Verify wordlist exists in container
	checkCmd := exec.Command(
		"docker", "exec",
		"ars0n-framework-v2-ffuf-1",
		"ls", "-l", "/wordlist.txt",
	)
	if out, err := checkCmd.CombinedOutput(); err != nil {
		log.Printf("[ERROR] Wordlist not found in container. Output: %s, Error: %v", string(out), err)
		return fmt.Errorf("wordlist not found in container: %v", err)
	} else {
		log.Printf("[DEBUG] Wordlist verified in container: %s", string(out))
	}

	for _, targetURL := range urlsToScan {
		completedFfuf++
		log.Printf("[INFO] Running ffuf scan for URL: %s (%d/%d)", targetURL, completedFfuf, len(urlsToScan))

		// Run ffuf scan
		fuzzyURL := fmt.Sprintf("%s/FUZZ", targetURL)
		cmd := exec.Command(
			"docker", "exec",
			"ars0n-framework-v2-ffuf-1",
			"ffuf",
			"-w", "/wordlist.txt",
			"-u", fuzzyURL,
			"-mc", "all",
			"-o", "/output.json",
			"-of", "json",
			"-ac",
			"-c",
			"-r",
			"-t", "50",
		)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		log.Printf("[DEBUG] Running ffuf command: %s", cmd.String())
		if err := cmd.Run(); err != nil {
			log.Printf("[ERROR] ffuf scan failed for URL %s (%d/%d): %v\nStderr: %s",
				targetURL, completedFfuf, len(urlsToScan), err, stderr.String())
			continue
		}
		log.Printf("[INFO] Completed ffuf scan for URL: %s (%d/%d)", targetURL, completedFfuf, len(urlsToScan))

		// Read and parse results
		resultBytes, err := os.ReadFile(filepath.Join(tempDir, "output.json"))
		if err != nil {
			log.Printf("[ERROR] Failed to read ffuf results file at %s: %v", filepath.Join(tempDir, "output.json"), err)
			return fmt.Errorf("failed to read ffuf results: %v", err)
		}
		log.Printf("[DEBUG] Read %d bytes from results file", len(resultBytes))

		var results struct {
			Results []FfufResult `json:"results"`
		}
		if err := json.Unmarshal(resultBytes, &results); err != nil {
			log.Printf("[ERROR] Failed to parse ffuf results JSON: %v\nContent: %s", err, string(resultBytes))
			return fmt.Errorf("failed to parse ffuf results: %v", err)
		}
		log.Printf("[DEBUG] Successfully parsed %d results from JSON", len(results.Results))

		// Filter and format results
		var endpoints []map[string]interface{}
		for _, result := range results.Results {
			endpoint := map[string]interface{}{
				"path":   result.Input.FUZZ,
				"status": result.Status,
				"size":   result.Length,
				"words":  result.Words,
				"lines":  result.Lines,
			}
			endpoints = append(endpoints, endpoint)
		}

		// Store results in database
		ffufResults := map[string]interface{}{
			"endpoints": endpoints,
		}
		ffufResultsJSON, err := json.Marshal(ffufResults)
		if err != nil {
			log.Printf("[ERROR] Failed to marshal ffuf results to JSON: %v", err)
			return fmt.Errorf("failed to marshal ffuf results: %v", err)
		}

		_, err = dbPool.Exec(context.Background(),
			`UPDATE target_urls 
			 SET ffuf_results = $1::jsonb 
			 WHERE url = $2 AND scope_target_id = $3`,
			string(ffufResultsJSON), targetURL, scopeTargetID)
		if err != nil {
			log.Printf("[ERROR] Failed to store ffuf results in database: %v", err)
			return fmt.Errorf("failed to store ffuf results: %v", err)
		}

		log.Printf("[INFO] Successfully completed ffuf scan for %s in %s. Found %d endpoints.",
			targetURL, time.Since(startTime), len(endpoints))
	}

	return nil
}
