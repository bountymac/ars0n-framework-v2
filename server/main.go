package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"ars0n-framework-v2-server/utils"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var dbPool *pgxpool.Pool

func main() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("Environment variable DATABASE_URL is not set")
	}

	var err error
	for i := 0; i < 10; i++ {
		dbPool, err = pgxpool.New(context.Background(), connStr)
		if err == nil {
			err = dbPool.Ping(context.Background())
		}
		if err == nil {
			fmt.Println("Connected to the database successfully!")
			break
		}
		log.Printf("Failed to connect to the database: %v. Retrying in 5 seconds...", err)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		log.Fatalf("Could not connect to the database: %v", err)
	}
	utils.InitDB(dbPool)
	defer dbPool.Close()

	createTables()

	r := mux.NewRouter()

	// Apply CORS middleware first
	r.Use(corsMiddleware)

	// Define routes
	r.HandleFunc("/scopetarget/add", utils.CreateScopeTarget).Methods("POST", "OPTIONS")
	r.HandleFunc("/scopetarget/read", utils.ReadScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/delete/{id}", utils.DeleteScopeTarget).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/activate", utils.ActivateScopeTarget).Methods("POST", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/amass", utils.GetAmassScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/amass/run", utils.RunAmassScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/amass/{scanID}", utils.GetAmassScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/amass/{scan_id}/dns", utils.GetDNSRecords).Methods("GET", "OPTIONS")
	r.HandleFunc("/amass/{scan_id}/ip", utils.GetIPs).Methods("GET", "OPTIONS")
	r.HandleFunc("/amass/{scan_id}/subdomain", utils.GetSubdomains).Methods("GET", "OPTIONS")
	r.HandleFunc("/amass/{scan_id}/cloud", utils.GetCloudDomains).Methods("GET", "OPTIONS")
	r.HandleFunc("/amass/{scan_id}/sp", utils.GetServiceProviders).Methods("GET", "OPTIONS")
	r.HandleFunc("/amass/{scan_id}/asn", utils.GetASNs).Methods("GET", "OPTIONS")
	r.HandleFunc("/amass/{scan_id}/subnet", utils.GetSubnets).Methods("GET", "OPTIONS")
	r.HandleFunc("/amass-intel/run", utils.RunAmassIntelScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/amass-intel/{scanID}", utils.GetAmassIntelScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/amass-intel", utils.GetAmassIntelScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/amass-intel/{scan_id}/networks", utils.GetIntelNetworkRanges).Methods("GET", "OPTIONS")
	r.HandleFunc("/amass-intel/{scan_id}/asn", utils.GetIntelASNData).Methods("GET", "OPTIONS")
	r.HandleFunc("/amass-intel/network-range/{id}", utils.DeleteIntelNetworkRange).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/amass-intel/scan/{scan_id}/network-ranges", utils.DeleteAllIntelNetworkRanges).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/httpx/run", utils.RunHttpxScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/httpx/{scanID}", utils.GetHttpxScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/httpx", utils.GetHttpxScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans", utils.GetAllScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/gau/run", utils.RunGauScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/gau/{scanID}", utils.GetGauScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/gau", utils.GetGauScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/sublist3r/run", utils.RunSublist3rScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/sublist3r/{scan_id}", utils.GetSublist3rScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/sublist3r", utils.GetSublist3rScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/assetfinder/run", utils.RunAssetfinderScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/assetfinder/{scan_id}", utils.GetAssetfinderScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/assetfinder", utils.GetAssetfinderScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/ctl/run", utils.RunCTLScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/ctl/{scan_id}", utils.GetCTLScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/ctl", utils.GetCTLScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/ctl-company/run", utils.RunCTLCompanyScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/ctl-company/{scan_id}", utils.GetCTLCompanyScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/ctl-company", utils.GetCTLCompanyScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/metabigor-company/run", utils.RunMetabigorCompanyScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/metabigor-company/{scan_id}", utils.GetMetabigorCompanyScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/metabigor-company", utils.GetMetabigorCompanyScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/metabigor-company/{scan_id}/networks", utils.GetMetabigorNetworkRanges).Methods("GET", "OPTIONS")
	r.HandleFunc("/metabigor/network-range/{id}", utils.DeleteMetabigorNetworkRange).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/metabigor/scan/{scan_id}/network-ranges", utils.DeleteAllMetabigorNetworkRanges).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/metabigor-company/{scan_id}/asn", utils.GetMetabigorASNData).Methods("GET", "OPTIONS")
	r.HandleFunc("/metabigor-netd/run", utils.RunMetabigorNetdScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/metabigor-asn/run", utils.RunMetabigorASNScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/metabigor-ip/run", utils.RunMetabigorIPIntelligence).Methods("POST", "OPTIONS")
	r.HandleFunc("/metabigor-ip/{scan_id}/intelligence", utils.GetMetabigorIPIntelligence).Methods("GET", "OPTIONS")
	r.HandleFunc("/subfinder/run", utils.RunSubfinderScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/subfinder/{scan_id}", utils.GetSubfinderScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/subfinder", utils.GetSubfinderScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/consolidate-subdomains/{id}", utils.HandleConsolidateSubdomains).Methods("GET", "OPTIONS")
	r.HandleFunc("/consolidated-subdomains/{id}", utils.GetConsolidatedSubdomains).Methods("GET", "OPTIONS")
	r.HandleFunc("/consolidate-company-domains/{id}", utils.HandleConsolidateCompanyDomains).Methods("GET", "OPTIONS")
	r.HandleFunc("/consolidated-company-domains/{id}", utils.GetConsolidatedCompanyDomains).Methods("GET", "OPTIONS")
	r.HandleFunc("/consolidate-network-ranges/{id}", utils.HandleConsolidateNetworkRanges).Methods("GET", "OPTIONS")
	r.HandleFunc("/consolidated-network-ranges/{id}", utils.GetConsolidatedNetworkRanges).Methods("GET", "OPTIONS")
	r.HandleFunc("/shuffledns/run", utils.RunShuffleDNSScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/shuffledns/{scan_id}", utils.GetShuffleDNSScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/shuffledns", utils.GetShuffleDNSScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/cewl/run", utils.RunCeWLScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/cewl/{scan_id}", utils.GetCeWLScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/cewl", utils.GetCeWLScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/cewl-urls/run", utils.RunCeWLScansForUrls).Methods("POST", "OPTIONS")
	r.HandleFunc("/cewl-wordlist/run", utils.RunShuffleDNSWithWordlist).Methods("POST", "OPTIONS")
	r.HandleFunc("/cewl-wordlist/{scan_id}", utils.GetShuffleDNSScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/scope-targets/{id}/shufflednscustom-scans", utils.GetShuffleDNSCustomScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/gospider/run", utils.RunGoSpiderScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/gospider/{scan_id}", utils.GetGoSpiderScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/gospider", utils.GetGoSpiderScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/subdomainizer/run", utils.RunSubdomainizerScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/subdomainizer/{scan_id}", utils.GetSubdomainizerScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/subdomainizer", utils.GetSubdomainizerScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/nuclei-screenshot/run", utils.RunNucleiScreenshotScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/nuclei-screenshot", utils.GetNucleiScreenshotScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/nuclei-screenshot/run", utils.RunNucleiScreenshotScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/nuclei-screenshot/{scan_id}", utils.GetNucleiScreenshotScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/scope-targets/{id}/target-urls", utils.GetTargetURLsForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/metadata/run", utils.RunMetaDataScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/metadata/{scan_id}", utils.GetMetaDataScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/metadata", utils.GetMetaDataScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/investigate/run", utils.RunInvestigateScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/investigate/{scan_id}", utils.GetInvestigateScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/investigate", utils.GetInvestigateScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/target-urls/{id}/roi-score", utils.UpdateTargetURLROIScore).Methods("PUT", "OPTIONS")
	r.HandleFunc("/user/settings", getUserSettings).Methods("GET", "OPTIONS")
	r.HandleFunc("/user/settings", updateUserSettings).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/export-data", utils.HandleExportData).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/auto-scan-state/{target_id}", getAutoScanState).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/auto-scan-state/{target_id}", updateAutoScanState).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/auto-scan-config", getAutoScanConfig).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/auto-scan-config", updateAutoScanConfig).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/auto-scan/session/start", startAutoScanSession).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/auto-scan/session/{id}", getAutoScanSession).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/auto-scan/sessions", listAutoScanSessions).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/auto-scan/session/{id}/cancel", cancelAutoScanSession).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/auto-scan/session/{id}/final-stats", updateAutoScanSessionFinalStats).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/google-dorking-domains", createGoogleDorkingDomain).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/google-dorking-domains/{target_id}", getGoogleDorkingDomains).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/google-dorking-domains/{domain_id}", deleteGoogleDorkingDomain).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/api/reverse-whois-domains", createReverseWhoisDomain).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/reverse-whois-domains/{target_id}", getReverseWhoisDomains).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/reverse-whois-domains/{domain_id}", deleteReverseWhoisDomain).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/api/api-keys", getAPIKeys).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/api-keys", createAPIKey).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/api-keys/{id}", updateAPIKey).Methods("PUT", "OPTIONS")
	r.HandleFunc("/api/api-keys/{id}", deleteAPIKey).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/securitytrails-company/run", utils.RunSecurityTrailsCompanyScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/securitytrails-company/status/{scan_id}", utils.GetSecurityTrailsCompanyScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/securitytrails-company", utils.GetSecurityTrailsCompanyScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/censys-company/run", utils.RunCensysCompanyScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/censys-company/status/{scan_id}", utils.GetCensysCompanyScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/censys-company", utils.GetCensysCompanyScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/shodan-company/run", utils.RunShodanCompanyScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/shodan-company/status/{scan_id}", utils.GetShodanCompanyScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/shodan-company", utils.GetShodanCompanyScansForScopeTarget).Methods("GET", "OPTIONS")

	// GitHub Recon routes
	r.HandleFunc("/github-recon/run", utils.RunGitHubReconScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/github-recon/status/{scan_id}", utils.GetGitHubReconScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/github-recon", utils.GetGitHubReconScansForScopeTarget).Methods("GET", "OPTIONS")

	// Company domain management routes
	r.HandleFunc("/api/company-domains/{scope_target_id}/{tool}", getCompanyDomainsByTool).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/company-domains/{scope_target_id}/{tool}/all", deleteAllCompanyDomainsFromTool).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/api/company-domains/{scope_target_id}/{tool}/{domain}", deleteCompanyDomainFromTool).Methods("DELETE", "OPTIONS")

	log.Println("API server started on :8443")
	http.ListenAndServe(":8443", r)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getUserSettings(w http.ResponseWriter, r *http.Request) {
	// Get settings from the database
	var settings map[string]interface{} = make(map[string]interface{})

	row := dbPool.QueryRow(context.Background(), `
		SELECT 
			amass_rate_limit,
			httpx_rate_limit,
			subfinder_rate_limit,
			gau_rate_limit,
			sublist3r_rate_limit,
			ctl_rate_limit,
			shuffledns_rate_limit,
			cewl_rate_limit,
			gospider_rate_limit,
			subdomainizer_rate_limit,
			nuclei_screenshot_rate_limit,
			custom_user_agent,
			custom_header
		FROM user_settings
		LIMIT 1
	`)

	var amassRateLimit, httpxRateLimit, subfinderRateLimit, gauRateLimit,
		sublist3rRateLimit, ctlRateLimit, shufflednsRateLimit,
		cewlRateLimit, gospiderRateLimit, subdomainizerRateLimit, nucleiScreenshotRateLimit int
	var customUserAgent, customHeader sql.NullString

	err := row.Scan(
		&amassRateLimit,
		&httpxRateLimit,
		&subfinderRateLimit,
		&gauRateLimit,
		&sublist3rRateLimit,
		&ctlRateLimit,
		&shufflednsRateLimit,
		&cewlRateLimit,
		&gospiderRateLimit,
		&subdomainizerRateLimit,
		&nucleiScreenshotRateLimit,
		&customUserAgent,
		&customHeader,
	)

	if err != nil {
		log.Printf("Error fetching settings: %v", err)
		// Return default settings if there's an error
		settings = map[string]interface{}{
			"amass_rate_limit":             10,
			"httpx_rate_limit":             150,
			"subfinder_rate_limit":         20,
			"gau_rate_limit":               10,
			"sublist3r_rate_limit":         10,
			"ctl_rate_limit":               10,
			"shuffledns_rate_limit":        10000,
			"cewl_rate_limit":              10,
			"gospider_rate_limit":          5,
			"subdomainizer_rate_limit":     5,
			"nuclei_screenshot_rate_limit": 20,
			"custom_user_agent":            "",
			"custom_header":                "",
		}
	} else {
		settings = map[string]interface{}{
			"amass_rate_limit":             amassRateLimit,
			"httpx_rate_limit":             httpxRateLimit,
			"subfinder_rate_limit":         subfinderRateLimit,
			"gau_rate_limit":               gauRateLimit,
			"sublist3r_rate_limit":         sublist3rRateLimit,
			"ctl_rate_limit":               ctlRateLimit,
			"shuffledns_rate_limit":        shufflednsRateLimit,
			"cewl_rate_limit":              cewlRateLimit,
			"gospider_rate_limit":          gospiderRateLimit,
			"subdomainizer_rate_limit":     subdomainizerRateLimit,
			"nuclei_screenshot_rate_limit": nucleiScreenshotRateLimit,
			"custom_user_agent":            customUserAgent.String,
			"custom_header":                customHeader.String,
		}
	}

	// Return settings as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func updateUserSettings(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	var settings map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&settings)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Log the received settings
	log.Printf("Received settings: %v", settings)

	// Update settings in the database
	_, err = dbPool.Exec(context.Background(), `
		UPDATE user_settings
		SET 
			amass_rate_limit = $1,
			httpx_rate_limit = $2,
			subfinder_rate_limit = $3,
			gau_rate_limit = $4,
			sublist3r_rate_limit = $5,
			ctl_rate_limit = $6,
			shuffledns_rate_limit = $7,
			cewl_rate_limit = $8,
			gospider_rate_limit = $9,
			subdomainizer_rate_limit = $10,
			nuclei_screenshot_rate_limit = $11,
			custom_user_agent = $12,
			custom_header = $13,
			updated_at = NOW()
	`,
		getIntSetting(settings, "amass_rate_limit", 10),
		getIntSetting(settings, "httpx_rate_limit", 150),
		getIntSetting(settings, "subfinder_rate_limit", 20),
		getIntSetting(settings, "gau_rate_limit", 10),
		getIntSetting(settings, "sublist3r_rate_limit", 10),
		getIntSetting(settings, "ctl_rate_limit", 10),
		getIntSetting(settings, "shuffledns_rate_limit", 10000),
		getIntSetting(settings, "cewl_rate_limit", 10),
		getIntSetting(settings, "gospider_rate_limit", 5),
		getIntSetting(settings, "subdomainizer_rate_limit", 5),
		getIntSetting(settings, "nuclei_screenshot_rate_limit", 20),
		settings["custom_user_agent"],
		settings["custom_header"],
	)

	if err != nil {
		log.Printf("Error updating settings: %v", err)
		http.Error(w, "Failed to update settings", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}

// Helper function to get integer settings with default values
func getIntSetting(settings map[string]interface{}, key string, defaultValue int) int {
	if val, ok := settings[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case string:
			if intVal, err := strconv.Atoi(v); err == nil {
				return intVal
			}
		}
	}
	return defaultValue
}

// getAutoScanState retrieves the current auto scan state for a target
func getAutoScanState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetID := vars["target_id"]

	// First try with new columns
	var state struct {
		ID            string    `json:"id"`
		ScopeTargetID string    `json:"scope_target_id"`
		CurrentStep   string    `json:"current_step"`
		IsPaused      bool      `json:"is_paused"`
		IsCancelled   bool      `json:"is_cancelled"`
		CreatedAt     time.Time `json:"created_at"`
		UpdatedAt     time.Time `json:"updated_at"`
	}

	err := dbPool.QueryRow(context.Background(), `
		SELECT id, scope_target_id, current_step, 
		COALESCE((SELECT column_name FROM information_schema.columns WHERE table_name='auto_scan_state' AND column_name='is_paused') IS NOT NULL AND is_paused, false) as is_paused,
		COALESCE((SELECT column_name FROM information_schema.columns WHERE table_name='auto_scan_state' AND column_name='is_cancelled') IS NOT NULL AND is_cancelled, false) as is_cancelled,
		created_at, updated_at
		FROM auto_scan_state
		WHERE scope_target_id = $1
	`, targetID).Scan(&state.ID, &state.ScopeTargetID, &state.CurrentStep, &state.IsPaused, &state.IsCancelled, &state.CreatedAt, &state.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			// No state found, return empty object with IDLE state
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"scope_target_id": targetID,
				"current_step":    "IDLE",
				"is_paused":       false,
				"is_cancelled":    false,
			})
			return
		}

		// If the error is about missing columns, try the fallback query
		if strings.Contains(err.Error(), "column") && (strings.Contains(err.Error(), "is_paused") || strings.Contains(err.Error(), "is_cancelled")) {
			var basicState struct {
				ID            string    `json:"id"`
				ScopeTargetID string    `json:"scope_target_id"`
				CurrentStep   string    `json:"current_step"`
				CreatedAt     time.Time `json:"created_at"`
				UpdatedAt     time.Time `json:"updated_at"`
			}

			fallbackErr := dbPool.QueryRow(context.Background(), `
				SELECT id, scope_target_id, current_step, created_at, updated_at
				FROM auto_scan_state
				WHERE scope_target_id = $1
			`, targetID).Scan(&basicState.ID, &basicState.ScopeTargetID, &basicState.CurrentStep, &basicState.CreatedAt, &basicState.UpdatedAt)

			if fallbackErr == nil {
				// Successfully got basic state, return it with default pause/cancel values
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":              basicState.ID,
					"scope_target_id": basicState.ScopeTargetID,
					"current_step":    basicState.CurrentStep,
					"is_paused":       false,
					"is_cancelled":    false,
					"created_at":      basicState.CreatedAt,
					"updated_at":      basicState.UpdatedAt,
				})
				return
			}

			// If even the fallback failed with no rows, return idle state
			if fallbackErr == pgx.ErrNoRows {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"scope_target_id": targetID,
					"current_step":    "IDLE",
					"is_paused":       false,
					"is_cancelled":    false,
				})
				return
			}
		}

		log.Printf("Error fetching auto scan state: %v", err)
		http.Error(w, "Failed to fetch auto scan state", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

// updateAutoScanState updates the current auto scan state for a target
func updateAutoScanState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetID := vars["target_id"]

	var requestData struct {
		CurrentStep string `json:"current_step"`
		IsPaused    bool   `json:"is_paused"`
		IsCancelled bool   `json:"is_cancelled"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if requestData.CurrentStep == "" {
		http.Error(w, "Current step is required", http.StatusBadRequest)
		return
	}

	// First check if columns exist
	var hasPausedColumn, hasCancelledColumn bool
	err = dbPool.QueryRow(context.Background(), `
		SELECT 
			EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='auto_scan_state' AND column_name='is_paused') as has_paused,
			EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='auto_scan_state' AND column_name='is_cancelled') as has_cancelled
	`).Scan(&hasPausedColumn, &hasCancelledColumn)

	if err != nil {
		log.Printf("Error checking for columns: %v", err)
		http.Error(w, "Failed to update auto scan state", http.StatusInternalServerError)
		return
	}

	var execErr error
	if hasPausedColumn && hasCancelledColumn {
		// Use upsert with all columns
		_, execErr = dbPool.Exec(context.Background(), `
			INSERT INTO auto_scan_state (scope_target_id, current_step, is_paused, is_cancelled)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (scope_target_id)
			DO UPDATE SET 
				current_step = $2, 
				is_paused = $3, 
				is_cancelled = $4, 
				updated_at = NOW()
		`, targetID, requestData.CurrentStep, requestData.IsPaused, requestData.IsCancelled)
	} else {
		// Use upsert with only current_step
		_, execErr = dbPool.Exec(context.Background(), `
			INSERT INTO auto_scan_state (scope_target_id, current_step)
			VALUES ($1, $2)
			ON CONFLICT (scope_target_id)
			DO UPDATE SET 
				current_step = $2, 
				updated_at = NOW()
		`, targetID, requestData.CurrentStep)

		// Try to add the missing columns
		_, alterErr := dbPool.Exec(context.Background(), `
			ALTER TABLE auto_scan_state 
			ADD COLUMN IF NOT EXISTS is_paused BOOLEAN DEFAULT false,
			ADD COLUMN IF NOT EXISTS is_cancelled BOOLEAN DEFAULT false;
		`)
		if alterErr != nil {
			log.Printf("Error adding missing columns: %v", alterErr)
		}
	}

	if execErr != nil {
		log.Printf("Error updating auto scan state: %v", execErr)
		http.Error(w, "Failed to update auto scan state", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":         true,
		"scope_target_id": targetID,
		"current_step":    requestData.CurrentStep,
		"is_paused":       requestData.IsPaused,
		"is_cancelled":    requestData.IsCancelled,
	})
}

// The createTables function is already defined in database.go
// func createTables() {
// 	// Create the tables if they don't exist
// 	_, err := dbPool.Exec(context.Background(), `
// 		CREATE TABLE IF NOT EXISTS scope_targets (
// 			id SERIAL PRIMARY KEY,
// 			type VARCHAR(255) NOT NULL,
// 			mode VARCHAR(255) NOT NULL,
// 			scope_target VARCHAR(255) NOT NULL,
// 			active BOOLEAN DEFAULT false,
// 			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
// 		);

