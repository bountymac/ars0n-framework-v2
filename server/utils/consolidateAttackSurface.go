package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	ROIScore            int                    `json:"roi_score"`

	// Cloud Asset fields
	CloudProvider    *string `json:"cloud_provider,omitempty"`
	CloudServiceType *string `json:"cloud_service_type,omitempty"`
	CloudRegion      *string `json:"cloud_region,omitempty"`

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
	query := `
		INSERT INTO consolidated_attack_surface_assets (
			scope_target_id, asset_type, asset_identifier, 
			asn_number, asn_organization, asn_description, asn_country
		)
		SELECT DISTINCT 
			$1::uuid, 'asn', asn_number,
			asn_number, organization, description, country
		FROM (
			SELECT asn_number, organization, description, country
			FROM intel_asn_data ia
			JOIN amass_intel_scans ais ON ia.scan_id = ais.scan_id
			WHERE ais.scope_target_id = $1::uuid AND ais.status = 'success'
			
			UNION
			
			SELECT asn, organization, description, country
			FROM consolidated_network_ranges
			WHERE scope_target_id = $1::uuid AND asn IS NOT NULL
		) asn_data
		ON CONFLICT (scope_target_id, asset_type, asset_identifier) DO UPDATE SET
			asn_organization = EXCLUDED.asn_organization,
			asn_description = EXCLUDED.asn_description,
			asn_country = EXCLUDED.asn_country,
			last_updated = NOW()
	`

	result, err := dbPool.Exec(context.Background(), query, scopeTargetID)
	if err != nil {
		return 0, err
	}

	return int(result.RowsAffected()), nil
}

func consolidateNetworkRanges(scopeTargetID string) (int, error) {
	query := `
		INSERT INTO consolidated_attack_surface_assets (
			scope_target_id, asset_type, asset_identifier, cidr_block
		)
		SELECT DISTINCT 
			$1::uuid, 'network_range', cidr_block, cidr_block
		FROM (
			SELECT cidr_block
			FROM intel_network_ranges inr
			JOIN amass_intel_scans ais ON inr.scan_id = ais.scan_id
			WHERE ais.scope_target_id = $1::uuid AND ais.status = 'success'
			
			UNION
			
			SELECT cidr_block
			FROM consolidated_network_ranges
			WHERE scope_target_id = $1::uuid
		) range_data
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
		return 0, err
	}

	return int(result.RowsAffected()), nil
}

func consolidateLiveWebServers(scopeTargetID string) (int, error) {
	// First, consolidate IP/Port live web servers
	ipPortQuery := `
		INSERT INTO consolidated_attack_surface_assets (
			scope_target_id, asset_type, asset_subtype, asset_identifier,
			ip_address, port, protocol, url, status_code, title, web_server,
			technologies, content_length, response_time_ms, screenshot_path, roi_score
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
			content_length, response_time_ms, screenshot_path, 50
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
			domain, url, roi_score
		)
		SELECT DISTINCT 
			$1::uuid, 'live_web_server', 'domain', 
			result,
			result, 'https://' || result, 50
		FROM investigate_scans
		WHERE scope_target_id = $1::uuid AND status = 'success' AND result IS NOT NULL
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
			content_length, screenshot_path, roi_score
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
			content_length, screenshot, COALESCE(roi_score, 50)
		FROM target_urls
		WHERE scope_target_id = $1::uuid AND no_longer_live = false
		ON CONFLICT (scope_target_id, asset_type, asset_identifier) DO UPDATE SET
			status_code = EXCLUDED.status_code,
			title = EXCLUDED.title,
			web_server = EXCLUDED.web_server,
			technologies = EXCLUDED.technologies,
			content_length = EXCLUDED.content_length,
			screenshot_path = EXCLUDED.screenshot_path,
			roi_score = EXCLUDED.roi_score,
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
			findings_json, roi_score, cloud_provider, cloud_service_type,
			cloud_region, last_updated, created_at
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

		err := rows.Scan(
			&asset.ID, &asset.ScopeTargetID, &asset.AssetType, &asset.AssetIdentifier, &asset.AssetSubtype,
			&asset.ASNNumber, &asset.ASNOrganization, &asset.ASNDescription, &asset.ASNCountry,
			&asset.CIDRBlock, &asset.IPAddress, &asset.IPType, &asset.URL, &asset.Domain, &asset.Port, &asset.Protocol,
			&asset.StatusCode, &asset.Title, &asset.WebServer, &technologies, &asset.ContentLength,
			&asset.ResponseTime, &asset.ScreenshotPath, &sslInfo, &httpHeaders,
			&findings, &asset.ROIScore, &asset.CloudProvider, &asset.CloudServiceType,
			&asset.CloudRegion, &asset.LastUpdated, &asset.CreatedAt,
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

		assets = append(assets, asset)
	}

	return assets, nil
}
