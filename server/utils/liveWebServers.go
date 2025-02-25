package utils

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
)

// HttpxScanStatus represents the status of a httpx scan
type HttpxScanStatus struct {
	ID            string         `json:"id"`
	ScanID        string         `json:"scan_id"`
	Domain        string         `json:"domain"`
	Status        string         `json:"status"`
	Result        sql.NullString `json:"result"`
	Error         sql.NullString `json:"error"`
	StdOut        sql.NullString `json:"stdout"`
	StdErr        sql.NullString `json:"stderr"`
	Command       sql.NullString `json:"command"`
	ExecTime      sql.NullString `json:"execution_time"`
	CreatedAt     time.Time      `json:"created_at"`
	ScopeTargetID string         `json:"scope_target_id"`
}

// TargetURL represents a target URL in the database
type TargetURL struct {
	ID                  string         `json:"id"`
	URL                 string         `json:"url"`
	Screenshot          sql.NullString `json:"screenshot"`
	StatusCode          int            `json:"status_code"`
	Title               sql.NullString `json:"title"`
	WebServer           sql.NullString `json:"web_server"`
	Technologies        []string       `json:"technologies"`
	ContentLength       int            `json:"content_length"`
	NewlyDiscovered     bool           `json:"newly_discovered"`
	NoLongerLive        bool           `json:"no_longer_live"`
	ScopeTargetID       string         `json:"scope_target_id"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	HasDeprecatedTLS    bool           `json:"has_deprecated_tls"`
	HasExpiredSSL       bool           `json:"has_expired_ssl"`
	HasMismatchedSSL    bool           `json:"has_mismatched_ssl"`
	HasRevokedSSL       bool           `json:"has_revoked_ssl"`
	HasSelfSignedSSL    bool           `json:"has_self_signed_ssl"`
	HasUntrustedRootSSL bool           `json:"has_untrusted_root_ssl"`
	HasWildcardTLS      bool           `json:"has_wildcard_tls"`
	FindingsJSON        []byte         `json:"findings_json"`
	HTTPResponse        sql.NullString `json:"http_response"`
	HTTPResponseHeaders []byte         `json:"http_response_headers"`
	DNSARecords         []string       `json:"dns_a_records"`
	DNSAAAARecords      []string       `json:"dns_aaaa_records"`
	DNSCNAMERecords     []string       `json:"dns_cname_records"`
	DNSMXRecords        []string       `json:"dns_mx_records"`
	DNSTXTRecords       []string       `json:"dns_txt_records"`
	DNSNSRecords        []string       `json:"dns_ns_records"`
	DNSPTRRecords       []string       `json:"dns_ptr_records"`
	DNSSRVRecords       []string       `json:"dns_srv_records"`
}

// RunHttpxScan handles the HTTP request to start a new httpx scan
func RunHttpxScan(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		FQDN string `json:"fqdn" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.FQDN == "" {
		http.Error(w, "Invalid request body. `fqdn` is required.", http.StatusBadRequest)
		return
	}

	domain := payload.FQDN
	wildcardDomain := fmt.Sprintf("*.%s", domain)

	// Get the scope target ID
	query := `SELECT id FROM scope_targets WHERE type = 'Wildcard' AND scope_target = $1`
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(), query, wildcardDomain).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] No matching wildcard scope target found for domain %s", domain)
		http.Error(w, "No matching wildcard scope target found.", http.StatusBadRequest)
		return
	}

	scanID := uuid.New().String()
	insertQuery := `INSERT INTO httpx_scans (scan_id, domain, status, scope_target_id) VALUES ($1, $2, $3, $4)`
	_, err = dbPool.Exec(context.Background(), insertQuery, scanID, domain, "pending", scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to create scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}

	go ExecuteAndParseHttpxScan(scanID, domain)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