// 		CREATE TABLE IF NOT EXISTS user_settings (
// 			id SERIAL PRIMARY KEY,
// 			amass_rate_limit INTEGER DEFAULT 10,
// 			httpx_rate_limit INTEGER DEFAULT 150,
// 			subfinder_rate_limit INTEGER DEFAULT 20,
// 			gau_rate_limit INTEGER DEFAULT 10,
// 			sublist3r_rate_limit INTEGER DEFAULT 10,
// 			assetfinder_rate_limit INTEGER DEFAULT 10,
// 			ctl_rate_limit INTEGER DEFAULT 10,
// 			shuffledns_rate_limit INTEGER DEFAULT 10,
// 			cewl_rate_limit INTEGER DEFAULT 10,
// 			gospider_rate_limit INTEGER DEFAULT 5,
// 			subdomainizer_rate_limit INTEGER DEFAULT 5,
// 			nuclei_screenshot_rate_limit INTEGER DEFAULT 20,
// 			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
// 			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
// 		);

// 		-- Insert default settings if none exist
// 		INSERT INTO user_settings (id)
// 		SELECT 1
// 		WHERE NOT EXISTS (SELECT 1 FROM user_settings WHERE id = 1);
// 	`)
// 	if err != nil {
// 		log.Fatalf("Error creating tables: %v", err)
// 	}
// }

