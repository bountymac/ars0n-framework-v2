package utils

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
)

type MetabigorCompanyScanStatus struct {
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

func RunMetabigorCompanyScan(w http.ResponseWriter, r *http.Request) {
	log.Printf("[METABIGOR-COMPANY] [INFO] Starting Metabigor Company scan request handling")
	var payload struct {
		CompanyName       string  `json:"company_name" binding:"required"`
		AutoScanSessionID *string `json:"auto_scan_session_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.CompanyName == "" {
		log.Printf("[METABIGOR-COMPANY] [ERROR] Invalid request body: %v", err)
		http.Error(w, "Invalid request body. `company_name` is required.", http.StatusBadRequest)
		return
	}

	companyName := payload.CompanyName
	log.Printf("[METABIGOR-COMPANY] [INFO] Processing Metabigor Company scan for company: %s", companyName)

	query := `SELECT id FROM scope_targets WHERE type = 'Company' AND scope_target = $1`
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(), query, companyName).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[METABIGOR-COMPANY] [ERROR] No matching company scope target found for company %s: %v", companyName, err)
		http.Error(w, "No matching company scope target found.", http.StatusBadRequest)
		return
	}
	log.Printf("[METABIGOR-COMPANY] [INFO] Found scope target ID: %s for company: %s", scopeTargetID, companyName)

	scanID := uuid.New().String()
	log.Printf("[METABIGOR-COMPANY] [INFO] Generated new scan ID: %s", scanID)

	createTableQuery := `
		CREATE TABLE IF NOT EXISTS metabigor_company_scans (
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
		log.Printf("[METABIGOR-COMPANY] [ERROR] Failed to create metabigor_company_scans table: %v", err)
		http.Error(w, "Failed to create scan table.", http.StatusInternalServerError)
		return
	}
	log.Printf("[METABIGOR-COMPANY] [INFO] Ensured metabigor_company_scans table exists")

	var insertQuery string
	var args []interface{}
	if payload.AutoScanSessionID != nil && *payload.AutoScanSessionID != "" {
		insertQuery = `INSERT INTO metabigor_company_scans (scan_id, company_name, status, scope_target_id, auto_scan_session_id) VALUES ($1, $2, $3, $4, $5)`
		args = []interface{}{scanID, companyName, "pending", scopeTargetID, *payload.AutoScanSessionID}
	} else {
		insertQuery = `INSERT INTO metabigor_company_scans (scan_id, company_name, status, scope_target_id) VALUES ($1, $2, $3, $4)`
		args = []interface{}{scanID, companyName, "pending", scopeTargetID}
	}
	_, err = dbPool.Exec(context.Background(), insertQuery, args...)
	if err != nil {
		log.Printf("[METABIGOR-COMPANY] [ERROR] Failed to create scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}
	log.Printf("[METABIGOR-COMPANY] [INFO] Successfully created Metabigor Company scan record in database")

	go ExecuteMetabigorCompanyScan(scanID, companyName)

	log.Printf("[METABIGOR-COMPANY] [INFO] Metabigor Company scan initiated successfully, returning scan ID: %s", scanID)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func ExecuteMetabigorCompanyScan(scanID, companyName string) {
	log.Printf("[METABIGOR-COMPANY] [INFO] Starting Metabigor Company scan execution for company %s (scan ID: %s)", companyName, scanID)
	startTime := time.Now()

	command := fmt.Sprintf("echo '%s' | /usr/bin/docker exec -i ars0n-framework-v2-metabigor-1 metabigor net --org -o -", companyName)

	log.Printf("[METABIGOR-COMPANY] [DEBUG] Executing command: %s", command)

	output, err := exec.Command("sh", "-c", command).CombinedOutput()
	if err != nil {
		log.Printf("[METABIGOR-COMPANY] [ERROR] Metabigor command failed: %v", err)
		log.Printf("[METABIGOR-COMPANY] [ERROR] Command output: %s", string(output))
		UpdateMetabigorCompanyScanStatus(scanID, "error", "", fmt.Sprintf("Metabigor command failed: %v\nOutput: %s", err, string(output)), command, time.Since(startTime).String())
		return
	}

	log.Printf("[METABIGOR-COMPANY] [DEBUG] Metabigor raw output: %s", string(output))

	// Process the output to extract network information (IP ranges, ASNs)
	lines := strings.Split(string(output), "\n")
	var networkRanges []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		log.Printf("[METABIGOR-COMPANY] [DEBUG] Processing line: %s", line)

		// Skip non-network lines (comments, headers, etc.)
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") || strings.Contains(line, "metabigor") {
			log.Printf("[METABIGOR-COMPANY] [DEBUG] Skipping non-network line: %s", line)
			continue
		}

		// Clean up the line
		networkInfo := strings.TrimSpace(line)

		// Validate that it looks like network information (IP ranges, ASNs, etc.)
		if networkInfo == "" {
			log.Printf("[METABIGOR-COMPANY] [DEBUG] Skipping empty line")
			continue
		}

		log.Printf("[METABIGOR-COMPANY] [DEBUG] Adding network info: %s", networkInfo)
		networkRanges = append(networkRanges, networkInfo)
	}

	// Sort the results
	sort.Strings(networkRanges)

	result := strings.Join(networkRanges, "\n")
	log.Printf("[METABIGOR-COMPANY] [DEBUG] Final processed result contains %d network entries", len(networkRanges))
	log.Printf("[METABIGOR-COMPANY] [DEBUG] Network information found: %v", networkRanges)

	UpdateMetabigorCompanyScanStatus(scanID, "success", result, "", command, time.Since(startTime).String())
	log.Printf("[METABIGOR-COMPANY] [INFO] Metabigor Company scan completed and results stored successfully for company %s", companyName)
}

func UpdateMetabigorCompanyScanStatus(scanID, status, result, stderr, command, execTime string) {
	log.Printf("[METABIGOR-COMPANY] [INFO] Updating Metabigor Company scan status for scan ID %s to %s", scanID, status)
	query := `UPDATE metabigor_company_scans SET status = $1, result = $2, error = $3, command = $4, execution_time = $5 WHERE scan_id = $6`

	_, err := dbPool.Exec(context.Background(), query, status, result, stderr, command, execTime, scanID)
	if err != nil {
		log.Printf("[METABIGOR-COMPANY] [ERROR] Failed to update Metabigor Company scan status for scan ID %s: %v", scanID, err)
		log.Printf("[METABIGOR-COMPANY] [ERROR] Update attempted with: status=%s, result_length=%d, error_length=%d, command_length=%d, execTime=%s",
			status, len(result), len(stderr), len(command), execTime)
	} else {
		log.Printf("[METABIGOR-COMPANY] [INFO] Successfully updated Metabigor Company scan status to %s for scan ID %s", status, scanID)
	}
}

func GetMetabigorCompanyScanStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scanID := vars["scan_id"]
	log.Printf("[METABIGOR-COMPANY] [INFO] Retrieving Metabigor Company scan status for scan ID: %s", scanID)

	var scan MetabigorCompanyScanStatus
	query := `SELECT id, scan_id, company_name, status, result, error, stdout, stderr, command, execution_time, created_at, scope_target_id, auto_scan_session_id FROM metabigor_company_scans WHERE scan_id = $1`
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
			log.Printf("[METABIGOR-COMPANY] [ERROR] Metabigor Company scan not found for scan ID: %s", scanID)
			http.Error(w, "Scan not found", http.StatusNotFound)
		} else {
			log.Printf("[METABIGOR-COMPANY] [ERROR] Failed to get Metabigor Company scan status for scan ID %s: %v", scanID, err)
			http.Error(w, "Failed to get scan status", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("[METABIGOR-COMPANY] [INFO] Successfully retrieved Metabigor Company scan status for scan ID %s: %s", scanID, scan.Status)
	if scan.Result.Valid {
		log.Printf("[METABIGOR-COMPANY] [DEBUG] Scan has valid results of length: %d bytes", len(scan.Result.String))
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
		log.Printf("[METABIGOR-COMPANY] [ERROR] Failed to encode Metabigor Company scan response: %v", err)
	} else {
		log.Printf("[METABIGOR-COMPANY] [INFO] Successfully sent Metabigor Company scan status response")
	}
}

func GetMetabigorCompanyScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]
	log.Printf("[METABIGOR-COMPANY] [INFO] Fetching Metabigor Company scans for scope target ID: %s", scopeTargetID)

	if scopeTargetID == "" {
		log.Printf("[METABIGOR-COMPANY] [ERROR] No scope target ID provided")
		http.Error(w, "No scope target ID provided", http.StatusBadRequest)
		return
	}

	// Ensure the table exists before trying to query it
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS metabigor_company_scans (
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
		log.Printf("[METABIGOR-COMPANY] [ERROR] Failed to create metabigor_company_scans table: %v", err)
		http.Error(w, "Failed to create scan table.", http.StatusInternalServerError)
		return
	}

	query := `SELECT id, scan_id, company_name, status, result, error, stdout, stderr, command, execution_time, created_at, scope_target_id, auto_scan_session_id FROM metabigor_company_scans WHERE scope_target_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[METABIGOR-COMPANY] [ERROR] Failed to get scans: %v", err)
		http.Error(w, "Failed to get scans", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []map[string]interface{}
	for rows.Next() {
		var scan MetabigorCompanyScanStatus
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
			log.Printf("[METABIGOR-COMPANY] [ERROR] Error scanning Metabigor Company scan row: %v", err)
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

	log.Printf("[METABIGOR-COMPANY] [INFO] Successfully retrieved %d Metabigor Company scans for scope target %s", len(scans), scopeTargetID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(scans); err != nil {
		log.Printf("[METABIGOR-COMPANY] [ERROR] Failed to encode scans response: %v", err)
	} else {
		log.Printf("[METABIGOR-COMPANY] [INFO] Successfully sent Metabigor Company scans response")
	}
}
