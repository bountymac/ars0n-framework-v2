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
	r.HandleFunc("/subfinder/run", utils.RunSubfinderScan).Methods("POST", "OPTIONS")
	r.HandleFunc("/subfinder/{scan_id}", utils.GetSubfinderScanStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/scopetarget/{id}/scans/subfinder", utils.GetSubfinderScansForScopeTarget).Methods("GET", "OPTIONS")
	r.HandleFunc("/consolidate-subdomains/{id}", utils.HandleConsolidateSubdomains).Methods("GET", "OPTIONS")
	r.HandleFunc("/consolidated-subdomains/{id}", utils.GetConsolidatedSubdomains).Methods("GET", "OPTIONS")
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

	log.Println("API server started on :8080")
	http.ListenAndServe(":8080", r)
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

	var state struct {
		ID            string    `json:"id"`
		ScopeTargetID string    `json:"scope_target_id"`
		CurrentStep   string    `json:"current_step"`
		CreatedAt     time.Time `json:"created_at"`
		UpdatedAt     time.Time `json:"updated_at"`
	}

	err := dbPool.QueryRow(context.Background(), `
		SELECT id, scope_target_id, current_step, created_at, updated_at
		FROM auto_scan_state
		WHERE scope_target_id = $1
	`, targetID).Scan(&state.ID, &state.ScopeTargetID, &state.CurrentStep, &state.CreatedAt, &state.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			// No state found, return empty object with IDLE state
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"scope_target_id": targetID,
				"current_step":    "IDLE",
			})
			return
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

	// Use upsert (insert or update)
	_, err = dbPool.Exec(context.Background(), `
		INSERT INTO auto_scan_state (scope_target_id, current_step)
		VALUES ($1, $2)
		ON CONFLICT (scope_target_id)
		DO UPDATE SET current_step = $2, updated_at = NOW()
	`, targetID, requestData.CurrentStep)

	if err != nil {
		log.Printf("Error updating auto scan state: %v", err)
		http.Error(w, "Failed to update auto scan state", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":         true,
		"scope_target_id": targetID,
		"current_step":    requestData.CurrentStep,
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
	_, err := dbPool.Exec(context.Background(), `
		UPDATE auto_scan_sessions SET status = 'cancelled', ended_at = NOW() WHERE id = $1
	`, sessionID)
	if err != nil {
		http.Error(w, "Failed to cancel session", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}

func updateAutoScanSessionFinalStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	var payload struct {
		FinalConsolidatedSubdomains *int `json:"final_consolidated_subdomains"`
		FinalLiveWebServers         *int `json:"final_live_web_servers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	_, err := dbPool.Exec(context.Background(), `
		UPDATE auto_scan_sessions
		SET final_consolidated_subdomains = $1, 
		    final_live_web_servers = $2, 
		    ended_at = COALESCE(ended_at, NOW()),
		    status = 'completed'
		WHERE id = $3
	`, payload.FinalConsolidatedSubdomains, payload.FinalLiveWebServers, sessionID)
	if err != nil {
		http.Error(w, "Failed to update session stats", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true}`))
}