func getAutoScanConfig(w http.ResponseWriter, r *http.Request) {
	row := dbPool.QueryRow(context.Background(), `
		SELECT amass, sublist3r, assetfinder, gau, ctl, subfinder, consolidate_httpx_round1, shuffledns, cewl, consolidate_httpx_round2, gospider, subdomainizer, consolidate_httpx_round3, nuclei_screenshot, metadata, max_consolidated_subdomains, max_live_web_servers
		FROM auto_scan_config
		LIMIT 1
	`)
	var config struct {
		Amass                     bool `json:"amass"`
		Sublist3r                 bool `json:"sublist3r"`
		Assetfinder               bool `json:"assetfinder"`
		Gau                       bool `json:"gau"`
		Ctl                       bool `json:"ctl"`
		Subfinder                 bool `json:"subfinder"`
		ConsolidateHttpxRound1    bool `json:"consolidate_httpx_round1"`
		Shuffledns                bool `json:"shuffledns"`
		Cewl                      bool `json:"cewl"`
		ConsolidateHttpxRound2    bool `json:"consolidate_httpx_round2"`
		Gospider                  bool `json:"gospider"`
		Subdomainizer             bool `json:"subdomainizer"`
		ConsolidateHttpxRound3    bool `json:"consolidate_httpx_round3"`
		NucleiScreenshot          bool `json:"nuclei_screenshot"`
		Metadata                  bool `json:"metadata"`
		MaxConsolidatedSubdomains int  `json:"maxConsolidatedSubdomains"`
		MaxLiveWebServers         int  `json:"maxLiveWebServers"`
	}
	err := row.Scan(
		&config.Amass,
		&config.Sublist3r,
		&config.Assetfinder,
		&config.Gau,
		&config.Ctl,
		&config.Subfinder,
		&config.ConsolidateHttpxRound1,
		&config.Shuffledns,
		&config.Cewl,
		&config.ConsolidateHttpxRound2,
		&config.Gospider,
		&config.Subdomainizer,
		&config.ConsolidateHttpxRound3,
		&config.NucleiScreenshot,
		&config.Metadata,
		&config.MaxConsolidatedSubdomains,
		&config.MaxLiveWebServers,
	)
	if err != nil {
		// Return defaults if not found
		config = struct {
			Amass                     bool `json:"amass"`
			Sublist3r                 bool `json:"sublist3r"`
			Assetfinder               bool `json:"assetfinder"`
			Gau                       bool `json:"gau"`
			Ctl                       bool `json:"ctl"`
			Subfinder                 bool `json:"subfinder"`
			ConsolidateHttpxRound1    bool `json:"consolidate_httpx_round1"`
			Shuffledns                bool `json:"shuffledns"`
			Cewl                      bool `json:"cewl"`
			ConsolidateHttpxRound2    bool `json:"consolidate_httpx_round2"`
			Gospider                  bool `json:"gospider"`
			Subdomainizer             bool `json:"subdomainizer"`
			ConsolidateHttpxRound3    bool `json:"consolidate_httpx_round3"`
			NucleiScreenshot          bool `json:"nuclei_screenshot"`
			Metadata                  bool `json:"metadata"`
			MaxConsolidatedSubdomains int  `json:"maxConsolidatedSubdomains"`
			MaxLiveWebServers         int  `json:"maxLiveWebServers"`
		}{
			Amass: true, Sublist3r: true, Assetfinder: true, Gau: true, Ctl: true, Subfinder: true, ConsolidateHttpxRound1: true, Shuffledns: true, Cewl: true, ConsolidateHttpxRound2: true, Gospider: true, Subdomainizer: true, ConsolidateHttpxRound3: true, NucleiScreenshot: true, Metadata: true, MaxConsolidatedSubdomains: 2500, MaxLiveWebServers: 500,
		}
	}
	log.Printf("[AutoScanConfig] GET: %+v", config)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

func updateAutoScanConfig(w http.ResponseWriter, r *http.Request) {
	var config struct {
		Amass                     bool `json:"amass"`
		Sublist3r                 bool `json:"sublist3r"`
		Assetfinder               bool `json:"assetfinder"`
		Gau                       bool `json:"gau"`
		Ctl                       bool `json:"ctl"`
		Subfinder                 bool `json:"subfinder"`
		ConsolidateHttpxRound1    bool `json:"consolidate_httpx_round1"`
		Shuffledns                bool `json:"shuffledns"`
		Cewl                      bool `json:"cewl"`
		ConsolidateHttpxRound2    bool `json:"consolidate_httpx_round2"`
		Gospider                  bool `json:"gospider"`
		Subdomainizer             bool `json:"subdomainizer"`
		ConsolidateHttpxRound3    bool `json:"consolidate_httpx_round3"`
		NucleiScreenshot          bool `json:"nuclei_screenshot"`
		Metadata                  bool `json:"metadata"`
		MaxConsolidatedSubdomains int  `json:"maxConsolidatedSubdomains"`
		MaxLiveWebServers         int  `json:"maxLiveWebServers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	log.Printf("[AutoScanConfig] POST: %+v", config)
	_, err := dbPool.Exec(context.Background(), `
		UPDATE auto_scan_config SET
			amass = $1,
			sublist3r = $2,
			assetfinder = $3,
			gau = $4,
			ctl = $5,
			subfinder = $6,
			consolidate_httpx_round1 = $7,
			shuffledns = $8,
			cewl = $9,
			consolidate_httpx_round2 = $10,
			gospider = $11,
			subdomainizer = $12,
			consolidate_httpx_round3 = $13,
			nuclei_screenshot = $14,
			metadata = $15,
			max_consolidated_subdomains = $16,
			max_live_web_servers = $17,
			updated_at = NOW()
		WHERE id = (SELECT id FROM auto_scan_config LIMIT 1)
	`,
		config.Amass,
		config.Sublist3r,
		config.Assetfinder,
		config.Gau,
		config.Ctl,
		config.Subfinder,
		config.ConsolidateHttpxRound1,
		config.Shuffledns,
		config.Cewl,
		config.ConsolidateHttpxRound2,
		config.Gospider,
		config.Subdomainizer,
		config.ConsolidateHttpxRound3,
		config.NucleiScreenshot,
		config.Metadata,
		config.MaxConsolidatedSubdomains,
		config.MaxLiveWebServers,
	)
	if err != nil {
		http.Error(w, "Failed to update config", http.StatusInternalServerError)
		return
	}
	getAutoScanConfig(w, r)
}

func startAutoScanSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScopeTargetID  string      `json:"scope_target_id"`
		ConfigSnapshot interface{} `json:"config_snapshot"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	var sessionID string
	err := dbPool.QueryRow(context.Background(), `
		INSERT INTO auto_scan_sessions (scope_target_id, config_snapshot, status, started_at)
		VALUES ($1, $2, 'running', NOW())
		RETURNING id
	`, req.ScopeTargetID, req.ConfigSnapshot).Scan(&sessionID)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"session_id": sessionID})
}

func getAutoScanSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	row := dbPool.QueryRow(context.Background(), `
		SELECT id, scope_target_id, config_snapshot, status, started_at, ended_at, steps_run, error_message, final_consolidated_subdomains, final_live_web_servers
		FROM auto_scan_sessions WHERE id = $1
	`, sessionID)
	var session struct {
		ID                          string      `json:"id"`
		ScopeTargetID               string      `json:"scope_target_id"`
		ConfigSnapshot              interface{} `json:"config_snapshot"`
		Status                      string      `json:"status"`
		StartedAt                   time.Time   `json:"started_at"`
		EndedAt                     *time.Time  `json:"ended_at"`
		StepsRun                    interface{} `json:"steps_run"`
		ErrorMessage                *string     `json:"error_message"`
		FinalConsolidatedSubdomains *int        `json:"final_consolidated_subdomains"`
		FinalLiveWebServers         *int        `json:"final_live_web_servers"`
	}
	err := row.Scan(&session.ID, &session.ScopeTargetID, &session.ConfigSnapshot, &session.Status, &session.StartedAt, &session.EndedAt, &session.StepsRun, &session.ErrorMessage, &session.FinalConsolidatedSubdomains, &session.FinalLiveWebServers)
	if err != nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func listAutoScanSessions(w http.ResponseWriter, r *http.Request) {
	targetID := r.URL.Query().Get("target_id")
	rows, err := dbPool.Query(context.Background(), `
		SELECT id, scope_target_id, config_snapshot, status, started_at, ended_at, steps_run, error_message, final_consolidated_subdomains, final_live_web_servers
		FROM auto_scan_sessions WHERE scope_target_id = $1 ORDER BY started_at DESC
	`, targetID)
	if err != nil {
		http.Error(w, "Failed to list sessions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var sessions []interface{}
	for rows.Next() {
		var session struct {
			ID                          string      `json:"id"`
			ScopeTargetID               string      `json:"scope_target_id"`
			ConfigSnapshot              interface{} `json:"config_snapshot"`
			Status                      string      `json:"status"`
			StartedAt                   time.Time   `json:"started_at"`
			EndedAt                     *time.Time  `json:"ended_at"`
			StepsRun                    interface{} `json:"steps_run"`
			ErrorMessage                *string     `json:"error_message"`
			FinalConsolidatedSubdomains *int        `json:"final_consolidated_subdomains"`
			FinalLiveWebServers         *int        `json:"final_live_web_servers"`
		}
		err := rows.Scan(&session.ID, &session.ScopeTargetID, &session.ConfigSnapshot, &session.Status, &session.StartedAt, &session.EndedAt, &session.StepsRun, &session.ErrorMessage, &session.FinalConsolidatedSubdomains, &session.FinalLiveWebServers)
		if err == nil {
			sessions = append(sessions, session)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func cancelAutoScanSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	log.Printf("Cancelling session %s", sessionID)

	// First, get the current status to see if it's already completed or cancelled
	var currentStatus string
	err := dbPool.QueryRow(context.Background(),
		`SELECT status FROM auto_scan_sessions WHERE id = $1`, sessionID).Scan(&currentStatus)

	if err == nil {
		log.Printf("Current status for session %s: %s", sessionID, currentStatus)

		// Don't overwrite completed status with cancelled
		if currentStatus == "completed" {
			log.Printf("Session %s is already completed, not updating to cancelled", sessionID)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true, "status": "completed", "message": "Session already completed"}`))
			return
		}
	}

	// Mark as cancelled or completed based on request type
	// If this is coming from the completed code path, mark as completed
	status := "cancelled"
	if r.URL.Query().Get("completed") == "true" {
		status = "completed"
	}

	log.Printf("Setting session %s status to %s", sessionID, status)
	_, err = dbPool.Exec(context.Background(), `
		UPDATE auto_scan_sessions SET status = $1, ended_at = NOW() WHERE id = $2
	`, status, sessionID)

	if err != nil {
		log.Printf("Error cancelling session: %v", err)
		http.Error(w, "Failed to cancel session", http.StatusInternalServerError)
		return
	}

	// Verify the update was successful
	var newStatus string
	err = dbPool.QueryRow(context.Background(), `SELECT status FROM auto_scan_sessions WHERE id = $1`,
		sessionID).Scan(&newStatus)
	if err != nil {
		log.Printf("Error verifying session update: %v", err)
	} else {
		log.Printf("Session %s status after update: %s", sessionID, newStatus)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"success": true, "status": "%s"}`, status)))
}

func updateAutoScanSessionFinalStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	log.Printf("Updating final stats for session %s", sessionID)

	var payload struct {
		FinalConsolidatedSubdomains *int   `json:"final_consolidated_subdomains"`
		FinalLiveWebServers         *int   `json:"final_live_web_servers"`
		ScopeTargetID               string `json:"scope_target_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Final stats: subdomains=%v, webservers=%v, scope_target_id=%s",
		payload.FinalConsolidatedSubdomains, payload.FinalLiveWebServers, payload.ScopeTargetID)

	// First, verify this session belongs to this scope target to prevent cross-target modification
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(), `
		SELECT scope_target_id FROM auto_scan_sessions WHERE id = $1
	`, sessionID).Scan(&scopeTargetID)

	if err != nil {
		log.Printf("Error fetching session: %v", err)
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Ensure the scope_target_id in the request matches the one in the database
	if payload.ScopeTargetID != "" && scopeTargetID != payload.ScopeTargetID {
		log.Printf("Scope target ID mismatch: %s (request) vs %s (database)", payload.ScopeTargetID, scopeTargetID)
		http.Error(w, "Scope target ID mismatch", http.StatusBadRequest)
		return
	}

	_, err = dbPool.Exec(context.Background(), `
		UPDATE auto_scan_sessions
		SET final_consolidated_subdomains = $1, 
		    final_live_web_servers = $2, 
		    ended_at = COALESCE(ended_at, NOW()),
		    status = 'completed'
		WHERE id = $3
	`, payload.FinalConsolidatedSubdomains, payload.FinalLiveWebServers, sessionID)

	if err != nil {
		log.Printf("Error updating session stats: %v", err)
		http.Error(w, "Failed to update session stats", http.StatusInternalServerError)
		return
	}

	// Verify the update was successful
	var status string
	err = dbPool.QueryRow(context.Background(), `SELECT status FROM auto_scan_sessions WHERE id = $1`, sessionID).Scan(&status)
	if err != nil {
		log.Printf("Error verifying session update: %v", err)
	} else {
		log.Printf("Session %s status after update: %s", sessionID, status)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true, "status": "completed"}`))
}