// ExecuteAndParseHttpxScan runs the httpx scan and processes its results
func ExecuteAndParseHttpxScan(scanID, domain string) {
	log.Printf("[INFO] Starting httpx scan for domain %s (scan ID: %s)", domain, scanID)
	startTime := time.Now()

	// Get scope target ID
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(),
		`SELECT scope_target_id FROM httpx_scans WHERE scan_id = $1`,
		scanID).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to get scope target ID: %v", err)
		UpdateHttpxScanStatus(scanID, "error", "", fmt.Sprintf("Failed to get scope target ID: %v", err), "", time.Since(startTime).String())
		return
	}

	// Get consolidated subdomains
	rows, err := dbPool.Query(context.Background(),
		`SELECT subdomain FROM consolidated_subdomains WHERE scope_target_id = $1`,
		scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to get consolidated subdomains: %v", err)
		UpdateHttpxScanStatus(scanID, "error", "", fmt.Sprintf("Failed to get consolidated subdomains: %v", err), "", time.Since(startTime).String())
		return
	}
	defer rows.Close()

	var domainsToScan []string
	for rows.Next() {
		var subdomain string
		if err := rows.Scan(&subdomain); err != nil {
			log.Printf("[ERROR] Failed to scan subdomain row: %v", err)
			continue
		}
		domainsToScan = append(domainsToScan, subdomain)
	}

	// If no consolidated subdomains found, use the base domain
	if len(domainsToScan) == 0 {
		log.Printf("[INFO] No consolidated subdomains found, using base domain: %s", domain)
		domainsToScan = []string{domain}
	}

	// Create temporary directory for domains file
	tempDir := "/tmp/httpx-temp"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Printf("[ERROR] Failed to create temp directory: %v", err)
		UpdateHttpxScanStatus(scanID, "error", "", fmt.Sprintf("Failed to create temp directory: %v", err), "", time.Since(startTime).String())
		return
	}
	defer os.RemoveAll(tempDir)

	// Write domains to file
	domainsFile := filepath.Join(tempDir, "domains.txt")
	if err := os.WriteFile(domainsFile, []byte(strings.Join(domainsToScan, "\n")), 0644); err != nil {
		log.Printf("[ERROR] Failed to write domains file: %v", err)
		UpdateHttpxScanStatus(scanID, "error", "", fmt.Sprintf("Failed to write domains file: %v", err), "", time.Since(startTime).String())
		return
	}

	// Run httpx scan
	cmd := exec.Command(
		"docker", "exec",
		"ars0n-framework-v2-httpx-1",
		"httpx",
		"-l", "/tmp/domains.txt",
		"-json",
		"-status-code",
		"-title",
		"-tech-detect",
		"-server",
		"-content-length",
		"-no-color",
		"-timeout", "10",
		"-retries", "2",
		"-mc", "100,101,200,201,202,203,204,205,206,207,208,226,300,301,302,303,304,305,307,308,400,401,402,403,404,405,406,407,408,409,410,411,412,413,414,415,416,417,418,421,422,423,424,426,428,429,431,451,500,501,502,503,504,505,506,507,508,510,511",
		"-o", "/tmp/httpx-output.json",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	execTime := time.Since(startTime).String()

	if err != nil {
		log.Printf("[ERROR] httpx scan failed for %s: %v", domain, err)
		UpdateHttpxScanStatus(scanID, "error", "", stderr.String(), cmd.String(), execTime)
		return
	}

	// Process results
	result := stdout.String()
	if result == "" {
		UpdateHttpxScanStatus(scanID, "completed", "", "No results found", cmd.String(), execTime)
		return
	}

	// Process results and update target URLs
	var liveURLs []string
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var httpxResult map[string]interface{}
		if err := json.Unmarshal([]byte(line), &httpxResult); err != nil {
			continue
		}

		if url, ok := httpxResult["url"].(string); ok {
			liveURLs = append(liveURLs, url)
			if err := UpdateTargetURLFromHttpx(scopeTargetID, httpxResult); err != nil {
				log.Printf("[WARN] Failed to update target URL for %s: %v", url, err)
			}
		}
	}

	// Mark URLs not found in this scan as no longer live
	if err := MarkOldTargetURLsAsNoLongerLive(scopeTargetID, liveURLs); err != nil {
		log.Printf("[WARN] Failed to mark old target URLs as no longer live: %v", err)
	}

	UpdateHttpxScanStatus(scanID, "success", result, stderr.String(), cmd.String(), execTime)
}

