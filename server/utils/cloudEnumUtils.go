package utils

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
)

type CloudEnumScanStatus struct {
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

type CloudEnumResult struct {
	Platform string `json:"platform"`
	Msg      string `json:"msg"`
	Target   string `json:"target"`
	Access   string `json:"access"`
}

func RunCloudEnumScan(w http.ResponseWriter, r *http.Request) {
	log.Printf("[CLOUD-ENUM] [INFO] Starting Cloud Enum scan request handling")
	var payload struct {
		CompanyName       string  `json:"company_name" binding:"required"`
		AutoScanSessionID *string `json:"auto_scan_session_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.CompanyName == "" {
		log.Printf("[CLOUD-ENUM] [ERROR] Invalid request body: %v", err)
		http.Error(w, "Invalid request body. `company_name` is required.", http.StatusBadRequest)
		return
	}

	companyName := payload.CompanyName
	log.Printf("[CLOUD-ENUM] [INFO] Processing Cloud Enum scan for company: %s", companyName)

	query := `SELECT id FROM scope_targets WHERE type = 'Company' AND scope_target = $1`
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(), query, companyName).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[CLOUD-ENUM] [ERROR] No matching company scope target found for company %s: %v", companyName, err)
		http.Error(w, "No matching company scope target found.", http.StatusBadRequest)
		return
	}
	log.Printf("[CLOUD-ENUM] [INFO] Found scope target ID: %s for company: %s", scopeTargetID, companyName)

	scanID := uuid.New().String()
	log.Printf("[CLOUD-ENUM] [INFO] Generated new scan ID: %s", scanID)

	createTableQuery := `
		CREATE TABLE IF NOT EXISTS cloud_enum_scans (
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
		log.Printf("[CLOUD-ENUM] [ERROR] Failed to create cloud_enum_scans table: %v", err)
		http.Error(w, "Failed to create scan table.", http.StatusInternalServerError)
		return
	}
	log.Printf("[CLOUD-ENUM] [INFO] Ensured cloud_enum_scans table exists")

	var insertQuery string
	var args []interface{}
	if payload.AutoScanSessionID != nil && *payload.AutoScanSessionID != "" {
		insertQuery = `INSERT INTO cloud_enum_scans (scan_id, company_name, status, scope_target_id, auto_scan_session_id) VALUES ($1, $2, $3, $4, $5)`
		args = []interface{}{scanID, companyName, "pending", scopeTargetID, *payload.AutoScanSessionID}
	} else {
		insertQuery = `INSERT INTO cloud_enum_scans (scan_id, company_name, status, scope_target_id) VALUES ($1, $2, $3, $4)`
		args = []interface{}{scanID, companyName, "pending", scopeTargetID}
	}
	_, err = dbPool.Exec(context.Background(), insertQuery, args...)
	if err != nil {
		log.Printf("[CLOUD-ENUM] [ERROR] Failed to create scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}
	log.Printf("[CLOUD-ENUM] [INFO] Successfully created Cloud Enum scan record in database")

	go ExecuteAndParseCloudEnumScan(scanID, companyName)

	log.Printf("[CLOUD-ENUM] [INFO] Cloud Enum scan initiated successfully, returning scan ID: %s", scanID)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func ExecuteAndParseCloudEnumScan(scanID, companyName string) {
	log.Printf("[CLOUD-ENUM] [INFO] Starting Cloud Enum scan execution for company %s (scan ID: %s)", companyName, scanID)
	startTime := time.Now()

	containerName := "ars0n-framework-v2-cloud_enum-1"
	logFile := fmt.Sprintf("/tmp/cloud_enum_%s.json", scanID)

	command := []string{
		"docker", "exec", containerName,
		"python", "cloud_enum.py",
		"-k", companyName,
		"-l", logFile,
		"-f", "json",
	}

	log.Printf("[CLOUD-ENUM] [DEBUG] Executing command: %v", command)
	cmd := exec.Command(command[0], command[1:]...)

	stdout, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[CLOUD-ENUM] [ERROR] Failed to execute cloud_enum: %v", err)
		UpdateCloudEnumScanStatus(scanID, "error", "", fmt.Sprintf("Failed to execute cloud_enum: %v", err), strings.Join(command, " "), time.Since(startTime).String())
		return
	}

	log.Printf("[CLOUD-ENUM] [DEBUG] Command stdout: %s", string(stdout))

	catCommand := []string{"docker", "exec", containerName, "cat", logFile}
	catCmd := exec.Command(catCommand[0], catCommand[1:]...)
	resultOutput, err := catCmd.Output()
	if err != nil {
		log.Printf("[CLOUD-ENUM] [ERROR] Failed to read results file: %v", err)
		UpdateCloudEnumScanStatus(scanID, "error", "", fmt.Sprintf("Failed to read results file: %v", err), strings.Join(command, " "), time.Since(startTime).String())
		return
	}

	resultStr := string(resultOutput)
	log.Printf("[CLOUD-ENUM] [DEBUG] Raw results length: %d bytes", len(resultStr))

	var cloudEnumResults []CloudEnumResult
	lines := strings.Split(resultStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#### CLOUD_ENUM") {
			continue
		}

		var result CloudEnumResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			log.Printf("[CLOUD-ENUM] [DEBUG] Skipping invalid JSON line: %s", line)
			continue
		}
		cloudEnumResults = append(cloudEnumResults, result)
	}

	resultJSON, err := json.Marshal(cloudEnumResults)
	if err != nil {
		log.Printf("[CLOUD-ENUM] [ERROR] Failed to marshal results: %v", err)
		UpdateCloudEnumScanStatus(scanID, "error", "", fmt.Sprintf("Failed to marshal results: %v", err), strings.Join(command, " "), time.Since(startTime).String())
		return
	}

	log.Printf("[CLOUD-ENUM] [DEBUG] Processed %d cloud resources", len(cloudEnumResults))
	UpdateCloudEnumScanStatus(scanID, "success", string(resultJSON), "", strings.Join(command, " "), time.Since(startTime).String())
	log.Printf("[CLOUD-ENUM] [INFO] Cloud Enum scan completed and results stored successfully for company %s", companyName)
}

func UpdateCloudEnumScanStatus(scanID, status, result, stderr, command, execTime string) {
	log.Printf("[CLOUD-ENUM] [INFO] Updating Cloud Enum scan status for scan ID %s to %s", scanID, status)
	query := `UPDATE cloud_enum_scans SET status = $1, result = $2, error = $3, command = $4, execution_time = $5 WHERE scan_id = $6`

	_, err := dbPool.Exec(context.Background(), query, status, result, stderr, command, execTime, scanID)
	if err != nil {
		log.Printf("[CLOUD-ENUM] [ERROR] Failed to update Cloud Enum scan status for scan ID %s: %v", scanID, err)
		log.Printf("[CLOUD-ENUM] [ERROR] Update attempted with: status=%s, result_length=%d, error_length=%d, command_length=%d, execTime=%s",
			status, len(result), len(stderr), len(command), execTime)
	} else {
		log.Printf("[CLOUD-ENUM] [INFO] Successfully updated Cloud Enum scan status to %s for scan ID %s", status, scanID)
	}
}

func GetCloudEnumScanStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scanID := vars["scan_id"]
	log.Printf("[CLOUD-ENUM] [INFO] Retrieving Cloud Enum scan status for scan ID: %s", scanID)

	var scan CloudEnumScanStatus
	query := `SELECT id, scan_id, company_name, status, result, error, stdout, stderr, command, execution_time, created_at, scope_target_id, auto_scan_session_id FROM cloud_enum_scans WHERE scan_id = $1`
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
			log.Printf("[CLOUD-ENUM] [ERROR] Cloud Enum scan not found for scan ID: %s", scanID)
			http.Error(w, "Scan not found", http.StatusNotFound)
		} else {
			log.Printf("[CLOUD-ENUM] [ERROR] Failed to get Cloud Enum scan status for scan ID %s: %v", scanID, err)
			http.Error(w, "Failed to get scan status", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("[CLOUD-ENUM] [INFO] Successfully retrieved Cloud Enum scan status for scan ID %s: %s", scanID, scan.Status)
	if scan.Result.Valid {
		log.Printf("[CLOUD-ENUM] [DEBUG] Scan has valid results of length: %d bytes", len(scan.Result.String))
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
		log.Printf("[CLOUD-ENUM] [ERROR] Failed to encode Cloud Enum scan response: %v", err)
	} else {
		log.Printf("[CLOUD-ENUM] [INFO] Successfully sent Cloud Enum scan status response")
	}
}

func GetCloudEnumScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]
	log.Printf("[CLOUD-ENUM] [INFO] Fetching Cloud Enum scans for scope target ID: %s", scopeTargetID)

	if scopeTargetID == "" {
		log.Printf("[CLOUD-ENUM] [ERROR] No scope target ID provided")
		http.Error(w, "No scope target ID provided", http.StatusBadRequest)
		return
	}

	createTableQuery := `
		CREATE TABLE IF NOT EXISTS cloud_enum_scans (
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
		log.Printf("[CLOUD-ENUM] [ERROR] Failed to create cloud_enum_scans table: %v", err)
		http.Error(w, "Failed to create scan table.", http.StatusInternalServerError)
		return
	}

	query := `SELECT id, scan_id, company_name, status, result, error, stdout, stderr, command, execution_time, created_at, scope_target_id, auto_scan_session_id FROM cloud_enum_scans WHERE scope_target_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[CLOUD-ENUM] [ERROR] Failed to get scans: %v", err)
		http.Error(w, "Failed to get scans", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []map[string]interface{}
	for rows.Next() {
		var scan CloudEnumScanStatus
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
			log.Printf("[CLOUD-ENUM] [ERROR] Error scanning Cloud Enum scan row: %v", err)
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

	log.Printf("[CLOUD-ENUM] [INFO] Successfully retrieved %d Cloud Enum scans for scope target %s", len(scans), scopeTargetID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(scans); err != nil {
		log.Printf("[CLOUD-ENUM] [ERROR] Failed to encode scans response: %v", err)
	} else {
		log.Printf("[CLOUD-ENUM] [INFO] Successfully sent Cloud Enum scans response")
	}
}