func createGoogleDorkingDomain(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScopeTargetID string `json:"scope_target_id"`
		Domain        string `json:"domain"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ScopeTargetID == "" || req.Domain == "" {
		http.Error(w, "scope_target_id and domain are required", http.StatusBadRequest)
		return
	}

	// Check if domain already exists for this scope target
	var existingCount int
	err := dbPool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM google_dorking_domains 
		WHERE scope_target_id = $1 AND LOWER(domain) = LOWER($2)
	`, req.ScopeTargetID, req.Domain).Scan(&existingCount)

	if err != nil {
		log.Printf("Error checking existing domain: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if existingCount > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("Domain \"%s\" already exists for this target", req.Domain),
		})
		return
	}

	// Insert the domain
	var domainID string
	err = dbPool.QueryRow(context.Background(), `
		INSERT INTO google_dorking_domains (scope_target_id, domain, created_at)
		VALUES ($1, $2, NOW())
		RETURNING id
	`, req.ScopeTargetID, req.Domain).Scan(&domainID)

	if err != nil {
		log.Printf("Error creating Google dorking domain: %v", err)
		http.Error(w, "Failed to create domain", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":              domainID,
		"scope_target_id": req.ScopeTargetID,
		"domain":          req.Domain,
		"success":         true,
	})
}

