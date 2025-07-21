package utils

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type DatabaseExportRequest struct {
	ScopeTargetIDs []string `json:"scope_target_ids"`
}

type DatabaseImportRequest struct {
	Data []byte `json:"data"`
}

type ExportData struct {
	ExportMetadata ExportMetadata                      `json:"export_metadata"`
	ScopeTargets   []map[string]interface{}            `json:"scope_targets"`
	TableData      map[string][]map[string]interface{} `json:"table_data"`
}

type ExportMetadata struct {
	ExportedAt     time.Time `json:"exported_at"`
	Version        string    `json:"version"`
	ScopeTargetIDs []string  `json:"scope_target_ids"`
	ScopeTargets   []string  `json:"scope_targets"`
	TotalRecords   int       `json:"total_records"`
	TablesExported []string  `json:"tables_exported"`
}

var exportTableQueries = map[string]string{
	"auto_scan_sessions": `
		SELECT id, scope_target_id, config_snapshot, status, started_at, ended_at, 
		       steps_run, error_message, final_consolidated_subdomains, final_live_web_servers
		FROM auto_scan_sessions 
		WHERE scope_target_id = ANY($1)`,

	"auto_scan_state": `
		SELECT id, scope_target_id, current_step, is_paused, is_cancelled, created_at, updated_at
		FROM auto_scan_state 
		WHERE scope_target_id = ANY($1)`,

	"amass_scans": `
		SELECT id, scan_id, domain, status, result, error, stdout, stderr, command, 
		       execution_time, created_at, scope_target_id, auto_scan_session_id
		FROM amass_scans 
		WHERE scope_target_id = ANY($1)`,

	"amass_intel_scans": `
		SELECT id, scan_id, company_name, status, result, error, stdout, stderr, command,
		       execution_time, created_at, scope_target_id, auto_scan_session_id
		FROM amass_intel_scans 
		WHERE scope_target_id = ANY($1)`,

	"amass_enum_company_scans": `
		SELECT id, scan_id, scope_target_id, domains, status, result, error, stdout, stderr,
		       command, execution_time, created_at
		FROM amass_enum_company_scans 
		WHERE scope_target_id = ANY($1)`,

	"httpx_scans": `
		SELECT id, scan_id, domain, status, result, error, stdout, stderr, command,
		       execution_time, created_at, scope_target_id, auto_scan_session_id
		FROM httpx_scans 
		WHERE scope_target_id = ANY($1)`,

	"gau_scans": `
		SELECT id, scan_id, domain, status, result, error, stdout, stderr, command,
		       execution_time, created_at, scope_target_id, auto_scan_session_id
		FROM gau_scans 
		WHERE scope_target_id = ANY($1)`,

	"sublist3r_scans": `
		SELECT id, scan_id, domain, status, result, error, stdout, stderr, command,
		       execution_time, created_at, scope_target_id, auto_scan_session_id
		FROM sublist3r_scans 
		WHERE scope_target_id = ANY($1)`,

	"assetfinder_scans": `
		SELECT id, scan_id, domain, status, result, error, stdout, stderr, command,
		       execution_time, created_at, scope_target_id, auto_scan_session_id
		FROM assetfinder_scans 
		WHERE scope_target_id = ANY($1)`,

	"ctl_scans": `
		SELECT id, scan_id, domain, status, result, error, stdout, stderr, command,
		       execution_time, created_at, scope_target_id, auto_scan_session_id
		FROM ctl_scans 
		WHERE scope_target_id = ANY($1)`,

	"subfinder_scans": `
		SELECT id, scan_id, domain, status, result, error, stdout, stderr, command,
		       execution_time, created_at, scope_target_id, auto_scan_session_id
		FROM subfinder_scans 
		WHERE scope_target_id = ANY($1)`,

	"shuffledns_scans": `
		SELECT id, scan_id, domain, status, result, error, stdout, stderr, command,
		       execution_time, created_at, scope_target_id, auto_scan_session_id
		FROM shuffledns_scans 
		WHERE scope_target_id = ANY($1)`,

	"cewl_scans": `
		SELECT id, scan_id, url, status, result, error, stdout, stderr, command,
		       execution_time, created_at, scope_target_id, auto_scan_session_id
		FROM cewl_scans 
		WHERE scope_target_id = ANY($1)`,

	"gospider_scans": `
		SELECT id, scan_id, domain, status, result, error, stdout, stderr, command,
		       execution_time, created_at, scope_target_id, auto_scan_session_id
		FROM gospider_scans 
		WHERE scope_target_id = ANY($1)`,

	"subdomainizer_scans": `
		SELECT id, scan_id, domain, status, result, error, stdout, stderr, command,
		       execution_time, created_at, scope_target_id, auto_scan_session_id
		FROM subdomainizer_scans 
		WHERE scope_target_id = ANY($1)`,

	"nuclei_screenshots": `
		SELECT id, scan_id, domain, status, result, error, stdout, stderr, command,
		       execution_time, created_at, scope_target_id, auto_scan_session_id
		FROM nuclei_screenshots 
		WHERE scope_target_id = ANY($1)`,

	"metadata_scans": `
		SELECT id, scan_id, domain, status, result, error, stdout, stderr, command,
		       execution_time, created_at, scope_target_id, auto_scan_session_id
		FROM metadata_scans 
		WHERE scope_target_id = ANY($1)`,

	"target_urls": `
		SELECT id, url, screenshot, status_code, title, web_server, technologies, content_length,
		       newly_discovered, no_longer_live, scope_target_id, created_at, updated_at,
		       has_deprecated_tls, has_expired_ssl, has_mismatched_ssl, has_revoked_ssl,
		       has_self_signed_ssl, has_untrusted_root_ssl, has_wildcard_tls, findings_json,
		       http_response, http_response_headers, dns_a_records, dns_aaaa_records,
		       dns_cname_records, dns_mx_records, dns_txt_records, dns_ns_records,
		       dns_ptr_records, dns_srv_records, katana_results, ffuf_results, roi_score, ip_address
		FROM target_urls 
		WHERE scope_target_id = ANY($1)`,

	"consolidated_subdomains": `
		SELECT id, scope_target_id, subdomain, created_at
		FROM consolidated_subdomains 
		WHERE scope_target_id = ANY($1)`,

	"consolidated_company_domains": `
		SELECT id, scope_target_id, domain, source, created_at
		FROM consolidated_company_domains 
		WHERE scope_target_id = ANY($1)`,

	"consolidated_network_ranges": `
		SELECT id, scope_target_id, cidr_block, asn, organization, description, country, source, scan_type, created_at
		FROM consolidated_network_ranges 
		WHERE scope_target_id = ANY($1)`,

	"google_dorking_domains": `
		SELECT id, scope_target_id, domain, created_at
		FROM google_dorking_domains 
		WHERE scope_target_id = ANY($1)`,

	"reverse_whois_domains": `
		SELECT id, scope_target_id, domain, created_at
		FROM reverse_whois_domains 
		WHERE scope_target_id = ANY($1)`,

	"consolidated_attack_surface_assets": `
		SELECT id, scope_target_id, asset_type, asset_identifier, asset_subtype, asn_number,
		       asn_organization, asn_description, asn_country, cidr_block, subnet_size,
		       responsive_ip_count, responsive_port_count, ip_address, ip_type, dnsx_a_records,
		       amass_a_records, httpx_sources, url, domain, port, protocol, status_code, title,
		       web_server, technologies, content_length, response_time_ms, screenshot_path,
		       ssl_info, http_response_headers, findings_json, cloud_provider, cloud_service_type,
		       cloud_region, fqdn, root_domain, subdomain, registrar, creation_date, expiration_date,
		       updated_date, name_servers, status, whois_info, ssl_certificate, ssl_expiry_date,
		       ssl_issuer, ssl_subject, ssl_version, ssl_cipher_suite, ssl_protocols, resolved_ips,
		       mail_servers, spf_record, dkim_record, dmarc_record, caa_records, txt_records,
		       mx_records, ns_records, a_records, aaaa_records, cname_records, ptr_records,
		       srv_records, soa_record, last_dns_scan, last_ssl_scan, last_whois_scan,
		       last_updated, created_at
		FROM consolidated_attack_surface_assets 
		WHERE scope_target_id = ANY($1)`,

	"nuclei_scans": `
		SELECT id, scan_id, scope_target_id, targets, templates, status, result, error, stdout, stderr,
		       command, execution_time, created_at, updated_at, auto_scan_session_id
		FROM nuclei_scans 
		WHERE scope_target_id = ANY($1)`,

	"ip_port_scans": `
		SELECT id, scan_id, scope_target_id, status, total_network_ranges, processed_network_ranges,
		       total_ips_discovered, total_ports_scanned, live_web_servers_found, error_message,
		       command, execution_time, created_at, auto_scan_session_id
		FROM ip_port_scans 
		WHERE scope_target_id = ANY($1)`,
}