// UpdateHttpxScanStatus updates the status of a httpx scan in the database
func UpdateHttpxScanStatus(scanID, status, result, stderr, command, execTime string) {
	log.Printf("[INFO] Updating httpx scan status for %s to %s", scanID, status)
	query := `UPDATE httpx_scans SET status = $1, result = $2, stderr = $3, command = $4, execution_time = $5 WHERE scan_id = $6`
	_, err := dbPool.Exec(context.Background(), query, status, result, stderr, command, execTime, scanID)
	if err != nil {
		log.Printf("[ERROR] Failed to update httpx scan status for %s: %v", scanID, err)
	}
}

// GetHttpxScanStatus retrieves the status of a httpx scan
func GetHttpxScanStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scanID := vars["scanID"]
	if scanID == "" {
		http.Error(w, "Scan ID is required", http.StatusBadRequest)
		return
	}

	var scan HttpxScanStatus
	query := `SELECT id, scan_id, domain, status, result, error, stdout, stderr, command, execution_time, created_at FROM httpx_scans WHERE scan_id = $1`
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
	)
	if err != nil {
		http.Error(w, "Scan not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(scan)
}

// GetHttpxScansForScopeTarget retrieves all httpx scans for a scope target
func GetHttpxScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]
	if scopeTargetID == "" {
		http.Error(w, "Scope target ID is required", http.StatusBadRequest)
		return
	}

	query := `SELECT * FROM httpx_scans WHERE scope_target_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		http.Error(w, "Failed to get scans", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []HttpxScanStatus
	for rows.Next() {
		var scan HttpxScanStatus
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
		)
		if err != nil {
			continue
		}
		scans = append(scans, scan)
	}

	json.NewEncoder(w).Encode(scans)
}

// ConsolidateSubdomains consolidates subdomains from various sources
func ConsolidateSubdomains(scopeTargetID string) ([]string, error) {
	log.Printf("[INFO] Starting consolidation for scope target ID: %s", scopeTargetID)

	var baseDomain string
	err := dbPool.QueryRow(context.Background(), `
		SELECT TRIM(LEADING '*.' FROM scope_target) 
		FROM scope_targets 
		WHERE id = $1`, scopeTargetID).Scan(&baseDomain)
	if err != nil {
		return nil, fmt.Errorf("failed to get base domain: %v", err)
	}
	log.Printf("[INFO] Base domain for consolidation: %s", baseDomain)

	uniqueSubdomains := make(map[string]bool)
	toolResults := make(map[string]int)

	// Special handling for Amass - get from subdomains table
	amassQuery := `
		SELECT s.subdomain 
		FROM subdomains s 
		JOIN amass_scans a ON s.scan_id = a.scan_id 
		WHERE a.scope_target_id = $1 
			AND a.status = 'success'
			AND a.created_at = (
				SELECT MAX(created_at) 
				FROM amass_scans 
				WHERE scope_target_id = $1 
					AND status = 'success'
			)`

	log.Printf("[DEBUG] Processing results from amass using subdomains table")
	amassRows, err := dbPool.Query(context.Background(), amassQuery, scopeTargetID)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("[ERROR] Failed to get Amass subdomains: %v", err)
	} else {
		count := 0
		for amassRows.Next() {
			var subdomain string
			if err := amassRows.Scan(&subdomain); err != nil {
				log.Printf("[ERROR] Failed to scan Amass subdomain: %v", err)
				continue
			}
			if strings.HasSuffix(subdomain, baseDomain) {
				if !uniqueSubdomains[subdomain] {
					log.Printf("[DEBUG] Found new subdomain from amass: %s", subdomain)
					count++
				}
				uniqueSubdomains[subdomain] = true
			}
		}
		amassRows.Close()
		toolResults["amass"] = count
		log.Printf("[INFO] Found %d new unique subdomains from amass", count)
	}

	// Handle other tools
	queries := []struct {
		query string
		table string
	}{
		{
			query: `
				SELECT result 
				FROM sublist3r_scans 
				WHERE scope_target_id = $1 
					AND status = 'completed' 
					AND result IS NOT NULL 
					AND result != '' 
				ORDER BY created_at DESC 
				LIMIT 1`,
			table: "sublist3r",
		},
		{
			query: `
				SELECT result 
				FROM assetfinder_scans 
				WHERE scope_target_id = $1 
					AND status = 'success' 
					AND result IS NOT NULL 
					AND result != '' 
				ORDER BY created_at DESC 
				LIMIT 1`,
			table: "assetfinder",
		},
		{
			query: `
				SELECT result 
				FROM ctl_scans 
				WHERE scope_target_id = $1 
					AND status = 'success' 
					AND result IS NOT NULL 
					AND result != '' 
				ORDER BY created_at DESC 
				LIMIT 1`,
			table: "ctl",
		},
		{
			query: `
				SELECT result 
				FROM subfinder_scans 
				WHERE scope_target_id = $1 
					AND status = 'success' 
					AND result IS NOT NULL 
					AND result != '' 
				ORDER BY created_at DESC 
				LIMIT 1`,
			table: "subfinder",
		},
		{
			query: `
				SELECT result 
				FROM gau_scans 
				WHERE scope_target_id = $1 
					AND status = 'success' 
					AND result IS NOT NULL 
					AND result != '' 
				ORDER BY created_at DESC 
				LIMIT 1`,
			table: "gau",
		},
		{
			query: `
				SELECT result 
				FROM shuffledns_scans 
				WHERE scope_target_id = $1 
					AND status = 'success' 
					AND result IS NOT NULL 
					AND result != '' 
				ORDER BY created_at DESC 
				LIMIT 1`,
			table: "shuffledns",
		},
		{
			query: `
				SELECT result 
				FROM shufflednscustom_scans 
				WHERE scope_target_id = $1 
					AND status = 'success' 
					AND result IS NOT NULL 
					AND result != '' 
				ORDER BY created_at DESC 
				LIMIT 1`,
			table: "shuffledns_custom",
		},
		{
			query: `
				SELECT result 
				FROM gospider_scans 
				WHERE scope_target_id = $1 
					AND status = 'success' 
					AND result IS NOT NULL 
					AND result != '' 
				ORDER BY created_at DESC 
				LIMIT 1`,
			table: "gospider",
		},
		{
			query: `
				SELECT result 
				FROM subdomainizer_scans 
				WHERE scope_target_id = $1 
					AND status = 'success' 
					AND result IS NOT NULL 
					AND result != '' 
				ORDER BY created_at DESC 
				LIMIT 1`,
			table: "subdomainizer",
		},
	}

	for _, q := range queries {
		log.Printf("[DEBUG] Processing results from %s", q.table)
		var result sql.NullString
		err := dbPool.QueryRow(context.Background(), q.query, scopeTargetID).Scan(&result)
		if err != nil {
			if err == pgx.ErrNoRows {
				log.Printf("[DEBUG] No results found for %s", q.table)
				continue
			}
			log.Printf("[ERROR] Failed to get results from %s: %v", q.table, err)
			continue
		}

		if !result.Valid || result.String == "" {
			log.Printf("[DEBUG] No valid results found for %s", q.table)
			continue
		}

		count := 0
		if q.table == "gau" {
			lines := strings.Split(result.String, "\n")
			log.Printf("[DEBUG] Processing %d lines from GAU", len(lines))
			for i, line := range lines {
				if line == "" {
					continue
				}
				var gauResult struct {
					URL string `json:"url"`
				}
				if err := json.Unmarshal([]byte(line), &gauResult); err != nil {
					log.Printf("[ERROR] Failed to parse GAU result line %d: %v", i, err)
					continue
				}
				if gauResult.URL == "" {
					continue
				}
				parsedURL, err := url.Parse(gauResult.URL)
				if err != nil {
					log.Printf("[ERROR] Failed to parse URL %s: %v", gauResult.URL, err)
					continue
				}
				hostname := parsedURL.Hostname()
				if strings.HasSuffix(hostname, baseDomain) {
					if !uniqueSubdomains[hostname] {
						log.Printf("[DEBUG] Found new subdomain from GAU: %s", hostname)
						count++
					}
					uniqueSubdomains[hostname] = true
				}
			}
		} else {
			lines := strings.Split(result.String, "\n")
			log.Printf("[DEBUG] Processing %d lines from %s", len(lines), q.table)
			for _, line := range lines {
				subdomain := strings.TrimSpace(line)
				if subdomain == "" {
					continue
				}
				if strings.HasSuffix(subdomain, baseDomain) {
					if !uniqueSubdomains[subdomain] {
						log.Printf("[DEBUG] Found new subdomain from %s: %s", q.table, subdomain)
						count++
					}
					uniqueSubdomains[subdomain] = true
				}
			}
		}
		toolResults[q.table] = count
		log.Printf("[INFO] Found %d new unique subdomains from %s", count, q.table)
	}

	var consolidatedSubdomains []string
	for subdomain := range uniqueSubdomains {
		consolidatedSubdomains = append(consolidatedSubdomains, subdomain)
	}
	sort.Strings(consolidatedSubdomains)

	log.Printf("[INFO] Tool contribution breakdown:")
	for tool, count := range toolResults {
		log.Printf("- %s: %d subdomains", tool, count)
	}
	log.Printf("[INFO] Total unique subdomains found: %d", len(consolidatedSubdomains))

	// Update database
	tx, err := dbPool.Begin(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(context.Background())

	_, err = tx.Exec(context.Background(), `DELETE FROM consolidated_subdomains WHERE scope_target_id = $1`, scopeTargetID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete old consolidated subdomains: %v", err)
	}

	for _, subdomain := range consolidatedSubdomains {
		_, err = tx.Exec(context.Background(),
			`INSERT INTO consolidated_subdomains (scope_target_id, subdomain) VALUES ($1, $2)
			ON CONFLICT (scope_target_id, subdomain) DO NOTHING`,
			scopeTargetID, subdomain)
		if err != nil {
			return nil, fmt.Errorf("failed to insert consolidated subdomain: %v", err)
		}
	}

	if err := tx.Commit(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	return consolidatedSubdomains, nil
}

// HandleConsolidateSubdomains handles the HTTP request to consolidate subdomains
func HandleConsolidateSubdomains(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]
	if scopeTargetID == "" {
		http.Error(w, "Scope target ID is required", http.StatusBadRequest)
		return
	}

	consolidatedSubdomains, err := ConsolidateSubdomains(scopeTargetID)
	if err != nil {
		http.Error(w, "Failed to consolidate subdomains", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":      len(consolidatedSubdomains),
		"subdomains": consolidatedSubdomains,
	})
}

// GetConsolidatedSubdomains retrieves consolidated subdomains for a scope target
func GetConsolidatedSubdomains(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]
	if scopeTargetID == "" {
		http.Error(w, "Scope target ID is required", http.StatusBadRequest)
		return
	}

	query := `SELECT subdomain FROM consolidated_subdomains WHERE scope_target_id = $1 ORDER BY subdomain ASC`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		http.Error(w, "Failed to get consolidated subdomains", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var subdomains []string
	for rows.Next() {
		var subdomain string
		if err := rows.Scan(&subdomain); err != nil {
			continue
		}
		subdomains = append(subdomains, subdomain)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":      len(subdomains),
		"subdomains": subdomains,
	})
}

// UpdateTargetURLFromHttpx updates target URL information from httpx scan results
func UpdateTargetURLFromHttpx(scopeTargetID string, httpxData map[string]interface{}) error {
	url, ok := httpxData["url"].(string)
	if !ok || url == "" {
		return fmt.Errorf("invalid or missing URL in httpx data")
	}

	url = NormalizeURL(url)
	var technologies []string
	if techInterface, ok := httpxData["tech"].([]interface{}); ok {
		for _, tech := range techInterface {
			if techStr, ok := tech.(string); ok {
				technologies = append(technologies, techStr)
			}
		}
	}

	// Check if target URL exists and update accordingly
	var existingID string
	var isNoLongerLive bool
	err := dbPool.QueryRow(context.Background(),
		`SELECT id, no_longer_live FROM target_urls WHERE url = $1`,
		url).Scan(&existingID, &isNoLongerLive)

	if err == pgx.ErrNoRows {
		// Insert new target URL
		_, err = dbPool.Exec(context.Background(),
			`INSERT INTO target_urls (
				url, status_code, title, web_server, technologies, 
				content_length, scope_target_id, newly_discovered, no_longer_live,
				findings_json
			) VALUES ($1, $2, $3, $4, $5::text[], $6, $7, true, false, $8::jsonb)`,
			url,
			httpxData["status_code"],
			httpxData["title"],
			httpxData["webserver"],
			technologies,
			httpxData["content_length"],
			scopeTargetID,
			"[]")
	} else if err == nil {
		// Update existing target URL
		updateQuery := `UPDATE target_urls SET 
			status_code = $1,
			title = $2,
			web_server = $3,
			technologies = $4::text[],
			content_length = $5,
			no_longer_live = false,
			newly_discovered = $6,
			updated_at = NOW(),
			findings_json = $7::jsonb
		WHERE id = $8`

		_, err = dbPool.Exec(context.Background(),
			updateQuery,
			httpxData["status_code"],
			httpxData["title"],
			httpxData["webserver"],
			technologies,
			httpxData["content_length"],
			isNoLongerLive, // If previously marked as no longer live, mark as newly discovered
			"[]",
			existingID)
	}

	return err
}

// MarkOldTargetURLsAsNoLongerLive marks URLs not found in recent scans as no longer live
func MarkOldTargetURLsAsNoLongerLive(scopeTargetID string, liveURLs []string) error {
	_, err := dbPool.Exec(context.Background(),
		`UPDATE target_urls SET 
			no_longer_live = true,
			newly_discovered = false,
			updated_at = NOW()
		WHERE scope_target_id = $1 
		AND url NOT IN (SELECT unnest($2::text[]))`,
		scopeTargetID, liveURLs)

	return err
}

// GetTargetURLsForScopeTarget retrieves all target URLs for a scope target
func GetTargetURLsForScopeTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]
	if scopeTargetID == "" {
		http.Error(w, "Scope target ID is required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT id, url, screenshot, status_code, title, web_server, 
			   technologies, content_length, newly_discovered, no_longer_live,
			   scope_target_id, created_at, updated_at,
			   has_deprecated_tls, has_expired_ssl, has_mismatched_ssl,
			   has_revoked_ssl, has_self_signed_ssl, has_untrusted_root_ssl,
			   has_wildcard_tls, findings_json, http_response, http_response_headers
		FROM target_urls 
		WHERE scope_target_id = $1 
		ORDER BY created_at DESC`

	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		http.Error(w, "Failed to get target URLs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var targetURLs []TargetURL
	for rows.Next() {
		var targetURL TargetURL
		err := rows.Scan(
			&targetURL.ID,
			&targetURL.URL,
			&targetURL.Screenshot,
			&targetURL.StatusCode,
			&targetURL.Title,
			&targetURL.WebServer,
			&targetURL.Technologies,
			&targetURL.ContentLength,
			&targetURL.NewlyDiscovered,
			&targetURL.NoLongerLive,
			&targetURL.ScopeTargetID,
			&targetURL.CreatedAt,
			&targetURL.UpdatedAt,
			&targetURL.HasDeprecatedTLS,
			&targetURL.HasExpiredSSL,
			&targetURL.HasMismatchedSSL,
			&targetURL.HasRevokedSSL,
			&targetURL.HasSelfSignedSSL,
			&targetURL.HasUntrustedRootSSL,
			&targetURL.HasWildcardTLS,
			&targetURL.FindingsJSON,
			&targetURL.HTTPResponse,
			&targetURL.HTTPResponseHeaders,
		)
		if err != nil {
			continue
		}
		targetURLs = append(targetURLs, targetURL)
	}

	json.NewEncoder(w).Encode(targetURLs)
}
