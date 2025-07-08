package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4"
)

type KatanaCompanyScanStatus struct {
	ID                string    `json:"id"`
	ScanID            string    `json:"scan_id"`
	ScopeTargetID     string    `json:"scope_target_id"`
	Domains           []string  `json:"domains"`
	Status            string    `json:"status"`
	Result            *string   `json:"result"`
	Error             *string   `json:"error"`
	StdOut            *string   `json:"stdout"`
	StdErr            *string   `json:"stderr"`
	Command           *string   `json:"command"`
	ExecTime          *string   `json:"execution_time"`
	CreatedAt         time.Time `json:"created_at"`
	AutoScanSessionID *string   `json:"auto_scan_session_id"`
}

type KatanaCloudAsset struct {
	Domain      string `json:"domain"`
	URL         string `json:"url"`
	Type        string `json:"type"`
	Service     string `json:"service"`
	Description string `json:"description"`
}

type KatanaCloudFinding struct {
	Domain        string `json:"domain"`
	URL           string `json:"url"`
	Type          string `json:"type"`
	Content       string `json:"content"`
	Description   string `json:"description"`
	CloudService  string `json:"cloud_service"`
	ContextBefore string `json:"context_before"`
	ContextAfter  string `json:"context_after"`
	MatchPosition int    `json:"match_position"`
}

