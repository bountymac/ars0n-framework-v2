package utils

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type AmassIntelScanStatus struct {
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
	AutoScanSessionID sql.NullString `json:"auto_scan_session_id"`
}

type IntelRootDomainResponse struct {
	Domain  string `json:"domain"`
	Source  string `json:"source"`
	RawData string `json:"raw_data,omitempty"`
	ScanID  string `json:"scan_id"`
}

type IntelWhoisResponse struct {
	Domain       string `json:"domain,omitempty"`
	Registrant   string `json:"registrant,omitempty"`
	Organization string `json:"organization,omitempty"`
	RawWhois     string `json:"raw_whois"`
	ScanID       string `json:"scan_id"`
}

func RunAmassIntelScan(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		CompanyName       string  `json:"company_name" binding:"required"`
		AutoScanSessionID *string `json:"auto_scan_session_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.CompanyName == "" {
		http.Error(w, "Invalid request body. `company_name` is required.", http.StatusBadRequest)
		return
	}

	companyName := payload.CompanyName

	query := `SELECT id FROM scope_targets WHERE type = 'Company' AND scope_target = $1`
	var requestID string
	err := dbPool.QueryRow(context.Background(), query, companyName).Scan(&requestID)
	if err != nil {
		log.Printf("[ERROR] No matching company scope target found for company %s", companyName)
		http.Error(w, "No matching company scope target found.", http.StatusBadRequest)
		return
	}

	scanID := uuid.New().String()
	var insertQuery string
	var args []interface{}
	if payload.AutoScanSessionID != nil && *payload.AutoScanSessionID != "" {
		insertQuery = `INSERT INTO amass_intel_scans (scan_id, company_name, status, scope_target_id, auto_scan_session_id) VALUES ($1, $2, $3, $4, $5)`
		args = []interface{}{scanID, companyName, "pending", requestID, *payload.AutoScanSessionID}
	} else {
		insertQuery = `INSERT INTO amass_intel_scans (scan_id, company_name, status, scope_target_id) VALUES ($1, $2, $3, $4)`
		args = []interface{}{scanID, companyName, "pending", requestID}
	}
	_, err = dbPool.Exec(context.Background(), insertQuery, args...)
	if err != nil {
		log.Printf("[ERROR] Failed to create amass intel scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}

	go ExecuteAmassIntelScan(scanID, companyName)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func ExecuteAmassIntelScan(scanID, companyName string) {
	log.Printf("[INFO] Starting Amass Intel scan for company %s (scan ID: %s)", companyName, scanID)
	startTime := time.Now()

	// Generate domain from company name: remove spaces, lowercase, add .com
	domain := strings.ToLower(strings.ReplaceAll(companyName, " ", "")) + ".com"
	log.Printf("[INFO] Generated domain for scan: %s", domain)

	cmd := exec.Command(
		"docker", "run", "--rm",
		"caffix/amass",
		"intel",
		"-org", companyName,
		"-whois",
		"-active",
		"-d", domain,
		"-timeout", "60",
	)

	log.Printf("[INFO] Executing command: %s", cmd.String())

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	execTime := time.Since(startTime).String()

	if err != nil {
		log.Printf("[ERROR] Amass Intel scan failed for %s: %v", companyName, err)
		log.Printf("[ERROR] stderr output: %s", stderr.String())
		UpdateIntelScanStatus(scanID, "error", "", stderr.String(), cmd.String(), execTime)
		return
	}

	result := stdout.String()
	log.Printf("[INFO] Amass Intel scan completed in %s for company %s", execTime, companyName)
	log.Printf("[DEBUG] Raw output length: %d bytes", len(result))

	if result != "" {
		log.Printf("[INFO] Starting to parse Intel results for scan %s", scanID)
		ParseAndStoreIntelResults(scanID, companyName, result)
		log.Printf("[INFO] Finished parsing Intel results for scan %s", scanID)
	} else {
		log.Printf("[WARN] No output from Amass Intel scan for company %s", companyName)
	}

	UpdateIntelScanStatus(scanID, "success", result, stderr.String(), cmd.String(), execTime)
	log.Printf("[INFO] Intel scan status updated for scan %s", scanID)
}

func ParseAndStoreIntelResults(scanID, companyName, result string) {
	log.Printf("[INFO] Starting to parse Intel results for scan %s on company %s", scanID, companyName)

	lines := strings.Split(result, "\n")
	log.Printf("[INFO] Processing %d lines of Intel output", len(lines))

	domainPattern := regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		log.Printf("[DEBUG] Processing line %d: %s", lineNum+1, line)

		if domainPattern.MatchString(line) {
			log.Printf("[DEBUG] Valid root domain found: %s", line)
			InsertIntelRootDomain(scanID, line, "intel", line)
		} else if strings.Contains(strings.ToLower(line), "whois") {
			log.Printf("[DEBUG] Potential WHOIS data found: %s", line)
			ParseWhoisCorrelation(scanID, line)
		}
	}
	log.Printf("[INFO] Completed parsing Intel results for scan %s", scanID)
}

func ParseWhoisCorrelation(scanID, line string) {
	parts := strings.Fields(line)
	if len(parts) > 1 {
		domain := parts[0]
		whoisData := strings.Join(parts[1:], " ")
		InsertIntelWhoisData(scanID, domain, "", "", whoisData)
	}
}

func InsertIntelRootDomain(scanID, domain, source, rawData string) {
	log.Printf("[INFO] Inserting Intel root domain: %s", domain)
	query := `INSERT INTO intel_root_domains (scan_id, domain, source, raw_data) VALUES ($1, $2, $3, $4)`
	_, err := dbPool.Exec(context.Background(), query, scanID, domain, source, rawData)
	if err != nil {
		log.Printf("[ERROR] Failed to insert Intel root domain: %v", err)
	} else {
		log.Printf("[INFO] Successfully inserted Intel root domain: %s", domain)
	}
}

func InsertIntelWhoisData(scanID, domain, registrant, organization, rawWhois string) {
	log.Printf("[INFO] Inserting Intel WHOIS data for domain: %s", domain)
	query := `INSERT INTO intel_whois_data (scan_id, domain, registrant, organization, raw_whois) VALUES ($1, $2, $3, $4, $5)`
	_, err := dbPool.Exec(context.Background(), query, scanID, domain, registrant, organization, rawWhois)
	if err != nil {
		log.Printf("[ERROR] Failed to insert Intel WHOIS data: %v", err)
	} else {
		log.Printf("[INFO] Successfully inserted Intel WHOIS data for domain: %s", domain)
	}
}

func UpdateIntelScanStatus(scanID, status, result, stderr, command, execTime string) {
	log.Printf("[INFO] Updating Intel scan status for %s to %s", scanID, status)
	query := `UPDATE amass_intel_scans SET status = $1, result = $2, stderr = $3, command = $4, execution_time = $5 WHERE scan_id = $6`
	_, err := dbPool.Exec(context.Background(), query, status, result, stderr, command, execTime, scanID)
	if err != nil {
		log.Printf("[ERROR] Failed to update Intel scan status for %s: %v", scanID, err)
	} else {
		log.Printf("[INFO] Successfully updated Intel scan status for %s", scanID)
	}
}

func GetAmassIntelScanStatus(w http.ResponseWriter, r *http.Request) {
	scanID := mux.Vars(r)["scanID"]
	if scanID == "" {
		http.Error(w, "Scan ID is required", http.StatusBadRequest)
		return
	}

	var scan AmassIntelScanStatus
	query := `SELECT id, scan_id, company_name, status, result, error, stdout, stderr, command, execution_time, created_at, auto_scan_session_id FROM amass_intel_scans WHERE scan_id = $1`
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
		&scan.AutoScanSessionID,
	)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch Intel scan status: %v", err)
		http.Error(w, "Scan not found.", http.StatusNotFound)
		return
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
		"auto_scan_session_id": nullStringToString(scan.AutoScanSessionID),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func GetAmassIntelScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	scopeTargetID := mux.Vars(r)["id"]
	if scopeTargetID == "" {
		http.Error(w, "Scope target ID is required", http.StatusBadRequest)
		return
	}

	query := `SELECT id, scan_id, company_name, status, result, error, stdout, stderr, command, execution_time, created_at, auto_scan_session_id 
              FROM amass_intel_scans WHERE scope_target_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch Intel scans for scope target ID %s: %v", scopeTargetID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []map[string]interface{}
	for rows.Next() {
		var scan AmassIntelScanStatus
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
			&scan.AutoScanSessionID,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		scans = append(scans, map[string]interface{}{
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
			"auto_scan_session_id": nullStringToString(scan.AutoScanSessionID),
		})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(scans)
}

func GetIntelRootDomains(w http.ResponseWriter, r *http.Request) {
	scanID := mux.Vars(r)["scan_id"]
	if scanID == "" || scanID == "No scans available" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]struct{}{})
		return
	}

	if _, err := uuid.Parse(scanID); err != nil {
		http.Error(w, "Invalid scan ID format", http.StatusBadRequest)
		return
	}

	query := `SELECT domain, source, raw_data, scan_id FROM intel_root_domains WHERE scan_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scanID)
	if err != nil {
		http.Error(w, "Failed to fetch Intel root domains", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var domains []IntelRootDomainResponse
	for rows.Next() {
		var domain IntelRootDomainResponse
		if err := rows.Scan(&domain.Domain, &domain.Source, &domain.RawData, &domain.ScanID); err != nil {
			http.Error(w, "Error scanning Intel root domain", http.StatusInternalServerError)
			return
		}
		domains = append(domains, domain)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(domains)
}

func GetIntelWhoisData(w http.ResponseWriter, r *http.Request) {
	scanID := mux.Vars(r)["scan_id"]
	if scanID == "" || scanID == "No scans available" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]struct{}{})
		return
	}

	if _, err := uuid.Parse(scanID); err != nil {
		http.Error(w, "Invalid scan ID format", http.StatusBadRequest)
		return
	}

	query := `SELECT domain, registrant, organization, raw_whois, scan_id FROM intel_whois_data WHERE scan_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scanID)
	if err != nil {
		http.Error(w, "Failed to fetch Intel WHOIS data", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var whoisData []IntelWhoisResponse
	for rows.Next() {
		var whois IntelWhoisResponse
		if err := rows.Scan(&whois.Domain, &whois.Registrant, &whois.Organization, &whois.RawWhois, &whois.ScanID); err != nil {
			http.Error(w, "Error scanning Intel WHOIS data", http.StatusInternalServerError)
			return
		}
		whoisData = append(whoisData, whois)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(whoisData)
}
