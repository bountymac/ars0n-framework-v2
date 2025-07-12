package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type AttackSurfaceAsset struct {
	ID              string  `json:"id"`
	ScopeTargetID   string  `json:"scope_target_id"`
	AssetType       string  `json:"asset_type"`
	AssetIdentifier string  `json:"asset_identifier"`
	AssetSubtype    *string `json:"asset_subtype,omitempty"`

	// ASN fields
	ASNNumber       *string `json:"asn_number,omitempty"`
	ASNOrganization *string `json:"asn_organization,omitempty"`
	ASNDescription  *string `json:"asn_description,omitempty"`
	ASNCountry      *string `json:"asn_country,omitempty"`

	// Network Range fields
	CIDRBlock *string `json:"cidr_block,omitempty"`

	// IP Address fields
	IPAddress *string `json:"ip_address,omitempty"`
	IPType    *string `json:"ip_type,omitempty"`

	// Live Web Server fields
	URL                 *string                `json:"url,omitempty"`
	Domain              *string                `json:"domain,omitempty"`
	Port                *int                   `json:"port,omitempty"`
	Protocol            *string                `json:"protocol,omitempty"`
	StatusCode          *int                   `json:"status_code,omitempty"`
	Title               *string                `json:"title,omitempty"`
	WebServer           *string                `json:"web_server,omitempty"`
	Technologies        []string               `json:"technologies,omitempty"`
	ContentLength       *int                   `json:"content_length,omitempty"`
	ResponseTime        *float64               `json:"response_time_ms,omitempty"`
	ScreenshotPath      *string                `json:"screenshot_path,omitempty"`
	SSLInfo             map[string]interface{} `json:"ssl_info,omitempty"`
	HTTPResponseHeaders map[string]interface{} `json:"http_response_headers,omitempty"`
	FindingsJSON        map[string]interface{} `json:"findings_json,omitempty"`

	// Cloud Asset fields
	CloudProvider    *string `json:"cloud_provider,omitempty"`
	CloudServiceType *string `json:"cloud_service_type,omitempty"`
	CloudRegion      *string `json:"cloud_region,omitempty"`

	// FQDN fields
	FQDN           *string                `json:"fqdn,omitempty"`
	RootDomain     *string                `json:"root_domain,omitempty"`
	Subdomain      *string                `json:"subdomain,omitempty"`
	Registrar      *string                `json:"registrar,omitempty"`
	CreationDate   *time.Time             `json:"creation_date,omitempty"`
	ExpirationDate *time.Time             `json:"expiration_date,omitempty"`
	UpdatedDate    *time.Time             `json:"updated_date,omitempty"`
	NameServers    []string               `json:"name_servers,omitempty"`
	Status         []string               `json:"status,omitempty"`
	WhoisInfo      map[string]interface{} `json:"whois_info,omitempty"`
	SSLCertificate map[string]interface{} `json:"ssl_certificate,omitempty"`
	SSLExpiryDate  *time.Time             `json:"ssl_expiry_date,omitempty"`
	SSLIssuer      *string                `json:"ssl_issuer,omitempty"`
	SSLSubject     *string                `json:"ssl_subject,omitempty"`
	SSLVersion     *string                `json:"ssl_version,omitempty"`
	SSLCipherSuite *string                `json:"ssl_cipher_suite,omitempty"`
	SSLProtocols   []string               `json:"ssl_protocols,omitempty"`
	ResolvedIPs    []string               `json:"resolved_ips,omitempty"`
	MailServers    []string               `json:"mail_servers,omitempty"`
	SPFRecord      *string                `json:"spf_record,omitempty"`
	DKIMRecord     *string                `json:"dkim_record,omitempty"`
	DMARCRecord    *string                `json:"dmarc_record,omitempty"`
	CAARecords     []string               `json:"caa_records,omitempty"`
	TXTRecords     []string               `json:"txt_records,omitempty"`
	MXRecords      []string               `json:"mx_records,omitempty"`
	NSRecords      []string               `json:"ns_records,omitempty"`
	ARecords       []string               `json:"a_records,omitempty"`
	AAAARecords    []string               `json:"aaaa_records,omitempty"`
	CNAMERecords   []string               `json:"cname_records,omitempty"`
	PTRRecords     []string               `json:"ptr_records,omitempty"`
	SRVRecords     []string               `json:"srv_records,omitempty"`
	SOARecord      map[string]interface{} `json:"soa_record,omitempty"`
	LastDNSScan    *time.Time             `json:"last_dns_scan,omitempty"`
	LastSSLScan    *time.Time             `json:"last_ssl_scan,omitempty"`
	LastWhoisScan  *time.Time             `json:"last_whois_scan,omitempty"`

	LastUpdated time.Time `json:"last_updated"`
	CreatedAt   time.Time `json:"created_at"`

	// Related data
	DNSRecords    []AttackSurfaceDNSRecord `json:"dns_records,omitempty"`
	Relationships []AssetRelationship      `json:"relationships,omitempty"`
}

type AttackSurfaceDNSRecord struct {
	ID          string    `json:"id"`
	AssetID     string    `json:"asset_id"`
	RecordType  string    `json:"record_type"`
	RecordValue string    `json:"record_value"`
	TTL         *int      `json:"ttl,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type AssetRelationship struct {
	ID               string                 `json:"id"`
	ParentAssetID    string                 `json:"parent_asset_id"`
	ChildAssetID     string                 `json:"child_asset_id"`
	RelationshipType string                 `json:"relationship_type"`
	RelationshipData map[string]interface{} `json:"relationship_data,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
}

type ConsolidationResult struct {
	TotalAssets        int                  `json:"total_assets"`
	ASNs               int                  `json:"asns"`
	NetworkRanges      int                  `json:"network_ranges"`
	IPAddresses        int                  `json:"ip_addresses"`
	LiveWebServers     int                  `json:"live_web_servers"`
	CloudAssets        int                  `json:"cloud_assets"`
	FQDNs              int                  `json:"fqdns"`
	TotalRelationships int                  `json:"total_relationships"`
	Assets             []AttackSurfaceAsset `json:"assets"`
	ExecutionTime      string               `json:"execution_time"`
	ConsolidatedAt     time.Time            `json:"consolidated_at"`
}