func getGoogleDorkingDomains(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetID := vars["target_id"]

	if targetID == "" {
		http.Error(w, "target_id is required", http.StatusBadRequest)
		return
	}

	rows, err := dbPool.Query(context.Background(), `
		SELECT id, scope_target_id, domain, created_at
		FROM google_dorking_domains
		WHERE scope_target_id = $1
		ORDER BY created_at DESC
	`, targetID)

	if err != nil {
		log.Printf("Error fetching Google dorking domains: %v", err)
		http.Error(w, "Failed to fetch domains", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var domains []map[string]interface{}
	for rows.Next() {
		var domain struct {
			ID            string    `json:"id"`
			ScopeTargetID string    `json:"scope_target_id"`
			Domain        string    `json:"domain"`
			CreatedAt     time.Time `json:"created_at"`
		}

		err := rows.Scan(&domain.ID, &domain.ScopeTargetID, &domain.Domain, &domain.CreatedAt)
		if err != nil {
			log.Printf("Error scanning Google dorking domain: %v", err)
			continue
		}

		domains = append(domains, map[string]interface{}{
			"id":              domain.ID,
			"scope_target_id": domain.ScopeTargetID,
			"domain":          domain.Domain,
			"created_at":      domain.CreatedAt,
		})
	}

	if domains == nil {
		domains = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domains)
}

func deleteGoogleDorkingDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	domainID := vars["domain_id"]

	if domainID == "" {
		http.Error(w, "domain_id is required", http.StatusBadRequest)
		return
	}

	result, err := dbPool.Exec(context.Background(), `
		DELETE FROM google_dorking_domains WHERE id = $1
	`, domainID)

	if err != nil {
		log.Printf("Error deleting Google dorking domain: %v", err)
		http.Error(w, "Failed to delete domain", http.StatusInternalServerError)
		return
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Domain not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Domain deleted successfully",
	})
}

func createReverseWhoisDomain(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScopeTargetID string `json:"scope_target_id"`
		Domain        string `json:"domain"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ScopeTargetID == "" || req.Domain == "" {
		http.Error(w, "scope_target_id and domain are required", http.StatusBadRequest)
		return
	}

	// Check if domain already exists for this scope target
	var existingCount int
	err := dbPool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM reverse_whois_domains 
		WHERE scope_target_id = $1 AND LOWER(domain) = LOWER($2)
	`, req.ScopeTargetID, req.Domain).Scan(&existingCount)

	if err != nil {
		log.Printf("Error checking existing reverse whois domain: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if existingCount > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("Domain \"%s\" already exists for this target", req.Domain),
		})
		return
	}

	// Insert the domain
	var domainID string
	err = dbPool.QueryRow(context.Background(), `
		INSERT INTO reverse_whois_domains (scope_target_id, domain, created_at)
		VALUES ($1, $2, NOW())
		RETURNING id
	`, req.ScopeTargetID, req.Domain).Scan(&domainID)

	if err != nil {
		log.Printf("Error creating reverse whois domain: %v", err)
		http.Error(w, "Failed to create domain", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":              domainID,
		"scope_target_id": req.ScopeTargetID,
		"domain":          req.Domain,
		"success":         true,
	})
}