func HandleDatabaseExport(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] Starting database export process")

	var req DatabaseExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.ScopeTargetIDs) == 0 {
		http.Error(w, "No scope targets specified", http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Database export request for %d scope targets", len(req.ScopeTargetIDs))

	exportData, err := exportDatabaseData(req.ScopeTargetIDs)
	if err != nil {
		log.Printf("[ERROR] Failed to export database data: %v", err)
		http.Error(w, fmt.Sprintf("Failed to export data: %v", err), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(exportData)
	if err != nil {
		log.Printf("[ERROR] Failed to marshal export data: %v", err)
		http.Error(w, "Failed to marshal export data", http.StatusInternalServerError)
		return
	}

	var compressedData bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressedData)
	if _, err := gzipWriter.Write(jsonData); err != nil {
		log.Printf("[ERROR] Failed to compress data: %v", err)
		http.Error(w, "Failed to compress data", http.StatusInternalServerError)
		return
	}
	gzipWriter.Close()

	filename := fmt.Sprintf("rs0n-export-%s.rs0n", time.Now().Format("2006-01-02-15-04-05"))

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", compressedData.Len()))

	if _, err := w.Write(compressedData.Bytes()); err != nil {
		log.Printf("[ERROR] Failed to write response: %v", err)
	}

	log.Printf("[INFO] Database export completed successfully. File: %s, Size: %d bytes", filename, compressedData.Len())
}

func HandleDatabaseImport(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] Starting database import process")

	if err := r.ParseMultipartForm(100 << 20); err != nil {
		log.Printf("[ERROR] Failed to parse multipart form: %v", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("[ERROR] Failed to get file from form: %v", err)
		http.Error(w, "Failed to get uploaded file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if !strings.HasSuffix(header.Filename, ".rs0n") {
		http.Error(w, "Invalid file type. Only .rs0n files are supported", http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Processing import file: %s", header.Filename)

	compressedData, err := io.ReadAll(file)
	if err != nil {
		log.Printf("[ERROR] Failed to read file: %v", err)
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		log.Printf("[ERROR] Failed to create gzip reader: %v", err)
		http.Error(w, "Invalid file format", http.StatusBadRequest)
		return
	}
	defer gzipReader.Close()

	jsonData, err := io.ReadAll(gzipReader)
	if err != nil {
		log.Printf("[ERROR] Failed to decompress data: %v", err)
		http.Error(w, "Failed to decompress data", http.StatusInternalServerError)
		return
	}

	var exportData ExportData
	if err := json.Unmarshal(jsonData, &exportData); err != nil {
		log.Printf("[ERROR] Failed to unmarshal export data: %v", err)
		http.Error(w, "Invalid export data format", http.StatusBadRequest)
		return
	}

	if err := importDatabaseData(&exportData); err != nil {
		log.Printf("[ERROR] Failed to import database data: %v", err)
		http.Error(w, fmt.Sprintf("Failed to import data: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":                "Database import completed successfully",
		"imported_scope_targets": len(exportData.ScopeTargets),
		"imported_tables":        len(exportData.TableData),
		"total_records":          exportData.ExportMetadata.TotalRecords,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("[INFO] Database import completed successfully. Imported %d scope targets", len(exportData.ScopeTargets))
}

func exportDatabaseData(scopeTargetIDs []string) (*ExportData, error) {
	exportData := &ExportData{
		TableData: make(map[string][]map[string]interface{}),
	}

	scopeTargets, err := getScopeTargetsForExport(scopeTargetIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get scope targets: %v", err)
	}
	exportData.ScopeTargets = scopeTargets

	var scopeTargetNames []string
	for _, target := range scopeTargets {
		if name, ok := target["scope_target"].(string); ok {
			scopeTargetNames = append(scopeTargetNames, name)
		}
	}

	totalRecords := len(scopeTargets)
	var tablesExported []string

	for tableName, query := range exportTableQueries {
		log.Printf("[INFO] Exporting data from table: %s", tableName)

		rows, err := dbPool.Query(context.Background(), query, scopeTargetIDs)
		if err != nil {
			log.Printf("[WARN] Failed to query table %s: %v", tableName, err)
			continue
		}

		tableData, err := rowsToMaps(rows)
		rows.Close()

		if err != nil {
			log.Printf("[WARN] Failed to process rows for table %s: %v", tableName, err)
			continue
		}

		if len(tableData) > 0 {
			exportData.TableData[tableName] = tableData
			totalRecords += len(tableData)
			tablesExported = append(tablesExported, tableName)
			log.Printf("[INFO] Exported %d records from table: %s", len(tableData), tableName)
		}
	}

	exportData.ExportMetadata = ExportMetadata{
		ExportedAt:     time.Now(),
		Version:        "1.0",
		ScopeTargetIDs: scopeTargetIDs,
		ScopeTargets:   scopeTargetNames,
		TotalRecords:   totalRecords,
		TablesExported: tablesExported,
	}

	return exportData, nil
}

func getScopeTargetsForExport(scopeTargetIDs []string) ([]map[string]interface{}, error) {
	query := `SELECT id, type, mode, scope_target, active, created_at FROM scope_targets WHERE id = ANY($1)`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToMaps(rows)
}

func rowsToMaps(rows pgx.Rows) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	fieldDescriptions := rows.FieldDescriptions()

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		record := make(map[string]interface{})
		for i, value := range values {
			fieldName := string(fieldDescriptions[i].Name)

			// Convert UUID byte arrays to strings during export
			if fieldDescriptions[i].DataTypeOID == 2950 { // UUID OID
				if bytes, ok := value.([]byte); ok && len(bytes) == 16 {
					if parsedUUID, err := uuid.FromBytes(bytes); err == nil {
						record[fieldName] = parsedUUID.String()
						continue
					}
				}
			}

			record[fieldName] = value
		}

		results = append(results, record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func importDatabaseData(exportData *ExportData) error {
	tx, err := dbPool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(context.Background())

	if err := importScopeTargets(tx, exportData.ScopeTargets); err != nil {
		return fmt.Errorf("failed to import scope targets: %v", err)
	}

	if err := importTableData(tx, exportData.TableData); err != nil {
		return fmt.Errorf("failed to import table data: %v", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

func convertUUIDValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	// If it's already a string, return as is
	if str, ok := value.(string); ok {
		return str
	}

	// If it's a byte slice (UUID as bytes), convert to UUID string
	if bytes, ok := value.([]interface{}); ok {
		if len(bytes) == 16 {
			// Convert []interface{} to []byte
			byteArray := make([]byte, 16)
			for i, b := range bytes {
				if num, ok := b.(float64); ok {
					byteArray[i] = byte(num)
				} else {
					return value // Return original if conversion fails
				}
			}

			// Parse bytes as UUID and return string representation
			if parsedUUID, err := uuid.FromBytes(byteArray); err == nil {
				return parsedUUID.String()
			}
		}
	}

	// If it's already a byte array
	if bytes, ok := value.([]byte); ok {
		if len(bytes) == 16 {
			if parsedUUID, err := uuid.FromBytes(bytes); err == nil {
				return parsedUUID.String()
			}
		}
	}

	return value
}

func convertRecordUUIDs(record map[string]interface{}) map[string]interface{} {
	// List of fields that are UUIDs
	uuidFields := []string{
		"id", "scope_target_id", "scan_id", "auto_scan_session_id",
		"ip_port_scan_id", "parent_asset_id", "child_asset_id", "asset_id",
	}

	for _, field := range uuidFields {
		if value, exists := record[field]; exists {
			record[field] = convertUUIDValue(value)
		}
	}

	return record
}

func importScopeTargets(tx pgx.Tx, scopeTargets []map[string]interface{}) error {
	for _, target := range scopeTargets {
		// Convert UUID fields
		target = convertRecordUUIDs(target)

		query := `
			INSERT INTO scope_targets (id, type, mode, scope_target, active, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO UPDATE SET
				type = EXCLUDED.type,
				mode = EXCLUDED.mode,
				scope_target = EXCLUDED.scope_target,
				active = EXCLUDED.active,
				created_at = EXCLUDED.created_at`

		_, err := tx.Exec(context.Background(), query,
			target["id"], target["type"], target["mode"],
			target["scope_target"], target["active"], target["created_at"])
		if err != nil {
			return fmt.Errorf("failed to insert scope target: %v", err)
		}
	}
	return nil
}

func importTableData(tx pgx.Tx, tableData map[string][]map[string]interface{}) error {
	tableOrder := []string{
		"auto_scan_sessions", "auto_scan_state", "amass_scans", "amass_intel_scans",
		"amass_enum_company_scans", "httpx_scans", "gau_scans", "sublist3r_scans",
		"assetfinder_scans", "ctl_scans", "subfinder_scans", "shuffledns_scans",
		"cewl_scans", "gospider_scans", "subdomainizer_scans", "nuclei_screenshots",
		"metadata_scans", "target_urls", "consolidated_subdomains", "consolidated_company_domains",
		"consolidated_network_ranges", "google_dorking_domains", "reverse_whois_domains",
		"consolidated_attack_surface_assets", "nuclei_scans", "ip_port_scans",
	}

	for _, tableName := range tableOrder {
		if records, exists := tableData[tableName]; exists {
			if err := importTableRecords(tx, tableName, records); err != nil {
				return fmt.Errorf("failed to import table %s: %v", tableName, err)
			}
		}
	}

	for tableName, records := range tableData {
		found := false
		for _, orderedTable := range tableOrder {
			if tableName == orderedTable {
				found = true
				break
			}
		}
		if !found {
			if err := importTableRecords(tx, tableName, records); err != nil {
				return fmt.Errorf("failed to import table %s: %v", tableName, err)
			}
		}
	}

	return nil
}

func importTableRecords(tx pgx.Tx, tableName string, records []map[string]interface{}) error {
	if len(records) == 0 {
		return nil
	}

	log.Printf("[INFO] Importing %d records into table: %s", len(records), tableName)

	for _, record := range records {
		if err := importSingleRecord(tx, tableName, record); err != nil {
			log.Printf("[WARN] Failed to import record in table %s: %v", tableName, err)
		}
	}

	return nil
}

func importSingleRecord(tx pgx.Tx, tableName string, record map[string]interface{}) error {
	// Convert UUID fields
	record = convertRecordUUIDs(record)

	var columns []string
	var placeholders []string
	var values []interface{}
	var updateClauses []string

	i := 1
	for column, value := range record {
		columns = append(columns, column)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, value)
		updateClauses = append(updateClauses, fmt.Sprintf("%s = EXCLUDED.%s", column, column))
		i++
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (%s) VALUES (%s)
		ON CONFLICT (id) DO UPDATE SET %s`,
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(updateClauses, ", "))

	_, err := tx.Exec(context.Background(), query, values...)
	return err
}

func GetScopeTargetsForExport(w http.ResponseWriter, r *http.Request) {
	log.Println("[INFO] Fetching scope targets for export")

	rows, err := dbPool.Query(context.Background(),
		`SELECT id, type, scope_target, active, created_at FROM scope_targets ORDER BY created_at DESC`)
	if err != nil {
		log.Printf("[ERROR] Failed to query scope targets: %v", err)
		http.Error(w, "Failed to fetch scope targets", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var targets []map[string]interface{}
	for rows.Next() {
		var id, targetType, scopeTarget string
		var active bool
		var createdAt time.Time

		if err := rows.Scan(&id, &targetType, &scopeTarget, &active, &createdAt); err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			continue
		}

		targets = append(targets, map[string]interface{}{
			"id":           id,
			"type":         targetType,
			"scope_target": scopeTarget,
			"active":       active,
			"created_at":   createdAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(targets)

	log.Printf("[INFO] Returned %d scope targets for export", len(targets))
}
