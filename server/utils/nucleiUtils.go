package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NucleiFinding represents a single Nuclei finding from JSON output
type NucleiFinding struct {
	TemplateID  string     `json:"templateID"`
	Info        NucleiInfo `json:"info"`
	Type        string     `json:"type"`
	Host        string     `json:"host"`
	Matched     string     `json:"matched"`
	IP          string     `json:"ip"`
	Timestamp   string     `json:"timestamp"`
	MatcherName string     `json:"matcher_name,omitempty"`
	Extracted   []string   `json:"extracted_results,omitempty"`
	CurlCommand string     `json:"curl-command,omitempty"`
	Request     string     `json:"request,omitempty"`
	Response    string     `json:"response,omitempty"`
}

// NucleiInfo represents the info section of a Nuclei finding
type NucleiInfo struct {
	Name        string   `json:"name"`
	Author      []string `json:"author,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Reference   []string `json:"reference,omitempty"`
	Severity    string   `json:"severity"`
	Description string   `json:"description,omitempty"`
}

// convertAttackSurfaceAssetsToTargets converts attack surface assets to Nuclei-compatible target format
func convertAttackSurfaceAssetsToTargets(assetIDs []string, scopeTargetID string, dbPool *pgxpool.Pool) ([]string, error) {
	var targets []string

	log.Printf("[DEBUG] Converting %d asset IDs to Nuclei targets", len(assetIDs))

	for _, assetID := range assetIDs {
		var assetType, assetIdentifier string
		var asnNumber, cidrBlock, ipAddress, url, fqdn *string

		err := dbPool.QueryRow(context.Background(), `
			SELECT asset_type, asset_identifier, asn_number, cidr_block, ip_address, url, fqdn
			FROM consolidated_attack_surface_assets 
			WHERE id = $1 AND scope_target_id = $2
		`, assetID, scopeTargetID).Scan(&assetType, &assetIdentifier, &asnNumber, &cidrBlock, &ipAddress, &url, &fqdn)

		if err != nil {
			log.Printf("[WARN] Failed to get asset %s: %v", assetID, err)
			continue
		}

		log.Printf("[DEBUG] Processing asset %s: type=%s, identifier=%s", assetID, assetType, assetIdentifier)

		// Convert each asset type to appropriate Nuclei target format
		switch assetType {
		case "asn":
			if asnNumber != nil {
				target := "AS" + *asnNumber
				targets = append(targets, target)
				log.Printf("[DEBUG] Added ASN target: %s", target)
			}
		case "network_range":
			if cidrBlock != nil {
				targets = append(targets, *cidrBlock)
				log.Printf("[DEBUG] Added network range target: %s", *cidrBlock)
			}
		case "ip_address":
			if ipAddress != nil {
				targets = append(targets, *ipAddress)
				log.Printf("[DEBUG] Added IP target: %s", *ipAddress)
			}
		case "live_web_server":
			if url != nil {
				targets = append(targets, *url)
				log.Printf("[DEBUG] Added live web server target: %s", *url)
			}
		case "fqdn":
			if fqdn != nil {
				targets = append(targets, *fqdn)
				log.Printf("[DEBUG] Added FQDN target: %s", *fqdn)
			}
		case "cloud_asset":
			// For cloud assets, use the URL or FQDN if available
			if url != nil {
				targets = append(targets, *url)
				log.Printf("[DEBUG] Added cloud asset target (URL): %s", *url)
			} else if fqdn != nil {
				targets = append(targets, *fqdn)
				log.Printf("[DEBUG] Added cloud asset target (FQDN): %s", *fqdn)
			}
		}
	}

	log.Printf("[DEBUG] Converted %d asset IDs to %d Nuclei targets", len(assetIDs), len(targets))
	return targets, nil
}

// executeNucleiScan executes a Nuclei scan with the given parameters
func executeNucleiScan(targets []string, templates []string, severities []string, uploadedTemplates []map[string]interface{}, outputFile string) error {
	log.Printf("[DEBUG] Starting Nuclei scan with %d targets", len(targets))
	log.Printf("[DEBUG] Targets: %v", targets)
	log.Printf("[DEBUG] Templates: %v", templates)
	log.Printf("[DEBUG] Severities: %v", severities)

	// Create targets file
	targetsFile := outputFile + ".targets"
	targetsContent := strings.Join(targets, "\n")
	if err := ioutil.WriteFile(targetsFile, []byte(targetsContent), 0644); err != nil {
		return fmt.Errorf("failed to write targets file: %v", err)
	}
	defer os.Remove(targetsFile)

	log.Printf("[DEBUG] Created targets file: %s", targetsFile)
	log.Printf("[DEBUG] Targets file content:\n%s", targetsContent)

	// Prepare Nuclei command arguments
	var args []string
	args = append(args, "-l", targetsFile, "-jsonl", "-o", outputFile)

	// Add template categories
	if len(templates) > 0 {
		for _, template := range templates {
			switch template {
			case "cves":
				args = append(args, "-tags", "cve")
			case "vulnerabilities":
				args = append(args, "-tags", "vuln")
			case "exposures":
				args = append(args, "-tags", "exposure")
			case "technologies":
				args = append(args, "-tags", "tech")
			case "misconfiguration":
				args = append(args, "-tags", "misconfig")
			case "takeovers":
				args = append(args, "-tags", "takeover")
			case "network":
				args = append(args, "-tags", "network")
			case "dns":
				args = append(args, "-tags", "dns")
			case "headless":
				args = append(args, "-tags", "headless")
			}
		}
	}

	// Add severity filters
	if len(severities) > 0 {
		severityArgs := []string{}
		for _, severity := range severities {
			severityArgs = append(severityArgs, "-severity", severity)
		}
		args = append(args, severityArgs...)
	}

	// Handle custom templates
	if len(uploadedTemplates) > 0 {
		customTemplatesDir := filepath.Join(os.TempDir(), "nuclei_custom_"+strconv.FormatInt(time.Now().Unix(), 10))
		if err := os.MkdirAll(customTemplatesDir, 0755); err != nil {
			return fmt.Errorf("failed to create custom templates directory: %v", err)
		}
		defer os.RemoveAll(customTemplatesDir)

		for i, template := range uploadedTemplates {
			if content, ok := template["content"].(string); ok {
				templateFile := filepath.Join(customTemplatesDir, fmt.Sprintf("custom_%d.yaml", i))
				if err := ioutil.WriteFile(templateFile, []byte(content), 0644); err != nil {
					log.Printf("[WARN] Failed to write custom template %d: %v", i, err)
					continue
				}
			}
		}

		// Add custom templates directory to command
		args = append(args, "-t", customTemplatesDir)
	}

	// Build docker exec command
	dockerArgs := []string{"exec", "-i", "ars0n-framework-v2-nuclei-1", "nuclei"}
	dockerArgs = append(dockerArgs, args...)

	// Execute Nuclei command via docker exec
	log.Printf("[INFO] Executing Nuclei command: docker %s", strings.Join(dockerArgs, " "))
	dockerCmd := exec.Command("docker", dockerArgs...)

	// Capture output for debugging
	output, err := dockerCmd.CombinedOutput()
	log.Printf("[DEBUG] Docker command output: %s", string(output))

	if err != nil {
		log.Printf("[ERROR] Nuclei command failed: %v, output: %s", err, string(output))
		return fmt.Errorf("nuclei execution failed: %v", err)
	}

	log.Printf("[INFO] Nuclei scan completed successfully")
	log.Printf("[DEBUG] Output file should be created at: %s", outputFile)

	// Check if output file exists and has content
	if fileInfo, err := os.Stat(outputFile); err == nil {
		log.Printf("[DEBUG] Output file exists, size: %d bytes", fileInfo.Size())
		if fileInfo.Size() > 0 {
			content, _ := ioutil.ReadFile(outputFile)
			contentLen := len(content)
			if contentLen > 500 {
				contentLen = 500
			}
			log.Printf("[DEBUG] Output file content (first %d chars): %s", contentLen, string(content[:contentLen]))
		}
	} else {
		log.Printf("[DEBUG] Output file does not exist: %v", err)
	}

	return nil
}

// parseNucleiResults parses the JSON output from Nuclei and returns findings
func parseNucleiResults(outputFile string) ([]NucleiFinding, error) {
	var findings []NucleiFinding

	log.Printf("[DEBUG] Parsing Nuclei results from file: %s", outputFile)

	content, err := ioutil.ReadFile(outputFile)
	if err != nil {
		log.Printf("[ERROR] Failed to read results file: %v", err)
		return findings, fmt.Errorf("failed to read results file: %v", err)
	}

	log.Printf("[DEBUG] Read %d bytes from results file", len(content))
	log.Printf("[DEBUG] Results file content:\n%s", string(content))

	// Parse JSON Lines format (one JSON object per line)
	lines := strings.Split(string(content), "\n")
	log.Printf("[DEBUG] Found %d lines in results file", len(lines))

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		log.Printf("[DEBUG] Parsing line %d: %s", lineNum+1, line)

		var finding NucleiFinding
		if err := json.Unmarshal([]byte(line), &finding); err != nil {
			log.Printf("[WARN] Failed to parse JSON on line %d: %v", lineNum+1, err)
			continue
		}

		log.Printf("[DEBUG] Successfully parsed finding: %s", finding.TemplateID)
		findings = append(findings, finding)
	}

	log.Printf("[INFO] Parsed %d findings from Nuclei results", len(findings))
	return findings, nil
}

// ExecuteNucleiScanForScopeTarget executes a complete Nuclei scan for a scope target
func ExecuteNucleiScanForScopeTarget(scopeTargetID string, selectedTargets []string, selectedTemplates []string, selectedSeverities []string, uploadedTemplates []map[string]interface{}, dbPool *pgxpool.Pool) (string, []NucleiFinding, error) {
	// Convert attack surface assets to Nuclei targets
	targets, err := convertAttackSurfaceAssetsToTargets(selectedTargets, scopeTargetID, dbPool)
	if err != nil {
		return "", nil, fmt.Errorf("failed to convert targets: %v", err)
	}

	if len(targets) == 0 {
		return "", nil, fmt.Errorf("no valid targets found")
	}

	// Create output file
	outputDir := filepath.Join(os.TempDir(), "nuclei_scans")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	outputFile := filepath.Join(outputDir, fmt.Sprintf("nuclei_scan_%s_%d.jsonl", scopeTargetID, time.Now().Unix()))

	// Execute the scan
	if err := executeNucleiScan(targets, selectedTemplates, selectedSeverities, uploadedTemplates, outputFile); err != nil {
		return "", nil, fmt.Errorf("scan execution failed: %v", err)
	}

	// Parse results
	findings, err := parseNucleiResults(outputFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse results: %v", err)
	}

	return outputFile, findings, nil
}