func getReverseWhoisDomains(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetID := vars["target_id"]

	if targetID == "" {
		http.Error(w, "target_id is required", http.StatusBadRequest)
		return
	}

	rows, err := dbPool.Query(context.Background(), `
		SELECT id, scope_target_id, domain, created_at
		FROM reverse_whois_domains
		WHERE scope_target_id = $1
		ORDER BY created_at DESC
	`, targetID)

	if err != nil {
		log.Printf("Error fetching reverse whois domains: %v", err)
		http.Error(w, "Failed to fetch domains", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var domains []map[string]interface{}
	for rows.Next() {
		var domain struct {
			ID            string    `json:"id"`
			ScopeTargetID string    `json:"scope_target_id"`
			Domain        string    `json:"domain"`
			CreatedAt     time.Time `json:"created_at"`
		}

		err := rows.Scan(&domain.ID, &domain.ScopeTargetID, &domain.Domain, &domain.CreatedAt)
		if err != nil {
			log.Printf("Error scanning reverse whois domain: %v", err)
			continue
		}

		domains = append(domains, map[string]interface{}{
			"id":              domain.ID,
			"scope_target_id": domain.ScopeTargetID,
			"domain":          domain.Domain,
			"created_at":      domain.CreatedAt,
		})
	}

	if domains == nil {
		domains = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domains)
}

func deleteReverseWhoisDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	domainID := vars["domain_id"]

	if domainID == "" {
		http.Error(w, "domain_id is required", http.StatusBadRequest)
		return
	}

	result, err := dbPool.Exec(context.Background(), `
		DELETE FROM reverse_whois_domains WHERE id = $1
	`, domainID)

	if err != nil {
		log.Printf("Error deleting reverse whois domain: %v", err)
		http.Error(w, "Failed to delete domain", http.StatusInternalServerError)
		return
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Domain not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Domain deleted successfully",
	})
}

