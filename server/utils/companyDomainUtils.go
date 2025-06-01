package utils

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
)

type CTLCompanyScanStatus struct {
	ID                string         `json:"id"`
	ScanID            string         `json:"scan_id"`
	CompanyName       string         `json:"company_name"`
	Status            string         `json:"status"`
	Result            sql.NullString `json:"result,omitempty"`
	Error             sql.NullString `json:"error,omitempty"`
	StdOut            sql.NullString `json:"stdout,omitempty"`
	StdErr            sql.NullString `json:"stderr,omitempty"`
	Command           sql.NullString `json:"command,omitempty"`
	ExecTime          sql.NullString `json:"execution_time,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	ScopeTargetID     string         `json:"scope_target_id"`
	AutoScanSessionID sql.NullString `json:"auto_scan_session_id"`
}

func RunCTLCompanyScan(w http.ResponseWriter, r *http.Request) {
	log.Printf("[CTL-COMPANY] [INFO] Starting CTL Company scan request handling")
	var payload struct {
		CompanyName       string  `json:"company_name" binding:"required"`
		AutoScanSessionID *string `json:"auto_scan_session_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.CompanyName == "" {
		log.Printf("[CTL-COMPANY] [ERROR] Invalid request body: %v", err)
		http.Error(w, "Invalid request body. `company_name` is required.", http.StatusBadRequest)
		return
	}

	companyName := payload.CompanyName
	log.Printf("[CTL-COMPANY] [INFO] Processing CTL Company scan for company: %s", companyName)

	query := `SELECT id FROM scope_targets WHERE type = 'Company' AND scope_target = $1`
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(), query, companyName).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[CTL-COMPANY] [ERROR] No matching company scope target found for company %s: %v", companyName, err)
		http.Error(w, "No matching company scope target found.", http.StatusBadRequest)
		return
	}
	log.Printf("[CTL-COMPANY] [INFO] Found scope target ID: %s for company: %s", scopeTargetID, companyName)

	scanID := uuid.New().String()
	log.Printf("[CTL-COMPANY] [INFO] Generated new scan ID: %s", scanID)

	createTableQuery := `
		CREATE TABLE IF NOT EXISTS ctl_company_scans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL UNIQUE,
			company_name TEXT NOT NULL,
			status VARCHAR(50) NOT NULL,
			result TEXT,
			error TEXT,
			stdout TEXT,
			stderr TEXT,
			command TEXT,
			execution_time TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			scope_target_id UUID REFERENCES scope_targets(id) ON DELETE CASCADE,
			auto_scan_session_id UUID REFERENCES auto_scan_sessions(id) ON DELETE SET NULL
		);`
	_, err = dbPool.Exec(context.Background(), createTableQuery)
	if err != nil {
		log.Printf("[CTL-COMPANY] [ERROR] Failed to create ctl_company_scans table: %v", err)
		http.Error(w, "Failed to create scan table.", http.StatusInternalServerError)
		return
	}
	log.Printf("[CTL-COMPANY] [INFO] Ensured ctl_company_scans table exists")

	var insertQuery string
	var args []interface{}
	if payload.AutoScanSessionID != nil && *payload.AutoScanSessionID != "" {
		insertQuery = `INSERT INTO ctl_company_scans (scan_id, company_name, status, scope_target_id, auto_scan_session_id) VALUES ($1, $2, $3, $4, $5)`
		args = []interface{}{scanID, companyName, "pending", scopeTargetID, *payload.AutoScanSessionID}
	} else {
		insertQuery = `INSERT INTO ctl_company_scans (scan_id, company_name, status, scope_target_id) VALUES ($1, $2, $3, $4)`
		args = []interface{}{scanID, companyName, "pending", scopeTargetID}
	}
	_, err = dbPool.Exec(context.Background(), insertQuery, args...)
	if err != nil {
		log.Printf("[CTL-COMPANY] [ERROR] Failed to create scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}
	log.Printf("[CTL-COMPANY] [INFO] Successfully created CTL Company scan record in database")

	go ExecuteAndParseCTLCompanyScan(scanID, companyName)

	log.Printf("[CTL-COMPANY] [INFO] CTL Company scan initiated successfully, returning scan ID: %s", scanID)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func ExecuteAndParseCTLCompanyScan(scanID, companyName string) {
	log.Printf("[CTL-COMPANY] [INFO] Starting CTL Company scan execution for company %s (scan ID: %s)", companyName, scanID)
	startTime := time.Now()

	url := fmt.Sprintf("https://crt.sh/?O=%s&output=json", companyName)
	log.Printf("[CTL-COMPANY] [DEBUG] Requesting URL: %s", url)

	client := &http.Client{Timeout: 60 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		log.Printf("[CTL-COMPANY] [ERROR] Failed to make request to crt.sh: %v", err)
		UpdateCTLCompanyScanStatus(scanID, "error", "", fmt.Sprintf("Failed to make request to crt.sh: %v", err), "", time.Since(startTime).String())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[CTL-COMPANY] [ERROR] crt.sh returned non-200 status code: %d", resp.StatusCode)
		UpdateCTLCompanyScanStatus(scanID, "error", "", fmt.Sprintf("crt.sh returned status code: %d", resp.StatusCode), "", time.Since(startTime).String())
		return
	}

	var results []struct {
		CommonName string `json:"common_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		log.Printf("[CTL-COMPANY] [ERROR] Failed to decode crt.sh response: %v", err)
		UpdateCTLCompanyScanStatus(scanID, "error", "", fmt.Sprintf("Failed to decode crt.sh response: %v", err), "", time.Since(startTime).String())
		return
	}

	log.Printf("[CTL-COMPANY] [DEBUG] Received %d certificate entries from crt.sh", len(results))

	uniqueRootDomains := make(map[string]bool)
	for _, result := range results {
		domain := strings.ToLower(strings.TrimPrefix(result.CommonName, "*."))
		log.Printf("[CTL-COMPANY] [DEBUG] Processing domain: %s", domain)

		if domain == "" {
			continue
		}

		// Skip entries that don't look like domains (contain spaces, commas, or "inc")
		if strings.Contains(domain, " ") || strings.Contains(domain, ",") || strings.Contains(domain, "inc") {
			log.Printf("[CTL-COMPANY] [DEBUG] Skipping non-domain entry: %s", domain)
			continue
		}

		// Only process entries that contain at least one dot and look like domains
		parts := strings.Split(domain, ".")
		if len(parts) >= 2 {
			// Validate that it's a proper domain format
			lastPart := parts[len(parts)-1]
			if len(lastPart) >= 2 && len(lastPart) <= 6 { // Valid TLD length
				rootDomain := parts[len(parts)-2] + "." + parts[len(parts)-1]
				log.Printf("[CTL-COMPANY] [DEBUG] Extracted root domain: %s from %s", rootDomain, domain)
				uniqueRootDomains[rootDomain] = true
			} else {
				log.Printf("[CTL-COMPANY] [DEBUG] Skipping invalid TLD: %s", domain)
			}
		} else {
			log.Printf("[CTL-COMPANY] [DEBUG] Skipping entry without valid domain structure: %s", domain)
		}
	}

	var rootDomains []string
	for rootDomain := range uniqueRootDomains {
		rootDomains = append(rootDomains, rootDomain)
	}
	sort.Strings(rootDomains)

	result := strings.Join(rootDomains, "\n")
	log.Printf("[CTL-COMPANY] [DEBUG] Final processed result contains %d unique root domains", len(rootDomains))
	log.Printf("[CTL-COMPANY] [DEBUG] Root domains found: %v", rootDomains)

	UpdateCTLCompanyScanStatus(scanID, "success", result, "", fmt.Sprintf("GET %s", url), time.Since(startTime).String())
	log.Printf("[CTL-COMPANY] [INFO] CTL Company scan completed and results stored successfully for company %s", companyName)
}

func UpdateCTLCompanyScanStatus(scanID, status, result, stderr, command, execTime string) {
	log.Printf("[CTL-COMPANY] [INFO] Updating CTL Company scan status for scan ID %s to %s", scanID, status)
	query := `UPDATE ctl_company_scans SET status = $1, result = $2, error = $3, command = $4, execution_time = $5 WHERE scan_id = $6`

	_, err := dbPool.Exec(context.Background(), query, status, result, stderr, command, execTime, scanID)
	if err != nil {
		log.Printf("[CTL-COMPANY] [ERROR] Failed to update CTL Company scan status for scan ID %s: %v", scanID, err)
		log.Printf("[CTL-COMPANY] [ERROR] Update attempted with: status=%s, result_length=%d, error_length=%d, command_length=%d, execTime=%s",
			status, len(result), len(stderr), len(command), execTime)
	} else {
		log.Printf("[CTL-COMPANY] [INFO] Successfully updated CTL Company scan status to %s for scan ID %s", status, scanID)
	}
}

func GetCTLCompanyScanStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scanID := vars["scan_id"]
	log.Printf("[CTL-COMPANY] [INFO] Retrieving CTL Company scan status for scan ID: %s", scanID)

	var scan CTLCompanyScanStatus
	query := `SELECT id, scan_id, company_name, status, result, error, stdout, stderr, command, execution_time, created_at, scope_target_id, auto_scan_session_id FROM ctl_company_scans WHERE scan_id = $1`
	err := dbPool.QueryRow(context.Background(), query, scanID).Scan(
		&scan.ID,
		&scan.ScanID,
		&scan.CompanyName,
		&scan.Status,
		&scan.Result,
		&scan.Error,
		&scan.StdOut,
		&scan.StdErr,
		&scan.Command,
		&scan.ExecTime,
		&scan.CreatedAt,
		&scan.ScopeTargetID,
		&scan.AutoScanSessionID,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			log.Printf("[CTL-COMPANY] [ERROR] CTL Company scan not found for scan ID: %s", scanID)
			http.Error(w, "Scan not found", http.StatusNotFound)
		} else {
			log.Printf("[CTL-COMPANY] [ERROR] Failed to get CTL Company scan status for scan ID %s: %v", scanID, err)
			http.Error(w, "Failed to get scan status", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("[CTL-COMPANY] [INFO] Successfully retrieved CTL Company scan status for scan ID %s: %s", scanID, scan.Status)
	if scan.Result.Valid {
		log.Printf("[CTL-COMPANY] [DEBUG] Scan has valid results of length: %d bytes", len(scan.Result.String))
	}

	response := map[string]interface{}{
		"id":                   scan.ID,
		"scan_id":              scan.ScanID,
		"company_name":         scan.CompanyName,
		"status":               scan.Status,
		"result":               nullStringToString(scan.Result),
		"error":                nullStringToString(scan.Error),
		"stdout":               nullStringToString(scan.StdOut),
		"stderr":               nullStringToString(scan.StdErr),
		"command":              nullStringToString(scan.Command),
		"execution_time":       nullStringToString(scan.ExecTime),
		"created_at":           scan.CreatedAt.Format(time.RFC3339),
		"scope_target_id":      scan.ScopeTargetID,
		"auto_scan_session_id": nullStringToString(scan.AutoScanSessionID),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[CTL-COMPANY] [ERROR] Failed to encode CTL Company scan response: %v", err)
	} else {
		log.Printf("[CTL-COMPANY] [INFO] Successfully sent CTL Company scan status response")
	}
}

func GetCTLCompanyScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]
	log.Printf("[CTL-COMPANY] [INFO] Fetching CTL Company scans for scope target ID: %s", scopeTargetID)

	if scopeTargetID == "" {
		log.Printf("[CTL-COMPANY] [ERROR] No scope target ID provided")
		http.Error(w, "No scope target ID provided", http.StatusBadRequest)
		return
	}

	// Ensure the table exists before trying to query it
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS ctl_company_scans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL UNIQUE,
			company_name TEXT NOT NULL,
			status VARCHAR(50) NOT NULL,
			result TEXT,
			error TEXT,
			stdout TEXT,
			stderr TEXT,
			command TEXT,
			execution_time TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			scope_target_id UUID REFERENCES scope_targets(id) ON DELETE CASCADE,
			auto_scan_session_id UUID REFERENCES auto_scan_sessions(id) ON DELETE SET NULL
		);`
	_, err := dbPool.Exec(context.Background(), createTableQuery)
	if err != nil {
		log.Printf("[CTL-COMPANY] [ERROR] Failed to create ctl_company_scans table: %v", err)
		http.Error(w, "Failed to create scan table.", http.StatusInternalServerError)
		return
	}

	query := `SELECT id, scan_id, company_name, status, result, error, stdout, stderr, command, execution_time, created_at, scope_target_id, auto_scan_session_id FROM ctl_company_scans WHERE scope_target_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[CTL-COMPANY] [ERROR] Failed to get scans: %v", err)
		http.Error(w, "Failed to get scans", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []map[string]interface{}
	for rows.Next() {
		var scan CTLCompanyScanStatus
		err := rows.Scan(
			&scan.ID,
			&scan.ScanID,
			&scan.CompanyName,
			&scan.Status,
			&scan.Result,
			&scan.Error,
			&scan.StdOut,
			&scan.StdErr,
			&scan.Command,
			&scan.ExecTime,
			&scan.CreatedAt,
			&scan.ScopeTargetID,
			&scan.AutoScanSessionID,
		)
		if err != nil {
			log.Printf("[CTL-COMPANY] [ERROR] Error scanning CTL Company scan row: %v", err)
			continue
		}

		scanMap := map[string]interface{}{
			"id":                   scan.ID,
			"scan_id":              scan.ScanID,
			"company_name":         scan.CompanyName,
			"status":               scan.Status,
			"result":               nullStringToString(scan.Result),
			"error":                nullStringToString(scan.Error),
			"stdout":               nullStringToString(scan.StdOut),
			"stderr":               nullStringToString(scan.StdErr),
			"command":              nullStringToString(scan.Command),
			"execution_time":       nullStringToString(scan.ExecTime),
			"created_at":           scan.CreatedAt.Format(time.RFC3339),
			"scope_target_id":      scan.ScopeTargetID,
			"auto_scan_session_id": nullStringToString(scan.AutoScanSessionID),
		}
		scans = append(scans, scanMap)
	}

	log.Printf("[CTL-COMPANY] [INFO] Successfully retrieved %d CTL Company scans for scope target %s", len(scans), scopeTargetID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(scans); err != nil {
		log.Printf("[CTL-COMPANY] [ERROR] Failed to encode scans response: %v", err)
	} else {
		log.Printf("[CTL-COMPANY] [INFO] Successfully sent CTL Company scans response")
	}
}