func RunKatanaCompanyScan(w http.ResponseWriter, r *http.Request) {
	log.Printf("[KATANA-COMPANY] [INFO] Starting Katana Company scan request handling")
	var payload struct {
		Domains           []string `json:"domains" binding:"required"`
		AutoScanSessionID *string  `json:"auto_scan_session_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || len(payload.Domains) == 0 {
		log.Printf("[KATANA-COMPANY] [ERROR] Invalid request body: %v", err)
		http.Error(w, "Invalid request body. `domains` array is required.", http.StatusBadRequest)
		return
	}

	scopeTargetID := mux.Vars(r)["scope_target_id"]
	if scopeTargetID == "" {
		http.Error(w, "Scope target ID is required", http.StatusBadRequest)
		return
	}

	var scopeTarget string
	err := dbPool.QueryRow(context.Background(),
		`SELECT scope_target FROM scope_targets WHERE id = $1 AND type = 'Company'`,
		scopeTargetID).Scan(&scopeTarget)
	if err != nil {
		log.Printf("[KATANA-COMPANY] [ERROR] No matching company scope target found for ID %s", scopeTargetID)
		http.Error(w, "No matching company scope target found.", http.StatusBadRequest)
		return
	}
	log.Printf("[KATANA-COMPANY] [INFO] Found company scope target: %s", scopeTarget)

	scanID := uuid.New().String()
	log.Printf("[KATANA-COMPANY] [INFO] Generated new scan ID: %s", scanID)

	domainsJSON, _ := json.Marshal(payload.Domains)

	var insertQuery string
	var args []interface{}
	if payload.AutoScanSessionID != nil && *payload.AutoScanSessionID != "" {
		insertQuery = `INSERT INTO katana_company_scans (scan_id, scope_target_id, domains, status, auto_scan_session_id) VALUES ($1, $2, $3, $4, $5)`
		args = []interface{}{scanID, scopeTargetID, string(domainsJSON), "pending", *payload.AutoScanSessionID}
	} else {
		insertQuery = `INSERT INTO katana_company_scans (scan_id, scope_target_id, domains, status) VALUES ($1, $2, $3, $4)`
		args = []interface{}{scanID, scopeTargetID, string(domainsJSON), "pending"}
	}
	_, err = dbPool.Exec(context.Background(), insertQuery, args...)
	if err != nil {
		log.Printf("[KATANA-COMPANY] [ERROR] Failed to create scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}

	// Verify the scan record was created
	var verifyID string
	err = dbPool.QueryRow(context.Background(),
		`SELECT scan_id FROM katana_company_scans WHERE scan_id = $1`,
		scanID).Scan(&verifyID)
	if err != nil {
		log.Printf("[KATANA-COMPANY] [ERROR] Failed to verify scan record creation: %v", err)
		http.Error(w, "Failed to verify scan record creation.", http.StatusInternalServerError)
		return
	}
	log.Printf("[KATANA-COMPANY] [INFO] Scan record verified in database with ID: %s", verifyID)

	go ExecuteKatanaCompanyScan(scanID, payload.Domains, scopeTargetID)

	log.Printf("[KATANA-COMPANY] [INFO] Katana Company scan initiated successfully, returning scan ID: %s", scanID)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func ExecuteKatanaCompanyScan(scanID string, domains []string, scopeTargetID string) {
	log.Printf("[KATANA-COMPANY] [INFO] Starting Katana Company scan execution (scan ID: %s) for %d domains", scanID, len(domains))
	startTime := time.Now()

	// Ensure scan status is always updated, even if function panics or errors
	var scanCompleted bool
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[KATANA-COMPANY] [ERROR] Panic during scan execution: %v", r)
			execTime := time.Since(startTime).String()
			UpdateKatanaCompanyScanStatus(scanID, "failed", "", fmt.Sprintf("Panic during execution: %v", r), "", execTime)
		} else if !scanCompleted {
			log.Printf("[KATANA-COMPANY] [ERROR] Scan did not complete normally")
			execTime := time.Since(startTime).String()
			UpdateKatanaCompanyScanStatus(scanID, "failed", "", "Scan did not complete normally", "", execTime)
		}
	}()

	UpdateKatanaCompanyScanStatus(scanID, "running", "", "", "", "")

	// Delete existing results for only the domains being scanned (domain-centric approach)
	for _, domain := range domains {
		log.Printf("[KATANA-COMPANY] [INFO] Clearing existing results for domain: %s", domain)

		// Delete existing cloud assets for this specific domain
		_, err := dbPool.Exec(context.Background(),
			`DELETE FROM katana_company_cloud_assets WHERE scope_target_id = $1 AND root_domain = $2`,
			scopeTargetID, domain)
		if err != nil {
			log.Printf("[KATANA-COMPANY] [ERROR] Failed to delete existing cloud assets for domain %s: %v", domain, err)
		}

		// Delete existing cloud findings for this specific domain
		_, err = dbPool.Exec(context.Background(),
			`DELETE FROM katana_company_cloud_findings WHERE scope_target_id = $1 AND root_domain = $2`,
			scopeTargetID, domain)
		if err != nil {
			log.Printf("[KATANA-COMPANY] [ERROR] Failed to delete existing cloud findings for domain %s: %v", domain, err)
		}

		// Delete existing domain result entry
		_, err = dbPool.Exec(context.Background(),
			`DELETE FROM katana_company_domain_results WHERE scope_target_id = $1 AND domain = $2`,
			scopeTargetID, domain)
		if err != nil {
			log.Printf("[KATANA-COMPANY] [ERROR] Failed to delete existing domain result for domain %s: %v", domain, err)
		}
	}

	var allCloudAssets []KatanaCloudAsset
	var allCloudFindings []KatanaCloudFinding
	var commandsExecuted []string

	for i, domain := range domains {
		log.Printf("[KATANA-COMPANY] [INFO] Processing domain %d/%d: %s", i+1, len(domains), domain)

		// Ensure domain has protocol
		targetURL := domain
		if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
			targetURL = "https://" + domain
		}

		cmd := exec.Command(
			"docker", "exec", "ars0n-framework-v2-katana-1",
			"katana",
			"-u", targetURL,
			"-d", "3",
			"-jc",
			"-j",
			"-v",
			"-timeout", "120",
			"-c", "20",
			"-p", "20",
			"-retry", "3",
			"-rd", "1",
			"-rl", "10",
		)

		commandsExecuted = append(commandsExecuted, cmd.String())
		log.Printf("[KATANA-COMPANY] [INFO] Executing command: %s", cmd.String())

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			log.Printf("[KATANA-COMPANY] [ERROR] Katana scan failed for domain %s: %v", domain, err)
			log.Printf("[KATANA-COMPANY] [ERROR] stderr output: %s", stderr.String())
			continue
		}

		result := stdout.String()
		log.Printf("[KATANA-COMPANY] [INFO] Katana scan completed for domain %s", domain)
		log.Printf("[KATANA-COMPANY] [DEBUG] Raw output length: %d bytes", len(result))

		// Store domain result in the new domain-centric table
		InsertKatanaDomainResult(scopeTargetID, domain, scanID, result)

		if result != "" {
			cloudAssets, cloudFindings := ParseKatanaResultsDomainCentric(scopeTargetID, domain, result)
			allCloudAssets = append(allCloudAssets, cloudAssets...)
			allCloudFindings = append(allCloudFindings, cloudFindings...)
		}
	}

	log.Printf("[KATANA-COMPANY] [INFO] Found %d cloud assets and %d cloud findings", len(allCloudAssets), len(allCloudFindings))

	// Deduplicate cloud assets before counting (for scan summary)
	// Use the actual cloud asset identifier as the unique key to preserve multiple assets from same URL
	uniqueCloudAssets := make(map[string]KatanaCloudAsset)
	for _, cloudAsset := range allCloudAssets {
		// Extract the cloud asset identifier from the description (after the last ": ")
		parts := strings.Split(cloudAsset.Description, ": ")
		cloudAssetIdentifier := cloudAsset.Description
		if len(parts) > 1 {
			cloudAssetIdentifier = parts[len(parts)-1]
		}
		key := fmt.Sprintf("%s_%s:%s", cloudAsset.Service, cloudAsset.Type, cloudAssetIdentifier)
		uniqueCloudAssets[key] = cloudAsset
	}

	// Convert back to slice for counting
	deduplicatedCloudAssets := make([]KatanaCloudAsset, 0, len(uniqueCloudAssets))
	for _, cloudAsset := range uniqueCloudAssets {
		deduplicatedCloudAssets = append(deduplicatedCloudAssets, cloudAsset)
	}

	// Deduplicate cloud findings before counting (for scan summary)
	uniqueCloudFindings := make(map[string]KatanaCloudFinding)
	for _, cloudFinding := range allCloudFindings {
		key := fmt.Sprintf("%s_%s_%s", cloudFinding.URL, cloudFinding.Type, cloudFinding.Content)
		uniqueCloudFindings[key] = cloudFinding
	}

	// Convert back to slice for counting
	deduplicatedCloudFindings := make([]KatanaCloudFinding, 0, len(uniqueCloudFindings))
	for _, cloudFinding := range uniqueCloudFindings {
		deduplicatedCloudFindings = append(deduplicatedCloudFindings, cloudFinding)
	}

	log.Printf("[KATANA-COMPANY] [INFO] Deduplicated cloud assets: %d unique assets from %d total discoveries", len(deduplicatedCloudAssets), len(allCloudAssets))
	log.Printf("[KATANA-COMPANY] [INFO] Deduplicated cloud findings: %d unique findings from %d total discoveries", len(deduplicatedCloudFindings), len(allCloudFindings))

	result := map[string]interface{}{
		"cloud_assets":    deduplicatedCloudAssets,
		"cloud_findings":  deduplicatedCloudFindings,
		"domains_scanned": len(domains),
		"summary": map[string]int{
			"total_cloud_assets":   len(deduplicatedCloudAssets),
			"aws_assets":           countAssetsByService(deduplicatedCloudAssets, "aws"),
			"gcp_assets":           countAssetsByService(deduplicatedCloudAssets, "gcp"),
			"azure_assets":         countAssetsByService(deduplicatedCloudAssets, "azure"),
			"other_assets":         countAssetsByService(deduplicatedCloudAssets, "other"),
			"total_cloud_findings": len(deduplicatedCloudFindings),
		},
	}

	resultJSON, _ := json.Marshal(result)
	execTime := time.Since(startTime).String()
	commandsStr := strings.Join(commandsExecuted, "; ")

	UpdateKatanaCompanyScanStatus(scanID, "success", string(resultJSON), "", commandsStr, execTime)
	scanCompleted = true // Mark scan as completed

	log.Printf("[KATANA-COMPANY] [INFO] Katana Company scan completed (scan ID: %s) in %s", scanID, execTime)
}

func ParseKatanaCloudResults(domain, result string) ([]KatanaCloudAsset, []KatanaCloudFinding) {
	log.Printf("[KATANA-COMPANY] [INFO] Starting to parse Katana results for domain %s", domain)

	// Use maps for deduplication
	uniqueAssets := make(map[string]KatanaCloudAsset)
	uniqueFindings := make(map[string]KatanaCloudFinding)

	lines := strings.Split(result, "\n")
	log.Printf("[KATANA-COMPANY] [INFO] Processing %d lines of output", len(lines))

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		log.Printf("[KATANA-COMPANY] [DEBUG] Processing line %d: %s", lineNum+1, line)

		var katanaResult struct {
			Timestamp string `json:"timestamp"`
			Request   struct {
				Method   string `json:"method"`
				Endpoint string `json:"endpoint"`
				Source   string `json:"source"`
			} `json:"request"`
			Response struct {
				StatusCode int                    `json:"status_code"`
				Headers    map[string]interface{} `json:"headers"`
				Body       string                 `json:"body"`
			} `json:"response"`
		}

		if err := json.Unmarshal([]byte(line), &katanaResult); err != nil {
			log.Printf("[KATANA-COMPANY] [DEBUG] Failed to parse JSON, treating as plain URL: %v", err)
			if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
				assets, findings := analyzeURLForCloudAssets(domain, line, "")
				addUniqueAssets(uniqueAssets, assets)
				addUniqueFindings(uniqueFindings, findings)
			}
			continue
		}

		assets, findings := analyzeURLForCloudAssets(domain, katanaResult.Request.Endpoint, katanaResult.Response.Body)
		addUniqueAssets(uniqueAssets, assets)
		addUniqueFindings(uniqueFindings, findings)

		// Also analyze response headers for cloud assets
		if katanaResult.Response.Headers != nil {
			headerAssets, headerFindings := analyzeHeadersForCloudAssets(domain, katanaResult.Request.Endpoint, katanaResult.Response.Headers)
			addUniqueAssets(uniqueAssets, headerAssets)
			addUniqueFindings(uniqueFindings, headerFindings)
		}

		if katanaResult.Request.Source != "" && katanaResult.Request.Source != katanaResult.Request.Endpoint {
			sourceAssets, sourceFindings := analyzeURLForCloudAssets(domain, katanaResult.Request.Source, "")
			addUniqueAssets(uniqueAssets, sourceAssets)
			addUniqueFindings(uniqueFindings, sourceFindings)
		}
	}

	// Convert maps back to slices
	var cloudAssets []KatanaCloudAsset
	var cloudFindings []KatanaCloudFinding

	for _, asset := range uniqueAssets {
		cloudAssets = append(cloudAssets, asset)
	}

	for _, finding := range uniqueFindings {
		cloudFindings = append(cloudFindings, finding)
	}

	log.Printf("[KATANA-COMPANY] [INFO] Completed parsing results for domain %s - found %d unique cloud assets, %d unique cloud findings", domain, len(cloudAssets), len(cloudFindings))
	return cloudAssets, cloudFindings
}

func addUniqueAssets(uniqueAssets map[string]KatanaCloudAsset, assets []KatanaCloudAsset) {
	for _, asset := range assets {
		// Create unique key based on the actual cloud asset URL
		// This allows multiple cloud assets from the same source URL to be preserved
		key := fmt.Sprintf("%s_%s:%s", asset.Service, asset.Type, asset.URL)
		uniqueAssets[key] = asset
	}
}

func addUniqueFindings(uniqueFindings map[string]KatanaCloudFinding, findings []KatanaCloudFinding) {
	for _, finding := range findings {
		// Create unique key based on cloud service, type, and content
		key := fmt.Sprintf("%s:%s:%s", finding.CloudService, finding.Type, finding.Content)
		uniqueFindings[key] = finding
	}
}

func analyzeURLForCloudAssets(domain, url, body string) ([]KatanaCloudAsset, []KatanaCloudFinding) {
	var assets []KatanaCloudAsset
	var findings []KatanaCloudFinding

	// Use a map to track unique assets within this URL
	uniqueURLAssets := make(map[string]bool)

	cloudPatterns := map[string]map[string]string{
		"aws": {
			"s3":                `([a-zA-Z0-9\-\.]+\.s3[\-\.][a-zA-Z0-9\-]*\.amazonaws\.com|[a-zA-Z0-9\-\.]+\.s3\.amazonaws\.com|s3\.amazonaws\.com\/[a-zA-Z0-9\-\.]+)`,
			"cloudfront":        `([a-zA-Z0-9\-\.]+\.cloudfront\.net)`,
			"lambda":            `([a-zA-Z0-9\-\.]+\.lambda\.amazonaws\.com)`,
			"apigateway":        `([a-zA-Z0-9\-\.]+\.execute-api\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"apigateway_v2":     `([a-zA-Z0-9\-\.]+\.execute-api\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"elasticbeanstalk":  `([a-zA-Z0-9\-\.]+\.elasticbeanstalk\.com)`,
			"elb":               `([a-zA-Z0-9\-\.]+\.elb\.amazonaws\.com)`,
			"alb_nlb":           `([a-zA-Z0-9\-\.]+\.elb\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"rds":               `([a-zA-Z0-9\-\.]+\.rds\.amazonaws\.com)`,
			"dynamodb":          `([a-zA-Z0-9\-\.]+\.dynamodb\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"ec2":               `(ec2-[0-9\-]+\.[a-zA-Z0-9\-]+\.compute\.amazonaws\.com)`,
			"ecs":               `([a-zA-Z0-9\-\.]+\.ecs\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"eks":               `([a-zA-Z0-9\-\.]+\.eks\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"sns":               `([a-zA-Z0-9\-\.]+\.sns\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"sqs":               `([a-zA-Z0-9\-\.]+\.sqs\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"ses":               `([a-zA-Z0-9\-\.]+\.ses\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"redshift":          `([a-zA-Z0-9\-\.]+\.redshift\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"elasticache":       `([a-zA-Z0-9\-\.]+\.cache\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"kinesis":           `([a-zA-Z0-9\-\.]+\.kinesis\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"amplify":           `([a-zA-Z0-9\-\.]+\.amplifyapp\.com)`,
			"appsync":           `([a-zA-Z0-9\-\.]+\.appsync-api\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"cloudwatch":        `([a-zA-Z0-9\-\.]+\.cloudwatch\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"cloudformation":    `([a-zA-Z0-9\-\.]+\.cloudformation\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"stepfunctions":     `([a-zA-Z0-9\-\.]+\.states\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"secretsmanager":    `([a-zA-Z0-9\-\.]+\.secretsmanager\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"iot":               `([a-zA-Z0-9\-\.]+\.iot\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"eventbridge":       `([a-zA-Z0-9\-\.]+\.events\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"xray":              `([a-zA-Z0-9\-\.]+\.xray\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"cloudsearch":       `([a-zA-Z0-9\-\.]+\.cloudsearch\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"elastictranscoder": `([a-zA-Z0-9\-\.]+\.elastictranscoder\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"elasticinference":  `([a-zA-Z0-9\-\.]+\.elasticinference\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"neptune":           `([a-zA-Z0-9\-\.]+\.neptune\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"workspaces":        `([a-zA-Z0-9\-\.]+\.workspaces\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"directconnect":     `([a-zA-Z0-9\-\.]+\.directconnect\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"mobilehub":         `([a-zA-Z0-9\-\.]+\.mobilehub\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"macie":             `([a-zA-Z0-9\-\.]+\.macie\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"sagemaker":         `([a-zA-Z0-9\-\.]+\.sagemaker\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"workdocs":          `([a-zA-Z0-9\-\.]+\.workdocs\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"transcribe":        `([a-zA-Z0-9\-\.]+\.transcribe\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"translate":         `([a-zA-Z0-9\-\.]+\.translate\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"appmesh":           `([a-zA-Z0-9\-\.]+\.appmesh\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"inspector":         `([a-zA-Z0-9\-\.]+\.inspector\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"glue":              `([a-zA-Z0-9\-\.]+\.glue\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"connect":           `([a-zA-Z0-9\-\.]+\.awsapps\.com)`,
			"chime":             `([a-zA-Z0-9\-\.]+\.chime\.aws)`,
			"efs":               `([a-zA-Z0-9\-\.]+\.efs\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"batch":             `([a-zA-Z0-9\-\.]+\.batch\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"kafka":             `([a-zA-Z0-9\-\.]+\.kafka\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"medialive":         `([a-zA-Z0-9\-\.]+\.medialive\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"snowball":          `([a-zA-Z0-9\-\.]+\.snowball\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"cloudtrail":        `([a-zA-Z0-9\-\.]+\.cloudtrail\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"datasync":          `([a-zA-Z0-9\-\.]+\.datasync\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
		},
		"gcp": {
			"storage":             `([a-zA-Z0-9\-\.]+\.storage\.googleapis\.com|storage\.googleapis\.com\/[a-zA-Z0-9\-\.]+)`,
			"appengine":           `([a-zA-Z0-9\-\.]+\.appspot\.com)`,
			"cloudfunctions":      `([a-zA-Z0-9\-\.]+\.cloudfunctions\.net)`,
			"cloudrun":            `([a-zA-Z0-9\-\.]+\.run\.app)`,
			"firestore":           `([a-zA-Z0-9\-\.]+\.firebaseio\.com)`,
			"firebase":            `([a-zA-Z0-9\-\.]+\.firebaseapp\.com)`,
			"gke":                 `([a-zA-Z0-9\-\.]+\.container\.googleapis\.com)`,
			"compute":             `([a-zA-Z0-9\-\.]+\.compute\.googleapis\.com)`,
			"sql":                 `([a-zA-Z0-9\-\.]+\.sql\.googleapis\.com)`,
			"bigquery":            `([a-zA-Z0-9\-\.]+\.bigquery\.googleapis\.com)`,
			"googleusercontent":   `([a-zA-Z0-9\-\.]+\.googleusercontent\.com)`,
			"pubsub":              `([a-zA-Z0-9\-\.]+\.pubsub\.googleapis\.com)`,
			"bigtable":            `([a-zA-Z0-9\-\.]+\.bigtable\.googleapis\.com)`,
			"spanner":             `([a-zA-Z0-9\-\.]+\.spanner\.googleapis\.com)`,
			"dataflow":            `([a-zA-Z0-9\-\.]+\.dataflow\.googleapis\.com)`,
			"identityplatform":    `([a-zA-Z0-9\-\.]+\.identityplatform\.googleapis\.com)`,
			"firestore_api":       `([a-zA-Z0-9\-\.]+\.firestore\.googleapis\.com)`,
			"datastore":           `([a-zA-Z0-9\-\.]+\.datastore\.googleapis\.com)`,
			"monitoring":          `([a-zA-Z0-9\-\.]+\.monitoring\.googleapis\.com)`,
			"logging":             `([a-zA-Z0-9\-\.]+\.logging\.googleapis\.com)`,
			"speech":              `([a-zA-Z0-9\-\.]+\.speech\.googleapis\.com)`,
			"ai":                  `([a-zA-Z0-9\-\.]+\.ai\.googleapis\.com)`,
			"filestore":           `([a-zA-Z0-9\-\.]+\.filestore\.googleapis\.com)`,
			"dataproc":            `([a-zA-Z0-9\-\.]+\.dataproc\.googleapis\.com)`,
			"texttospeech":        `([a-zA-Z0-9\-\.]+\.texttospeech\.googleapis\.com)`,
			"language":            `([a-zA-Z0-9\-\.]+\.language\.googleapis\.com)`,
			"vision":              `([a-zA-Z0-9\-\.]+\.vision\.googleapis\.com)`,
			"automl":              `([a-zA-Z0-9\-\.]+\.automl\.googleapis\.com)`,
			"memcached":           `([a-zA-Z0-9\-\.]+\.memcached\.googleapis\.com)`,
			"iap":                 `([a-zA-Z0-9\-\.]+\.iap\.googleapis\.com)`,
			"networkintelligence": `([a-zA-Z0-9\-\.]+\.networkintelligence\.googleapis\.com)`,
			"vertexai":            `([a-zA-Z0-9\-\.]+\.vertexai\.googleapis\.com)`,
		},
		"azure": {
			"blob":                `([a-zA-Z0-9\-\.]+\.blob\.core\.windows\.net)`,
			"webapp":              `([a-zA-Z0-9\-\.]+\.azurewebsites\.net)`,
			"function":            `([a-zA-Z0-9\-\.]+\.azurewebsites\.net)`,
			"cosmosdb":            `([a-zA-Z0-9\-\.]+\.documents\.azure\.com)`,
			"servicebus":          `([a-zA-Z0-9\-\.]+\.servicebus\.windows\.net)`,
			"keyvault":            `([a-zA-Z0-9\-\.]+\.vault\.azure\.net)`,
			"sql":                 `([a-zA-Z0-9\-\.]+\.database\.windows\.net)`,
			"redis":               `([a-zA-Z0-9\-\.]+\.redis\.cache\.windows\.net)`,
			"cdn":                 `([a-zA-Z0-9\-\.]+\.azureedge\.net)`,
			"activedirectory":     `([a-zA-Z0-9\-\.]+\.microsoftonline\.com|[a-zA-Z0-9\-\.]+\.onmicrosoft\.com)`,
			"vm":                  `([a-zA-Z0-9\-\.]+\.cloudapp\.azure\.com)`,
			"virtualnetwork":      `([a-zA-Z0-9\-\.]+\.virtualnetwork\.azure\.com)`,
			"azurecontainer":      `([a-zA-Z0-9\-\.]+\.azurecontainer\.io)`,
			"eventgrid":           `([a-zA-Z0-9\-\.]+\.eventgrid\.azure\.net)`,
			"wvd":                 `([a-zA-Z0-9\-\.]+\.wvd\.microsoft\.com)`,
			"devops":              `([a-zA-Z0-9\-\.]+\.dev\.azure\.com)`,
			"logic":               `([a-zA-Z0-9\-\.]+\.logic\.azure\.com)`,
			"loadbalancer":        `([a-zA-Z0-9\-\.]+\.loadbalancer\.azure\.com)`,
			"backup":              `([a-zA-Z0-9\-\.]+\.backup\.azure\.com)`,
			"monitor":             `([a-zA-Z0-9\-\.]+\.monitor\.azure\.com)`,
			"firewallmanager":     `([a-zA-Z0-9\-\.]+\.firewallmanager\.azure\.net)`,
			"synapse":             `([a-zA-Z0-9\-\.]+\.synapse\.azure\.com)`,
			"virtualwan":          `([a-zA-Z0-9\-\.]+\.virtualwan\.azure\.com)`,
			"b2clogin":            `([a-zA-Z0-9\-\.]+\.b2clogin\.com)`,
			"applicationinsights": `([a-zA-Z0-9\-\.]+\.applicationinsights\.azure\.com)`,
			"managedhsm":          `([a-zA-Z0-9\-\.]+\.managedhsm\.azure\.net)`,
			"purview":             `([a-zA-Z0-9\-\.]+\.purview\.azure\.com)`,
			"datalake":            `([a-zA-Z0-9\-\.]+\.datalake\.azure\.net)`,
			"azconfig":            `([a-zA-Z0-9\-\.]+\.azconfig\.io)`,
			"azureapi":            `([a-zA-Z0-9\-\.]+\.azure-api\.net)`,
			"firewall":            `([a-zA-Z0-9\-\.]+\.firewall\.azure\.net)`,
			"sites":               `([a-zA-Z0-9\-\.]+\.sites\.azure\.com)`,
			"azuremicroservices":  `([a-zA-Z0-9\-\.]+\.azuremicroservices\.io)`,
			"search":              `([a-zA-Z0-9\-\.]+\.search\.windows\.net)`,
			"media":               `([a-zA-Z0-9\-\.]+\.media\.azure\.net)`,
		},
		"other": {
			"ibm_cloud":     `([a-zA-Z0-9\-\.]+\.bluemix\.net)`,
			"ibm_cloud_s3":  `([a-zA-Z0-9\-\.]+\.s3-api\.us-geo\.objectstorage\.softlayer\.net)`,
			"alibaba_cloud": `([a-zA-Z0-9\-\.]+\.aliyuncs\.com)`,
			"oracle_cloud":  `([a-zA-Z0-9\-\.]+\.oraclecloud\.com)`,
			"salesforce":    `([a-zA-Z0-9\-\.]+\.force\.com)`,
			"tencent_cloud": `([a-zA-Z0-9\-\.]+\.tencentcloudapi\.com)`,
		},
	}

	for cloudProvider, services := range cloudPatterns {
		for serviceName, pattern := range services {
			re := regexp.MustCompile(pattern)
			matches := re.FindAllString(url, -1)
			for _, match := range matches {
				// Create unique key to avoid duplicates within the same URL
				key := fmt.Sprintf("%s_%s:%s", cloudProvider, serviceName, match)
				if !uniqueURLAssets[key] {
					uniqueURLAssets[key] = true
					assets = append(assets, KatanaCloudAsset{
						Domain:      domain,
						URL:         fmt.Sprintf("https://%s", match),
						Type:        "cloud_service",
						Service:     fmt.Sprintf("%s_%s", cloudProvider, serviceName),
						Description: fmt.Sprintf("Found %s %s service: %s (source: %s)", cloudProvider, serviceName, match, url),
					})
				}
			}
		}
	}

	if body != "" {
		bodyFindings := analyzeBodyForCloudFindings(domain, url, body)
		findings = append(findings, bodyFindings...)
	}

	return assets, findings
}

// calculateEntropy calculates the Shannon entropy of a string to determine randomness
func calculateEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	// Count character frequencies
	freq := make(map[rune]int)
	for _, char := range s {
		freq[char]++
	}

	// Calculate entropy
	var entropy float64
	length := float64(len(s))
	for _, count := range freq {
		p := float64(count) / length
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

// isLikelyAWSSecret checks if a 40-char string is likely an AWS secret key
func isLikelyAWSSecret(candidate, context string) bool {
	// Must be exactly 40 characters
	if len(candidate) != 40 {
		return false
	}

	// Calculate entropy - AWS secrets should have high randomness
	entropy := calculateEntropy(candidate)
	if entropy < 4.5 { // Threshold for randomness
		return false
	}

	// Check if this appears to be part of a larger high-entropy token
	// Look for long alphanumeric strings surrounding the candidate
	candidateIndex := strings.Index(context, candidate)
	if candidateIndex != -1 {
		// Check characters before and after the candidate
		start := candidateIndex
		end := candidateIndex + len(candidate)

		// Count high-entropy characters before the candidate
		beforeCount := 0
		for i := start - 1; i >= 0 && i >= start-100; i-- {
			char := context[i]
			if (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') ||
				(char >= '0' && char <= '9') || char == '_' || char == '-' || char == '.' {
				beforeCount++
			} else {
				break
			}
		}

		// Count high-entropy characters after the candidate
		afterCount := 0
		for i := end; i < len(context) && i < end+100; i++ {
			char := context[i]
			if (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') ||
				(char >= '0' && char <= '9') || char == '_' || char == '-' || char == '.' {
				afterCount++
			} else {
				break
			}
		}

		// If surrounded by long strings of alphanumeric characters,
		// it's likely part of a JWT or other large token
		if beforeCount > 20 || afterCount > 20 {
			return false
		}

		// If the total continuous alphanumeric sequence is very long, reject it
		totalLength := beforeCount + len(candidate) + afterCount
		if totalLength > 100 {
			return false
		}
	}

	// Check for patterns that indicate it's NOT a secret
	lowerCandidate := strings.ToLower(candidate)

	// Exclude common programming patterns
	excludePatterns := []string{
		"internal", "memo", "react", "context", "component",
		"function", "object", "element", "instance", "prototype",
		"undefined", "null", "true", "false", "constructor",
	}

	for _, pattern := range excludePatterns {
		if strings.Contains(lowerCandidate, pattern) {
			return false
		}
	}

	// Look for positive context indicators
	lowerContext := strings.ToLower(context)
	positiveIndicators := []string{
		"secret", "key", "aws", "access", "credential", "token",
		"config", "env", "password", "auth", "api",
	}

	for _, indicator := range positiveIndicators {
		if strings.Contains(lowerContext, indicator) {
			return true
		}
	}

	// If no positive context but high entropy, be more conservative
	return entropy > 5.2 // Higher threshold without context
}

func analyzeBodyForCloudFindings(domain, url, body string) []KatanaCloudFinding {
	var findings []KatanaCloudFinding

	// Use a map to track unique findings within this body
	uniqueBodyFindings := make(map[string]bool)

	patterns := map[string]map[string]string{
		"aws": {
			"access_key":            `AKIA[0-9A-Z]{16}`,
			"secret_key":            `\b[A-Za-z0-9/+=]{40}\b`,
			"s3_bucket":             `s3://[a-zA-Z0-9\-\.]+`,
			"s3_url":                `https://[a-zA-Z0-9\-\.]+\.s3[\-\.][a-zA-Z0-9\-]*\.amazonaws\.com|https://[a-zA-Z0-9\-\.]+\.s3\.amazonaws\.com`,
			"lambda_arn":            `arn:aws:lambda:[a-zA-Z0-9\-]+:[0-9]+:function:[a-zA-Z0-9\-_]+`,
			"iam_role_arn":          `arn:aws:iam::[0-9]+:role\/[a-zA-Z0-9\-_\/]+`,
			"iam_user_arn":          `arn:aws:iam::[0-9]+:user\/[a-zA-Z0-9\-_\/]+`,
			"sns_topic_arn":         `arn:aws:sns:[a-zA-Z0-9\-]+:[0-9]+:[a-zA-Z0-9\-_]+`,
			"sqs_queue_arn":         `arn:aws:sqs:[a-zA-Z0-9\-]+:[0-9]+:[a-zA-Z0-9\-_]+`,
			"kinesis_stream_arn":    `arn:aws:kinesis:[a-zA-Z0-9\-]+:[0-9]+:stream\/[a-zA-Z0-9\-_]+`,
			"s3_bucket_arn":         `arn:aws:s3:::[a-zA-Z0-9\-\.]+`,
			"dynamodb_table_arn":    `arn:aws:dynamodb:[a-zA-Z0-9\-]+:[0-9]+:table\/[a-zA-Z0-9\-_]+`,
			"secrets_manager_arn":   `arn:aws:secretsmanager:[a-zA-Z0-9\-]+:[0-9]+:secret:[a-zA-Z0-9\-_\/]+`,
			"parameter_store":       `arn:aws:ssm:[a-zA-Z0-9\-]+:[0-9]+:parameter\/[a-zA-Z0-9\-_\/]+`,
			"ec2_instance_id":       `i-[0-9a-f]{8,17}`,
			"vpc_id":                `vpc-[0-9a-f]{8,17}`,
			"subnet_id":             `subnet-[0-9a-f]{8,17}`,
			"security_group_id":     `sg-[0-9a-f]{8,17}`,
			"cognito_user_pool":     `[a-z]{2}-[a-z]+-[0-9]+_[A-Za-z0-9]{9}`,
			"cognito_identity_pool": `[a-z]{2}-[a-z]+-[0-9]+:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`,
			"api_gateway":           `[a-z0-9]{10}\.execute-api\.[a-z0-9\-]+\.amazonaws\.com`,
			"cloudfront_dist":       `[a-z0-9]{14}\.cloudfront\.net`,
			"route53_hosted_zone":   `Z[0-9A-Z]{13}`,
		},
		"gcp": {
			"project_id":       `[a-z][a-z0-9\-]{4,28}[a-z0-9]\.googleapis\.com`,
			"service_account":  `[a-zA-Z0-9\-]+@[a-zA-Z0-9\-]+\.iam\.gserviceaccount\.com`,
			"firebase_config":  `"apiKey"\s*:\s*"[A-Za-z0-9_-]+",\s*"authDomain"\s*:\s*"[a-zA-Z0-9\-]+\.firebaseapp\.com",\s*"projectId"\s*:\s*"[a-zA-Z0-9\-]+"`,
			"storage_bucket":   `gs://[a-zA-Z0-9\-\.]+`,
			"firebase_api_key": `AIza[0-9A-Za-z_-]{35}`,
			"gcp_api_key":      `AIza[0-9A-Za-z_-]{35}`,
			"client_id":        `[0-9]+-[a-zA-Z0-9_-]+\.apps\.googleusercontent\.com`,
			"client_secret":    `GOCSPX-[a-zA-Z0-9_-]{28}`,
		},
		"azure": {
			"storage_account": `[a-z0-9]{3,24}\.blob\.core\.windows\.net`,
			"key_vault":       `[a-zA-Z0-9\-]{3,24}\.vault\.azure\.net`,
			"app_service":     `[a-zA-Z0-9\-]+\.azurewebsites\.net`,
			"cosmos_db":       `[a-zA-Z0-9\-]+\.documents\.azure\.com`,
			"tenant_id":       `(?:tenant["\s]*[:=]["\s]*|TenantId["\s]*[:=]["\s]*)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`,
			"client_id":       `(?:client["\s]*[:=]["\s]*|ClientId["\s]*[:=]["\s]*)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`,
			"subscription_id": `(?:subscription["\s]*[:=]["\s]*|SubscriptionId["\s]*[:=]["\s]*)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`,
		},
	}

	for cloudProvider, detectionPatterns := range patterns {
		for findingType, pattern := range detectionPatterns {
			re := regexp.MustCompile(pattern)
			matches := re.FindAllStringIndex(body, -1)
			matchStrings := re.FindAllString(body, -1)

			for i, match := range matches {
				if i >= len(matchStrings) {
					continue
				}

				matchText := sanitizeUTF8(matchStrings[i])
				startPos := match[0]
				endPos := match[1]

				// Special validation for AWS secret keys to reduce false positives
				if cloudProvider == "aws" && findingType == "secret_key" {
					// Extract broader context for validation (500 chars)
					contextStart := startPos - 500
					if contextStart < 0 {
						contextStart = 0
					}
					contextEnd := endPos + 500
					if contextEnd > len(body) {
						contextEnd = len(body)
					}

					fullContext := body[contextStart:contextEnd]
					if !isLikelyAWSSecret(matchText, fullContext) {
						continue // Skip this match as it's likely a false positive
					}
				}

				// Create unique key to avoid duplicates within the same body
				key := fmt.Sprintf("%s:%s:%s:%d", cloudProvider, findingType, matchText, startPos)
				if !uniqueBodyFindings[key] {
					uniqueBodyFindings[key] = true

					// Extract context before and after (250 characters each)
					contextStart := startPos - 250
					if contextStart < 0 {
						contextStart = 0
					}
					contextEnd := endPos + 250
					if contextEnd > len(body) {
						contextEnd = len(body)
					}

					contextBefore := ""
					contextAfter := ""

					if startPos > 0 {
						contextBefore = sanitizeUTF8(body[contextStart:startPos])
					}
					if endPos < len(body) {
						contextAfter = sanitizeUTF8(body[endPos:contextEnd])
					}

					findings = append(findings, KatanaCloudFinding{
						Domain:        domain,
						URL:           url,
						Type:          findingType,
						Content:       matchText,
						Description:   fmt.Sprintf("Found %s %s in response body", cloudProvider, findingType),
						CloudService:  cloudProvider,
						ContextBefore: contextBefore,
						ContextAfter:  contextAfter,
						MatchPosition: startPos,
					})
				}
			}
		}
	}

	return findings
}

func analyzeHeadersForCloudAssets(domain, url string, headers map[string]interface{}) ([]KatanaCloudAsset, []KatanaCloudFinding) {
	var assets []KatanaCloudAsset
	var findings []KatanaCloudFinding

	// Use a map to track unique assets within headers
	uniqueHeaderAssets := make(map[string]bool)

	// Headers to check for cloud assets
	headersToCheck := []string{
		"content-security-policy",
		"strict-transport-security",
		"referrer-policy",
		"x-amz-*",
		"x-goog-*",
		"x-ms-*",
	}

	cloudPatterns := map[string]map[string]string{
		"aws": {
			"s3":                `([a-zA-Z0-9\-\.]+\.s3[\-\.][a-zA-Z0-9\-]*\.amazonaws\.com|[a-zA-Z0-9\-\.]+\.s3\.amazonaws\.com)`,
			"cloudfront":        `([a-zA-Z0-9\-\.]+\.cloudfront\.net)`,
			"lambda":            `([a-zA-Z0-9\-\.]+\.lambda\.amazonaws\.com)`,
			"apigateway":        `([a-zA-Z0-9\-\.]+\.execute-api\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"elasticbeanstalk":  `([a-zA-Z0-9\-\.]+\.elasticbeanstalk\.com)`,
			"elb":               `([a-zA-Z0-9\-\.]+\.elb\.amazonaws\.com)`,
			"alb_nlb":           `([a-zA-Z0-9\-\.]+\.elb\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"ec2":               `(ec2-[0-9\-]+\.[a-zA-Z0-9\-]+\.compute\.amazonaws\.com)`,
			"rds":               `([a-zA-Z0-9\-\.]+\.rds\.amazonaws\.com)`,
			"dynamodb":          `([a-zA-Z0-9\-\.]+\.dynamodb\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"amplify":           `([a-zA-Z0-9\-\.]+\.amplifyapp\.com)`,
			"appsync":           `([a-zA-Z0-9\-\.]+\.appsync-api\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"cloudwatch":        `([a-zA-Z0-9\-\.]+\.cloudwatch\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"cloudformation":    `([a-zA-Z0-9\-\.]+\.cloudformation\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"stepfunctions":     `([a-zA-Z0-9\-\.]+\.states\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"secretsmanager":    `([a-zA-Z0-9\-\.]+\.secretsmanager\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"iot":               `([a-zA-Z0-9\-\.]+\.iot\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"eventbridge":       `([a-zA-Z0-9\-\.]+\.events\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"xray":              `([a-zA-Z0-9\-\.]+\.xray\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"cloudsearch":       `([a-zA-Z0-9\-\.]+\.cloudsearch\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"elastictranscoder": `([a-zA-Z0-9\-\.]+\.elastictranscoder\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"elasticinference":  `([a-zA-Z0-9\-\.]+\.elasticinference\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"neptune":           `([a-zA-Z0-9\-\.]+\.neptune\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"workspaces":        `([a-zA-Z0-9\-\.]+\.workspaces\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"directconnect":     `([a-zA-Z0-9\-\.]+\.directconnect\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"mobilehub":         `([a-zA-Z0-9\-\.]+\.mobilehub\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"macie":             `([a-zA-Z0-9\-\.]+\.macie\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"sagemaker":         `([a-zA-Z0-9\-\.]+\.sagemaker\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"workdocs":          `([a-zA-Z0-9\-\.]+\.workdocs\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"transcribe":        `([a-zA-Z0-9\-\.]+\.transcribe\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"translate":         `([a-zA-Z0-9\-\.]+\.translate\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"appmesh":           `([a-zA-Z0-9\-\.]+\.appmesh\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"inspector":         `([a-zA-Z0-9\-\.]+\.inspector\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"glue":              `([a-zA-Z0-9\-\.]+\.glue\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"connect":           `([a-zA-Z0-9\-\.]+\.awsapps\.com)`,
			"chime":             `([a-zA-Z0-9\-\.]+\.chime\.aws)`,
			"efs":               `([a-zA-Z0-9\-\.]+\.efs\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"batch":             `([a-zA-Z0-9\-\.]+\.batch\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"kafka":             `([a-zA-Z0-9\-\.]+\.kafka\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"medialive":         `([a-zA-Z0-9\-\.]+\.medialive\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"snowball":          `([a-zA-Z0-9\-\.]+\.snowball\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"cloudtrail":        `([a-zA-Z0-9\-\.]+\.cloudtrail\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
			"datasync":          `([a-zA-Z0-9\-\.]+\.datasync\.[a-zA-Z0-9\-]+\.amazonaws\.com)`,
		},
		"gcp": {
			"storage":             `([a-zA-Z0-9\-\.]+\.storage\.googleapis\.com)`,
			"appengine":           `([a-zA-Z0-9\-\.]+\.appspot\.com)`,
			"cloudfunctions":      `([a-zA-Z0-9\-\.]+\.cloudfunctions\.net)`,
			"cloudrun":            `([a-zA-Z0-9\-\.]+\.run\.app)`,
			"firebase":            `([a-zA-Z0-9\-\.]+\.firebaseapp\.com)`,
			"gke":                 `([a-zA-Z0-9\-\.]+\.container\.googleapis\.com)`,
			"compute":             `([a-zA-Z0-9\-\.]+\.compute\.googleapis\.com)`,
			"sql":                 `([a-zA-Z0-9\-\.]+\.sql\.googleapis\.com)`,
			"bigquery":            `([a-zA-Z0-9\-\.]+\.bigquery\.googleapis\.com)`,
			"googleusercontent":   `([a-zA-Z0-9\-\.]+\.googleusercontent\.com)`,
			"pubsub":              `([a-zA-Z0-9\-\.]+\.pubsub\.googleapis\.com)`,
			"bigtable":            `([a-zA-Z0-9\-\.]+\.bigtable\.googleapis\.com)`,
			"spanner":             `([a-zA-Z0-9\-\.]+\.spanner\.googleapis\.com)`,
			"dataflow":            `([a-zA-Z0-9\-\.]+\.dataflow\.googleapis\.com)`,
			"identityplatform":    `([a-zA-Z0-9\-\.]+\.identityplatform\.googleapis\.com)`,
			"firestore_api":       `([a-zA-Z0-9\-\.]+\.firestore\.googleapis\.com)`,
			"datastore":           `([a-zA-Z0-9\-\.]+\.datastore\.googleapis\.com)`,
			"monitoring":          `([a-zA-Z0-9\-\.]+\.monitoring\.googleapis\.com)`,
			"logging":             `([a-zA-Z0-9\-\.]+\.logging\.googleapis\.com)`,
			"speech":              `([a-zA-Z0-9\-\.]+\.speech\.googleapis\.com)`,
			"ai":                  `([a-zA-Z0-9\-\.]+\.ai\.googleapis\.com)`,
			"filestore":           `([a-zA-Z0-9\-\.]+\.filestore\.googleapis\.com)`,
			"dataproc":            `([a-zA-Z0-9\-\.]+\.dataproc\.googleapis\.com)`,
			"texttospeech":        `([a-zA-Z0-9\-\.]+\.texttospeech\.googleapis\.com)`,
			"language":            `([a-zA-Z0-9\-\.]+\.language\.googleapis\.com)`,
			"vision":              `([a-zA-Z0-9\-\.]+\.vision\.googleapis\.com)`,
			"automl":              `([a-zA-Z0-9\-\.]+\.automl\.googleapis\.com)`,
			"memcached":           `([a-zA-Z0-9\-\.]+\.memcached\.googleapis\.com)`,
			"iap":                 `([a-zA-Z0-9\-\.]+\.iap\.googleapis\.com)`,
			"networkintelligence": `([a-zA-Z0-9\-\.]+\.networkintelligence\.googleapis\.com)`,
			"vertexai":            `([a-zA-Z0-9\-\.]+\.vertexai\.googleapis\.com)`,
		},
		"azure": {
			"blob":                `([a-zA-Z0-9\-\.]+\.blob\.core\.windows\.net)`,
			"webapp":              `([a-zA-Z0-9\-\.]+\.azurewebsites\.net)`,
			"cosmosdb":            `([a-zA-Z0-9\-\.]+\.documents\.azure\.com)`,
			"keyvault":            `([a-zA-Z0-9\-\.]+\.vault\.azure\.net)`,
			"sql":                 `([a-zA-Z0-9\-\.]+\.database\.windows\.net)`,
			"cdn":                 `([a-zA-Z0-9\-\.]+\.azureedge\.net)`,
			"activedirectory":     `([a-zA-Z0-9\-\.]+\.microsoftonline\.com|[a-zA-Z0-9\-\.]+\.onmicrosoft\.com)`,
			"vm":                  `([a-zA-Z0-9\-\.]+\.cloudapp\.azure\.com)`,
			"virtualnetwork":      `([a-zA-Z0-9\-\.]+\.virtualnetwork\.azure\.com)`,
			"azurecontainer":      `([a-zA-Z0-9\-\.]+\.azurecontainer\.io)`,
			"eventgrid":           `([a-zA-Z0-9\-\.]+\.eventgrid\.azure\.net)`,
			"wvd":                 `([a-zA-Z0-9\-\.]+\.wvd\.microsoft\.com)`,
			"devops":              `([a-zA-Z0-9\-\.]+\.dev\.azure\.com)`,
			"logic":               `([a-zA-Z0-9\-\.]+\.logic\.azure\.com)`,
			"loadbalancer":        `([a-zA-Z0-9\-\.]+\.loadbalancer\.azure\.com)`,
			"backup":              `([a-zA-Z0-9\-\.]+\.backup\.azure\.com)`,
			"monitor":             `([a-zA-Z0-9\-\.]+\.monitor\.azure\.com)`,
			"firewallmanager":     `([a-zA-Z0-9\-\.]+\.firewallmanager\.azure\.net)`,
			"synapse":             `([a-zA-Z0-9\-\.]+\.synapse\.azure\.com)`,
			"virtualwan":          `([a-zA-Z0-9\-\.]+\.virtualwan\.azure\.com)`,
			"b2clogin":            `([a-zA-Z0-9\-\.]+\.b2clogin\.com)`,
			"applicationinsights": `([a-zA-Z0-9\-\.]+\.applicationinsights\.azure\.com)`,
			"managedhsm":          `([a-zA-Z0-9\-\.]+\.managedhsm\.azure\.net)`,
			"purview":             `([a-zA-Z0-9\-\.]+\.purview\.azure\.com)`,
			"datalake":            `([a-zA-Z0-9\-\.]+\.datalake\.azure\.net)`,
			"azconfig":            `([a-zA-Z0-9\-\.]+\.azconfig\.io)`,
			"azureapi":            `([a-zA-Z0-9\-\.]+\.azure-api\.net)`,
			"firewall":            `([a-zA-Z0-9\-\.]+\.firewall\.azure\.net)`,
			"sites":               `([a-zA-Z0-9\-\.]+\.sites\.azure\.com)`,
			"azuremicroservices":  `([a-zA-Z0-9\-\.]+\.azuremicroservices\.io)`,
			"search":              `([a-zA-Z0-9\-\.]+\.search\.windows\.net)`,
			"media":               `([a-zA-Z0-9\-\.]+\.media\.azure\.net)`,
		},
		"other": {
			"ibm_cloud":     `([a-zA-Z0-9\-\.]+\.bluemix\.net)`,
			"ibm_cloud_s3":  `([a-zA-Z0-9\-\.]+\.s3-api\.us-geo\.objectstorage\.softlayer\.net)`,
			"alibaba_cloud": `([a-zA-Z0-9\-\.]+\.aliyuncs\.com)`,
			"oracle_cloud":  `([a-zA-Z0-9\-\.]+\.oraclecloud\.com)`,
			"salesforce":    `([a-zA-Z0-9\-\.]+\.force\.com)`,
			"tencent_cloud": `([a-zA-Z0-9\-\.]+\.tencentcloudapi\.com)`,
		},
	}

	for headerName, headerValue := range headers {
		headerNameLower := strings.ToLower(headerName)

		// Check if this header should be analyzed
		shouldCheck := false
		for _, checkHeader := range headersToCheck {
			if strings.Contains(headerNameLower, checkHeader) || checkHeader == headerNameLower {
				shouldCheck = true
				break
			}
		}

		if !shouldCheck {
			continue
		}

		// Convert header value to string
		headerValueStr := ""
		if str, ok := headerValue.(string); ok {
			headerValueStr = str
		} else if strPtr, ok := headerValue.(*string); ok && strPtr != nil {
			headerValueStr = *strPtr
		} else {
			continue
		}

		// Search for cloud patterns in the header value
		for cloudProvider, services := range cloudPatterns {
			for serviceName, pattern := range services {
				re := regexp.MustCompile(pattern)
				matches := re.FindAllString(headerValueStr, -1)
				for _, match := range matches {
					// Create unique key to avoid duplicates within headers
					key := fmt.Sprintf("%s_%s:%s:%s", cloudProvider, serviceName, headerName, match)
					if !uniqueHeaderAssets[key] {
						uniqueHeaderAssets[key] = true
						assets = append(assets, KatanaCloudAsset{
							Domain:      domain,
							URL:         fmt.Sprintf("https://%s", match),
							Type:        "cloud_service_header",
							Service:     fmt.Sprintf("%s_%s", cloudProvider, serviceName),
							Description: fmt.Sprintf("Found %s %s in %s header: %s (source: %s)", cloudProvider, serviceName, headerName, match, url),
						})
					}
				}
			}
		}
	}

	return assets, findings
}

func countAssetsByService(assets []KatanaCloudAsset, serviceType string) int {
	count := 0
	for _, asset := range assets {
		if strings.Contains(asset.Service, serviceType) {
			count++
		}
	}
	return count
}

func UpdateKatanaCompanyScanStatus(scanID, status, result, stderr, command, execTime string) {
	log.Printf("[KATANA-COMPANY] [INFO] Updating scan status for %s to %s", scanID, status)

	// Convert strings to pointers for nullable fields
	var resultPtr, stderrPtr, commandPtr, execTimePtr *string
	if result != "" {
		resultPtr = &result
	}
	if stderr != "" {
		stderrPtr = &stderr
	}
	if command != "" {
		commandPtr = &command
	}
	if execTime != "" {
		execTimePtr = &execTime
	}

	query := `UPDATE katana_company_scans SET status = $1, result = $2, stderr = $3, command = $4, execution_time = $5 WHERE scan_id = $6`
	_, err := dbPool.Exec(context.Background(), query, status, resultPtr, stderrPtr, commandPtr, execTimePtr, scanID)
	if err != nil {
		log.Printf("[KATANA-COMPANY] [ERROR] Failed to update scan status for %s: %v", scanID, err)
	} else {
		log.Printf("[KATANA-COMPANY] [INFO] Successfully updated scan status for %s", scanID)
	}
}

func GetKatanaCompanyScanStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scanID := vars["scan_id"]

	var scan KatanaCompanyScanStatus
	query := `SELECT id, scan_id, scope_target_id, domains, status, result, error, stdout, stderr, command, execution_time, created_at, auto_scan_session_id FROM katana_company_scans WHERE scan_id = $1`

	var domainsJSON string
	err := dbPool.QueryRow(context.Background(), query, scanID).Scan(
		&scan.ID,
		&scan.ScanID,
		&scan.ScopeTargetID,
		&domainsJSON,
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
		if err == pgx.ErrNoRows {
			log.Printf("[KATANA-COMPANY] [ERROR] Katana Company scan not found for scan ID: %s", scanID)
			http.Error(w, "Scan not found", http.StatusNotFound)
		} else {
			log.Printf("[KATANA-COMPANY] [ERROR] Failed to get Katana Company scan status for scan ID %s: %v", scanID, err)
			http.Error(w, "Failed to get scan status", http.StatusInternalServerError)
		}
		return
	}

	json.Unmarshal([]byte(domainsJSON), &scan.Domains)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scan)
}

func GetKatanaCompanyScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]
	log.Printf("[KATANA-COMPANY] [INFO] Retrieving Katana Company scans for scope target ID: %s", scopeTargetID)

	query := `SELECT id, scan_id, scope_target_id, domains, status, result, error, stdout, stderr, command, execution_time, created_at, auto_scan_session_id FROM katana_company_scans WHERE scope_target_id = $1 ORDER BY created_at DESC`

	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[KATANA-COMPANY] [ERROR] Failed to get Katana Company scans for scope target %s: %v", scopeTargetID, err)
		http.Error(w, "Failed to get scans", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []KatanaCompanyScanStatus
	for rows.Next() {
		var scan KatanaCompanyScanStatus
		var domainsJSON string
		err := rows.Scan(
			&scan.ID,
			&scan.ScanID,
			&scan.ScopeTargetID,
			&domainsJSON,
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
			log.Printf("[KATANA-COMPANY] [ERROR] Failed to scan row: %v", err)
			continue
		}

		json.Unmarshal([]byte(domainsJSON), &scan.Domains)
		scans = append(scans, scan)
	}

	log.Printf("[KATANA-COMPANY] [INFO] Successfully retrieved %d Katana Company scans for scope target %s", len(scans), scopeTargetID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scans)
}

// InsertKatanaDomainResult stores domain scan results in the domain-centric table
func InsertKatanaDomainResult(scopeTargetID, domain, scanID, rawOutput string) {
	_, err := dbPool.Exec(context.Background(),
		`INSERT INTO katana_company_domain_results (scope_target_id, domain, last_scan_id, raw_output, last_scanned_at, updated_at) 
		 VALUES ($1, $2, $3, $4, NOW(), NOW())
		 ON CONFLICT (scope_target_id, domain) 
		 DO UPDATE SET last_scan_id = $3, raw_output = $4, last_scanned_at = NOW(), updated_at = NOW()`,
		scopeTargetID, domain, scanID, sanitizeUTF8(rawOutput))
	if err != nil {
		log.Printf("[KATANA-COMPANY] [ERROR] Failed to insert/update domain result: %v", err)
	}
}

// ParseKatanaResultsDomainCentric parses results and stores them in domain-centric tables
func ParseKatanaResultsDomainCentric(scopeTargetID, domain, result string) ([]KatanaCloudAsset, []KatanaCloudFinding) {
	log.Printf("[KATANA-COMPANY] [INFO] Starting to parse results for domain %s using domain-centric approach", domain)

	var cloudAssets []KatanaCloudAsset
	var cloudFindings []KatanaCloudFinding

	// Use maps for deduplication during parsing
	uniqueAssets := make(map[string]KatanaCloudAsset)
	uniqueFindings := make(map[string]KatanaCloudFinding)

	lines := strings.Split(result, "\n")
	log.Printf("[KATANA-COMPANY] [INFO] Processing %d lines of output for domain-centric storage", len(lines))

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		log.Printf("[KATANA-COMPANY] [DEBUG] Processing line %d: %s", lineNum+1, line)

		var katanaResult struct {
			Timestamp string `json:"timestamp"`
			Request   struct {
				Method   string `json:"method"`
				Endpoint string `json:"endpoint"`
				Source   string `json:"source"`
			} `json:"request"`
			Response struct {
				StatusCode int                    `json:"status_code"`
				Headers    map[string]interface{} `json:"headers"`
				Body       string                 `json:"body"`
			} `json:"response"`
		}

		if err := json.Unmarshal([]byte(line), &katanaResult); err != nil {
			log.Printf("[KATANA-COMPANY] [DEBUG] Failed to parse JSON, treating as plain URL: %v", err)
			if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
				assets, findings := analyzeURLForCloudAssets(domain, line, "")
				addUniqueAssets(uniqueAssets, assets)
				addUniqueFindings(uniqueFindings, findings)
			}
			continue
		}

		assets, findings := analyzeURLForCloudAssets(domain, katanaResult.Request.Endpoint, katanaResult.Response.Body)
		addUniqueAssets(uniqueAssets, assets)
		addUniqueFindings(uniqueFindings, findings)

		// Also analyze headers
		headerAssets, headerFindings := analyzeHeadersForCloudAssets(domain, katanaResult.Request.Endpoint, katanaResult.Response.Headers)
		addUniqueAssets(uniqueAssets, headerAssets)
		addUniqueFindings(uniqueFindings, headerFindings)
	}

	// Convert unique maps back to slices and insert into domain-centric tables
	for _, asset := range uniqueAssets {
		// Extract source URL from description
		sourceURL := ""
		if sourceMatch := regexp.MustCompile(`\(source: (.+?)\)`).FindStringSubmatch(asset.Description); len(sourceMatch) > 1 {
			sourceURL = sourceMatch[1]
		}

		// Insert into domain-centric cloud assets table
		_, err := dbPool.Exec(context.Background(),
			`INSERT INTO katana_company_cloud_assets (scope_target_id, root_domain, asset_domain, asset_url, asset_type, service, description, source_url, last_scanned_at) 
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
			 ON CONFLICT (scope_target_id, root_domain, asset_url, asset_type) 
			 DO UPDATE SET service = $6, description = $7, source_url = $8, last_scanned_at = NOW()`,
			scopeTargetID, domain, asset.Domain, asset.URL, asset.Type, asset.Service, asset.Description, sourceURL)
		if err != nil {
			log.Printf("[KATANA-COMPANY] [ERROR] Failed to insert cloud asset: %v", err)
		}

		cloudAssets = append(cloudAssets, asset)
	}

	for _, finding := range uniqueFindings {
		// Insert into domain-centric cloud findings table
		_, err := dbPool.Exec(context.Background(),
			`INSERT INTO katana_company_cloud_findings (scope_target_id, root_domain, finding_domain, finding_url, finding_type, content, description, cloud_service, context_before, context_after, match_position, last_scanned_at) 
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
			 ON CONFLICT (scope_target_id, root_domain, finding_url, finding_type, content) 
			 DO UPDATE SET description = $7, cloud_service = $8, context_before = $9, context_after = $10, match_position = $11, last_scanned_at = NOW()`,
			scopeTargetID, domain, finding.Domain, finding.URL, finding.Type, finding.Content, finding.Description, finding.CloudService, finding.ContextBefore, finding.ContextAfter, finding.MatchPosition)
		if err != nil {
			log.Printf("[KATANA-COMPANY] [ERROR] Failed to insert cloud finding: %v", err)
		}

		cloudFindings = append(cloudFindings, finding)
	}

	log.Printf("[KATANA-COMPANY] [INFO] Completed parsing results for domain %s - found %d unique cloud assets, %d cloud findings",
		domain, len(cloudAssets), len(cloudFindings))

	return cloudAssets, cloudFindings
}

// GetKatanaCompanyCloudAssets retrieves all cloud assets for a scope target (domain-centric approach)
func GetKatanaCompanyCloudAssets(w http.ResponseWriter, r *http.Request) {
	scanID := mux.Vars(r)["scan_id"]
	if scanID == "" {
		http.Error(w, "Scan ID is required", http.StatusBadRequest)
		return
	}

	// Get scope_target_id from scan
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(),
		`SELECT scope_target_id FROM katana_company_scans WHERE scan_id = $1`,
		scanID).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[KATANA-COMPANY] [ERROR] Failed to get scope target ID for scan %s: %v", scanID, err)
		http.Error(w, "Scan not found", http.StatusNotFound)
		return
	}

	// Fetch all cloud assets for this scope target (domain-centric approach)
	rows, err := dbPool.Query(context.Background(),
		`SELECT id, root_domain, asset_domain, asset_url, asset_type, service, description, source_url, last_scanned_at 
		 FROM katana_company_cloud_assets 
		 WHERE scope_target_id = $1 
		 ORDER BY last_scanned_at DESC, asset_url`,
		scopeTargetID)
	if err != nil {
		log.Printf("[KATANA-COMPANY] [ERROR] Failed to fetch cloud assets: %v", err)
		http.Error(w, "Failed to fetch cloud assets", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var cloudAssets []map[string]interface{}
	for rows.Next() {
		var id, rootDomain, assetDomain, assetURL, assetType, service, description string
		var sourceURL *string
		var lastScannedAt time.Time

		err := rows.Scan(&id, &rootDomain, &assetDomain, &assetURL, &assetType, &service, &description, &sourceURL, &lastScannedAt)
		if err != nil {
			log.Printf("[KATANA-COMPANY] [ERROR] Error scanning cloud asset row: %v", err)
			continue
		}

		sourceURLStr := ""
		if sourceURL != nil {
			sourceURLStr = *sourceURL
		}

		cloudAssets = append(cloudAssets, map[string]interface{}{
			"id":              id,
			"root_domain":     rootDomain,
			"domain":          assetDomain,
			"url":             assetURL,
			"type":            assetType,
			"service":         service,
			"description":     description,
			"source_url":      sourceURLStr,
			"created_at":      lastScannedAt,
			"last_scanned_at": lastScannedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cloudAssets)
}

// GetKatanaCompanyCloudFindings retrieves all cloud findings for a scope target (domain-centric approach)
func GetKatanaCompanyCloudFindings(w http.ResponseWriter, r *http.Request) {
	scanID := mux.Vars(r)["scan_id"]
	if scanID == "" {
		http.Error(w, "Scan ID is required", http.StatusBadRequest)
		return
	}

	// Get scope_target_id from scan
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(),
		`SELECT scope_target_id FROM katana_company_scans WHERE scan_id = $1`,
		scanID).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[KATANA-COMPANY] [ERROR] Failed to get scope target ID for scan %s: %v", scanID, err)
		http.Error(w, "Scan not found", http.StatusNotFound)
		return
	}

	// Fetch all cloud findings for this scope target (domain-centric approach)
	rows, err := dbPool.Query(context.Background(),
		`SELECT id, root_domain, finding_domain, finding_url, finding_type, content, description, cloud_service, context_before, context_after, match_position, last_scanned_at 
		 FROM katana_company_cloud_findings 
		 WHERE scope_target_id = $1 
		 ORDER BY last_scanned_at DESC, finding_url`,
		scopeTargetID)
	if err != nil {
		log.Printf("[KATANA-COMPANY] [ERROR] Failed to fetch cloud findings: %v", err)
		http.Error(w, "Failed to fetch cloud findings", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var cloudFindings []map[string]interface{}
	for rows.Next() {
		var id, rootDomain, findingDomain, findingURL, findingType, content, description, cloudService, contextBefore, contextAfter string
		var matchPosition int
		var lastScannedAt time.Time

		err := rows.Scan(&id, &rootDomain, &findingDomain, &findingURL, &findingType, &content, &description, &cloudService, &contextBefore, &contextAfter, &matchPosition, &lastScannedAt)
		if err != nil {
			log.Printf("[KATANA-COMPANY] [ERROR] Error scanning cloud finding row: %v", err)
			continue
		}

		cloudFindings = append(cloudFindings, map[string]interface{}{
			"id":              id,
			"root_domain":     rootDomain,
			"domain":          findingDomain,
			"url":             findingURL,
			"type":            findingType,
			"content":         content,
			"description":     description,
			"cloud_service":   cloudService,
			"context_before":  contextBefore,
			"context_after":   contextAfter,
			"match_position":  matchPosition,
			"created_at":      lastScannedAt,
			"last_scanned_at": lastScannedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cloudFindings)
}

// GetKatanaCompanyRawResults retrieves raw domain results for a scope target
func GetKatanaCompanyRawResults(w http.ResponseWriter, r *http.Request) {
	scanID := mux.Vars(r)["scan_id"]
	if scanID == "" {
		http.Error(w, "Scan ID is required", http.StatusBadRequest)
		return
	}

	// Get domain parameter from query string (optional)
	domain := r.URL.Query().Get("domain")

	// Get scope_target_id from scan
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(),
		`SELECT scope_target_id FROM katana_company_scans WHERE scan_id = $1`,
		scanID).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[KATANA-COMPANY] [ERROR] Failed to get scope target ID for scan %s: %v", scanID, err)
		http.Error(w, "Scan not found", http.StatusNotFound)
		return
	}

	var query string
	var args []interface{}

	if domain != "" {
		// Fetch raw results for specific domain
		query = `SELECT domain, raw_output, last_scanned_at, last_scan_id 
		         FROM katana_company_domain_results 
		         WHERE scope_target_id = $1 AND domain = $2`
		args = []interface{}{scopeTargetID, domain}
	} else {
		// Fetch all domains from scan configuration and join with results to show scan status
		query = `
		WITH scan_domains AS (
			SELECT DISTINCT jsonb_array_elements_text(domains) as domain
			FROM katana_company_scans 
			WHERE scan_id = $1
		)
		SELECT 
			sd.domain,
			'' as raw_output,
			kcdr.last_scanned_at,
			kcdr.last_scan_id,
			CASE WHEN kcdr.domain IS NOT NULL THEN true ELSE false END as has_been_scanned
		FROM scan_domains sd
		LEFT JOIN katana_company_domain_results kcdr 
			ON sd.domain = kcdr.domain AND kcdr.scope_target_id = $2
		ORDER BY has_been_scanned DESC, sd.domain ASC`
		args = []interface{}{scanID, scopeTargetID}
	}

	rows, err := dbPool.Query(context.Background(), query, args...)
	if err != nil {
		log.Printf("[KATANA-COMPANY] [ERROR] Failed to fetch raw domain results: %v", err)
		http.Error(w, "Failed to fetch raw domain results", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var rawResults []map[string]interface{}
	for rows.Next() {
		var domainName, rawOutput, lastScanID string
		var lastScannedAt *time.Time
		var hasBeenScanned bool

		if domain != "" {
			// For specific domain queries, use the original 4-column format
			var lastScannedAtNonNull time.Time
			err := rows.Scan(&domainName, &rawOutput, &lastScannedAtNonNull, &lastScanID)
			if err != nil {
				log.Printf("[KATANA-COMPANY] [ERROR] Error scanning raw domain result row: %v", err)
				continue
			}
			lastScannedAt = &lastScannedAtNonNull
			hasBeenScanned = true
		} else {
			// For general domain list queries, use the 5-column format with scan status
			err := rows.Scan(&domainName, &rawOutput, &lastScannedAt, &lastScanID, &hasBeenScanned)
			if err != nil {
				log.Printf("[KATANA-COMPANY] [ERROR] Error scanning raw domain result row: %v", err)
				continue
			}
		}

		result := map[string]interface{}{
			"domain":           domainName,
			"raw_output":       rawOutput,
			"last_scan_id":     lastScanID,
			"has_been_scanned": hasBeenScanned,
		}

		if lastScannedAt != nil {
			result["last_scanned_at"] = *lastScannedAt
		} else {
			result["last_scanned_at"] = nil
		}

		rawResults = append(rawResults, result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rawResults)
}

func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}

	v := make([]rune, 0, len(s))
	for i, r := range s {
		if r == utf8.RuneError {
			_, size := utf8.DecodeRuneInString(s[i:])
			if size == 1 {
				continue
			}
		}
		v = append(v, r)
	}
	return string(v)
}

// GetKatanaCompanyCloudAssetsByTarget retrieves all cloud assets for a scope target (all scans)
func GetKatanaCompanyCloudAssetsByTarget(w http.ResponseWriter, r *http.Request) {
	scopeTargetID := mux.Vars(r)["scope_target_id"]
	if scopeTargetID == "" {
		http.Error(w, "Scope target ID is required", http.StatusBadRequest)
		return
	}

	// Fetch all cloud assets for this scope target (across all scans)
	rows, err := dbPool.Query(context.Background(),
		`SELECT id, root_domain, asset_domain, asset_url, asset_type, service, description, source_url, last_scanned_at 
		 FROM katana_company_cloud_assets 
		 WHERE scope_target_id = $1 
		 ORDER BY last_scanned_at DESC, asset_url`,
		scopeTargetID)
	if err != nil {
		log.Printf("[KATANA-COMPANY] [ERROR] Failed to fetch cloud assets: %v", err)
		http.Error(w, "Failed to fetch cloud assets", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var cloudAssets []map[string]interface{}
	for rows.Next() {
		var id, rootDomain, assetDomain, assetURL, assetType, service, description string
		var sourceURL *string
		var lastScannedAt time.Time

		err := rows.Scan(&id, &rootDomain, &assetDomain, &assetURL, &assetType, &service, &description, &sourceURL, &lastScannedAt)
		if err != nil {
			log.Printf("[KATANA-COMPANY] [ERROR] Error scanning cloud asset row: %v", err)
			continue
		}

		sourceURLStr := ""
		if sourceURL != nil {
			sourceURLStr = *sourceURL
		}

		cloudAssets = append(cloudAssets, map[string]interface{}{
			"id":              id,
			"root_domain":     rootDomain,
			"domain":          assetDomain,
			"url":             assetURL,
			"type":            assetType,
			"service":         service,
			"description":     description,
			"source_url":      sourceURLStr,
			"created_at":      lastScannedAt,
			"last_scanned_at": lastScannedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cloudAssets)
}

// GetKatanaCompanyCloudFindingsByTarget retrieves all cloud findings for a scope target (all scans)
func GetKatanaCompanyCloudFindingsByTarget(w http.ResponseWriter, r *http.Request) {
	scopeTargetID := mux.Vars(r)["scope_target_id"]
	if scopeTargetID == "" {
		http.Error(w, "Scope target ID is required", http.StatusBadRequest)
		return
	}

	// Fetch all cloud findings for this scope target (across all scans)
	rows, err := dbPool.Query(context.Background(),
		`SELECT id, root_domain, finding_domain, finding_url, finding_type, content, description, cloud_service, context_before, context_after, match_position, last_scanned_at 
		 FROM katana_company_cloud_findings 
		 WHERE scope_target_id = $1 
		 ORDER BY last_scanned_at DESC, finding_url`,
		scopeTargetID)
	if err != nil {
		log.Printf("[KATANA-COMPANY] [ERROR] Failed to fetch cloud findings: %v", err)
		http.Error(w, "Failed to fetch cloud findings", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var cloudFindings []map[string]interface{}
	for rows.Next() {
		var id, rootDomain, findingDomain, findingURL, findingType, content, description, cloudService, contextBefore, contextAfter string
		var matchPosition int
		var lastScannedAt time.Time

		err := rows.Scan(&id, &rootDomain, &findingDomain, &findingURL, &findingType, &content, &description, &cloudService, &contextBefore, &contextAfter, &matchPosition, &lastScannedAt)
		if err != nil {
			log.Printf("[KATANA-COMPANY] [ERROR] Error scanning cloud finding row: %v", err)
			continue
		}

		cloudFindings = append(cloudFindings, map[string]interface{}{
			"id":              id,
			"root_domain":     rootDomain,
			"domain":          findingDomain,
			"url":             findingURL,
			"type":            findingType,
			"content":         content,
			"description":     description,
			"cloud_service":   cloudService,
			"context_before":  contextBefore,
			"context_after":   contextAfter,
			"match_position":  matchPosition,
			"created_at":      lastScannedAt,
			"last_scanned_at": lastScannedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cloudFindings)
}

// GetKatanaCompanyRawResultsByTarget retrieves all raw domain results for a scope target (all scans)
func GetKatanaCompanyRawResultsByTarget(w http.ResponseWriter, r *http.Request) {
	scopeTargetID := mux.Vars(r)["scope_target_id"]
	if scopeTargetID == "" {
		http.Error(w, "Scope target ID is required", http.StatusBadRequest)
		return
	}

	// Get domain parameter from query string (optional)
	domain := r.URL.Query().Get("domain")

	var query string
	var args []interface{}

	if domain != "" {
		// Fetch raw results for specific domain
		query = `SELECT domain, raw_output, last_scanned_at, last_scan_id 
		         FROM katana_company_domain_results 
		         WHERE scope_target_id = $1 AND domain = $2`
		args = []interface{}{scopeTargetID, domain}
	} else {
		// Fetch all domains that have ever been scanned for this scope target
		query = `SELECT domain, '' as raw_output, last_scanned_at, last_scan_id, true as has_been_scanned
		         FROM katana_company_domain_results 
		         WHERE scope_target_id = $1
		         ORDER BY last_scanned_at DESC, domain ASC`
		args = []interface{}{scopeTargetID}
	}

	rows, err := dbPool.Query(context.Background(), query, args...)
	if err != nil {
		log.Printf("[KATANA-COMPANY] [ERROR] Failed to fetch raw results: %v", err)
		http.Error(w, "Failed to fetch raw results", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var rawResults []map[string]interface{}
	for rows.Next() {
		var domainName, rawOutput string
		var lastScannedAt time.Time
		var lastScanID *string
		var hasBeenScanned bool

		if domain != "" {
			// Single domain query
			err := rows.Scan(&domainName, &rawOutput, &lastScannedAt, &lastScanID)
			if err != nil {
				log.Printf("[KATANA-COMPANY] [ERROR] Error scanning raw result row: %v", err)
				continue
			}
			hasBeenScanned = true
		} else {
			// All domains query
			err := rows.Scan(&domainName, &rawOutput, &lastScannedAt, &lastScanID, &hasBeenScanned)
			if err != nil {
				log.Printf("[KATANA-COMPANY] [ERROR] Error scanning raw result row: %v", err)
				continue
			}
		}

		lastScanIDStr := ""
		if lastScanID != nil {
			lastScanIDStr = *lastScanID
		}

		rawResults = append(rawResults, map[string]interface{}{
			"domain":           domainName,
			"raw_output":       rawOutput,
			"last_scanned_at":  lastScannedAt,
			"last_scan_id":     lastScanIDStr,
			"has_been_scanned": hasBeenScanned,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rawResults)
}