func getAPIKeys(w http.ResponseWriter, r *http.Request) {
	rows, err := dbPool.Query(context.Background(), `
		SELECT id, tool_name, api_key_name, api_key_value, created_at, updated_at
		FROM api_keys
		ORDER BY tool_name, api_key_name
	`)
	if err != nil {
		log.Printf("Error fetching API keys: %v", err)
		http.Error(w, "Failed to fetch API keys", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var apiKeys []map[string]interface{}
	for rows.Next() {
		var id, toolName, apiKeyName, apiKeyValue string
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &toolName, &apiKeyName, &apiKeyValue, &createdAt, &updatedAt)
		if err != nil {
			log.Printf("Error scanning API key row: %v", err)
			continue
		}

		// Parse the key_values JSON
		var keyValues struct {
			APIKey    string `json:"api_key"`
			AppID     string `json:"app_id"`
			AppSecret string `json:"app_secret"`
		}
		if err := json.Unmarshal([]byte(apiKeyValue), &keyValues); err != nil {
			log.Printf("Error parsing key values: %v", err)
			continue
		}

		// Mask sensitive values
		if keyValues.APIKey != "" {
			if len(keyValues.APIKey) > 4 {
				keyValues.APIKey = strings.Repeat("*", len(keyValues.APIKey)-4) + keyValues.APIKey[len(keyValues.APIKey)-4:]
			} else {
				keyValues.APIKey = strings.Repeat("*", len(keyValues.APIKey))
			}
		}
		if keyValues.AppID != "" {
			if len(keyValues.AppID) > 4 {
				keyValues.AppID = strings.Repeat("*", len(keyValues.AppID)-4) + keyValues.AppID[len(keyValues.AppID)-4:]
			} else {
				keyValues.AppID = strings.Repeat("*", len(keyValues.AppID))
			}
		}
		if keyValues.AppSecret != "" {
			if len(keyValues.AppSecret) > 4 {
				keyValues.AppSecret = strings.Repeat("*", len(keyValues.AppSecret)-4) + keyValues.AppSecret[len(keyValues.AppSecret)-4:]
			} else {
				keyValues.AppSecret = strings.Repeat("*", len(keyValues.AppSecret))
			}
		}

		apiKeys = append(apiKeys, map[string]interface{}{
			"id":           id,
			"tool_name":    toolName,
			"api_key_name": apiKeyName,
			"key_values":   keyValues,
			"created_at":   createdAt,
			"updated_at":   updatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiKeys)
}

func createAPIKey(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ToolName  string `json:"tool_name"`
		KeyName   string `json:"api_key_name"`
		KeyValues struct {
			APIKey    string `json:"api_key"`
			AppID     string `json:"app_id"`
			AppSecret string `json:"app_secret"`
		} `json:"key_values"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Printf("[ERROR] Failed to decode API key request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Log the incoming request data
	log.Printf("[DEBUG] Incoming API key request:")
	log.Printf("  Tool Name: %s", request.ToolName)
	log.Printf("  Key Name: %s", request.KeyName)
	log.Printf("  Key Values:")
	log.Printf("    API Key: %s", request.KeyValues.APIKey)
	log.Printf("    App ID: %s", request.KeyValues.AppID)
	log.Printf("    App Secret: %s", request.KeyValues.AppSecret)

	// Validate required fields
	if request.ToolName == "" || request.KeyName == "" {
		log.Printf("[ERROR] Missing required fields - Tool Name: %v, Key Name: %v", request.ToolName == "", request.KeyName == "")
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Validate key values based on tool type
	if request.ToolName == "Censys" {
		if request.KeyValues.AppID == "" || request.KeyValues.AppSecret == "" {
			log.Printf("[ERROR] Missing Censys credentials - App ID: %v, App Secret: %v", request.KeyValues.AppID == "", request.KeyValues.AppSecret == "")
			http.Error(w, "Missing Censys credentials", http.StatusBadRequest)
			return
		}
	} else {
		if request.KeyValues.APIKey == "" {
			log.Printf("[ERROR] Missing API key for tool: %s", request.ToolName)
			http.Error(w, "Missing API key", http.StatusBadRequest)
			return
		}
	}

	// Convert key_values to JSON string
	keyValuesJSON, err := json.Marshal(request.KeyValues)
	if err != nil {
		log.Printf("[ERROR] Failed to marshal key values: %v", err)
		http.Error(w, "Failed to process key values", http.StatusInternalServerError)
		return
	}

	// Log the data being stored
	log.Printf("[DEBUG] Storing API key in database:")
	log.Printf("  Tool Name: %s", request.ToolName)
	log.Printf("  Key Name: %s", request.KeyName)
	log.Printf("  Key Values JSON: %s", string(keyValuesJSON))

	// Try to insert the API key
	_, err = dbPool.Exec(context.Background(), `
		INSERT INTO api_keys (tool_name, api_key_name, api_key_value)
		VALUES ($1, $2, $3)
	`, request.ToolName, request.KeyName, string(keyValuesJSON))

	if err != nil {
		// Check if this is a unique constraint violation
		if strings.Contains(err.Error(), "unique constraint") {
			log.Printf("[ERROR] API key with name '%s' already exists for tool '%s'", request.KeyName, request.ToolName)
			http.Error(w, fmt.Sprintf("An API key with name '%s' already exists for %s", request.KeyName, request.ToolName), http.StatusConflict)
			return
		}
		// Any other database error
		log.Printf("[ERROR] Failed to store API key: %v", err)
		http.Error(w, "Failed to store API key", http.StatusInternalServerError)
		return
	}

	log.Printf("[DEBUG] API key stored successfully")
	w.WriteHeader(http.StatusCreated)
}

func updateAPIKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var request struct {
		APIKeyName  string `json:"api_key_name"`
		APIKeyValue string `json:"api_key_value"`
	}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.APIKeyName == "" || request.APIKeyValue == "" {
		http.Error(w, "api_key_name and api_key_value are required", http.StatusBadRequest)
		return
	}

	result, err := dbPool.Exec(context.Background(), `
		UPDATE api_keys 
		SET api_key_name = $1, api_key_value = $2, updated_at = NOW()
		WHERE id = $3
	`, request.APIKeyName, request.APIKeyValue, id)

	if err != nil {
		log.Printf("Error updating API key: %v", err)
		http.Error(w, "Failed to update API key", http.StatusInternalServerError)
		return
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "API key not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "API key updated successfully"})
}

func deleteAPIKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	result, err := dbPool.Exec(context.Background(), `
		DELETE FROM api_keys WHERE id = $1
	`, id)

	if err != nil {
		log.Printf("Error deleting API key: %v", err)
		http.Error(w, "Failed to delete API key", http.StatusInternalServerError)
		return
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "API key not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "API key deleted successfully"})
}

func getCompanyDomainsByTool(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["scope_target_id"]
	tool := vars["tool"]

	if scopeTargetID == "" || tool == "" {
		http.Error(w, "scope_target_id and tool are required", http.StatusBadRequest)
		return
	}

	var domains []string
	var err error

	switch tool {
	case "google_dorking":
		domains, err = utils.GetGoogleDorkingDomainsForTool(scopeTargetID)
	case "reverse_whois":
		domains, err = utils.GetReverseWhoisDomainsForTool(scopeTargetID)
	case "ctl_company":
		domains, err = utils.GetCTLCompanyDomainsForTool(scopeTargetID)
	case "securitytrails_company":
		domains, err = utils.GetSecurityTrailsCompanyDomainsForTool(scopeTargetID)
	case "censys_company":
		domains, err = utils.GetCensysCompanyDomainsForTool(scopeTargetID)
	case "github_recon":
		domains, err = utils.GetGitHubReconDomainsForTool(scopeTargetID)
	case "shodan_company":
		domains, err = utils.GetShodanCompanyDomainsForTool(scopeTargetID)
	default:
		http.Error(w, "Invalid tool specified", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("Error fetching domains for tool %s: %v", tool, err)
		http.Error(w, "Failed to fetch domains", http.StatusInternalServerError)
		return
	}

	if domains == nil {
		domains = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"domains": domains,
		"count":   len(domains),
	})
}

func deleteCompanyDomainFromTool(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["scope_target_id"]
	tool := vars["tool"]
	domain := vars["domain"]

	log.Printf("[DOMAIN-API] [DEBUG] Individual delete request: scope_target_id=%s, tool=%s, domain='%s'", scopeTargetID, tool, domain)

	if scopeTargetID == "" || tool == "" || domain == "" {
		log.Printf("[DOMAIN-API] [ERROR] Missing required parameters: scope_target_id='%s', tool='%s', domain='%s'", scopeTargetID, tool, domain)
		http.Error(w, "scope_target_id, tool, and domain are required", http.StatusBadRequest)
		return
	}

	var err error
	var success bool

	log.Printf("[DOMAIN-API] [DEBUG] Processing individual delete for tool: %s, domain: '%s'", tool, domain)

	switch tool {
	case "google_dorking":
		success, err = utils.DeleteGoogleDorkingDomainFromTool(scopeTargetID, domain)
	case "reverse_whois":
		success, err = utils.DeleteReverseWhoisDomainFromTool(scopeTargetID, domain)
	case "ctl_company":
		success, err = utils.DeleteCTLCompanyDomainFromTool(scopeTargetID, domain)
	case "securitytrails_company":
		success, err = utils.DeleteSecurityTrailsCompanyDomainFromTool(scopeTargetID, domain)
	case "censys_company":
		success, err = utils.DeleteCensysCompanyDomainFromTool(scopeTargetID, domain)
	case "github_recon":
		success, err = utils.DeleteGitHubReconDomainFromTool(scopeTargetID, domain)
	case "shodan_company":
		success, err = utils.DeleteShodanCompanyDomainFromTool(scopeTargetID, domain)
	default:
		log.Printf("[DOMAIN-API] [ERROR] Invalid tool specified: %s", tool)
		http.Error(w, "Invalid tool specified", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("[DOMAIN-API] [ERROR] Error deleting domain '%s' from tool %s: %v", domain, tool, err)
		http.Error(w, "Failed to delete domain", http.StatusInternalServerError)
		return
	}

	if !success {
		log.Printf("[DOMAIN-API] [WARNING] Domain '%s' not found in tool %s", domain, tool)
		http.Error(w, "Domain not found", http.StatusNotFound)
		return
	}

	log.Printf("[DOMAIN-API] [INFO] Successfully deleted domain '%s' from tool %s", domain, tool)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Domain deleted successfully",
	})
}

func deleteAllCompanyDomainsFromTool(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["scope_target_id"]
	tool := vars["tool"]

	log.Printf("[DOMAIN-API] [DEBUG] Delete all domains request: scope_target_id=%s, tool=%s", scopeTargetID, tool)
	log.Printf("[DOMAIN-API] [DEBUG] Request URL: %s", r.URL.Path)

	if scopeTargetID == "" || tool == "" {
		log.Printf("[DOMAIN-API] [ERROR] Missing parameters: scope_target_id=%s, tool=%s", scopeTargetID, tool)
		http.Error(w, "scope_target_id and tool are required", http.StatusBadRequest)
		return
	}

	var err error
	var count int64

	log.Printf("[DOMAIN-API] [DEBUG] Processing delete all for tool: %s", tool)

	switch tool {
	case "google_dorking":
		count, err = utils.DeleteAllGoogleDorkingDomainsFromTool(scopeTargetID)
	case "reverse_whois":
		count, err = utils.DeleteAllReverseWhoisDomainsFromTool(scopeTargetID)
	case "ctl_company":
		log.Printf("[DOMAIN-API] [DEBUG] Calling DeleteAllCTLCompanyDomainsFromTool")
		count, err = utils.DeleteAllCTLCompanyDomainsFromTool(scopeTargetID)
	case "securitytrails_company":
		count, err = utils.DeleteAllSecurityTrailsCompanyDomainsFromTool(scopeTargetID)
	case "censys_company":
		count, err = utils.DeleteAllCensysCompanyDomainsFromTool(scopeTargetID)
	case "github_recon":
		count, err = utils.DeleteAllGitHubReconDomainsFromTool(scopeTargetID)
	case "shodan_company":
		count, err = utils.DeleteAllShodanCompanyDomainsFromTool(scopeTargetID)
	default:
		log.Printf("[DOMAIN-API] [ERROR] Invalid tool specified: %s", tool)
		http.Error(w, "Invalid tool specified", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("[DOMAIN-API] [ERROR] Error deleting all domains from tool %s: %v", tool, err)
		http.Error(w, "Failed to delete domains", http.StatusInternalServerError)
		return
	}

	log.Printf("[DOMAIN-API] [INFO] Successfully deleted %d domains from tool %s", count, tool)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Deleted %d domains successfully", count),
		"count":   count,
	})
}