func ConsolidateAttackSurface(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["scope_target_id"]

	if scopeTargetID == "" {
		http.Error(w, "Missing scope_target_id", http.StatusBadRequest)
		return
	}

	log.Printf("[ATTACK SURFACE] Starting consolidation for scope target: %s", scopeTargetID)
	startTime := time.Now()

	// Clear existing data
	log.Printf("[ATTACK SURFACE] Clearing existing attack surface data...")
	err := clearExistingAttackSurfaceData(scopeTargetID)
	if err != nil {
		log.Printf("Error clearing existing attack surface data: %v", err)
		http.Error(w, "Failed to clear existing data", http.StatusInternalServerError)
		return
	}
	log.Printf("[ATTACK SURFACE] Successfully cleared existing data")

	// Consolidate each asset type
	log.Printf("[ATTACK SURFACE] Consolidating ASNs...")
	asns, err := consolidateASNs(scopeTargetID)
	if err != nil {
		log.Printf("Error consolidating ASNs: %v", err)
		http.Error(w, "Failed to consolidate ASNs", http.StatusInternalServerError)
		return
	}
	log.Printf("[ATTACK SURFACE] Consolidated %d ASNs", asns)

	log.Printf("[ATTACK SURFACE] Consolidating network ranges...")
	networkRanges, err := consolidateNetworkRanges(scopeTargetID)
	if err != nil {
		log.Printf("Error consolidating network ranges: %v", err)
		http.Error(w, "Failed to consolidate network ranges", http.StatusInternalServerError)
		return
	}
	log.Printf("[ATTACK SURFACE] Consolidated %d network ranges", networkRanges)

	log.Printf("[ATTACK SURFACE] Consolidating IP addresses...")
	ipAddresses, err := consolidateIPAddresses(scopeTargetID)
	if err != nil {
		log.Printf("Error consolidating IP addresses: %v", err)
		http.Error(w, "Failed to consolidate IP addresses", http.StatusInternalServerError)
		return
	}
	log.Printf("[ATTACK SURFACE] Consolidated %d IP addresses", ipAddresses)

	log.Printf("[ATTACK SURFACE] Consolidating live web servers...")
	liveWebServers, err := consolidateLiveWebServers(scopeTargetID)
	if err != nil {
		log.Printf("Error consolidating live web servers: %v", err)
		http.Error(w, "Failed to consolidate live web servers", http.StatusInternalServerError)
		return
	}
	log.Printf("[ATTACK SURFACE] Consolidated %d live web servers", liveWebServers)

	log.Printf("[ATTACK SURFACE] Consolidating cloud assets...")
	cloudAssets, err := consolidateCloudAssets(scopeTargetID)
	if err != nil {
		log.Printf("Error consolidating cloud assets: %v", err)
		http.Error(w, "Failed to consolidate cloud assets", http.StatusInternalServerError)
		return
	}
	log.Printf("[ATTACK SURFACE] Consolidated %d cloud assets", cloudAssets)

	// Consolidate FQDNs
	log.Printf("[ATTACK SURFACE] Consolidating FQDNs...")
	fqdns, err := consolidateFQDNs(scopeTargetID)
	if err != nil {
		log.Printf("Error consolidating FQDNs: %v", err)
		http.Error(w, "Failed to consolidate FQDNs", http.StatusInternalServerError)
		return
	}
	log.Printf("[ATTACK SURFACE] Consolidated %d FQDNs", fqdns)

	// Create relationships between assets
	log.Printf("[ATTACK SURFACE] Creating asset relationships...")
	relationshipCount, err := createAssetRelationships(scopeTargetID)
	if err != nil {
		log.Printf("Error creating asset relationships: %v", err)
		http.Error(w, "Failed to create asset relationships", http.StatusInternalServerError)
		return
	}
	log.Printf("[ATTACK SURFACE] Created %d asset relationships", relationshipCount)

	// Fetch all consolidated assets
	log.Printf("[ATTACK SURFACE] Fetching consolidated assets...")
	assets, err := fetchConsolidatedAssets(scopeTargetID)
	if err != nil {
		log.Printf("Error fetching consolidated assets: %v", err)
		http.Error(w, "Failed to fetch consolidated assets", http.StatusInternalServerError)
		return
	}

	executionTime := time.Since(startTime)

	result := ConsolidationResult{
		TotalAssets:        len(assets),
		ASNs:               asns,
		NetworkRanges:      networkRanges,
		IPAddresses:        ipAddresses,
		LiveWebServers:     liveWebServers,
		CloudAssets:        cloudAssets,
		FQDNs:              fqdns,
		TotalRelationships: relationshipCount,
		Assets:             assets,
		ExecutionTime:      executionTime.String(),
		ConsolidatedAt:     time.Now(),
	}

	// Log the final results
	log.Printf("[ATTACK SURFACE] ✅ CONSOLIDATION COMPLETE!")
	log.Printf("[ATTACK SURFACE] Summary for scope target %s:", scopeTargetID)
	log.Printf("[ATTACK SURFACE]   • Total Assets: %d", len(assets))
	log.Printf("[ATTACK SURFACE]   • ASNs: %d", asns)
	log.Printf("[ATTACK SURFACE]   • Network Ranges: %d", networkRanges)
	log.Printf("[ATTACK SURFACE]   • IP Addresses: %d", ipAddresses)
	log.Printf("[ATTACK SURFACE]   • Live Web Servers: %d", liveWebServers)
	log.Printf("[ATTACK SURFACE]   • Cloud Assets: %d", cloudAssets)
	log.Printf("[ATTACK SURFACE]   • FQDNs: %d", fqdns)
	log.Printf("[ATTACK SURFACE]   • Asset Relationships: %d", relationshipCount)
	log.Printf("[ATTACK SURFACE]   • Execution Time: %s", executionTime.String())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func GetAttackSurfaceAssetCounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	scopeTargetID := vars["scope_target_id"]

	if scopeTargetID == "" {
		http.Error(w, "scope_target_id is required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT 
			asset_type,
			COUNT(*) as count
		FROM consolidated_attack_surface_assets
		WHERE scope_target_id = $1::uuid
		GROUP BY asset_type
	`

	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("Error querying attack surface asset counts: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	counts := map[string]int{
		"asns":             0,
		"network_ranges":   0,
		"ip_addresses":     0,
		"live_web_servers": 0,
		"cloud_assets":     0,
		"fqdns":            0,
	}

	for rows.Next() {
		var assetType string
		var count int

		if err := rows.Scan(&assetType, &count); err != nil {
			log.Printf("Error scanning attack surface asset count row: %v", err)
			continue
		}

		switch assetType {
		case "asn":
			counts["asns"] = count
		case "network_range":
			counts["network_ranges"] = count
		case "ip_address":
			counts["ip_addresses"] = count
		case "live_web_server":
			counts["live_web_servers"] = count
		case "cloud_asset":
			counts["cloud_assets"] = count
		case "fqdn":
			counts["fqdns"] = count
		}
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating attack surface asset count rows: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(counts)
}

func clearExistingAttackSurfaceData(scopeTargetID string) error {
	queries := []string{
		"DELETE FROM consolidated_attack_surface_metadata WHERE asset_id IN (SELECT id FROM consolidated_attack_surface_assets WHERE scope_target_id = $1::uuid)",
		"DELETE FROM consolidated_attack_surface_dns_records WHERE asset_id IN (SELECT id FROM consolidated_attack_surface_assets WHERE scope_target_id = $1::uuid)",
		"DELETE FROM consolidated_attack_surface_relationships WHERE parent_asset_id IN (SELECT id FROM consolidated_attack_surface_assets WHERE scope_target_id = $1::uuid) OR child_asset_id IN (SELECT id FROM consolidated_attack_surface_assets WHERE scope_target_id = $1::uuid)",
		"DELETE FROM consolidated_attack_surface_assets WHERE scope_target_id = $1::uuid",
	}

	for _, query := range queries {
		_, err := dbPool.Exec(context.Background(), query, scopeTargetID)
		if err != nil {
			return fmt.Errorf("failed to execute query %s: %v", query, err)
		}
	}

	return nil
}

func consolidateASNs(scopeTargetID string) (int, error) {
	log.Printf("[ASN CONSOLIDATION] Starting ASN consolidation for scope target: %s", scopeTargetID)

	// Helper function to normalize ASN format
	normalizeASN := func(asn string) string {
		// Remove 'AS' prefix if present and trim whitespace
		normalized := strings.TrimSpace(strings.TrimPrefix(strings.ToUpper(asn), "AS"))
		// Ensure it's a valid ASN number (numeric)
		if _, err := strconv.Atoi(normalized); err == nil {
			return normalized
		}
		// If not numeric, return original but cleaned
		return strings.TrimSpace(asn)
	}

	// Query to get ASNs from each source with detailed logging
	query := `
		WITH amass_intel_asns AS (
			SELECT asn_number, organization, description, country, 'Amass Intel' as source
			FROM intel_asn_data ia
			JOIN amass_intel_scans ais ON ia.scan_id = ais.scan_id
			WHERE ais.scope_target_id = $1::uuid AND ais.status = 'success'
		),
		metabigor_asns AS (
			SELECT 
				jsonb_array_elements_text(result::jsonb->'asns') as asn_number,
				'Unknown' as organization,
				'Discovered by Metabigor' as description,
				'Unknown' as country,
				'Metabigor' as source
			FROM metabigor_company_scans
			WHERE scope_target_id = $1::uuid AND status = 'success' AND result IS NOT NULL
				AND result::jsonb ? 'asns'
		),
		                      amass_enum_asns AS (
                              SELECT
                                      unnest(regexp_matches(result, 'AS\d+', 'g')) as asn_number,
                                      'Unknown' as organization,
                                      'Discovered by Amass Enum' as description,
                                      'Unknown' as country,
                                      'Amass Enum' as source
                              FROM amass_enum_company_scans
                              WHERE scope_target_id = $1::uuid AND status = 'success' AND result IS NOT NULL
                      ),
		                      wildcard_asns AS (
                              SELECT
                                      unnest(regexp_matches(result, 'AS\d+', 'g')) as asn_number,
                                      'Unknown' as organization,
                                      'Discovered by Wildcard Amass' as description,
                                      'Unknown' as country,
                                      'Wildcard Amass' as source
                              FROM amass_scans am
                              JOIN scope_targets st ON am.scope_target_id = st.id
                              WHERE st.type = 'Wildcard'
                                      AND st.scope_target IN (
                                              SELECT DISTINCT domain
                                              FROM consolidated_company_domains
                                              WHERE scope_target_id = $1::uuid
                                      )
                                      AND am.status = 'success' AND am.result IS NOT NULL
                      ),
		network_range_asns AS (
			SELECT asn as asn_number, organization, description, country, 'Network Ranges' as source
			FROM consolidated_network_ranges
			WHERE scope_target_id = $1::uuid AND asn IS NOT NULL
		),
		all_asns AS (
			SELECT * FROM amass_intel_asns
			UNION ALL
			SELECT * FROM metabigor_asns
			UNION ALL
			SELECT * FROM amass_enum_asns
			UNION ALL
			SELECT * FROM wildcard_asns
			UNION ALL
			SELECT * FROM network_range_asns
		)
		SELECT asn_number, organization, description, country, source
		FROM all_asns
		WHERE asn_number IS NOT NULL AND asn_number != ''
		ORDER BY source, asn_number
	`

	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[ASN CONSOLIDATION] Error querying ASN sources: %v", err)
		return 0, err
	}
	defer rows.Close()

	// Track ASNs by source for logging
	sourceCounts := make(map[string]int)
	allASNs := make(map[string]map[string]string) // normalized_asn_number -> {organization, description, country, source}

	for rows.Next() {
		var asnNumber, organization, description, country, source string
		err := rows.Scan(&asnNumber, &organization, &description, &country, &source)
		if err != nil {
			log.Printf("[ASN CONSOLIDATION] Error scanning ASN row: %v", err)
			continue
		}

		// Normalize the ASN number
		normalizedASN := normalizeASN(asnNumber)
		if normalizedASN == "" {
			continue
		}

		sourceCounts[source]++
		allASNs[normalizedASN] = map[string]string{
			"organization": organization,
			"description":  description,
			"country":      country,
			"source":       source,
		}

		log.Printf("[ASN CONSOLIDATION] Found ASN %s (normalized: %s) from %s (Org: %s, Country: %s)",
			asnNumber, normalizedASN, source, organization, country)
	}

	// Log summary by source
	log.Printf("[ASN CONSOLIDATION] ASN discovery summary:")
	for source, count := range sourceCounts {
		log.Printf("[ASN CONSOLIDATION]   • %s: %d ASNs", source, count)
	}

	// Log unique ASNs
	log.Printf("[ASN CONSOLIDATION] Unique ASNs found: %d", len(allASNs))
	for asnNumber, details := range allASNs {
		log.Printf("[ASN CONSOLIDATION]   • %s (Org: %s, Country: %s, Source: %s)",
			asnNumber, details["organization"], details["country"], details["source"])
	}

	// Now insert the consolidated ASNs with proper deduplication and normalization
	insertQuery := `
		INSERT INTO consolidated_attack_surface_assets (
			scope_target_id, asset_type, asset_identifier, 
			asn_number, asn_organization, asn_description, asn_country
		)
		SELECT DISTINCT ON (normalized_asn)
			$1::uuid, 'asn', normalized_asn,
			normalized_asn, 
			COALESCE(NULLIF(organization, 'Unknown'), 'Unknown') as organization,
			COALESCE(NULLIF(description, 'Unknown'), 'Discovered') as description,
			COALESCE(NULLIF(country, 'Unknown'), 'Unknown') as country
		FROM (
			-- 1. Amass Intel ASN data (highest priority)
			SELECT 
				TRIM(BOTH 'AS' FROM UPPER(asn_number)) as normalized_asn,
				organization, description, country, 1 as priority
			FROM intel_asn_data ia
			JOIN amass_intel_scans ais ON ia.scan_id = ais.scan_id
			WHERE ais.scope_target_id = $1::uuid AND ais.status = 'success'
				AND asn_number IS NOT NULL AND asn_number != ''
			
			UNION ALL
			
			-- 2. Network range ASN data (second priority)
			SELECT 
				TRIM(BOTH 'AS' FROM UPPER(asn)) as normalized_asn,
				organization, description, country, 2 as priority
			FROM consolidated_network_ranges
			WHERE scope_target_id = $1::uuid AND asn IS NOT NULL AND asn != ''
			
			UNION ALL
			
			-- 3. Metabigor ASN data (third priority)
			SELECT 
				TRIM(BOTH 'AS' FROM UPPER(jsonb_array_elements_text(result::jsonb->'asns'))) as normalized_asn,
				'Unknown' as organization,
				'Discovered by Metabigor' as description,
				'Unknown' as country,
				3 as priority
			FROM metabigor_company_scans
			WHERE scope_target_id = $1::uuid AND status = 'success' AND result IS NOT NULL
				AND result::jsonb ? 'asns'
			
			UNION ALL
			
			-- 4. Amass Enum raw results (fourth priority)
			SELECT 
				TRIM(BOTH 'AS' FROM UPPER(unnest(regexp_matches(result, 'AS\d+', 'g')))) as normalized_asn,
				'Unknown' as organization,
				'Discovered by Amass Enum' as description,
				'Unknown' as country,
				4 as priority
			FROM amass_enum_company_scans
			WHERE scope_target_id = $1::uuid AND status = 'success' AND result IS NOT NULL
			
			UNION ALL
			
			-- 5. Wildcard Amass scans (lowest priority)
			SELECT 
				TRIM(BOTH 'AS' FROM UPPER(unnest(regexp_matches(result, 'AS\d+', 'g')))) as normalized_asn,
				'Unknown' as organization,
				'Discovered by Wildcard Amass' as description,
				'Unknown' as country,
				5 as priority
			FROM amass_scans am
			JOIN scope_targets st ON am.scope_target_id = st.id
			WHERE st.type = 'Wildcard' 
				AND st.scope_target IN (
					SELECT DISTINCT domain 
					FROM consolidated_company_domains 
					WHERE scope_target_id = $1::uuid
				)
				AND am.status = 'success' AND am.result IS NOT NULL
		) asn_data
		WHERE normalized_asn IS NOT NULL AND normalized_asn != ''
			AND normalized_asn ~ '^[0-9]+$'
		ORDER BY normalized_asn, priority
		ON CONFLICT (scope_target_id, asset_type, asset_identifier) DO UPDATE SET
			asn_organization = EXCLUDED.asn_organization,
			asn_description = EXCLUDED.asn_description,
			asn_country = EXCLUDED.asn_country,
			last_updated = NOW()
	`

	result, err := dbPool.Exec(context.Background(), insertQuery, scopeTargetID)
	if err != nil {
		log.Printf("[ASN CONSOLIDATION] Error inserting consolidated ASNs: %v", err)
		return 0, err
	}

	insertedCount := int(result.RowsAffected())
	log.Printf("[ASN CONSOLIDATION] ✅ Successfully inserted/updated %d ASN records", insertedCount)
	log.Printf("[ASN CONSOLIDATION] ASN consolidation complete for scope target: %s", scopeTargetID)

	return insertedCount, nil
}

func consolidateNetworkRanges(scopeTargetID string) (int, error) {
	query := `
		INSERT INTO consolidated_attack_surface_assets (
			scope_target_id, asset_type, asset_identifier, cidr_block
		)
		SELECT DISTINCT 
			$1::uuid, 'network_range', cidr_block, cidr_block
		FROM (
			-- 1. Amass Intel network ranges (highest priority)
			SELECT cidr_block
			FROM intel_network_ranges inr
			JOIN amass_intel_scans ais ON inr.scan_id = ais.scan_id
			WHERE ais.scope_target_id = $1::uuid AND ais.status = 'success'
			
			UNION
			
			-- 2. Previously consolidated network ranges
			SELECT cidr_block
			FROM consolidated_network_ranges
			WHERE scope_target_id = $1::uuid
			
			UNION
			
			-- 3. Amass Enum company scan raw results (extract CIDR blocks)
			SELECT unnest(regexp_matches(result, '\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2}\b', 'g')) as cidr_block
			FROM amass_enum_company_scans
			WHERE scope_target_id = $1::uuid AND status = 'success' AND result IS NOT NULL
			
			UNION
			
			-- 4. Wildcard Amass scans for company root domains
			SELECT unnest(regexp_matches(result, '\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2}\b', 'g')) as cidr_block
			FROM amass_scans am
			JOIN scope_targets st ON am.scope_target_id = st.id
			WHERE st.type = 'Wildcard'
				AND st.scope_target IN (
					SELECT DISTINCT domain 
					FROM consolidated_company_domains 
					WHERE scope_target_id = $1::uuid
				)
				AND am.status = 'success' AND am.result IS NOT NULL
		) range_data
		WHERE cidr_block IS NOT NULL AND cidr_block != ''
		ON CONFLICT (scope_target_id, asset_type, asset_identifier) DO UPDATE SET
			last_updated = NOW()
	`

	result, err := dbPool.Exec(context.Background(), query, scopeTargetID)
	if err != nil {
		return 0, err
	}

	return int(result.RowsAffected()), nil
}

func consolidateIPAddresses(scopeTargetID string) (int, error) {
	log.Printf("[IP CONSOLIDATION] Starting IP address consolidation for scope target: %s", scopeTargetID)

	// First, let's check what data we have
	debugQuery := `
		SELECT 
			'discovered_live_ips' as source,
			COUNT(*) as count
		FROM discovered_live_ips dli
		JOIN ip_port_scans ips ON dli.scan_id = ips.scan_id
		WHERE ips.scope_target_id = $1::uuid AND ips.status = 'success'
		
		UNION ALL
		
		SELECT 
			'live_web_servers' as source,
			COUNT(*) as count
		FROM live_web_servers lws
		JOIN ip_port_scans ips ON lws.scan_id = ips.scan_id
		WHERE ips.scope_target_id = $1::uuid AND ips.status = 'success'
		
		UNION ALL
		
		SELECT 
			'live_web_servers_all' as source,
			COUNT(*) as count
		FROM live_web_servers lws
		WHERE lws.scan_id IN (
			SELECT scan_id FROM ip_port_scans WHERE scope_target_id = $1::uuid
		)
		
		UNION ALL
		
		SELECT 
			'ip_port_scans_success' as source,
			COUNT(*) as count
		FROM ip_port_scans
		WHERE scope_target_id = $1::uuid AND status = 'success'
		
		UNION ALL
		
		SELECT 
			'ip_port_scans_all' as source,
			COUNT(*) as count
		FROM ip_port_scans
		WHERE scope_target_id = $1::uuid
	`

	debugRows, err := dbPool.Query(context.Background(), debugQuery, scopeTargetID)
	if err != nil {
		log.Printf("[IP CONSOLIDATION] Error in debug query: %v", err)
	} else {
		defer debugRows.Close()
		for debugRows.Next() {
			var source, count string
			if err := debugRows.Scan(&source, &count); err == nil {
				log.Printf("[IP CONSOLIDATION] Debug - %s: %s records", source, count)
			}
		}
	}

	query := `
		INSERT INTO consolidated_attack_surface_assets (
			scope_target_id, asset_type, asset_identifier, ip_address, ip_type
		)
		SELECT DISTINCT 
			$1::uuid, 'ip_address', ip_address::text, ip_address,
			CASE 
				WHEN family(ip_address) = 4 THEN 'ipv4'
				WHEN family(ip_address) = 6 THEN 'ipv6'
				ELSE 'unknown'
			END
		FROM (
			SELECT DISTINCT ip_address
			FROM discovered_live_ips dli
			JOIN ip_port_scans ips ON dli.scan_id = ips.scan_id
			WHERE ips.scope_target_id = $1::uuid AND ips.status = 'success'
			
			UNION
			
			SELECT DISTINCT ip_address
			FROM live_web_servers lws
			JOIN ip_port_scans ips ON lws.scan_id = ips.scan_id
			WHERE ips.scope_target_id = $1::uuid AND ips.status = 'success'
		) ip_data
		ON CONFLICT (scope_target_id, asset_type, asset_identifier) DO UPDATE SET
			last_updated = NOW()
	`

	result, err := dbPool.Exec(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[IP CONSOLIDATION] Error inserting consolidated IP addresses: %v", err)
		return 0, err
	}

	insertedCount := int(result.RowsAffected())
	log.Printf("[IP CONSOLIDATION] ✅ Successfully inserted/updated %d IP address records", insertedCount)
	log.Printf("[IP CONSOLIDATION] IP address consolidation complete for scope target: %s", scopeTargetID)

	return insertedCount, nil
}

func consolidateLiveWebServers(scopeTargetID string) (int, error) {
	// First, consolidate IP/Port live web servers
	ipPortQuery := `
		INSERT INTO consolidated_attack_surface_assets (
			scope_target_id, asset_type, asset_subtype, asset_identifier,
			ip_address, port, protocol, url, status_code, title, web_server,
			technologies, content_length, response_time_ms, screenshot_path
		)
		SELECT DISTINCT 
			$1::uuid, 'live_web_server', 'ip_port', 
			ip_address::text || ':' || port::text || '/' || protocol,
			ip_address, port, protocol, url, status_code, title, server_header,
			CASE 
				WHEN technologies IS NOT NULL THEN 
					CASE 
						WHEN jsonb_typeof(technologies) = 'array' THEN 
							ARRAY(SELECT jsonb_array_elements_text(technologies))
						WHEN jsonb_typeof(technologies) = 'string' THEN 
							ARRAY[technologies #>> '{}']
						ELSE ARRAY[]::text[]
					END
				ELSE ARRAY[]::text[]
			END,
			content_length, response_time_ms, screenshot_path
		FROM live_web_servers lws
		JOIN ip_port_scans ips ON lws.scan_id = ips.scan_id
		WHERE ips.scope_target_id = $1::uuid AND ips.status = 'success'
		ON CONFLICT (scope_target_id, asset_type, asset_identifier) DO UPDATE SET
			status_code = EXCLUDED.status_code,
			title = EXCLUDED.title,
			web_server = EXCLUDED.web_server,
			technologies = EXCLUDED.technologies,
			content_length = EXCLUDED.content_length,
			response_time_ms = EXCLUDED.response_time_ms,
			screenshot_path = EXCLUDED.screenshot_path,
			last_updated = NOW()
	`

	ipPortResult, err := dbPool.Exec(context.Background(), ipPortQuery, scopeTargetID)
	if err != nil {
		return 0, err
	}

	// Then, consolidate Domain live web servers from investigate scans
	domainInvestigateQuery := `
		INSERT INTO consolidated_attack_surface_assets (
			scope_target_id, asset_type, asset_subtype, asset_identifier,
			domain, url
		)
		SELECT DISTINCT 
			$1::uuid, 'live_web_server', 'domain', 
			domain_name,
			domain_name, 'https://' || domain_name
		FROM (
			SELECT DISTINCT
				jsonb_array_elements(result::jsonb)->>'domain' as domain_name
			FROM investigate_scans
			WHERE scope_target_id = $1::uuid AND status = 'success' AND result IS NOT NULL
				AND result::jsonb IS NOT NULL
		) domain_data
		WHERE domain_name IS NOT NULL AND domain_name != ''
		ON CONFLICT (scope_target_id, asset_type, asset_identifier) DO UPDATE SET
			last_updated = NOW()
	`

	domainInvestigateResult, err := dbPool.Exec(context.Background(), domainInvestigateQuery, scopeTargetID)
	if err != nil {
		return 0, err
	}

	// Finally, consolidate Domain live web servers from target_urls (wildcard targets)
	domainTargetQuery := `
		INSERT INTO consolidated_attack_surface_assets (
			scope_target_id, asset_type, asset_subtype, asset_identifier,
			domain, url, status_code, title, web_server, technologies,
			content_length, screenshot_path
		)
		SELECT DISTINCT 
			$1::uuid, 'live_web_server', 'domain', 
			url,
			CASE 
				WHEN url LIKE 'http://%' THEN substring(url from 8)
				WHEN url LIKE 'https://%' THEN substring(url from 9)
				ELSE url
			END,
			url, status_code, title, web_server, technologies,
			content_length, screenshot
		FROM target_urls
		WHERE scope_target_id = $1::uuid AND no_longer_live = false
		ON CONFLICT (scope_target_id, asset_type, asset_identifier) DO UPDATE SET
			status_code = EXCLUDED.status_code,
			title = EXCLUDED.title,
			web_server = EXCLUDED.web_server,
			technologies = EXCLUDED.technologies,
			content_length = EXCLUDED.content_length,
			screenshot_path = EXCLUDED.screenshot_path,
			last_updated = NOW()
	`

	domainTargetResult, err := dbPool.Exec(context.Background(), domainTargetQuery, scopeTargetID)
	if err != nil {
		return 0, err
	}

	totalRows := int(ipPortResult.RowsAffected()) + int(domainInvestigateResult.RowsAffected()) + int(domainTargetResult.RowsAffected())
	return totalRows, nil
}

func consolidateCloudAssets(scopeTargetID string) (int, error) {
	// Consolidate all cloud assets in one query to avoid conflicts
	consolidatedQuery := `
		INSERT INTO consolidated_attack_surface_assets (
			scope_target_id, asset_type, asset_identifier,
			domain, url, cloud_provider, cloud_service_type
		)
		SELECT 
			$1::uuid, 'cloud_asset', 
			asset_identifier,
			string_agg(DISTINCT domain_name, ', ') FILTER (WHERE domain_name IS NOT NULL),
			string_agg(DISTINCT url_value, ', ') FILTER (WHERE url_value IS NOT NULL),
			string_agg(DISTINCT cloud_provider, ', '),
			string_agg(DISTINCT service_type, ', ')
		FROM (
			-- Amass Enum cloud domains
			SELECT 
				cloud_domain as asset_identifier,
				cloud_domain as domain_name,
				NULL as url_value,
				type as cloud_provider,
				'domain' as service_type
			FROM amass_enum_company_cloud_domains
			WHERE scope_target_id = $1::uuid
			
			UNION ALL
			
			-- Cloud Enum results (AWS)
			SELECT 
				jsonb_array_elements_text(result::jsonb->'aws') as asset_identifier,
				jsonb_array_elements_text(result::jsonb->'aws') as domain_name,
				NULL as url_value,
				'aws' as cloud_provider,
				'domain' as service_type
			FROM cloud_enum_scans
			WHERE scope_target_id = $1::uuid AND status = 'success' AND result IS NOT NULL
				AND result::jsonb ? 'aws'
			
			UNION ALL
			
			-- Cloud Enum results (GCP)
			SELECT 
				jsonb_array_elements_text(result::jsonb->'gcp') as asset_identifier,
				jsonb_array_elements_text(result::jsonb->'gcp') as domain_name,
				NULL as url_value,
				'gcp' as cloud_provider,
				'domain' as service_type
			FROM cloud_enum_scans
			WHERE scope_target_id = $1::uuid AND status = 'success' AND result IS NOT NULL
				AND result::jsonb ? 'gcp'
			
			UNION ALL
			
			-- Cloud Enum results (Azure)
			SELECT 
				jsonb_array_elements_text(result::jsonb->'azure') as asset_identifier,
				jsonb_array_elements_text(result::jsonb->'azure') as domain_name,
				NULL as url_value,
				'azure' as cloud_provider,
				'domain' as service_type
			FROM cloud_enum_scans
			WHERE scope_target_id = $1::uuid AND status = 'success' AND result IS NOT NULL
				AND result::jsonb ? 'azure'
			
			UNION ALL
			
			-- Katana cloud assets
			SELECT 
				asset_url as asset_identifier,
				asset_domain as domain_name,
				asset_url as url_value,
				CASE 
					WHEN service ILIKE '%aws%' THEN 'aws'
					WHEN service ILIKE '%gcp%' OR service ILIKE '%google%' THEN 'gcp'
					WHEN service ILIKE '%azure%' THEN 'azure'
					ELSE 'unknown'
				END as cloud_provider,
				service as service_type
			FROM katana_company_cloud_assets
			WHERE scope_target_id = $1::uuid
		) all_cloud_data
		WHERE asset_identifier IS NOT NULL
		GROUP BY asset_identifier
		ON CONFLICT (scope_target_id, asset_type, asset_identifier) DO UPDATE SET
			domain = EXCLUDED.domain,
			url = EXCLUDED.url,
			cloud_provider = EXCLUDED.cloud_provider,
			cloud_service_type = EXCLUDED.cloud_service_type,
			last_updated = NOW()
	`

	result, err := dbPool.Exec(context.Background(), consolidatedQuery, scopeTargetID)
	if err != nil {
		return 0, err
	}

	return int(result.RowsAffected()), nil
}

func consolidateFQDNs(scopeTargetID string) (int, error) {
	log.Printf("[FQDN CONSOLIDATION] Starting FQDN consolidation for scope target: %s", scopeTargetID)

	// Consolidate FQDNs from various sources
	consolidatedQuery := `
		INSERT INTO consolidated_attack_surface_assets (
			scope_target_id, asset_type, asset_identifier, fqdn, root_domain, subdomain,
			registrar, creation_date, expiration_date, updated_date, name_servers, status,
			whois_info, ssl_certificate, ssl_expiry_date, ssl_issuer, ssl_subject, ssl_version,
			ssl_cipher_suite, ssl_protocols, resolved_ips, mail_servers, spf_record, dkim_record,
			dmarc_record, caa_records, txt_records, mx_records, ns_records, a_records,
			aaaa_records, cname_records, ptr_records, srv_records, soa_record,
			last_dns_scan, last_ssl_scan, last_whois_scan
		)
		WITH fqdn_sources AS (
			-- 1. Consolidated subdomains (basic FQDN data)
			SELECT
				subdomain as fqdn,
				subdomain as root_domain,
				NULL as subdomain,
				NULL as registrar,
				NULL::DATE as creation_date,
				NULL::DATE as expiration_date,
				NULL::DATE as updated_date,
				NULL::TEXT[] as name_servers,
				NULL::TEXT[] as status,
				NULL::JSONB as whois_info,
				NULL::JSONB as ssl_certificate,
				NULL::DATE as ssl_expiry_date,
				NULL as ssl_issuer,
				NULL as ssl_subject,
				NULL as ssl_version,
				NULL as ssl_cipher_suite,
				NULL::TEXT[] as ssl_protocols,
				NULL::TEXT[] as resolved_ips,
				NULL::TEXT[] as mail_servers,
				NULL as spf_record,
				NULL as dkim_record,
				NULL as dmarc_record,
				NULL::TEXT[] as caa_records,
				NULL::TEXT[] as txt_records,
				NULL::TEXT[] as mx_records,
				NULL::TEXT[] as ns_records,
				NULL::TEXT[] as a_records,
				NULL::TEXT[] as aaaa_records,
				NULL::TEXT[] as cname_records,
				NULL::TEXT[] as ptr_records,
				NULL::TEXT[] as srv_records,
				NULL::JSONB as soa_record,
				NULL::TIMESTAMP as last_dns_scan,
				NULL::TIMESTAMP as last_ssl_scan,
				NULL::TIMESTAMP as last_whois_scan
			FROM consolidated_subdomains
			WHERE scope_target_id = $1::uuid
			
			UNION ALL
			
			-- 2. DNSx Company DNS records (rich DNS data)
			SELECT
				record as fqdn,
				root_domain,
				CASE
					WHEN record != root_domain AND position('.' in record) > 0 THEN
						substring(record from 1 for position('.' in record) - 1)
					ELSE NULL
				END as subdomain,
				NULL as registrar,
				NULL::DATE as creation_date,
				NULL::DATE as expiration_date,
				NULL::DATE as updated_date,
				NULL::TEXT[] as name_servers,
				NULL::TEXT[] as status,
				NULL::JSONB as whois_info,
				NULL::JSONB as ssl_certificate,
				NULL::DATE as ssl_expiry_date,
				NULL as ssl_issuer,
				NULL as ssl_subject,
				NULL as ssl_version,
				NULL as ssl_cipher_suite,
				NULL::TEXT[] as ssl_protocols,
				NULL::TEXT[] as resolved_ips,
				NULL::TEXT[] as mail_servers,
				NULL as spf_record,
				NULL as dkim_record,
				NULL as dmarc_record,
				NULL::TEXT[] as caa_records,
				NULL::TEXT[] as txt_records,
				NULL::TEXT[] as mx_records,
				NULL::TEXT[] as ns_records,
				NULL::TEXT[] as a_records,
				NULL::TEXT[] as aaaa_records,
				NULL::TEXT[] as cname_records,
				NULL::TEXT[] as ptr_records,
				NULL::TEXT[] as srv_records,
				NULL::JSONB as soa_record,
				last_scanned_at as last_dns_scan,
				NULL as last_ssl_scan,
				NULL as last_whois_scan
			FROM dnsx_company_dns_records
			WHERE scope_target_id = $1::uuid
			
			UNION ALL
			
			-- 3. Amass Enum Company DNS records
			SELECT
				record as fqdn,
				root_domain,
				CASE
					WHEN record != root_domain AND position('.' in record) > 0 THEN
						substring(record from 1 for position('.' in record) - 1)
					ELSE NULL
				END as subdomain,
				NULL as registrar,
				NULL::DATE as creation_date,
				NULL::DATE as expiration_date,
				NULL::DATE as updated_date,
				NULL::TEXT[] as name_servers,
				NULL::TEXT[] as status,
				NULL::JSONB as whois_info,
				NULL::JSONB as ssl_certificate,
				NULL::DATE as ssl_expiry_date,
				NULL as ssl_issuer,
				NULL as ssl_subject,
				NULL as ssl_version,
				NULL as ssl_cipher_suite,
				NULL::TEXT[] as ssl_protocols,
				NULL::TEXT[] as resolved_ips,
				NULL::TEXT[] as mail_servers,
				NULL as spf_record,
				NULL as dkim_record,
				NULL as dmarc_record,
				NULL::TEXT[] as caa_records,
				NULL::TEXT[] as txt_records,
				NULL::TEXT[] as mx_records,
				NULL::TEXT[] as ns_records,
				NULL::TEXT[] as a_records,
				NULL::TEXT[] as aaaa_records,
				NULL::TEXT[] as cname_records,
				NULL::TEXT[] as ptr_records,
				NULL::TEXT[] as srv_records,
				NULL::JSONB as soa_record,
				last_scanned_at as last_dns_scan,
				NULL as last_ssl_scan,
				NULL as last_whois_scan
			FROM amass_enum_company_dns_records
			WHERE scope_target_id = $1::uuid
			
			UNION ALL
			
			-- 4. Target URLs (wildcard targets with rich data)
			SELECT
				CASE
					WHEN url LIKE 'http://%' THEN substring(url from 8)
					WHEN url LIKE 'https://%' THEN substring(url from 9)
					ELSE url
				END as fqdn,
				CASE
					WHEN url LIKE 'http://%' THEN substring(url from 8)
					WHEN url LIKE 'https://%' THEN substring(url from 9)
					ELSE url
				END as root_domain,
				NULL as subdomain,
				NULL as registrar,
				NULL::DATE as creation_date,
				NULL::DATE as expiration_date,
				NULL::DATE as updated_date,
				NULL::TEXT[] as name_servers,
				NULL::TEXT[] as status,
				NULL::JSONB as whois_info,
				NULL::JSONB as ssl_certificate,
				NULL::DATE as ssl_expiry_date,
				NULL as ssl_issuer,
				NULL as ssl_subject,
				NULL as ssl_version,
				NULL as ssl_cipher_suite,
				NULL::TEXT[] as ssl_protocols,
				NULL::TEXT[] as resolved_ips,
				NULL::TEXT[] as mail_servers,
				NULL as spf_record,
				NULL as dkim_record,
				NULL as dmarc_record,
				NULL::TEXT[] as caa_records,
				NULL::TEXT[] as txt_records,
				NULL::TEXT[] as mx_records,
				NULL::TEXT[] as ns_records,
				NULL::TEXT[] as a_records,
				NULL::TEXT[] as aaaa_records,
				NULL::TEXT[] as cname_records,
				NULL::TEXT[] as ptr_records,
				NULL::TEXT[] as srv_records,
				NULL::JSONB as soa_record,
				NULL::TIMESTAMP as last_dns_scan,
				NULL::TIMESTAMP as last_ssl_scan,
				NULL::TIMESTAMP as last_whois_scan
			FROM target_urls
			WHERE scope_target_id = $1::uuid AND no_longer_live = false
			
			UNION ALL
			
			-- 5. Domains from live web servers (IP/Port scan results)
			SELECT
				url as fqdn,
				CASE
					WHEN url LIKE 'http://%' THEN substring(url from 8)
					WHEN url LIKE 'https://%' THEN substring(url from 9)
					ELSE url
				END as root_domain,
				NULL as subdomain,
				NULL as registrar,
				NULL::DATE as creation_date,
				NULL::DATE as expiration_date,
				NULL::DATE as updated_date,
				NULL::TEXT[] as name_servers,
				NULL::TEXT[] as status,
				NULL::JSONB as whois_info,
				NULL::JSONB as ssl_certificate,
				NULL::DATE as ssl_expiry_date,
				NULL as ssl_issuer,
				NULL as ssl_subject,
				NULL as ssl_version,
				NULL as ssl_cipher_suite,
				NULL::TEXT[] as ssl_protocols,
				NULL::TEXT[] as resolved_ips,
				NULL::TEXT[] as mail_servers,
				NULL as spf_record,
				NULL as dkim_record,
				NULL as dmarc_record,
				NULL::TEXT[] as caa_records,
				NULL::TEXT[] as txt_records,
				NULL::TEXT[] as mx_records,
				NULL::TEXT[] as ns_records,
				NULL::TEXT[] as a_records,
				NULL::TEXT[] as aaaa_records,
				NULL::TEXT[] as cname_records,
				NULL::TEXT[] as ptr_records,
				NULL::TEXT[] as srv_records,
				NULL::JSONB as soa_record,
				NULL::TIMESTAMP as last_dns_scan,
				NULL::TIMESTAMP as last_ssl_scan,
				NULL::TIMESTAMP as last_whois_scan
			FROM live_web_servers lws
			JOIN ip_port_scans ips ON lws.scan_id = ips.scan_id
			WHERE ips.scope_target_id = $1::uuid AND ips.status = 'success'
				AND lws.url IS NOT NULL AND lws.url != ''
			
			UNION ALL
			
			-- 6. Consolidated company domains
			SELECT
				domain as fqdn,
				domain as root_domain,
				NULL as subdomain,
				NULL as registrar,
				NULL::DATE as creation_date,
				NULL::DATE as expiration_date,
				NULL::DATE as updated_date,
				NULL::TEXT[] as name_servers,
				NULL::TEXT[] as status,
				NULL::JSONB as whois_info,
				NULL::JSONB as ssl_certificate,
				NULL::DATE as ssl_expiry_date,
				NULL as ssl_issuer,
				NULL as ssl_subject,
				NULL as ssl_version,
				NULL as ssl_cipher_suite,
				NULL::TEXT[] as ssl_protocols,
				NULL::TEXT[] as resolved_ips,
				NULL::TEXT[] as mail_servers,
				NULL as spf_record,
				NULL as dkim_record,
				NULL as dmarc_record,
				NULL::TEXT[] as caa_records,
				NULL::TEXT[] as txt_records,
				NULL::TEXT[] as mx_records,
				NULL::TEXT[] as ns_records,
				NULL::TEXT[] as a_records,
				NULL::TEXT[] as aaaa_records,
				NULL::TEXT[] as cname_records,
				NULL::TEXT[] as ptr_records,
				NULL::TEXT[] as srv_records,
				NULL::JSONB as soa_record,
				NULL::TIMESTAMP as last_dns_scan,
				NULL::TIMESTAMP as last_ssl_scan,
				NULL::TIMESTAMP as last_whois_scan
			FROM consolidated_company_domains
			WHERE scope_target_id = $1::uuid
			
			UNION ALL
			
			-- 7. Domains from wildcard targets created from root domains
			SELECT
				scope_target as fqdn,
				scope_target as root_domain,
				NULL as subdomain,
				NULL as registrar,
				NULL::DATE as creation_date,
				NULL::DATE as expiration_date,
				NULL::DATE as updated_date,
				NULL::TEXT[] as name_servers,
				NULL::TEXT[] as status,
				NULL::JSONB as whois_info,
				NULL::JSONB as ssl_certificate,
				NULL::DATE as ssl_expiry_date,
				NULL as ssl_issuer,
				NULL as ssl_subject,
				NULL as ssl_version,
				NULL as ssl_cipher_suite,
				NULL::TEXT[] as ssl_protocols,
				NULL::TEXT[] as resolved_ips,
				NULL::TEXT[] as mail_servers,
				NULL as spf_record,
				NULL as dkim_record,
				NULL as dmarc_record,
				NULL::TEXT[] as caa_records,
				NULL::TEXT[] as txt_records,
				NULL::TEXT[] as mx_records,
				NULL::TEXT[] as ns_records,
				NULL::TEXT[] as a_records,
				NULL::TEXT[] as aaaa_records,
				NULL::TEXT[] as cname_records,
				NULL::TEXT[] as ptr_records,
				NULL::TEXT[] as srv_records,
				NULL::JSONB as soa_record,
				NULL::TIMESTAMP as last_dns_scan,
				NULL::TIMESTAMP as last_ssl_scan,
				NULL::TIMESTAMP as last_whois_scan
			FROM scope_targets
			WHERE type = 'Wildcard'
				AND scope_target IN (
					SELECT DISTINCT domain
					FROM consolidated_company_domains
					WHERE scope_target_id = $1::uuid
				)
		)
		SELECT DISTINCT ON (fqdn)
			$1::uuid, 'fqdn', fqdn_data.fqdn, fqdn_data.fqdn,
			fqdn_data.root_domain, fqdn_data.subdomain, fqdn_data.registrar,
			fqdn_data.creation_date, fqdn_data.expiration_date, fqdn_data.updated_date,
			fqdn_data.name_servers, fqdn_data.status, fqdn_data.whois_info,
			fqdn_data.ssl_certificate, fqdn_data.ssl_expiry_date, fqdn_data.ssl_issuer,
			fqdn_data.ssl_subject, fqdn_data.ssl_version, fqdn_data.ssl_cipher_suite,
			fqdn_data.ssl_protocols, fqdn_data.resolved_ips, fqdn_data.mail_servers,
			fqdn_data.spf_record, fqdn_data.dkim_record, fqdn_data.dmarc_record,
			fqdn_data.caa_records, fqdn_data.txt_records, fqdn_data.mx_records,
			fqdn_data.ns_records, fqdn_data.a_records, fqdn_data.aaaa_records,
			fqdn_data.cname_records, fqdn_data.ptr_records, fqdn_data.srv_records,
			fqdn_data.soa_record, fqdn_data.last_dns_scan, fqdn_data.last_ssl_scan,
			fqdn_data.last_whois_scan
				FROM fqdn_sources fqdn_data
		WHERE fqdn_data.fqdn IS NOT NULL AND fqdn_data.fqdn != ''
		ON CONFLICT (scope_target_id, asset_type, asset_identifier) DO UPDATE SET
			root_domain = EXCLUDED.root_domain,
			subdomain = EXCLUDED.subdomain,
			registrar = EXCLUDED.registrar,
			creation_date = EXCLUDED.creation_date,
			expiration_date = EXCLUDED.expiration_date,
			updated_date = EXCLUDED.updated_date,
			name_servers = EXCLUDED.name_servers,
			status = EXCLUDED.status,
			whois_info = EXCLUDED.whois_info,
			ssl_certificate = EXCLUDED.ssl_certificate,
			ssl_expiry_date = EXCLUDED.ssl_expiry_date,
			ssl_issuer = EXCLUDED.ssl_issuer,
			ssl_subject = EXCLUDED.ssl_subject,
			ssl_version = EXCLUDED.ssl_version,
			ssl_cipher_suite = EXCLUDED.ssl_cipher_suite,
			ssl_protocols = EXCLUDED.ssl_protocols,
			resolved_ips = EXCLUDED.resolved_ips,
			mail_servers = EXCLUDED.mail_servers,
			spf_record = EXCLUDED.spf_record,
			dkim_record = EXCLUDED.dkim_record,
			dmarc_record = EXCLUDED.dmarc_record,
			caa_records = EXCLUDED.caa_records,
			txt_records = EXCLUDED.txt_records,
			mx_records = EXCLUDED.mx_records,
			ns_records = EXCLUDED.ns_records,
			a_records = EXCLUDED.a_records,
			aaaa_records = EXCLUDED.aaaa_records,
			cname_records = EXCLUDED.cname_records,
			ptr_records = EXCLUDED.ptr_records,
			srv_records = EXCLUDED.srv_records,
			soa_record = EXCLUDED.soa_record,
			last_dns_scan = EXCLUDED.last_dns_scan,
			last_ssl_scan = EXCLUDED.last_ssl_scan,
			last_whois_scan = EXCLUDED.last_whois_scan,
			last_updated = NOW()
	`

	result, err := dbPool.Exec(context.Background(), consolidatedQuery, scopeTargetID)
	if err != nil {
		log.Printf("[FQDN CONSOLIDATION] Error inserting consolidated FQDNs: %v", err)
		return 0, err
	}

	insertedCount := int(result.RowsAffected())
	log.Printf("[FQDN CONSOLIDATION] ✅ Successfully inserted/updated %d FQDN records", insertedCount)
	log.Printf("[FQDN CONSOLIDATION] FQDN consolidation complete for scope target: %s", scopeTargetID)

	return insertedCount, nil
}

func createAssetRelationships(scopeTargetID string) (int, error) {
	totalRelationships := 0

	// Create relationships: IP addresses belong to network ranges
	ipToNetworkQuery := `
		INSERT INTO consolidated_attack_surface_relationships (
			parent_asset_id, child_asset_id, relationship_type
		)
		SELECT DISTINCT 
			nr.id, ip.id, 'contains'
		FROM consolidated_attack_surface_assets nr
		JOIN consolidated_attack_surface_assets ip ON nr.scope_target_id = ip.scope_target_id
		WHERE nr.scope_target_id = $1::uuid 
			AND nr.asset_type = 'network_range'
			AND ip.asset_type = 'ip_address'
			AND ip.ip_address::inet <<= nr.cidr_block::cidr
		ON CONFLICT (parent_asset_id, child_asset_id, relationship_type) DO NOTHING
	`

	ipToNetworkResult, err := dbPool.Exec(context.Background(), ipToNetworkQuery, scopeTargetID)
	if err != nil {
		return 0, err
	}
	totalRelationships += int(ipToNetworkResult.RowsAffected())

	// Create relationships: Network ranges belong to ASNs
	networkToASNQuery := `
		INSERT INTO consolidated_attack_surface_relationships (
			parent_asset_id, child_asset_id, relationship_type
		)
		SELECT DISTINCT 
			asn.id, nr.id, 'contains'
		FROM consolidated_attack_surface_assets asn
		JOIN consolidated_attack_surface_assets nr ON asn.scope_target_id = nr.scope_target_id
		JOIN consolidated_network_ranges cnr ON nr.cidr_block = cnr.cidr_block
		WHERE asn.scope_target_id = $1::uuid 
			AND asn.asset_type = 'asn'
			AND nr.asset_type = 'network_range'
			AND cnr.scope_target_id = $1::uuid
			AND cnr.asn = asn.asn_number
		ON CONFLICT (parent_asset_id, child_asset_id, relationship_type) DO NOTHING
	`

	networkToASNResult, err := dbPool.Exec(context.Background(), networkToASNQuery, scopeTargetID)
	if err != nil {
		return totalRelationships, err
	}
	totalRelationships += int(networkToASNResult.RowsAffected())

	// Create relationships: Live web servers to IP addresses
	liveWebServerToIPQuery := `
		INSERT INTO consolidated_attack_surface_relationships (
			parent_asset_id, child_asset_id, relationship_type
		)
		SELECT DISTINCT 
			lws.id, ip.id, 'hosted_on'
		FROM consolidated_attack_surface_assets lws
		JOIN consolidated_attack_surface_assets ip ON lws.scope_target_id = ip.scope_target_id
		WHERE lws.scope_target_id = $1::uuid 
			AND lws.asset_type = 'live_web_server'
			AND ip.asset_type = 'ip_address'
			AND lws.ip_address = ip.ip_address
		ON CONFLICT (parent_asset_id, child_asset_id, relationship_type) DO NOTHING
	`

	liveWebServerToIPResult, err := dbPool.Exec(context.Background(), liveWebServerToIPQuery, scopeTargetID)
	if err != nil {
		return totalRelationships, err
	}
	totalRelationships += int(liveWebServerToIPResult.RowsAffected())

	return totalRelationships, nil
}

func fetchConsolidatedAssets(scopeTargetID string) ([]AttackSurfaceAsset, error) {
	query := `
		SELECT 
			id, scope_target_id, asset_type, asset_identifier, asset_subtype,
			asn_number, asn_organization, asn_description, asn_country,
			cidr_block, ip_address::text, ip_type, url, domain, port, protocol,
			status_code, title, web_server, technologies, content_length,
			response_time_ms, screenshot_path, ssl_info, http_response_headers,
			findings_json, cloud_provider, cloud_service_type,
			cloud_region, fqdn, root_domain, subdomain, registrar, creation_date,
			expiration_date, updated_date, name_servers, status, whois_info,
			ssl_certificate, ssl_expiry_date, ssl_issuer, ssl_subject, ssl_version,
			ssl_cipher_suite, ssl_protocols, resolved_ips, mail_servers, spf_record,
			dkim_record, dmarc_record, caa_records, txt_records, mx_records,
			ns_records, a_records, aaaa_records, cname_records, ptr_records,
			srv_records, soa_record, last_dns_scan, last_ssl_scan, last_whois_scan,
			last_updated, created_at
		FROM consolidated_attack_surface_assets
		WHERE scope_target_id = $1::uuid
		ORDER BY asset_type, asset_identifier
	`

	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []AttackSurfaceAsset

	for rows.Next() {
		var asset AttackSurfaceAsset
		var technologies []string
		var sslInfo, httpHeaders, findings []byte
		var whoisInfo, sslCertificate, soaRecord []byte

		err := rows.Scan(
			&asset.ID, &asset.ScopeTargetID, &asset.AssetType, &asset.AssetIdentifier, &asset.AssetSubtype,
			&asset.ASNNumber, &asset.ASNOrganization, &asset.ASNDescription, &asset.ASNCountry,
			&asset.CIDRBlock, &asset.IPAddress, &asset.IPType, &asset.URL, &asset.Domain, &asset.Port, &asset.Protocol,
			&asset.StatusCode, &asset.Title, &asset.WebServer, &technologies, &asset.ContentLength,
			&asset.ResponseTime, &asset.ScreenshotPath, &sslInfo, &httpHeaders,
			&findings, &asset.CloudProvider, &asset.CloudServiceType,
			&asset.CloudRegion, &asset.FQDN, &asset.RootDomain, &asset.Subdomain, &asset.Registrar, &asset.CreationDate,
			&asset.ExpirationDate, &asset.UpdatedDate, &asset.NameServers, &asset.Status, &whoisInfo,
			&sslCertificate, &asset.SSLExpiryDate, &asset.SSLIssuer, &asset.SSLSubject, &asset.SSLVersion,
			&asset.SSLCipherSuite, &asset.SSLProtocols, &asset.ResolvedIPs, &asset.MailServers, &asset.SPFRecord,
			&asset.DKIMRecord, &asset.DMARCRecord, &asset.CAARecords, &asset.TXTRecords, &asset.MXRecords,
			&asset.NSRecords, &asset.ARecords, &asset.AAAARecords, &asset.CNAMERecords, &asset.PTRRecords,
			&asset.SRVRecords, &soaRecord, &asset.LastDNSScan, &asset.LastSSLScan, &asset.LastWhoisScan,
			&asset.LastUpdated, &asset.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		asset.Technologies = technologies

		if len(sslInfo) > 0 {
			json.Unmarshal(sslInfo, &asset.SSLInfo)
		}
		if len(httpHeaders) > 0 {
			json.Unmarshal(httpHeaders, &asset.HTTPResponseHeaders)
		}
		if len(findings) > 0 {
			json.Unmarshal(findings, &asset.FindingsJSON)
		}
		if len(whoisInfo) > 0 {
			json.Unmarshal(whoisInfo, &asset.WhoisInfo)
		}
		if len(sslCertificate) > 0 {
			json.Unmarshal(sslCertificate, &asset.SSLCertificate)
		}
		if len(soaRecord) > 0 {
			json.Unmarshal(soaRecord, &asset.SOARecord)
		}

		assets = append(assets, asset)
	}

	return assets, nil
}

func fetchAssetRelationships(assetID string) ([]AssetRelationship, error) {
	query := `
		SELECT 
			id, parent_asset_id, child_asset_id, relationship_type, 
			relationship_data, created_at
		FROM consolidated_attack_surface_relationships
		WHERE parent_asset_id = $1::uuid OR child_asset_id = $1::uuid
		ORDER BY relationship_type
	`

	rows, err := dbPool.Query(context.Background(), query, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []AssetRelationship

	for rows.Next() {
		var rel AssetRelationship
		var relationshipData []byte

		err := rows.Scan(
			&rel.ID, &rel.ParentAssetID, &rel.ChildAssetID, &rel.RelationshipType,
			&relationshipData, &rel.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(relationshipData) > 0 {
			json.Unmarshal(relationshipData, &rel.RelationshipData)
		}

		relationships = append(relationships, rel)
	}

	return relationships, nil
}

func GetAttackSurfaceAssets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["scope_target_id"]

	if scopeTargetID == "" {
		http.Error(w, "Scope target ID is required", http.StatusBadRequest)
		return
	}

	assets, err := fetchConsolidatedAssets(scopeTargetID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching attack surface assets: %v", err), http.StatusInternalServerError)
		return
	}

	for i := range assets {
		relationships, err := fetchAssetRelationships(assets[i].ID)
		if err != nil {
			log.Printf("Error fetching relationships for asset %s: %v", assets[i].ID, err)
			continue
		}
		assets[i].Relationships = relationships
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"assets": assets,
		"total":  len(assets),
	})
}
