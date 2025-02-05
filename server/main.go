package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ASN struct {
	ID        string    `json:"id"`
	ScanID    string    `json:"scan_id"`
	Number    string    `json:"number"`
	RawData   string    `json:"raw_data"`
	CreatedAt time.Time `json:"created_at"`
}

type Subnet struct {
	ID        string    `json:"id"`
	ScanID    string    `json:"scan_id"`
	CIDR      string    `json:"cidr"`
	RawData   string    `json:"raw_data"`
	CreatedAt time.Time `json:"created_at"`
}

type DNSRecord struct {
	ID        string    `json:"id"`
	ScanID    string    `json:"scan_id"`
	Record    string    `json:"record"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

type IPAddress struct {
	ID        string    `json:"id"`
	ScanID    string    `json:"scan_id"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
}

type Subdomain struct {
	ID        string    `json:"id"`
	ScanID    string    `json:"scan_id"`
	Subdomain string    `json:"subdomain"`
	CreatedAt time.Time `json:"created_at"`
}

type CloudDomain struct {
	ID        string    `json:"id"`
	Domain    string    `json:"domain"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

type RequestPayload struct {
	Type        string `json:"type"`
	Mode        string `json:"mode"`
	ScopeTarget string `json:"scope_target"`
	Active      bool   `json:"active"`
}

type ResponsePayload struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Mode        string `json:"mode"`
	ScopeTarget string `json:"scope_target"`
	Active      bool   `json:"active"`
}

type AmassScanStatus struct {
	ID        string         `json:"id"`
	ScanID    string         `json:"scan_id"`
	Domain    string         `json:"domain"`
	Status    string         `json:"status"`
	Result    sql.NullString `json:"result,omitempty"`
	Error     sql.NullString `json:"error,omitempty"`
	StdOut    sql.NullString `json:"stdout,omitempty"`
	StdErr    sql.NullString `json:"stderr,omitempty"`
	Command   sql.NullString `json:"command,omitempty"`
	ExecTime  sql.NullString `json:"execution_time,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type ServiceProvider struct {
	ID        string    `json:"id"`
	ScanID    string    `json:"scan_id"`
	Provider  string    `json:"provider"`
	RawData   string    `json:"raw_data"`
	CreatedAt time.Time `json:"created_at"`
}

type HttpxScanStatus struct {
	ID        string         `json:"id"`
	ScanID    string         `json:"scan_id"`
	Domain    string         `json:"domain"`
	Status    string         `json:"status"`
	Result    sql.NullString `json:"result,omitempty"`
	Error     sql.NullString `json:"error,omitempty"`
	StdOut    sql.NullString `json:"stdout,omitempty"`
	StdErr    sql.NullString `json:"stderr,omitempty"`
	Command   sql.NullString `json:"command,omitempty"`
	ExecTime  sql.NullString `json:"execution_time,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type ScanSummary struct {
	ID        string    `json:"id"`
	ScanID    string    `json:"scan_id"`
	Domain    string    `json:"domain"`
	Status    string    `json:"status"`
	Result    string    `json:"result,omitempty"`
	Error     string    `json:"error,omitempty"`
	StdOut    string    `json:"stdout,omitempty"`
	StdErr    string    `json:"stderr,omitempty"`
	Command   string    `json:"command,omitempty"`
	ExecTime  string    `json:"execution_time,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ScanType  string    `json:"scan_type"`
}

type GauScanStatus struct {
	ID        string         `json:"id"`
	ScanID    string         `json:"scan_id"`
	Domain    string         `json:"domain"`
	Status    string         `json:"status"`
	Result    sql.NullString `json:"result,omitempty"`
	Error     sql.NullString `json:"error,omitempty"`
	StdOut    sql.NullString `json:"stdout,omitempty"`
	StdErr    sql.NullString `json:"stderr,omitempty"`
	Command   sql.NullString `json:"command,omitempty"`
	ExecTime  sql.NullString `json:"execution_time,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type Sublist3rScanStatus struct {
	ID            string         `json:"id"`
	ScanID        string         `json:"scan_id"`
	Domain        string         `json:"domain"`
	Status        string         `json:"status"`
	Result        sql.NullString `json:"result,omitempty"`
	Error         sql.NullString `json:"error,omitempty"`
	StdOut        sql.NullString `json:"stdout,omitempty"`
	StdErr        sql.NullString `json:"stderr,omitempty"`
	Command       sql.NullString `json:"command,omitempty"`
	ExecTime      sql.NullString `json:"execution_time,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	ScopeTargetID string         `json:"scope_target_id"`
}

type AssetfinderScanStatus struct {
	ID            string         `json:"id"`
	ScanID        string         `json:"scan_id"`
	Domain        string         `json:"domain"`
	Status        string         `json:"status"`
	Result        sql.NullString `json:"result,omitempty"`
	Error         sql.NullString `json:"error,omitempty"`
	StdOut        sql.NullString `json:"stdout,omitempty"`
	StdErr        sql.NullString `json:"stderr,omitempty"`
	Command       sql.NullString `json:"command,omitempty"`
	ExecTime      sql.NullString `json:"execution_time,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	ScopeTargetID string         `json:"scope_target_id"`
}

type CTLScanStatus struct {
	ID            string         `json:"id"`
	ScanID        string         `json:"scan_id"`
	Domain        string         `json:"domain"`
	Status        string         `json:"status"`
	Result        sql.NullString `json:"result,omitempty"`
	Error         sql.NullString `json:"error,omitempty"`
	StdOut        sql.NullString `json:"stdout,omitempty"`
	StdErr        sql.NullString `json:"stderr,omitempty"`
	Command       sql.NullString `json:"command,omitempty"`
	ExecTime      sql.NullString `json:"execution_time,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	ScopeTargetID string         `json:"scope_target_id"`
}

type SubfinderScanStatus struct {
	ID            string         `json:"id"`
	ScanID        string         `json:"scan_id"`
	Domain        string         `json:"domain"`
	Status        string         `json:"status"`
	Result        sql.NullString `json:"result,omitempty"`
	Error         sql.NullString `json:"error,omitempty"`
	StdOut        sql.NullString `json:"stdout,omitempty"`
	StdErr        sql.NullString `json:"stderr,omitempty"`
	Command       sql.NullString `json:"command,omitempty"`
	ExecTime      sql.NullString `json:"execution_time,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	ScopeTargetID string         `json:"scope_target_id"`
}

type ShuffleDNSScanStatus struct {
	ID            string         `json:"id"`
	ScanID        string         `json:"scan_id"`
	Domain        string         `json:"domain"`
	Status        string         `json:"status"`
	Result        sql.NullString `json:"result,omitempty"`
	Error         sql.NullString `json:"error,omitempty"`
	StdOut        sql.NullString `json:"stdout,omitempty"`
	StdErr        sql.NullString `json:"stderr,omitempty"`
	Command       sql.NullString `json:"command,omitempty"`
	ExecTime      sql.NullString `json:"execution_time,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	ScopeTargetID string         `json:"scope_target_id"`
}

type CeWLScanStatus struct {
	ID            string         `json:"id"`
	ScanID        string         `json:"scan_id"`
	URL           string         `json:"url"`
	Status        string         `json:"status"`
	Result        sql.NullString `json:"result,omitempty"`
	Error         sql.NullString `json:"error,omitempty"`
	StdOut        sql.NullString `json:"stdout,omitempty"`
	StdErr        sql.NullString `json:"stderr,omitempty"`
	Command       sql.NullString `json:"command,omitempty"`
	ExecTime      sql.NullString `json:"execution_time,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	ScopeTargetID string         `json:"scope_target_id"`
}

type GoSpiderScanStatus struct {
	ID            string         `json:"id"`
	ScanID        string         `json:"scan_id"`
	Domain        string         `json:"domain"`
	Status        string         `json:"status"`
	Result        sql.NullString `json:"result,omitempty"`
	Error         sql.NullString `json:"error,omitempty"`
	StdOut        sql.NullString `json:"stdout,omitempty"`
	StdErr        sql.NullString `json:"stderr,omitempty"`
	Command       sql.NullString `json:"command,omitempty"`
	ExecTime      sql.NullString `json:"execution_time,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	ScopeTargetID string         `json:"scope_target_id"`
}

type SubdomainizerScanStatus struct {
	ID            string         `json:"id"`
	ScanID        string         `json:"scan_id"`
	Domain        string         `json:"domain"`
	Status        string         `json:"status"`
	Result        sql.NullString `json:"result,omitempty"`
	Error         sql.NullString `json:"error,omitempty"`
	StdOut        sql.NullString `json:"stdout,omitempty"`
	StdErr        sql.NullString `json:"stderr,omitempty"`
	Command       sql.NullString `json:"command,omitempty"`
	ExecTime      sql.NullString `json:"execution_time,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	ScopeTargetID string         `json:"scope_target_id"`
}

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
	defer dbPool.Close()

	createTables()

	r := mux.NewRouter()
	r.HandleFunc("/scopetarget/add", createScopeTarget).Methods("POST")
	r.HandleFunc("/scopetarget/read", readScopeTarget).Methods("GET")
	r.HandleFunc("/scopetarget/delete/{id}", deleteScopeTarget).Methods("DELETE")
	r.HandleFunc("/scopetarget/{id}/activate", activateScopeTarget).Methods("POST")
	r.HandleFunc("/scopetarget/{id}/scans/amass", getAmassScansForScopeTarget).Methods("GET")
	r.HandleFunc("/amass/run", runAmassScan).Methods("POST")
	r.HandleFunc("/amass/{scanID}", getAmassScanStatus).Methods("GET")
	r.HandleFunc("/amass/{scan_id}/dns", getDNSRecords).Methods("GET")
	r.HandleFunc("/amass/{scan_id}/ip", getIPs).Methods("GET")
	r.HandleFunc("/amass/{scan_id}/subdomain", getSubdomains).Methods("GET")
	r.HandleFunc("/amass/{scan_id}/cloud", getCloudDomains).Methods("GET")
	r.HandleFunc("/amass/{scan_id}/sp", getServiceProviders).Methods("GET")
	r.HandleFunc("/amass/{scan_id}/asn", getASNs).Methods("GET")
	r.HandleFunc("/amass/{scan_id}/subnet", getSubnets).Methods("GET")
	r.HandleFunc("/httpx/run", runHttpxScan).Methods("POST")
	r.HandleFunc("/httpx/{scanID}", getHttpxScanStatus).Methods("GET")
	r.HandleFunc("/scopetarget/{id}/scans/httpx", getHttpxScansForScopeTarget).Methods("GET")
	r.HandleFunc("/scopetarget/{id}/scans", getAllScansForScopeTarget).Methods("GET")
	r.HandleFunc("/gau/run", runGauScan).Methods("POST")
	r.HandleFunc("/gau/{scanID}", getGauScanStatus).Methods("GET")
	r.HandleFunc("/scopetarget/{id}/scans/gau", getGauScansForScopeTarget).Methods("GET")
	r.HandleFunc("/sublist3r/run", runSublist3rScan).Methods("POST")
	r.HandleFunc("/sublist3r/{scan_id}", getSublist3rScanStatus).Methods("GET")
	r.HandleFunc("/scopetarget/{id}/scans/sublist3r", getSublist3rScansForScopeTarget).Methods("GET")
	r.HandleFunc("/assetfinder/run", runAssetfinderScan).Methods("POST")
	r.HandleFunc("/assetfinder/{scan_id}", getAssetfinderScanStatus).Methods("GET")
	r.HandleFunc("/scopetarget/{id}/scans/assetfinder", getAssetfinderScansForScopeTarget).Methods("GET")
	r.HandleFunc("/ctl/run", runCTLScan).Methods("POST")
	r.HandleFunc("/ctl/{scan_id}", getCTLScanStatus).Methods("GET")
	r.HandleFunc("/scopetarget/{id}/scans/ctl", getCTLScansForScopeTarget).Methods("GET")
	r.HandleFunc("/subfinder/run", runSubfinderScan).Methods("POST")
	r.HandleFunc("/subfinder/{scan_id}", getSubfinderScanStatus).Methods("GET")
	r.HandleFunc("/scopetarget/{id}/scans/subfinder", getSubfinderScansForScopeTarget).Methods("GET")
	r.HandleFunc("/consolidate-subdomains/{id}", handleConsolidateSubdomains).Methods("GET")
	r.HandleFunc("/consolidated-subdomains/{id}", getConsolidatedSubdomains).Methods("GET")
	r.HandleFunc("/shuffledns/run", runShuffleDNSScan).Methods("POST")
	r.HandleFunc("/shuffledns/{scan_id}", getShuffleDNSScanStatus).Methods("GET")
	r.HandleFunc("/scopetarget/{id}/scans/shuffledns", getShuffleDNSScansForScopeTarget).Methods("GET")
	r.HandleFunc("/cewl/run", runCeWLScan).Methods("POST")
	r.HandleFunc("/cewl/{scan_id}", getCeWLScanStatus).Methods("GET")
	r.HandleFunc("/scopetarget/{id}/scans/cewl", getCeWLScansForScopeTarget).Methods("GET")
	r.HandleFunc("/cewl-urls/run", runCeWLScansForUrls).Methods("POST")
	r.HandleFunc("/cewl-wordlist/run", runShuffleDNSWithWordlist).Methods("POST")
	r.HandleFunc("/cewl-wordlist/{scan_id}", getShuffleDNSScanStatus).Methods("GET")
	r.HandleFunc("/api/scope-targets/{id}/shufflednscustom-scans", getShuffleDNSCustomScansForScopeTarget).Methods("GET")
	r.HandleFunc("/gospider/run", runGoSpiderScan).Methods("POST")
	r.HandleFunc("/gospider/{scan_id}", getGoSpiderScanStatus).Methods("GET")
	r.HandleFunc("/scopetarget/{id}/scans/gospider", getGoSpiderScansForScopeTarget).Methods("GET")
	r.HandleFunc("/subdomainizer/run", runSubdomainizerScan).Methods("POST")
	r.HandleFunc("/subdomainizer/{scan_id}", getSubdomainizerScanStatus).Methods("GET")
	r.HandleFunc("/scopetarget/{id}/scans/subdomainizer", getSubdomainizerScansForScopeTarget).Methods("GET")

	handlerWithCORS := corsMiddleware(r)

	log.Println("API server started on :8080")
	http.ListenAndServe(":8080", handlerWithCORS)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func createTables() {
	queries := []string{
		`CREATE EXTENSION IF NOT EXISTS pgcrypto;`,
		`DROP TABLE IF EXISTS requests CASCADE;`,
		`CREATE TABLE IF NOT EXISTS scope_targets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			type VARCHAR(50) NOT NULL,
			mode VARCHAR(50) NOT NULL,
			scope_target TEXT NOT NULL,
			active BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS amass_scans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL UNIQUE, 
			domain TEXT NOT NULL,
			status VARCHAR(50) NOT NULL,
			result TEXT,
			error TEXT,
			stdout TEXT,
			stderr TEXT,
			command TEXT,
			execution_time TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			scope_target_id UUID REFERENCES scope_targets(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS dns_records (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL,
			record TEXT NOT NULL,
			record_type TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS ips (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL,
			ip_address TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS subdomains (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL,
			subdomain TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS cloud_domains (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL,
			domain TEXT NOT NULL,
			type TEXT NOT NULL CHECK (type IN ('aws', 'gcp', 'azu')),
			created_at TIMESTAMP DEFAULT NOW(),
			FOREIGN KEY (scan_id) REFERENCES amass_scans(scan_id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS asns (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL,
			number TEXT NOT NULL,
			raw_data TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS subnets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL,
			cidr TEXT NOT NULL,
			raw_data TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS service_providers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL,
			provider TEXT NOT NULL,
			raw_data TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			FOREIGN KEY (scan_id) REFERENCES amass_scans(scan_id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS httpx_scans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL UNIQUE, 
			domain TEXT NOT NULL,
			status VARCHAR(50) NOT NULL,
			result TEXT,
			error TEXT,
			stdout TEXT,
			stderr TEXT,
			command TEXT,
			execution_time TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			scope_target_id UUID REFERENCES scope_targets(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS gau_scans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL UNIQUE, 
			domain TEXT NOT NULL,
			status VARCHAR(50) NOT NULL,
			result TEXT,
			error TEXT,
			stdout TEXT,
			stderr TEXT,
			command TEXT,
			execution_time TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			scope_target_id UUID REFERENCES scope_targets(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS sublist3r_scans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL UNIQUE,
			domain TEXT NOT NULL,
			status VARCHAR(50) NOT NULL,
			result TEXT,
			error TEXT,
			stdout TEXT,
			stderr TEXT,
			command TEXT,
			execution_time TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			scope_target_id UUID REFERENCES scope_targets(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS assetfinder_scans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL UNIQUE,
			domain TEXT NOT NULL,
			status VARCHAR(50) NOT NULL,
			result TEXT,
			error TEXT,
			stdout TEXT,
			stderr TEXT,
			command TEXT,
			execution_time TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			scope_target_id UUID REFERENCES scope_targets(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS ctl_scans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL UNIQUE,
			domain TEXT NOT NULL,
			status VARCHAR(50) NOT NULL,
			result TEXT,
			error TEXT,
			stdout TEXT,
			stderr TEXT,
			command TEXT,
			execution_time TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			scope_target_id UUID REFERENCES scope_targets(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS subfinder_scans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL UNIQUE,
			domain TEXT NOT NULL,
			status VARCHAR(50) NOT NULL,
			result TEXT,
			error TEXT,
			stdout TEXT,
			stderr TEXT,
			command TEXT,
			execution_time TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			scope_target_id UUID REFERENCES scope_targets(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS consolidated_subdomains (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scope_target_id UUID REFERENCES scope_targets(id) ON DELETE CASCADE,
			subdomain TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(scope_target_id, subdomain)
		);`,
		`CREATE TABLE IF NOT EXISTS shuffledns_scans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL UNIQUE,
			domain TEXT NOT NULL,
			status VARCHAR(50) NOT NULL,
			result TEXT,
			error TEXT,
			stdout TEXT,
			stderr TEXT,
			command TEXT,
			execution_time TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			scope_target_id UUID REFERENCES scope_targets(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS cewl_scans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL UNIQUE,
			url TEXT NOT NULL,
			status VARCHAR(50) NOT NULL,
			result TEXT,
			error TEXT,
			stdout TEXT,
			stderr TEXT,
			command TEXT,
			execution_time TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			scope_target_id UUID REFERENCES scope_targets(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS shufflednscustom_scans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL UNIQUE,
			domain TEXT NOT NULL,
			status VARCHAR(50) NOT NULL,
			result TEXT,
			error TEXT,
			stdout TEXT,
			stderr TEXT,
			command TEXT,
			execution_time TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			scope_target_id UUID REFERENCES scope_targets(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS gospider_scans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL UNIQUE,
			domain TEXT NOT NULL,
			status VARCHAR(50) NOT NULL,
			result TEXT,
			error TEXT,
			stdout TEXT,
			stderr TEXT,
			command TEXT,
			execution_time TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			scope_target_id UUID REFERENCES scope_targets(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS subdomainizer_scans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL UNIQUE,
			domain TEXT NOT NULL,
			status VARCHAR(50) NOT NULL,
			result TEXT,
			error TEXT,
			stdout TEXT,
			stderr TEXT,
			command TEXT,
			execution_time TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			scope_target_id UUID REFERENCES scope_targets(id) ON DELETE CASCADE
		);`,
	}

	for _, query := range queries {
		_, err := dbPool.Exec(context.Background(), query)
		if err != nil {
			log.Fatalf("[ERROR] Failed to execute query: %s, error: %v", query, err)
		}
	}

	deletePendingScansQuery := `DELETE FROM amass_scans WHERE status = 'pending';`
	_, err := dbPool.Exec(context.Background(), deletePendingScansQuery)
	if err != nil {
		log.Fatalf("[ERROR] Failed to delete pending Amass scans: %v", err)
	}
	log.Println("[INFO] Deleted any Amass scans with status 'pending'")
}

func parseAndStoreResults(scanID, domain, result string) {
	log.Printf("[INFO] Starting to parse results for scan %s on domain %s", scanID, domain)

	patterns := map[string]*regexp.Regexp{
		"service_provider": regexp.MustCompile(`(\d+)\s+\(ASN\)\s+-->\s+managed_by\s+-->\s+(.+?)\s+\(RIROrganization\)`),
		"asn_announces":    regexp.MustCompile(`(\d+)\s+\(ASN\)\s+-->\s+announces\s+-->\s+([^\s]+)\s+\(Netblock\)`),
		"subnet_contains":  regexp.MustCompile(`([^\s]+)\s+\(Netblock\)\s+-->\s+contains\s+-->\s+([^\s]+)\s+\(IPAddress\)`),
		"subdomain":        regexp.MustCompile(`([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`),
		"ipv4":             regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+)\s+\(IPAddress\)`),
		"dns_a":            regexp.MustCompile(`a_record`),
		"dns_aaaa":         regexp.MustCompile(`aaaa_record`),
		"dns_cname":        regexp.MustCompile(`cname_record`),
		"dns_mx":           regexp.MustCompile(`mx_record`),
		"dns_txt":          regexp.MustCompile(`txt_record`),
		"dns_ns":           regexp.MustCompile(`ns_record`),
		"dns_srv":          regexp.MustCompile(`srv_record`),
		"dns_ptr":          regexp.MustCompile(`ptr_record`),
		"dns_spf":          regexp.MustCompile(`spf_record`),
		"dns_soa":          regexp.MustCompile(`soa_record`),
	}

	lines := strings.Split(result, "\n")
	log.Printf("[INFO] Processing %d lines of output", len(lines))

	for lineNum, line := range lines {
		log.Printf("[DEBUG] Processing line %d: %s", lineNum+1, line)

		// Parse ASN and Service Provider information
		if matches := patterns["service_provider"].FindStringSubmatch(line); len(matches) > 2 {
			asn := matches[1]
			provider := matches[2]
			log.Printf("[DEBUG] Found ASN %s with provider %s", asn, provider)
			insertASN(scanID, asn, line)
			insertServiceProvider(scanID, provider, line)
		}

		// Parse ASN announcements
		if matches := patterns["asn_announces"].FindStringSubmatch(line); len(matches) > 2 {
			asn := matches[1]
			subnet := matches[2]
			log.Printf("[DEBUG] Found ASN %s announcing subnet %s", asn, subnet)
			insertASN(scanID, asn, line)
			insertSubnet(scanID, subnet, line)
		}

		// Parse subnet contains IP
		if matches := patterns["subnet_contains"].FindStringSubmatch(line); len(matches) > 2 {
			subnet := matches[1]
			ip := matches[2]
			log.Printf("[DEBUG] Found subnet %s containing IP %s", subnet, ip)
			insertSubnet(scanID, subnet, line)
			insertIP(scanID, ip)
		}

		// Parse subdomains
		if matches := patterns["subdomain"].FindAllString(line, -1); len(matches) > 0 {
			log.Printf("[DEBUG] Found potential subdomain matches: %v", matches)
			for _, subdomain := range matches {
				if strings.Contains(subdomain, domain) {
					log.Printf("[DEBUG] Valid subdomain found: %s", subdomain)
					insertSubdomain(scanID, subdomain)
				} else if isCloudDomain(subdomain) {
					log.Printf("[DEBUG] Cloud domain found: %s", subdomain)
					insertCloudDomain(scanID, subdomain)
				}
			}
		}

		// Parse IP addresses
		if matches := patterns["ipv4"].FindStringSubmatch(line); len(matches) > 1 {
			ip := matches[1]
			log.Printf("[DEBUG] Found IPv4 address: %s", ip)
			insertIP(scanID, ip)
		}

		// Parse DNS records
		for recordType, pattern := range map[string]*regexp.Regexp{
			"A":     patterns["dns_a"],
			"AAAA":  patterns["dns_aaaa"],
			"CNAME": patterns["dns_cname"],
			"MX":    patterns["dns_mx"],
			"TXT":   patterns["dns_txt"],
			"NS":    patterns["dns_ns"],
			"SRV":   patterns["dns_srv"],
			"PTR":   patterns["dns_ptr"],
			"SPF":   patterns["dns_spf"],
			"SOA":   patterns["dns_soa"],
		} {
			if pattern.MatchString(line) {
				log.Printf("[DEBUG] Found DNS record type %s: %s", recordType, line)
				insertDNSRecord(scanID, line, recordType)
			}
		}
	}
	log.Printf("[INFO] Completed parsing results for scan %s", scanID)
}

func insertServiceProvider(scanID, provider, rawData string) {
	query := `INSERT INTO service_providers (scan_id, provider, raw_data) VALUES ($1, $2, $3)`
	_, err := dbPool.Exec(context.Background(), query, scanID, provider, rawData)
	if err != nil {
		log.Printf("[ERROR] Failed to insert service provider: %v", err)
	} else {
		log.Printf("[INFO] Successfully inserted service provider: %s", provider)
	}
}

func insertASN(scanID, asn, rawData string) {
	query := `INSERT INTO asns (scan_id, number, raw_data) VALUES ($1, $2, $3)`
	_, err := dbPool.Exec(context.Background(), query, scanID, asn, rawData)
	if err != nil {
		log.Printf("Failed to insert ASN: %v (ASN: %s)", err, asn)
	} else {
		log.Printf("Successfully inserted ASN: %s", asn)
	}
}

func insertSubnet(scanID, cidr, rawData string) {
	query := `INSERT INTO subnets (scan_id, cidr, raw_data) VALUES ($1, $2, $3)`
	_, err := dbPool.Exec(context.Background(), query, scanID, cidr, rawData)
	if err != nil {
		log.Printf("Failed to insert Subnet: %v (Subnet: %s)", err, cidr)
	} else {
		log.Printf("Successfully inserted Subnet: %s", cidr)
	}
}

func getASNs(w http.ResponseWriter, r *http.Request) {
	scanID := mux.Vars(r)["scan_id"]
	if scanID == "" || scanID == "No scans available" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]struct{}{})
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(scanID); err != nil {
		http.Error(w, "Invalid scan ID format", http.StatusBadRequest)
		return
	}

	query := `SELECT number, raw_data FROM asns WHERE scan_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scanID)
	if err != nil {
		http.Error(w, "Failed to fetch ASNs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type ASNResponse struct {
		Number  string `json:"number"`
		RawData string `json:"raw_data"`
	}

	var asns []ASNResponse
	for rows.Next() {
		var asn ASNResponse
		if err := rows.Scan(&asn.Number, &asn.RawData); err != nil {
			http.Error(w, "Error scanning ASN", http.StatusInternalServerError)
			return
		}
		asns = append(asns, asn)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(asns)
}

func getSubnets(w http.ResponseWriter, r *http.Request) {
	scanID := mux.Vars(r)["scan_id"]
	if scanID == "" || scanID == "No scans available" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]struct{}{})
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(scanID); err != nil {
		http.Error(w, "Invalid scan ID format", http.StatusBadRequest)
		return
	}

	query := `SELECT cidr, raw_data FROM subnets WHERE scan_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scanID)
	if err != nil {
		http.Error(w, "Failed to fetch subnets", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type SubnetResponse struct {
		CIDR    string `json:"cidr"`
		RawData string `json:"raw_data"`
	}

	var subnets []SubnetResponse
	for rows.Next() {
		var subnet SubnetResponse
		if err := rows.Scan(&subnet.CIDR, &subnet.RawData); err != nil {
			http.Error(w, "Error scanning subnet", http.StatusInternalServerError)
			return
		}
		subnets = append(subnets, subnet)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subnets)
}

func getAmassScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	scopeTargetID := mux.Vars(r)["id"]
	if scopeTargetID == "" {
		http.Error(w, "Scope target ID is required", http.StatusBadRequest)
		return
	}

	query := `SELECT id, scan_id, domain, status, result, error, stdout, stderr, command, execution_time, created_at 
              FROM amass_scans WHERE scope_target_id = $1`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch scans for scope target ID %s: %v", scopeTargetID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []map[string]interface{}
	for rows.Next() {
		var scan AmassScanStatus
		err := rows.Scan(
			&scan.ID,
			&scan.ScanID,
			&scan.Domain,
			&scan.Status,
			&scan.Result,
			&scan.Error,
			&scan.StdOut,
			&scan.StdErr,
			&scan.Command,
			&scan.ExecTime,
			&scan.CreatedAt,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		scans = append(scans, map[string]interface{}{
			"id":             scan.ID,
			"scan_id":        scan.ScanID,
			"domain":         scan.Domain,
			"status":         scan.Status,
			"result":         nullStringToString(scan.Result),
			"error":          nullStringToString(scan.Error),
			"stdout":         nullStringToString(scan.StdOut),
			"stderr":         nullStringToString(scan.StdErr),
			"command":        nullStringToString(scan.Command),
			"execution_time": nullStringToString(scan.ExecTime),
			"created_at":     scan.CreatedAt.Format(time.RFC3339),
		})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(scans)
}

func runAmassScan(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		FQDN string `json:"fqdn" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.FQDN == "" {
		http.Error(w, "Invalid request body. `fqdn` is required.", http.StatusBadRequest)
		return
	}

	domain := payload.FQDN
	wildcardDomain := fmt.Sprintf("*.%s", domain)

	query := `SELECT id FROM scope_targets WHERE type = 'Wildcard' AND scope_target = $1`
	var requestID string
	err := dbPool.QueryRow(context.Background(), query, wildcardDomain).Scan(&requestID)
	if err != nil {
		log.Printf("[ERROR] No matching wildcard scope target found for domain %s", domain)
		http.Error(w, "No matching wildcard scope target found.", http.StatusBadRequest)
		return
	}

	scanID := uuid.New().String()
	insertQuery := `INSERT INTO amass_scans (scan_id, domain, status, scope_target_id) VALUES ($1, $2, $3, $4)`
	_, err = dbPool.Exec(context.Background(), insertQuery, scanID, domain, "pending", requestID)
	if err != nil {
		log.Printf("[ERROR] Failed to create scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}

	go executeAndParseAmassScan(scanID, domain)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func executeAndParseAmassScan(scanID, domain string) {
	log.Printf("[INFO] Starting Amass scan for domain %s (scan ID: %s)", domain, scanID)
	startTime := time.Now()

	cmd := exec.Command(
		"docker", "run", "--rm",
		"caffix/amass",
		"enum", "-active", "-alts", "-brute", "-nocolor",
		"-min-for-recursive", "2", "-timeout", "60",
		"-d", domain,
		"-r", "8.8.8.8", "1.1.1.1", "9.9.9.9", "64.6.64.6",
		"208.67.222.222", "208.67.220.220", "8.26.56.26", "8.20.247.20",
		"185.228.168.9", "185.228.169.9", "76.76.19.19", "76.223.122.150",
		"198.101.242.72", "176.103.130.130", "176.103.130.131",
		"94.140.14.14", "94.140.15.15", "1.0.0.1", "77.88.8.8", "77.88.8.1",
	)

	log.Printf("[INFO] Executing command: %s", cmd.String())

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	execTime := time.Since(startTime).String()

	if err != nil {
		log.Printf("[ERROR] Amass scan failed for %s: %v", domain, err)
		log.Printf("[ERROR] stderr output: %s", stderr.String())
		updateScanStatus(scanID, "error", "", stderr.String(), cmd.String(), execTime)
		return
	}

	result := stdout.String()
	log.Printf("[INFO] Amass scan completed in %s for domain %s", execTime, domain)
	log.Printf("[DEBUG] Raw output length: %d bytes", len(result))

	if result != "" {
		log.Printf("[INFO] Starting to parse results for scan %s", scanID)
		parseAndStoreResults(scanID, domain, result)
		log.Printf("[INFO] Finished parsing results for scan %s", scanID)
	} else {
		log.Printf("[WARN] No output from Amass scan for domain %s", domain)
	}

	updateScanStatus(scanID, "success", result, stderr.String(), cmd.String(), execTime)
	log.Printf("[INFO] Scan status updated for scan %s", scanID)
}

func isCloudDomain(domain string) bool {
	awsDomains := []string{
		"amazonaws.com", "awsdns", "cloudfront.net",
	}

	googleDomains := []string{
		"google.com", "gcloud.com", "appspot.com",
		"googleapis.com", "gcp.com", "withgoogle.com",
	}

	azureDomains := []string{
		"azure.com", "cloudapp.azure.com", "windows.net",
		"microsoft.com", "trafficmanager.net", "azureedge.net", "azure.net",
		"api.applicationinsights.io", "signalr.net", "microsoftonline.com",
		"azurewebsites.net", "azure-api.net", "redis.cache.windows.net",
		"media.azure.net", "appserviceenvironment.net",
	}

	for _, awsDomain := range awsDomains {
		if strings.Contains(domain, awsDomain) {
			return true
		}
	}
	for _, googleDomain := range googleDomains {
		if strings.Contains(domain, googleDomain) {
			return true
		}
	}
	for _, azureDomain := range azureDomains {
		if strings.Contains(domain, azureDomain) {
			return true
		}
	}

	return false
}

func updateScanStatus(scanID, status, result, stderr, command, execTime string) {
	log.Printf("[INFO] Updating scan status for %s to %s", scanID, status)
	query := `UPDATE amass_scans SET status = $1, result = $2, stderr = $3, command = $4, execution_time = $5 WHERE scan_id = $6`
	_, err := dbPool.Exec(context.Background(), query, status, result, stderr, command, execTime, scanID)
	if err != nil {
		log.Printf("[ERROR] Failed to update scan status for %s: %v", scanID, err)
	} else {
		log.Printf("[INFO] Successfully updated scan status for %s", scanID)
	}
}

func insertCloudDomain(scanID, domain string) {
	var cloudType string

	awsDomains := []string{
		"amazonaws.com", "awsdns", "cloudfront.net",
	}

	googleDomains := []string{
		"google.com", "gcloud.com", "appspot.com",
		"googleapis.com", "gcp.com", "withgoogle.com",
	}

	azureDomains := []string{
		"azure.com", "cloudapp.azure.com", "windows.net",
		"microsoft.com", "trafficmanager.net", "azureedge.net", "azure.net",
		"api.applicationinsights.io", "signalr.net", "microsoftonline.com",
		"azurewebsites.net", "azure-api.net", "redis.cache.windows.net",
		"media.azure.net", "appserviceenvironment.net",
	}

	matchFound := false

	for _, awsDomain := range awsDomains {
		if strings.Contains(domain, awsDomain) {
			cloudType = "aws"
			matchFound = true
			break
		}
	}
	if !matchFound {
		for _, googleDomain := range googleDomains {
			if strings.Contains(domain, googleDomain) {
				cloudType = "gcp"
				matchFound = true
				break
			}
		}
	}
	if !matchFound {
		for _, azureDomain := range azureDomains {
			if strings.Contains(domain, azureDomain) {
				cloudType = "azure"
				matchFound = true
				break
			}
		}
	}

	if !matchFound {
		log.Printf("[DEBUG] Domain %s does not match any known cloud provider", domain)
		cloudType = "Unknown"
	}

	query := `INSERT INTO cloud_domains (scan_id, domain, type) VALUES ($1, $2, $3)`
	_, err := dbPool.Exec(context.Background(), query, scanID, domain, cloudType)
	if err != nil {
		log.Printf("[ERROR] Failed to insert cloud domain %s: %v", domain, err)
		return
	}

	log.Printf("[DEBUG] Successfully inserted cloud domain %s with type %s", domain, cloudType)
}

func insertSubdomain(scanID, subdomain string) {
	log.Printf("[DEBUG] Checking if subdomain %s for scanID %s is already stored in the database", subdomain, scanID)

	checkQuery := `SELECT COUNT(*) FROM subdomains WHERE scan_id = $1 AND subdomain = $2`
	var count int
	err := dbPool.QueryRow(context.Background(), checkQuery, scanID, subdomain).Scan(&count)
	if err != nil {
		log.Printf("[ERROR] Failed to check existence of subdomain %s for scanID %s: %v", subdomain, scanID, err)
		return
	}

	if count > 0 {
		log.Printf("[DEBUG] Subdomain %s for scanID %s already exists. Skipping insertion.", subdomain, scanID)
		return
	}
	log.Printf("[DEBUG] Attempting to insert subdomain %s for scanID: %s", subdomain, scanID)
	insertQuery := `INSERT INTO subdomains (scan_id, subdomain) VALUES ($1, $2)`
	_, err = dbPool.Exec(context.Background(), insertQuery, scanID, subdomain)
	if err != nil {
		log.Printf("[ERROR] Failed to insert subdomain %s for scan %s: %v", subdomain, scanID, err)
		return
	}

	log.Printf("[DEBUG] Successfully inserted subdomain %s for scanID: %s", subdomain, scanID)
}

func getAmassScanStatus(w http.ResponseWriter, r *http.Request) {
	scanID := mux.Vars(r)["scanID"]
	if scanID == "" {
		http.Error(w, "Scan ID is required", http.StatusBadRequest)
		return
	}

	var scan AmassScanStatus
	query := `SELECT id, scan_id, domain, status, result, error, stdout, stderr, command, execution_time, created_at FROM amass_scans WHERE scan_id = $1`
	err := dbPool.QueryRow(context.Background(), query, scanID).Scan(
		&scan.ID,
		&scan.ScanID,
		&scan.Domain,
		&scan.Status,
		&scan.Result,
		&scan.Error,
		&scan.StdOut,
		&scan.StdErr,
		&scan.Command,
		&scan.ExecTime,
		&scan.CreatedAt,
	)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch scan status: %v", err)
		http.Error(w, "Scan not found.", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"id":             scan.ID,
		"scan_id":        scan.ScanID,
		"domain":         scan.Domain,
		"status":         scan.Status,
		"result":         nullStringToString(scan.Result),
		"error":          nullStringToString(scan.Error),
		"stdout":         nullStringToString(scan.StdOut),
		"stderr":         nullStringToString(scan.StdErr),
		"command":        nullStringToString(scan.Command),
		"execution_time": nullStringToString(scan.ExecTime),
		"created_at":     scan.CreatedAt.Format(time.RFC3339),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func createScopeTarget(w http.ResponseWriter, r *http.Request) {
	var payload RequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `INSERT INTO scope_targets (type, mode, scope_target, active) VALUES ($1, $2, $3, $4)`
	_, err := dbPool.Exec(context.Background(), query, payload.Type, payload.Mode, payload.ScopeTarget, payload.Active)
	if err != nil {
		log.Printf("Error inserting into database: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Request saved successfully"})
}

func readScopeTarget(w http.ResponseWriter, r *http.Request) {
	rows, err := dbPool.Query(context.Background(), `SELECT id, type, mode, scope_target, active FROM scope_targets`)
	if err != nil {
		log.Printf("Error querying database: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []ResponsePayload
	for rows.Next() {
		var res ResponsePayload
		if err := rows.Scan(&res.ID, &res.Type, &res.Mode, &res.ScopeTarget, &res.Active); err != nil {
			log.Printf("Error scanning row: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		results = append(results, res)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(results)
}

func deleteScopeTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "ID is required in the path", http.StatusBadRequest)
		return
	}

	query := `DELETE FROM scope_targets WHERE id = $1`
	_, err := dbPool.Exec(context.Background(), query, id)
	if err != nil {
		log.Printf("Error deleting from database: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Request deleted successfully"})
}

func insertDNSRecord(scanID, record, recordType string) {
	log.Printf("[DEBUG] Inserting DNS record type %s for scan %s: %s", recordType, scanID, record)
	query := `INSERT INTO dns_records (scan_id, record, record_type) VALUES ($1, $2, $3)`
	_, err := dbPool.Exec(context.Background(), query, scanID, record, recordType)
	if err != nil {
		log.Printf("[ERROR] Failed to insert DNS record: %v", err)
	} else {
		log.Printf("[DEBUG] Successfully inserted DNS record type %s for scan %s", recordType, scanID)
	}
}

func insertIP(scanID, ip string) {
	log.Printf("[DEBUG] Inserting IP address for scan %s: %s", scanID, ip)
	query := `INSERT INTO ips (scan_id, ip_address) VALUES ($1, $2)`
	_, err := dbPool.Exec(context.Background(), query, scanID, ip)
	if err != nil {
		log.Printf("[ERROR] Failed to insert IP address %s: %v", ip, err)
	} else {
		log.Printf("[DEBUG] Successfully inserted IP address %s for scan %s", ip, scanID)
	}
}

func getDNSRecords(w http.ResponseWriter, r *http.Request) {
	scanID := mux.Vars(r)["scan_id"]
	if scanID == "" || scanID == "No scans available" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]struct{}{})
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(scanID); err != nil {
		http.Error(w, "Invalid scan ID format", http.StatusBadRequest)
		return
	}

	query := `SELECT id, scan_id, record, record_type, created_at FROM dns_records WHERE scan_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scanID)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch DNS records for scan %s: %v", scanID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var dnsRecords []DNSRecord
	for rows.Next() {
		var dnsRecord DNSRecord
		if err := rows.Scan(&dnsRecord.ID, &dnsRecord.ScanID, &dnsRecord.Record, &dnsRecord.Type, &dnsRecord.CreatedAt); err != nil {
			log.Printf("[ERROR] Failed to scan DNS record row: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		dnsRecords = append(dnsRecords, dnsRecord)
	}

	if err := rows.Err(); err != nil {
		log.Printf("[ERROR] Error iterating over rows: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(dnsRecords); err != nil {
		log.Printf("[ERROR] Failed to encode DNS records: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func getIPs(w http.ResponseWriter, r *http.Request) {
	scanID := mux.Vars(r)["scan_id"]
	query := `SELECT ip_address FROM ips WHERE scan_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scanID)
	if err != nil {
		http.Error(w, "Failed to fetch IPs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var ips []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			http.Error(w, "Error scanning IP", http.StatusInternalServerError)
			return
		}
		ips = append(ips, ip)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ips)
}

func getSubdomains(w http.ResponseWriter, r *http.Request) {
	scanID := mux.Vars(r)["scan_id"]
	if scanID == "" {
		http.Error(w, "Scan ID is required", http.StatusBadRequest)
		return
	}

	// Explicitly initialize to an empty slice
	subdomains := []string{}

	subdomainQuery := `SELECT subdomain FROM subdomains WHERE scan_id = $1 ORDER BY created_at DESC`
	subRows, err := dbPool.Query(context.Background(), subdomainQuery, scanID)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch subdomains for scan %s: %v", scanID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer subRows.Close()

	for subRows.Next() {
		var subdomain string
		if err := subRows.Scan(&subdomain); err != nil {
			log.Printf("[ERROR] Failed to scan subdomain row: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		subdomains = append(subdomains, subdomain)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(subdomains); err != nil {
		log.Printf("[ERROR] Failed to encode subdomains: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func getCloudDomains(w http.ResponseWriter, r *http.Request) {
	scanID := mux.Vars(r)["scan_id"]
	if scanID == "" {
		http.Error(w, "Scan ID is required", http.StatusBadRequest)
		return
	}

	var awsDomains, gcpDomains, azureDomains []string

	cloudDomainQuery := `SELECT domain, type FROM cloud_domains WHERE scan_id = $1 ORDER BY domain ASC`
	cloudRows, err := dbPool.Query(context.Background(), cloudDomainQuery, scanID)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch cloud domains for scan %s: %v", scanID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer cloudRows.Close()

	for cloudRows.Next() {
		var domain, domainType string
		if err := cloudRows.Scan(&domain, &domainType); err != nil {
			log.Printf("[ERROR] Failed to scan cloud domain row: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		switch domainType {
		case "aws":
			awsDomains = append(awsDomains, domain)
		case "gcp":
			gcpDomains = append(gcpDomains, domain)
		case "azure":
			azureDomains = append(azureDomains, domain)
		}
	}

	response := map[string][]string{
		"aws_domains":   awsDomains,
		"gcp_domains":   gcpDomains,
		"azure_domains": azureDomains,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[ERROR] Failed to encode cloud domains: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func getServiceProviders(w http.ResponseWriter, r *http.Request) {
	scanID := mux.Vars(r)["scan_id"]
	if scanID == "" || scanID == "No scans available" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]struct{}{})
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(scanID); err != nil {
		http.Error(w, "Invalid scan ID format", http.StatusBadRequest)
		return
	}

	query := `SELECT provider, raw_data FROM service_providers WHERE scan_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scanID)
	if err != nil {
		http.Error(w, "Failed to fetch service providers", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type ServiceProviderResponse struct {
		Provider string `json:"provider"`
		RawData  string `json:"raw_data"`
	}

	var providers []ServiceProviderResponse
	for rows.Next() {
		var provider ServiceProviderResponse
		if err := rows.Scan(&provider.Provider, &provider.RawData); err != nil {
			http.Error(w, "Error scanning service provider", http.StatusInternalServerError)
			return
		}
		providers = append(providers, provider)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(providers)
}

func getLatestAmassSubdomains(scopeTargetID string) ([]string, error) {
	query := `
		WITH latest_scan AS (
			SELECT scan_id 
			FROM amass_scans 
			WHERE scope_target_id = $1 
			AND status = 'success'
			ORDER BY created_at DESC 
			LIMIT 1
		)
		SELECT subdomain 
		FROM subdomains 
		WHERE scan_id IN (SELECT scan_id FROM latest_scan)
		ORDER BY subdomain ASC;
	`

	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subdomains: %v", err)
	}
	defer rows.Close()

	var subdomains []string
	for rows.Next() {
		var subdomain string
		if err := rows.Scan(&subdomain); err != nil {
			return nil, fmt.Errorf("failed to scan subdomain: %v", err)
		}
		subdomains = append(subdomains, subdomain)
	}

	return subdomains, nil
}

func runHttpxScan(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		FQDN string `json:"fqdn" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.FQDN == "" {
		http.Error(w, "Invalid request body. `fqdn` is required.", http.StatusBadRequest)
		return
	}

	domain := payload.FQDN
	wildcardDomain := fmt.Sprintf("*.%s", domain)

	// Get the scope target ID
	query := `SELECT id FROM scope_targets WHERE type = 'Wildcard' AND scope_target = $1`
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(), query, wildcardDomain).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] No matching wildcard scope target found for domain %s", domain)
		http.Error(w, "No matching wildcard scope target found.", http.StatusBadRequest)
		return
	}

	// Get subdomains from latest amass scan
	subdomains, err := getLatestAmassSubdomains(scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to get subdomains: %v", err)
	}

	// If no subdomains found, use the main domain
	domainsToScan := subdomains
	if len(domainsToScan) == 0 {
		domainsToScan = []string{domain}
	}

	scanID := uuid.New().String()
	insertQuery := `INSERT INTO httpx_scans (scan_id, domain, status, scope_target_id) VALUES ($1, $2, $3, $4)`
	_, err = dbPool.Exec(context.Background(), insertQuery, scanID, domain, "pending", scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to create scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}

	go executeAndParseHttpxScan(scanID, domain, domainsToScan)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func executeAndParseHttpxScan(scanID, domain string, domainsToScan []string) {
	log.Printf("[INFO] Starting httpx scan for domain %s (scan ID: %s)", domain, scanID)
	startTime := time.Now()

	mountDir := "/tmp/httpx-mounts"
	if err := os.MkdirAll(mountDir, 0755); err != nil {
		log.Printf("[ERROR] Failed to create mount directory: %v", err)
		updateHttpxScanStatus(scanID, "error", "", fmt.Sprintf("Failed to create mount directory: %v", err), "", time.Since(startTime).String())
		return
	}

	mountPath := filepath.Join(mountDir, "domains.txt")
	domainsFile, err := os.Create(mountPath)
	if err != nil {
		log.Printf("[ERROR] Failed to create domains file: %v", err)
		updateHttpxScanStatus(scanID, "error", "", fmt.Sprintf("Failed to create domains file: %v", err), "", time.Since(startTime).String())
		return
	}
	defer os.Remove(mountPath)

	for _, subdomain := range domainsToScan {
		if _, err := domainsFile.WriteString(subdomain + "\n"); err != nil {
			log.Printf("[ERROR] Failed to write subdomain %s to file: %v", subdomain, err)
			updateHttpxScanStatus(scanID, "error", "", fmt.Sprintf("Failed to write to domains file: %v", err), "", time.Since(startTime).String())
			return
		}
	}
	domainsFile.Close()

	if err := os.Chmod(mountPath, 0644); err != nil {
		log.Printf("[ERROR] Failed to set file permissions: %v", err)
		updateHttpxScanStatus(scanID, "error", "", fmt.Sprintf("Failed to set file permissions: %v", err), "", time.Since(startTime).String())
		return
	}

	// Verify file contents
	content, err := os.ReadFile(mountPath)
	if err != nil {
		log.Printf("[ERROR] Failed to read domains file: %v", err)
	} else {
		log.Printf("[DEBUG] Domains file contents:\n%s", string(content))
	}

	// Run httpx with correct flags
	cmd := exec.Command(
		"httpx",
		"-l", mountPath,
		"-json",
		"-status-code",
		"-title",
		"-tech-detect",
		"-server",
		"-content-length",
		"-no-color",
		"-timeout", "10",
		"-retries", "2",
		"-mc", "100,101,200,201,202,203,204,205,206,207,208,226,300,301,302,303,304,305,307,308,400,401,402,403,404,405,406,407,408,409,410,411,412,413,414,415,416,417,418,421,422,423,424,426,428,429,431,451,500,501,502,503,504,505,506,507,508,510,511",
		"-o", "/tmp/httpx-output.json",
	)

	log.Printf("[INFO] Executing command: %s", cmd.String())

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	execTime := time.Since(startTime).String()

	if err != nil {
		log.Printf("[ERROR] httpx scan failed for %s: %v", domain, err)
		log.Printf("[ERROR] stderr output: %s", stderr.String())
		updateHttpxScanStatus(scanID, "error", "", stderr.String(), cmd.String(), execTime)
		return
	}

	// Read the output file
	outputContent, err := os.ReadFile("/tmp/httpx-output.json")
	if err != nil {
		log.Printf("[ERROR] Failed to read output file: %v", err)
		updateHttpxScanStatus(scanID, "error", "", fmt.Sprintf("Failed to read output file: %v", err), cmd.String(), execTime)
		return
	}

	result := string(outputContent)
	log.Printf("[INFO] httpx scan completed in %s for domain %s", execTime, domain)
	log.Printf("[DEBUG] Raw output length: %d bytes", len(result))
	if stderr.Len() > 0 {
		log.Printf("[DEBUG] stderr output: %s", stderr.String())
	}

	if result == "" {
		log.Printf("[WARN] No output from httpx scan")
		updateHttpxScanStatus(scanID, "completed", "", "No results found", cmd.String(), execTime)
	} else {
		log.Printf("[DEBUG] httpx output: %s", result)
		updateHttpxScanStatus(scanID, "success", result, stderr.String(), cmd.String(), execTime)
	}

	log.Printf("[INFO] Scan status updated for scan %s", scanID)
}

func updateHttpxScanStatus(scanID, status, result, stderr, command, execTime string) {
	log.Printf("[INFO] Updating httpx scan status for %s to %s", scanID, status)
	query := `UPDATE httpx_scans SET status = $1, result = $2, stderr = $3, command = $4, execution_time = $5 WHERE scan_id = $6`
	_, err := dbPool.Exec(context.Background(), query, status, result, stderr, command, execTime, scanID)
	if err != nil {
		log.Printf("[ERROR] Failed to update httpx scan status for %s: %v", scanID, err)
	} else {
		log.Printf("[INFO] Successfully updated httpx scan status for %s", scanID)
	}
}

func getHttpxScanStatus(w http.ResponseWriter, r *http.Request) {
	scanID := mux.Vars(r)["scanID"]
	if scanID == "" {
		http.Error(w, "Scan ID is required", http.StatusBadRequest)
		return
	}

	var scan HttpxScanStatus
	query := `SELECT id, scan_id, domain, status, result, error, stdout, stderr, command, execution_time, created_at FROM httpx_scans WHERE scan_id = $1`
	err := dbPool.QueryRow(context.Background(), query, scanID).Scan(
		&scan.ID,
		&scan.ScanID,
		&scan.Domain,
		&scan.Status,
		&scan.Result,
		&scan.Error,
		&scan.StdOut,
		&scan.StdErr,
		&scan.Command,
		&scan.ExecTime,
		&scan.CreatedAt,
	)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch httpx scan status: %v", err)
		http.Error(w, "Scan not found.", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"id":             scan.ID,
		"scan_id":        scan.ScanID,
		"domain":         scan.Domain,
		"status":         scan.Status,
		"result":         nullStringToString(scan.Result),
		"error":          nullStringToString(scan.Error),
		"stdout":         nullStringToString(scan.StdOut),
		"stderr":         nullStringToString(scan.StdErr),
		"command":        nullStringToString(scan.Command),
		"execution_time": nullStringToString(scan.ExecTime),
		"created_at":     scan.CreatedAt.Format(time.RFC3339),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func getHttpxScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	scopeTargetID := mux.Vars(r)["id"]
	if scopeTargetID == "" {
		http.Error(w, "Scope target ID is required", http.StatusBadRequest)
		return
	}

	query := `SELECT id, scan_id, domain, status, result, error, stdout, stderr, command, execution_time, created_at 
              FROM httpx_scans WHERE scope_target_id = $1`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch scans for scope target ID %s: %v", scopeTargetID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []map[string]interface{}
	for rows.Next() {
		var scan HttpxScanStatus
		err := rows.Scan(
			&scan.ID,
			&scan.ScanID,
			&scan.Domain,
			&scan.Status,
			&scan.Result,
			&scan.Error,
			&scan.StdOut,
			&scan.StdErr,
			&scan.Command,
			&scan.ExecTime,
			&scan.CreatedAt,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		scans = append(scans, map[string]interface{}{
			"id":             scan.ID,
			"scan_id":        scan.ScanID,
			"domain":         scan.Domain,
			"status":         scan.Status,
			"result":         nullStringToString(scan.Result),
			"error":          nullStringToString(scan.Error),
			"stdout":         nullStringToString(scan.StdOut),
			"stderr":         nullStringToString(scan.StdErr),
			"command":        nullStringToString(scan.Command),
			"execution_time": nullStringToString(scan.ExecTime),
			"created_at":     scan.CreatedAt.Format(time.RFC3339),
		})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(scans)
}

func activateScopeTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "ID is required in the path", http.StatusBadRequest)
		return
	}

	// Start a transaction
	tx, err := dbPool.Begin(context.Background())
	if err != nil {
		log.Printf("[ERROR] Failed to begin transaction: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(context.Background())

	// First, deactivate all scope targets
	_, err = tx.Exec(context.Background(), `UPDATE scope_targets SET active = false`)
	if err != nil {
		log.Printf("[ERROR] Failed to deactivate scope targets: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Then, activate the selected scope target
	result, err := tx.Exec(context.Background(), `UPDATE scope_targets SET active = true WHERE id = $1`, id)
	if err != nil {
		log.Printf("[ERROR] Failed to activate scope target: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Scope target not found", http.StatusNotFound)
		return
	}

	// Commit the transaction
	if err := tx.Commit(context.Background()); err != nil {
		log.Printf("[ERROR] Failed to commit transaction: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Scope target activated successfully"})
}

func getAllScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	scopeTargetID := mux.Vars(r)["id"]
	if scopeTargetID == "" {
		http.Error(w, "Scope target ID is required", http.StatusBadRequest)
		return
	}

	// Query for Amass scans
	amassQuery := `
		SELECT id, scan_id, domain, status, result, error, stdout, stderr, command, execution_time, created_at 
		FROM amass_scans 
		WHERE scope_target_id = $1
	`
	amassRows, err := dbPool.Query(context.Background(), amassQuery, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch Amass scans: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer amassRows.Close()

	// Query for httpx scans
	httpxQuery := `
		SELECT id, scan_id, domain, status, result, error, stdout, stderr, command, execution_time, created_at 
		FROM httpx_scans 
		WHERE scope_target_id = $1
	`
	httpxRows, err := dbPool.Query(context.Background(), httpxQuery, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch httpx scans: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer httpxRows.Close()

	// Query for GAU scans
	gauQuery := `
		SELECT id, scan_id, domain, status, result, error, stdout, stderr, command, execution_time, created_at 
		FROM gau_scans 
		WHERE scope_target_id = $1
	`
	gauRows, err := dbPool.Query(context.Background(), gauQuery, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch GAU scans: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer gauRows.Close()

	var allScans []ScanSummary

	// Process Amass scans
	for amassRows.Next() {
		var scan AmassScanStatus
		err := amassRows.Scan(
			&scan.ID,
			&scan.ScanID,
			&scan.Domain,
			&scan.Status,
			&scan.Result,
			&scan.Error,
			&scan.StdOut,
			&scan.StdErr,
			&scan.Command,
			&scan.ExecTime,
			&scan.CreatedAt,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to scan Amass row: %v", err)
			continue
		}

		allScans = append(allScans, ScanSummary{
			ID:        scan.ID,
			ScanID:    scan.ScanID,
			Domain:    scan.Domain,
			Status:    scan.Status,
			Result:    nullStringToString(scan.Result),
			Error:     nullStringToString(scan.Error),
			StdOut:    nullStringToString(scan.StdOut),
			StdErr:    nullStringToString(scan.StdErr),
			Command:   nullStringToString(scan.Command),
			ExecTime:  nullStringToString(scan.ExecTime),
			CreatedAt: scan.CreatedAt,
			ScanType:  "amass",
		})
	}

	// Process httpx scans
	for httpxRows.Next() {
		var scan HttpxScanStatus
		err := httpxRows.Scan(
			&scan.ID,
			&scan.ScanID,
			&scan.Domain,
			&scan.Status,
			&scan.Result,
			&scan.Error,
			&scan.StdOut,
			&scan.StdErr,
			&scan.Command,
			&scan.ExecTime,
			&scan.CreatedAt,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to scan httpx row: %v", err)
			continue
		}

		allScans = append(allScans, ScanSummary{
			ID:        scan.ID,
			ScanID:    scan.ScanID,
			Domain:    scan.Domain,
			Status:    scan.Status,
			Result:    nullStringToString(scan.Result),
			Error:     nullStringToString(scan.Error),
			StdOut:    nullStringToString(scan.StdOut),
			StdErr:    nullStringToString(scan.StdErr),
			Command:   nullStringToString(scan.Command),
			ExecTime:  nullStringToString(scan.ExecTime),
			CreatedAt: scan.CreatedAt,
			ScanType:  "httpx",
		})
	}

	// Process GAU scans
	for gauRows.Next() {
		var scan GauScanStatus
		err := gauRows.Scan(
			&scan.ID,
			&scan.ScanID,
			&scan.Domain,
			&scan.Status,
			&scan.Result,
			&scan.Error,
			&scan.StdOut,
			&scan.StdErr,
			&scan.Command,
			&scan.ExecTime,
			&scan.CreatedAt,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to scan GAU row: %v", err)
			continue
		}

		allScans = append(allScans, ScanSummary{
			ID:        scan.ID,
			ScanID:    scan.ScanID,
			Domain:    scan.Domain,
			Status:    scan.Status,
			Result:    nullStringToString(scan.Result),
			Error:     nullStringToString(scan.Error),
			StdOut:    nullStringToString(scan.StdOut),
			StdErr:    nullStringToString(scan.StdErr),
			Command:   nullStringToString(scan.Command),
			ExecTime:  nullStringToString(scan.ExecTime),
			CreatedAt: scan.CreatedAt,
			ScanType:  "gau",
		})
	}

	// Sort all scans by creation date, newest first
	sort.Slice(allScans, func(i, j int) bool {
		return allScans[i].CreatedAt.After(allScans[j].CreatedAt)
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(allScans)
}

func runGauScan(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		FQDN string `json:"fqdn" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.FQDN == "" {
		http.Error(w, "Invalid request body. `fqdn` is required.", http.StatusBadRequest)
		return
	}

	domain := payload.FQDN
	wildcardDomain := fmt.Sprintf("*.%s", domain)

	// Get the scope target ID
	query := `SELECT id FROM scope_targets WHERE type = 'Wildcard' AND scope_target = $1`
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(), query, wildcardDomain).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] No matching wildcard scope target found for domain %s", domain)
		http.Error(w, "No matching wildcard scope target found.", http.StatusBadRequest)
		return
	}

	scanID := uuid.New().String()
	insertQuery := `INSERT INTO gau_scans (scan_id, domain, status, scope_target_id) VALUES ($1, $2, $3, $4)`
	_, err = dbPool.Exec(context.Background(), insertQuery, scanID, domain, "pending", scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to create scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}

	go executeAndParseGauScan(scanID, domain)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func executeAndParseGauScan(scanID, domain string) {
	log.Printf("[INFO] Starting GAU scan for domain %s (scan ID: %s)", domain, scanID)
	startTime := time.Now()

	cmd := exec.Command(
		"docker", "run", "--rm",
		"sxcurity/gau:latest",
		domain,
		"--providers", "wayback",
		"--json",
		"--verbose",
		"--subs",
		"--threads", "10",
		"--timeout", "60",
		"--retries", "2",
	)

	log.Printf("[INFO] Executing command: %s", cmd.String())

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	execTime := time.Since(startTime).String()

	if err != nil {
		log.Printf("[ERROR] GAU scan failed for %s: %v", domain, err)
		log.Printf("[ERROR] stderr output: %s", stderr.String())
		updateGauScanStatus(scanID, "error", "", stderr.String(), cmd.String(), execTime)
		return
	}

	result := stdout.String()
	log.Printf("[INFO] GAU scan completed in %s for domain %s", execTime, domain)
	log.Printf("[DEBUG] Raw output length: %d bytes", len(result))
	if stderr.Len() > 0 {
		log.Printf("[DEBUG] stderr output: %s", stderr.String())
	}

	// Check if we have actual results
	if result == "" {
		// Try a second attempt with different flags
		cmd = exec.Command(
			"docker", "run", "--rm",
			"sxcurity/gau:latest",
			domain,
			"--providers", "wayback,otx,urlscan",
			"--subs",
			"--threads", "5",
			"--timeout", "30",
			"--retries", "3",
		)

		log.Printf("[INFO] No results from first attempt, trying second attempt with command: %s", cmd.String())

		stdout.Reset()
		stderr.Reset()
		err = cmd.Run()

		if err == nil {
			result = stdout.String()
		}
	}

	if result == "" {
		log.Printf("[WARN] No output from GAU scan after retries")
		updateGauScanStatus(scanID, "completed", "", "No results found after multiple attempts", cmd.String(), execTime)
	} else {
		log.Printf("[DEBUG] GAU output: %s", result)
		updateGauScanStatus(scanID, "success", result, stderr.String(), cmd.String(), execTime)
	}

	log.Printf("[INFO] Scan status updated for scan %s", scanID)
}

func updateGauScanStatus(scanID, status, result, stderr, command, execTime string) {
	log.Printf("[INFO] Updating GAU scan status for %s to %s", scanID, status)
	query := `UPDATE gau_scans SET status = $1, result = $2, stderr = $3, command = $4, execution_time = $5 WHERE scan_id = $6`
	_, err := dbPool.Exec(context.Background(), query, status, result, stderr, command, execTime, scanID)
	if err != nil {
		log.Printf("[ERROR] Failed to update GAU scan status for %s: %v", scanID, err)
	} else {
		log.Printf("[INFO] Successfully updated GAU scan status for %s", scanID)
	}
}

func getGauScanStatus(w http.ResponseWriter, r *http.Request) {
	scanID := mux.Vars(r)["scanID"]
	if scanID == "" {
		http.Error(w, "Scan ID is required", http.StatusBadRequest)
		return
	}

	var scan GauScanStatus
	query := `SELECT id, scan_id, domain, status, result, error, stdout, stderr, command, execution_time, created_at FROM gau_scans WHERE scan_id = $1`
	err := dbPool.QueryRow(context.Background(), query, scanID).Scan(
		&scan.ID,
		&scan.ScanID,
		&scan.Domain,
		&scan.Status,
		&scan.Result,
		&scan.Error,
		&scan.StdOut,
		&scan.StdErr,
		&scan.Command,
		&scan.ExecTime,
		&scan.CreatedAt,
	)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch GAU scan status: %v", err)
		http.Error(w, "Scan not found.", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"id":             scan.ID,
		"scan_id":        scan.ScanID,
		"domain":         scan.Domain,
		"status":         scan.Status,
		"result":         nullStringToString(scan.Result),
		"error":          nullStringToString(scan.Error),
		"stdout":         nullStringToString(scan.StdOut),
		"stderr":         nullStringToString(scan.StdErr),
		"command":        nullStringToString(scan.Command),
		"execution_time": nullStringToString(scan.ExecTime),
		"created_at":     scan.CreatedAt.Format(time.RFC3339),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func getGauScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	scopeTargetID := mux.Vars(r)["id"]
	if scopeTargetID == "" {
		http.Error(w, "Scope target ID is required", http.StatusBadRequest)
		return
	}

	query := `SELECT id, scan_id, domain, status, result, error, stdout, stderr, command, execution_time, created_at 
              FROM gau_scans WHERE scope_target_id = $1`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch scans for scope target ID %s: %v", scopeTargetID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []map[string]interface{}
	for rows.Next() {
		var scan GauScanStatus
		err := rows.Scan(
			&scan.ID,
			&scan.ScanID,
			&scan.Domain,
			&scan.Status,
			&scan.Result,
			&scan.Error,
			&scan.StdOut,
			&scan.StdErr,
			&scan.Command,
			&scan.ExecTime,
			&scan.CreatedAt,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		scans = append(scans, map[string]interface{}{
			"id":             scan.ID,
			"scan_id":        scan.ScanID,
			"domain":         scan.Domain,
			"status":         scan.Status,
			"result":         nullStringToString(scan.Result),
			"error":          nullStringToString(scan.Error),
			"stdout":         nullStringToString(scan.StdOut),
			"stderr":         nullStringToString(scan.StdErr),
			"command":        nullStringToString(scan.Command),
			"execution_time": nullStringToString(scan.ExecTime),
			"created_at":     scan.CreatedAt.Format(time.RFC3339),
		})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(scans)
}

func runSublist3rScan(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Received request to run Sublist3r scan")
	var requestData struct {
		FQDN string `json:"fqdn"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		log.Printf("[ERROR] Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	domain := requestData.FQDN
	wildcardDomain := "*." + domain
	log.Printf("[INFO] Processing Sublist3r scan request for domain: %s", domain)

	query := `SELECT id FROM scope_targets WHERE type = 'Wildcard' AND scope_target = $1`
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(), query, wildcardDomain).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] No matching wildcard scope target found for domain %s: %v", domain, err)
		http.Error(w, "No matching wildcard scope target found.", http.StatusBadRequest)
		return
	}
	log.Printf("[INFO] Found matching scope target ID: %s", scopeTargetID)

	scanID := uuid.New().String()
	log.Printf("[INFO] Generated new scan ID: %s", scanID)

	insertQuery := `INSERT INTO sublist3r_scans (scan_id, domain, status, scope_target_id) VALUES ($1, $2, $3, $4)`
	_, err = dbPool.Exec(context.Background(), insertQuery, scanID, domain, "pending", scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to create Sublist3r scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}
	log.Printf("[INFO] Successfully created Sublist3r scan record in database")

	go executeAndParseSublist3rScan(scanID, domain)

	log.Printf("[INFO] Initiated Sublist3r scan with ID: %s for domain: %s", scanID, domain)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func executeAndParseSublist3rScan(scanID, domain string) {
	log.Printf("[INFO] Starting Sublist3r scan for domain %s (scan ID: %s)", domain, scanID)
	log.Printf("[DEBUG] Initializing scan variables and preparing command")
	startTime := time.Now()

	log.Printf("[DEBUG] Constructing docker command for Sublist3r")
	cmd := exec.Command(
		"docker", "exec",
		"ars0n-framework-v2-sublist3r-1",
		"python", "/app/sublist3r.py",
		"-d", domain,
		"-v",
		"-t", "50",
		"-o", "/dev/stdout",
	)

	log.Printf("[DEBUG] Docker command constructed: %s", cmd.String())
	log.Printf("[DEBUG] Setting up stdout and stderr buffers")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Printf("[INFO] Executing Sublist3r command at %s", time.Now().Format(time.RFC3339))
	log.Printf("[DEBUG] Command working directory: %s", cmd.Dir)
	log.Printf("[DEBUG] Command environment variables: %v", cmd.Env)

	err := cmd.Run()
	execTime := time.Since(startTime).String()
	log.Printf("[INFO] Command execution completed in %s", execTime)

	if err != nil {
		log.Printf("[ERROR] Sublist3r scan failed with error: %v", err)
		log.Printf("[ERROR] Error type: %T", err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.Printf("[ERROR] Exit code: %d", exitErr.ExitCode())
		}
		log.Printf("[ERROR] Stderr output length: %d bytes", stderr.Len())
		log.Printf("[ERROR] Stderr output content: %s", stderr.String())
		log.Printf("[ERROR] Stdout output length: %d bytes", stdout.Len())
		log.Printf("[DEBUG] Updating scan status to error state")
		updateSublist3rScanStatus(scanID, "error", "", stderr.String(), cmd.String(), execTime)
		return
	}

	log.Printf("[INFO] Sublist3r scan completed successfully in %s", execTime)
	log.Printf("[DEBUG] Processing scan output")

	// Process the output
	lines := strings.Split(stdout.String(), "\n")
	log.Printf("[INFO] Processing %d lines of output", len(lines))

	// Use a map to handle deduplication
	uniqueSubdomains := make(map[string]bool)
	for _, line := range lines {
		// Clean the line by removing ANSI color codes and other control characters
		cleanLine := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`).ReplaceAllString(line, "")
		cleanLine = strings.TrimSpace(cleanLine)

		// Skip empty lines, banner lines, and status messages
		if cleanLine == "" ||
			strings.Contains(cleanLine, "Sublist3r") ||
			strings.Contains(cleanLine, "==") ||
			strings.Contains(cleanLine, "Total Unique Subdomains Found:") ||
			strings.HasPrefix(cleanLine, "[-]") ||
			strings.HasPrefix(cleanLine, "[!]") ||
			strings.HasPrefix(cleanLine, "[~]") ||
			strings.HasPrefix(cleanLine, "[+]") {
			continue
		}

		// Remove "SSL Certificates: " prefix if present
		cleanLine = strings.TrimPrefix(cleanLine, "SSL Certificates: ")

		// If the line is a valid subdomain of our target domain, add it to our map
		if strings.HasSuffix(cleanLine, domain) {
			uniqueSubdomains[cleanLine] = true
		}
	}

	// Convert map keys to slice
	var finalSubdomains []string
	for subdomain := range uniqueSubdomains {
		finalSubdomains = append(finalSubdomains, subdomain)
	}

	// Sort the results for consistency
	sort.Strings(finalSubdomains)

	// Join the results with newlines
	result := strings.Join(finalSubdomains, "\n")
	log.Printf("[DEBUG] Final result string length: %d bytes", len(result))

	log.Printf("[INFO] Updating scan status in database for scan ID: %s", scanID)
	updateSublist3rScanStatus(scanID, "completed", result, stderr.String(), cmd.String(), execTime)
	log.Printf("[INFO] Sublist3r scan completed successfully for domain %s (scan ID: %s)", domain, scanID)
	log.Printf("[INFO] Total execution time including processing: %s", time.Since(startTime))
}

func updateSublist3rScanStatus(scanID, status, result, stderr, command, execTime string) {
	log.Printf("[INFO] Updating Sublist3r scan status for scan ID %s to %s", scanID, status)
	query := `UPDATE sublist3r_scans SET status = $1, result = $2, stderr = $3, command = $4, execution_time = $5 WHERE scan_id = $6`
	_, err := dbPool.Exec(context.Background(), query, status, result, stderr, command, execTime, scanID)
	if err != nil {
		log.Printf("[ERROR] Failed to update Sublist3r scan status: %v", err)
		return
	}
	log.Printf("[INFO] Successfully updated Sublist3r scan status for scan ID %s", scanID)
}

func getSublist3rScanStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scanID := vars["scan_id"]

	var scan Sublist3rScanStatus
	query := `SELECT * FROM sublist3r_scans WHERE scan_id = $1`
	err := dbPool.QueryRow(context.Background(), query, scanID).Scan(
		&scan.ID,
		&scan.ScanID,
		&scan.Domain,
		&scan.Status,
		&scan.Result,
		&scan.Error,
		&scan.StdOut,
		&scan.StdErr,
		&scan.Command,
		&scan.ExecTime,
		&scan.CreatedAt,
		&scan.ScopeTargetID,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "Scan not found", http.StatusNotFound)
		} else {
			log.Printf("[ERROR] Failed to get scan status: %v", err)
			http.Error(w, "Failed to get scan status", http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"id":              scan.ID,
		"scan_id":         scan.ScanID,
		"domain":          scan.Domain,
		"status":          scan.Status,
		"result":          nullStringToString(scan.Result),
		"error":           nullStringToString(scan.Error),
		"stdout":          nullStringToString(scan.StdOut),
		"stderr":          nullStringToString(scan.StdErr),
		"command":         nullStringToString(scan.Command),
		"execution_time":  nullStringToString(scan.ExecTime),
		"created_at":      scan.CreatedAt.Format(time.RFC3339),
		"scope_target_id": scan.ScopeTargetID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getSublist3rScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]

	if scopeTargetID == "" {
		log.Printf("[ERROR] No scope target ID provided")
		http.Error(w, "No scope target ID provided", http.StatusBadRequest)
		return
	}

	query := `SELECT * FROM sublist3r_scans WHERE scope_target_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to get scans: %v", err)
		http.Error(w, "Failed to get scans", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []map[string]interface{}
	for rows.Next() {
		var scan Sublist3rScanStatus
		var scopeTargetID string
		err := rows.Scan(
			&scan.ID,
			&scan.ScanID,
			&scan.Domain,
			&scan.Status,
			&scan.Result,
			&scan.Error,
			&scan.StdOut,
			&scan.StdErr,
			&scan.Command,
			&scan.ExecTime,
			&scan.CreatedAt,
			&scopeTargetID,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			continue
		}

		scans = append(scans, map[string]interface{}{
			"id":              scan.ID,
			"scan_id":         scan.ScanID,
			"domain":          scan.Domain,
			"status":          scan.Status,
			"result":          nullStringToString(scan.Result),
			"error":           nullStringToString(scan.Error),
			"stdout":          nullStringToString(scan.StdOut),
			"stderr":          nullStringToString(scan.StdErr),
			"command":         nullStringToString(scan.Command),
			"execution_time":  nullStringToString(scan.ExecTime),
			"created_at":      scan.CreatedAt.Format(time.RFC3339),
			"scope_target_id": scopeTargetID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scans)
}

func runAssetfinderScan(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		FQDN string `json:"fqdn" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.FQDN == "" {
		http.Error(w, "Invalid request body. `fqdn` is required.", http.StatusBadRequest)
		return
	}

	domain := payload.FQDN
	wildcardDomain := fmt.Sprintf("*.%s", domain)

	query := `SELECT id FROM scope_targets WHERE type = 'Wildcard' AND scope_target = $1`
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(), query, wildcardDomain).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] No matching wildcard scope target found for domain %s", domain)
		http.Error(w, "No matching wildcard scope target found.", http.StatusBadRequest)
		return
	}

	scanID := uuid.New().String()
	insertQuery := `INSERT INTO assetfinder_scans (scan_id, domain, status, scope_target_id) VALUES ($1, $2, $3, $4)`
	_, err = dbPool.Exec(context.Background(), insertQuery, scanID, domain, "pending", scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to create scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}

	go executeAndParseAssetfinderScan(scanID, domain)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func executeAndParseAssetfinderScan(scanID, domain string) {
	log.Printf("[INFO] Starting Assetfinder scan for domain %s (scan ID: %s)", domain, scanID)
	startTime := time.Now()

	cmd := exec.Command(
		"docker", "exec",
		"ars0n-framework-v2-assetfinder-1",
		"assetfinder",
		"--subs-only",
		domain,
	)

	log.Printf("[INFO] Executing command: %s", cmd.String())

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	execTime := time.Since(startTime).String()

	if err != nil {
		log.Printf("[ERROR] Assetfinder scan failed for %s: %v", domain, err)
		log.Printf("[ERROR] stderr output: %s", stderr.String())
		updateAssetfinderScanStatus(scanID, "error", "", stderr.String(), cmd.String(), execTime)
		return
	}

	result := stdout.String()
	log.Printf("[INFO] Assetfinder scan completed in %s for domain %s", execTime, domain)
	log.Printf("[DEBUG] Raw output length: %d bytes", len(result))

	if result == "" {
		log.Printf("[WARN] No output from Assetfinder scan")
		updateAssetfinderScanStatus(scanID, "completed", "", "No results found", cmd.String(), execTime)
	} else {
		log.Printf("[DEBUG] Assetfinder output: %s", result)
		updateAssetfinderScanStatus(scanID, "success", result, stderr.String(), cmd.String(), execTime)
	}

	log.Printf("[INFO] Scan status updated for scan %s", scanID)
}

func updateAssetfinderScanStatus(scanID, status, result, stderr, command, execTime string) {
	log.Printf("[INFO] Updating Assetfinder scan status for %s to %s", scanID, status)
	query := `UPDATE assetfinder_scans SET status = $1, result = $2, stderr = $3, command = $4, execution_time = $5 WHERE scan_id = $6`
	_, err := dbPool.Exec(context.Background(), query, status, result, stderr, command, execTime, scanID)
	if err != nil {
		log.Printf("[ERROR] Failed to update Assetfinder scan status for %s: %v", scanID, err)
	} else {
		log.Printf("[INFO] Successfully updated Assetfinder scan status for %s", scanID)
	}
}

func getAssetfinderScanStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scanID := vars["scan_id"]

	var scan AssetfinderScanStatus
	query := `SELECT * FROM assetfinder_scans WHERE scan_id = $1`
	err := dbPool.QueryRow(context.Background(), query, scanID).Scan(
		&scan.ID,
		&scan.ScanID,
		&scan.Domain,
		&scan.Status,
		&scan.Result,
		&scan.Error,
		&scan.StdOut,
		&scan.StdErr,
		&scan.Command,
		&scan.ExecTime,
		&scan.CreatedAt,
		&scan.ScopeTargetID,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "Scan not found", http.StatusNotFound)
		} else {
			log.Printf("[ERROR] Failed to get scan status: %v", err)
			http.Error(w, "Failed to get scan status", http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"id":              scan.ID,
		"scan_id":         scan.ScanID,
		"domain":          scan.Domain,
		"status":          scan.Status,
		"result":          nullStringToString(scan.Result),
		"error":           nullStringToString(scan.Error),
		"stdout":          nullStringToString(scan.StdOut),
		"stderr":          nullStringToString(scan.StdErr),
		"command":         nullStringToString(scan.Command),
		"execution_time":  nullStringToString(scan.ExecTime),
		"created_at":      scan.CreatedAt.Format(time.RFC3339),
		"scope_target_id": scan.ScopeTargetID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getAssetfinderScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]

	if scopeTargetID == "" {
		log.Printf("[ERROR] No scope target ID provided")
		http.Error(w, "No scope target ID provided", http.StatusBadRequest)
		return
	}

	query := `SELECT * FROM assetfinder_scans WHERE scope_target_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to get scans: %v", err)
		http.Error(w, "Failed to get scans", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []map[string]interface{}
	for rows.Next() {
		var scan AssetfinderScanStatus
		err := rows.Scan(
			&scan.ID,
			&scan.ScanID,
			&scan.Domain,
			&scan.Status,
			&scan.Result,
			&scan.Error,
			&scan.StdOut,
			&scan.StdErr,
			&scan.Command,
			&scan.ExecTime,
			&scan.CreatedAt,
			&scan.ScopeTargetID,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			continue
		}

		scans = append(scans, map[string]interface{}{
			"id":              scan.ID,
			"scan_id":         scan.ScanID,
			"domain":          scan.Domain,
			"status":          scan.Status,
			"result":          nullStringToString(scan.Result),
			"error":           nullStringToString(scan.Error),
			"stdout":          nullStringToString(scan.StdOut),
			"stderr":          nullStringToString(scan.StdErr),
			"command":         nullStringToString(scan.Command),
			"execution_time":  nullStringToString(scan.ExecTime),
			"created_at":      scan.CreatedAt.Format(time.RFC3339),
			"scope_target_id": scan.ScopeTargetID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scans)
}

func runCTLScan(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		FQDN string `json:"fqdn" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.FQDN == "" {
		http.Error(w, "Invalid request body. `fqdn` is required.", http.StatusBadRequest)
		return
	}

	domain := payload.FQDN
	wildcardDomain := fmt.Sprintf("*.%s", domain)

	query := `SELECT id FROM scope_targets WHERE type = 'Wildcard' AND scope_target = $1`
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(), query, wildcardDomain).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] No matching wildcard scope target found for domain %s", domain)
		http.Error(w, "No matching wildcard scope target found.", http.StatusBadRequest)
		return
	}

	scanID := uuid.New().String()
	insertQuery := `INSERT INTO ctl_scans (scan_id, domain, status, scope_target_id) VALUES ($1, $2, $3, $4)`
	_, err = dbPool.Exec(context.Background(), insertQuery, scanID, domain, "pending", scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to create scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}

	go executeAndParseCTLScan(scanID, domain)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func executeAndParseCTLScan(scanID, domain string) {
	log.Printf("[INFO] Starting CTL scan for domain %s (scan ID: %s)", domain, scanID)
	startTime := time.Now()

	// Use HTTPS API instead of PostgreSQL
	url := fmt.Sprintf("https://crt.sh/?q=%%.%s&output=json", domain)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("[ERROR] CTL scan failed for %s: %v", domain, err)
		updateCTLScanStatus(scanID, "error", "", err.Error(), url, time.Since(startTime).String())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("HTTP request failed with status code: %d", resp.StatusCode)
		log.Printf("[ERROR] CTL scan failed for %s: %v", domain, err)
		updateCTLScanStatus(scanID, "error", "", err.Error(), url, time.Since(startTime).String())
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ERROR] Failed to read response body: %v", err)
		updateCTLScanStatus(scanID, "error", "", err.Error(), url, time.Since(startTime).String())
		return
	}

	type CertEntry struct {
		NameValue string `json:"name_value"`
	}

	var entries []CertEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		log.Printf("[ERROR] Failed to parse JSON response: %v", err)
		updateCTLScanStatus(scanID, "error", "", err.Error(), url, time.Since(startTime).String())
		return
	}

	// Process and clean the results
	var subdomains []string
	seen := make(map[string]bool)
	for _, entry := range entries {
		// Split on newlines as some entries contain multiple domains
		for _, line := range strings.Split(entry.NameValue, "\n") {
			// Clean up the subdomain
			subdomain := strings.TrimSpace(line)
			subdomain = strings.TrimPrefix(subdomain, "*.")
			subdomain = strings.ToLower(subdomain)

			// Only include subdomains of our target domain
			if strings.HasSuffix(subdomain, domain) && !seen[subdomain] {
				seen[subdomain] = true
				subdomains = append(subdomains, subdomain)
			}
		}
	}

	// Sort the results
	sort.Strings(subdomains)
	result := strings.Join(subdomains, "\n")

	execTime := time.Since(startTime).String()
	log.Printf("[INFO] CTL scan completed in %s for domain %s", execTime, domain)
	log.Printf("[DEBUG] Raw output length: %d bytes", len(result))

	if result == "" {
		log.Printf("[WARN] No output from CTL scan")
		updateCTLScanStatus(scanID, "completed", "", "No results found", url, execTime)
	} else {
		log.Printf("[DEBUG] CTL output: %s", result)
		updateCTLScanStatus(scanID, "success", result, "", url, execTime)
	}

	log.Printf("[INFO] Scan status updated for scan %s", scanID)
}

func updateCTLScanStatus(scanID, status, result, stderr, command, execTime string) {
	log.Printf("[INFO] Updating CTL scan status for %s to %s", scanID, status)
	query := `UPDATE ctl_scans SET status = $1, result = $2, stderr = $3, command = $4, execution_time = $5 WHERE scan_id = $6`
	_, err := dbPool.Exec(context.Background(), query, status, result, stderr, command, execTime, scanID)
	if err != nil {
		log.Printf("[ERROR] Failed to update CTL scan status for %s: %v", scanID, err)
	} else {
		log.Printf("[INFO] Successfully updated CTL scan status for %s", scanID)
	}
}

func getCTLScanStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scanID := vars["scan_id"]

	var scan CTLScanStatus
	query := `SELECT * FROM ctl_scans WHERE scan_id = $1`
	err := dbPool.QueryRow(context.Background(), query, scanID).Scan(
		&scan.ID,
		&scan.ScanID,
		&scan.Domain,
		&scan.Status,
		&scan.Result,
		&scan.Error,
		&scan.StdOut,
		&scan.StdErr,
		&scan.Command,
		&scan.ExecTime,
		&scan.CreatedAt,
		&scan.ScopeTargetID,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "Scan not found", http.StatusNotFound)
		} else {
			log.Printf("[ERROR] Failed to get scan status: %v", err)
			http.Error(w, "Failed to get scan status", http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"id":              scan.ID,
		"scan_id":         scan.ScanID,
		"domain":          scan.Domain,
		"status":          scan.Status,
		"result":          nullStringToString(scan.Result),
		"error":           nullStringToString(scan.Error),
		"stdout":          nullStringToString(scan.StdOut),
		"stderr":          nullStringToString(scan.StdErr),
		"command":         nullStringToString(scan.Command),
		"execution_time":  nullStringToString(scan.ExecTime),
		"created_at":      scan.CreatedAt.Format(time.RFC3339),
		"scope_target_id": scan.ScopeTargetID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getCTLScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]

	if scopeTargetID == "" {
		log.Printf("[ERROR] No scope target ID provided")
		http.Error(w, "No scope target ID provided", http.StatusBadRequest)
		return
	}

	query := `SELECT * FROM ctl_scans WHERE scope_target_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to get scans: %v", err)
		http.Error(w, "Failed to get scans", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []map[string]interface{}
	for rows.Next() {
		var scan CTLScanStatus
		err := rows.Scan(
			&scan.ID,
			&scan.ScanID,
			&scan.Domain,
			&scan.Status,
			&scan.Result,
			&scan.Error,
			&scan.StdOut,
			&scan.StdErr,
			&scan.Command,
			&scan.ExecTime,
			&scan.CreatedAt,
			&scan.ScopeTargetID,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			continue
		}

		scans = append(scans, map[string]interface{}{
			"id":              scan.ID,
			"scan_id":         scan.ScanID,
			"domain":          scan.Domain,
			"status":          scan.Status,
			"result":          nullStringToString(scan.Result),
			"error":           nullStringToString(scan.Error),
			"stdout":          nullStringToString(scan.StdOut),
			"stderr":          nullStringToString(scan.StdErr),
			"command":         nullStringToString(scan.Command),
			"execution_time":  nullStringToString(scan.ExecTime),
			"created_at":      scan.CreatedAt.Format(time.RFC3339),
			"scope_target_id": scan.ScopeTargetID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scans)
}

func runSubfinderScan(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		FQDN string `json:"fqdn" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.FQDN == "" {
		http.Error(w, "Invalid request body. `fqdn` is required.", http.StatusBadRequest)
		return
	}

	domain := payload.FQDN
	wildcardDomain := fmt.Sprintf("*.%s", domain)

	query := `SELECT id FROM scope_targets WHERE type = 'Wildcard' AND scope_target = $1`
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(), query, wildcardDomain).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] No matching wildcard scope target found for domain %s", domain)
		http.Error(w, "No matching wildcard scope target found.", http.StatusBadRequest)
		return
	}

	scanID := uuid.New().String()
	insertQuery := `INSERT INTO subfinder_scans (scan_id, domain, status, scope_target_id) VALUES ($1, $2, $3, $4)`
	_, err = dbPool.Exec(context.Background(), insertQuery, scanID, domain, "pending", scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to create scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}

	go executeAndParseSubfinderScan(scanID, domain)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func executeAndParseSubfinderScan(scanID, domain string) {
	log.Printf("[INFO] Starting Subfinder scan for domain %s (scan ID: %s)", domain, scanID)
	startTime := time.Now()

	cmd := exec.Command(
		"docker", "exec",
		"ars0n-framework-v2-subfinder-1",
		"subfinder",
		"-d", domain,
		"-silent",
	)

	log.Printf("[INFO] Executing command: %s", cmd.String())

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	execTime := time.Since(startTime).String()

	if err != nil {
		log.Printf("[ERROR] Subfinder scan failed for %s: %v", domain, err)
		log.Printf("[ERROR] stderr output: %s", stderr.String())
		updateSubfinderScanStatus(scanID, "error", "", stderr.String(), cmd.String(), execTime)
		return
	}

	result := stdout.String()
	log.Printf("[INFO] Subfinder scan completed in %s for domain %s", execTime, domain)
	log.Printf("[DEBUG] Raw output length: %d bytes", len(result))

	if result == "" {
		log.Printf("[WARN] No output from Subfinder scan")
		updateSubfinderScanStatus(scanID, "completed", "", "No results found", cmd.String(), execTime)
	} else {
		log.Printf("[DEBUG] Subfinder output: %s", result)
		updateSubfinderScanStatus(scanID, "success", result, stderr.String(), cmd.String(), execTime)
	}

	log.Printf("[INFO] Scan status updated for scan %s", scanID)
}

func updateSubfinderScanStatus(scanID, status, result, stderr, command, execTime string) {
	log.Printf("[INFO] Updating Subfinder scan status for %s to %s", scanID, status)
	query := `UPDATE subfinder_scans SET status = $1, result = $2, stderr = $3, command = $4, execution_time = $5 WHERE scan_id = $6`
	_, err := dbPool.Exec(context.Background(), query, status, result, stderr, command, execTime, scanID)
	if err != nil {
		log.Printf("[ERROR] Failed to update Subfinder scan status for %s: %v", scanID, err)
	} else {
		log.Printf("[INFO] Successfully updated Subfinder scan status for %s", scanID)
	}
}

func getSubfinderScanStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scanID := vars["scan_id"]

	var scan SubfinderScanStatus
	query := `SELECT * FROM subfinder_scans WHERE scan_id = $1`
	err := dbPool.QueryRow(context.Background(), query, scanID).Scan(
		&scan.ID,
		&scan.ScanID,
		&scan.Domain,
		&scan.Status,
		&scan.Result,
		&scan.Error,
		&scan.StdOut,
		&scan.StdErr,
		&scan.Command,
		&scan.ExecTime,
		&scan.CreatedAt,
		&scan.ScopeTargetID,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "Scan not found", http.StatusNotFound)
		} else {
			log.Printf("[ERROR] Failed to get scan status: %v", err)
			http.Error(w, "Failed to get scan status", http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"id":              scan.ID,
		"scan_id":         scan.ScanID,
		"domain":          scan.Domain,
		"status":          scan.Status,
		"result":          nullStringToString(scan.Result),
		"error":           nullStringToString(scan.Error),
		"stdout":          nullStringToString(scan.StdOut),
		"stderr":          nullStringToString(scan.StdErr),
		"command":         nullStringToString(scan.Command),
		"execution_time":  nullStringToString(scan.ExecTime),
		"created_at":      scan.CreatedAt.Format(time.RFC3339),
		"scope_target_id": scan.ScopeTargetID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getSubfinderScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]

	if scopeTargetID == "" {
		log.Printf("[ERROR] No scope target ID provided")
		http.Error(w, "No scope target ID provided", http.StatusBadRequest)
		return
	}

	query := `SELECT * FROM subfinder_scans WHERE scope_target_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to get scans: %v", err)
		http.Error(w, "Failed to get scans", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []map[string]interface{}
	for rows.Next() {
		var scan SubfinderScanStatus
		err := rows.Scan(
			&scan.ID,
			&scan.ScanID,
			&scan.Domain,
			&scan.Status,
			&scan.Result,
			&scan.Error,
			&scan.StdOut,
			&scan.StdErr,
			&scan.Command,
			&scan.ExecTime,
			&scan.CreatedAt,
			&scan.ScopeTargetID,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			continue
		}

		scans = append(scans, map[string]interface{}{
			"id":              scan.ID,
			"scan_id":         scan.ScanID,
			"domain":          scan.Domain,
			"status":          scan.Status,
			"result":          nullStringToString(scan.Result),
			"error":           nullStringToString(scan.Error),
			"stdout":          nullStringToString(scan.StdOut),
			"stderr":          nullStringToString(scan.StdErr),
			"command":         nullStringToString(scan.Command),
			"execution_time":  nullStringToString(scan.ExecTime),
			"created_at":      scan.CreatedAt.Format(time.RFC3339),
			"scope_target_id": scan.ScopeTargetID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scans)
}

func consolidateSubdomains(scopeTargetID string) ([]string, error) {
	log.Printf("[INFO] Starting consolidation for scope target ID: %s", scopeTargetID)

	var baseDomain string
	err := dbPool.QueryRow(context.Background(), `
		SELECT TRIM(LEADING '*.' FROM scope_target) 
			FROM scope_targets 
			WHERE id = $1`, scopeTargetID).Scan(&baseDomain)
	if err != nil {
		log.Printf("[ERROR] Failed to get base domain: %v", err)
		return nil, fmt.Errorf("failed to get base domain: %v", err)
	}
	log.Printf("[INFO] Base domain for consolidation: %s", baseDomain)

	uniqueSubdomains := make(map[string]bool)
	toolResults := make(map[string]int)

	// Special handling for Amass - get from subdomains table
	amassQuery := `
		SELECT s.subdomain 
		FROM subdomains s 
		JOIN amass_scans a ON s.scan_id = a.scan_id 
		WHERE a.scope_target_id = $1 
			AND a.status = 'success'
			AND a.created_at = (
				SELECT MAX(created_at) 
				FROM amass_scans 
				WHERE scope_target_id = $1 
					AND status = 'success'
			)`

	log.Printf("[DEBUG] Processing results from amass using subdomains table")
	amassRows, err := dbPool.Query(context.Background(), amassQuery, scopeTargetID)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("[ERROR] Failed to get Amass subdomains: %v", err)
	} else {
		count := 0
		for amassRows.Next() {
			var subdomain string
			if err := amassRows.Scan(&subdomain); err != nil {
				log.Printf("[ERROR] Failed to scan Amass subdomain: %v", err)
				continue
			}
			if strings.HasSuffix(subdomain, baseDomain) {
				if !uniqueSubdomains[subdomain] {
					log.Printf("[DEBUG] Found new subdomain from amass: %s", subdomain)
					count++
				}
				uniqueSubdomains[subdomain] = true
			}
		}
		amassRows.Close()
		toolResults["amass"] = count
		log.Printf("[INFO] Found %d new unique subdomains from amass", count)
	}

	// Handle other tools
	queries := []struct {
		query string
		table string
	}{
		{
			query: `
				SELECT result 
				FROM sublist3r_scans 
				WHERE scope_target_id = $1 
					AND status = 'completed' 
					AND result IS NOT NULL 
					AND result != '' 
				ORDER BY created_at DESC 
				LIMIT 1`,
			table: "sublist3r",
		},
		{
			query: `
				SELECT result 
				FROM assetfinder_scans 
				WHERE scope_target_id = $1 
					AND status = 'success' 
					AND result IS NOT NULL 
					AND result != '' 
				ORDER BY created_at DESC 
				LIMIT 1`,
			table: "assetfinder",
		},
		{
			query: `
				SELECT result 
				FROM ctl_scans 
				WHERE scope_target_id = $1 
					AND status = 'success' 
					AND result IS NOT NULL 
					AND result != '' 
				ORDER BY created_at DESC 
				LIMIT 1`,
			table: "ctl",
		},
		{
			query: `
				SELECT result 
				FROM subfinder_scans 
				WHERE scope_target_id = $1 
					AND status = 'success' 
					AND result IS NOT NULL 
					AND result != '' 
				ORDER BY created_at DESC 
				LIMIT 1`,
			table: "subfinder",
		},
		{
			query: `
				SELECT result 
				FROM gau_scans 
				WHERE scope_target_id = $1 
					AND status = 'success' 
					AND result IS NOT NULL 
					AND result != '' 
				ORDER BY created_at DESC 
				LIMIT 1`,
			table: "gau",
		},
		{
			query: `
				SELECT result 
				FROM shuffledns_scans 
				WHERE scope_target_id = $1 
					AND status = 'success' 
					AND result IS NOT NULL 
					AND result != '' 
				ORDER BY created_at DESC 
				LIMIT 1`,
			table: "shuffledns",
		},
		{
			query: `
				SELECT result 
				FROM shufflednscustom_scans 
				WHERE scope_target_id = $1 
					AND status = 'success' 
					AND result IS NOT NULL 
					AND result != '' 
				ORDER BY created_at DESC 
				LIMIT 1`,
			table: "shuffledns_custom",
		},
	}

	for _, q := range queries {
		log.Printf("[DEBUG] Processing results from %s", q.table)
		var result sql.NullString
		err := dbPool.QueryRow(context.Background(), q.query, scopeTargetID).Scan(&result)
		if err != nil {
			if err == pgx.ErrNoRows {
				log.Printf("[DEBUG] No results found for %s", q.table)
				continue
			}
			log.Printf("[ERROR] Failed to get results from %s: %v", q.table, err)
			continue
		}

		if !result.Valid || result.String == "" {
			log.Printf("[DEBUG] No valid results found for %s", q.table)
			continue
		}

		count := 0
		if q.table == "gau" {
			lines := strings.Split(result.String, "\n")
			log.Printf("[DEBUG] Processing %d lines from GAU", len(lines))
			for i, line := range lines {
				if line == "" {
					continue
				}
				var gauResult struct {
					URL string `json:"url"`
				}
				if err := json.Unmarshal([]byte(line), &gauResult); err != nil {
					log.Printf("[ERROR] Failed to parse GAU result line %d: %v", i, err)
					continue
				}
				if gauResult.URL == "" {
					continue
				}
				parsedURL, err := url.Parse(gauResult.URL)
				if err != nil {
					log.Printf("[ERROR] Failed to parse URL %s: %v", gauResult.URL, err)
					continue
				}
				hostname := parsedURL.Hostname()
				if strings.HasSuffix(hostname, baseDomain) {
					if !uniqueSubdomains[hostname] {
						log.Printf("[DEBUG] Found new subdomain from GAU: %s", hostname)
						count++
					}
					uniqueSubdomains[hostname] = true
				}
			}
		} else {
			lines := strings.Split(result.String, "\n")
			log.Printf("[DEBUG] Processing %d lines from %s", len(lines), q.table)
			for _, line := range lines {
				subdomain := strings.TrimSpace(line)
				if subdomain == "" {
					continue
				}
				if strings.HasSuffix(subdomain, baseDomain) {
					if !uniqueSubdomains[subdomain] {
						log.Printf("[DEBUG] Found new subdomain from %s: %s", q.table, subdomain)
						count++
					}
					uniqueSubdomains[subdomain] = true
				}
			}
		}
		toolResults[q.table] = count
		log.Printf("[INFO] Found %d new unique subdomains from %s", count, q.table)
	}

	var consolidatedSubdomains []string
	for subdomain := range uniqueSubdomains {
		consolidatedSubdomains = append(consolidatedSubdomains, subdomain)
	}
	sort.Strings(consolidatedSubdomains)

	log.Printf("[INFO] Tool contribution breakdown:")
	for tool, count := range toolResults {
		log.Printf("- %s: %d subdomains", tool, count)
	}
	log.Printf("[INFO] Total unique subdomains found: %d", len(consolidatedSubdomains))

	tx, err := dbPool.Begin(context.Background())
	if err != nil {
		log.Printf("[ERROR] Failed to begin transaction: %v", err)
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(context.Background())

	_, err = tx.Exec(context.Background(), `DELETE FROM consolidated_subdomains WHERE scope_target_id = $1`, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to delete old consolidated subdomains: %v", err)
		return nil, fmt.Errorf("failed to delete old consolidated subdomains: %v", err)
	}
	log.Printf("[INFO] Cleared old consolidated subdomains")

	for _, subdomain := range consolidatedSubdomains {
		_, err = tx.Exec(context.Background(),
			`INSERT INTO consolidated_subdomains (scope_target_id, subdomain) VALUES ($1, $2)
			ON CONFLICT (scope_target_id, subdomain) DO NOTHING`,
			scopeTargetID, subdomain)
		if err != nil {
			log.Printf("[ERROR] Failed to insert consolidated subdomain %s: %v", subdomain, err)
			return nil, fmt.Errorf("failed to insert consolidated subdomain: %v", err)
		}
	}
	log.Printf("[INFO] Inserted %d consolidated subdomains into database", len(consolidatedSubdomains))

	if err := tx.Commit(context.Background()); err != nil {
		log.Printf("[ERROR] Failed to commit transaction: %v", err)
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}
	log.Printf("[INFO] Successfully completed consolidation")

	return consolidatedSubdomains, nil
}

func handleConsolidateSubdomains(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]
	if scopeTargetID == "" {
		http.Error(w, "Scope target ID is required", http.StatusBadRequest)
		return
	}

	consolidatedSubdomains, err := consolidateSubdomains(scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to consolidate subdomains: %v", err)
		http.Error(w, "Failed to consolidate subdomains", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":      len(consolidatedSubdomains),
		"subdomains": consolidatedSubdomains,
	})
}

func getConsolidatedSubdomains(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]
	if scopeTargetID == "" {
		http.Error(w, "Scope target ID is required", http.StatusBadRequest)
		return
	}

	rows, err := dbPool.Query(context.Background(),
		`SELECT subdomain FROM consolidated_subdomains 
		WHERE scope_target_id = $1 
		ORDER BY subdomain ASC`, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to get consolidated subdomains: %v", err)
		http.Error(w, "Failed to get consolidated subdomains", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var subdomains []string
	for rows.Next() {
		var subdomain string
		if err := rows.Scan(&subdomain); err != nil {
			log.Printf("[ERROR] Failed to scan subdomain row: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		subdomains = append(subdomains, subdomain)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":      len(subdomains),
		"subdomains": subdomains,
	})
}

func runShuffleDNSScan(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		FQDN string `json:"fqdn" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.FQDN == "" {
		http.Error(w, "Invalid request body. `fqdn` is required.", http.StatusBadRequest)
		return
	}

	domain := payload.FQDN
	wildcardDomain := fmt.Sprintf("*.%s", domain)

	query := `SELECT id FROM scope_targets WHERE type = 'Wildcard' AND scope_target = $1`
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(), query, wildcardDomain).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] No matching wildcard scope target found for domain %s", domain)
		http.Error(w, "No matching wildcard scope target found.", http.StatusBadRequest)
		return
	}

	scanID := uuid.New().String()
	insertQuery := `INSERT INTO shuffledns_scans (scan_id, domain, status, scope_target_id) VALUES ($1, $2, $3, $4)`
	_, err = dbPool.Exec(context.Background(), insertQuery, scanID, domain, "pending", scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to create scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}

	go executeAndParseShuffleDNSScan(scanID, domain)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func executeAndParseShuffleDNSScan(scanID, domain string) {
	log.Printf("[INFO] Starting ShuffleDNS scan for domain %s (scan ID: %s)", domain, scanID)
	startTime := time.Now()

	// Create temporary directory for wordlist and resolvers
	tempDir := "/tmp/shuffledns-temp"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Printf("[ERROR] Failed to create temp directory: %v", err)
		updateShuffleDNSScanStatus(scanID, "error", "", fmt.Sprintf("Failed to create temp directory: %v", err), "", time.Since(startTime).String())
		return
	}
	defer os.RemoveAll(tempDir)

	// Write domain to a temporary file
	domainFile := filepath.Join(tempDir, "domain.txt")
	if err := os.WriteFile(domainFile, []byte(domain), 0644); err != nil {
		log.Printf("[ERROR] Failed to write domain file: %v", err)
		updateShuffleDNSScanStatus(scanID, "error", "", fmt.Sprintf("Failed to write domain file: %v", err), "", time.Since(startTime).String())
		return
	}

	cmd := exec.Command(
		"docker", "exec",
		"ars0n-framework-v2-shuffledns-1",
		"shuffledns",
		"-d", domain,
		"-w", "/app/wordlists/all.txt",
		"-r", "/app/wordlists/resolvers.txt",
		"-silent",
		"-massdns", "/usr/local/bin/massdns",
		"-mode", "bruteforce",
	)

	log.Printf("[INFO] Executing command: %s", cmd.String())

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	execTime := time.Since(startTime).String()

	if err != nil {
		log.Printf("[ERROR] ShuffleDNS scan failed for %s: %v", domain, err)
		log.Printf("[ERROR] stderr output: %s", stderr.String())
		updateShuffleDNSScanStatus(scanID, "error", "", stderr.String(), cmd.String(), execTime)
		return
	}

	result := stdout.String()
	log.Printf("[INFO] ShuffleDNS scan completed in %s for domain %s", execTime, domain)
	log.Printf("[DEBUG] Raw output length: %d bytes", len(result))

	if result == "" {
		log.Printf("[WARN] No output from ShuffleDNS scan")
		updateShuffleDNSScanStatus(scanID, "completed", "", "No results found", cmd.String(), execTime)
	} else {
		log.Printf("[DEBUG] ShuffleDNS output: %s", result)
		updateShuffleDNSScanStatus(scanID, "success", result, stderr.String(), cmd.String(), execTime)
	}

	log.Printf("[INFO] Scan status updated for scan %s", scanID)
}

func updateShuffleDNSScanStatus(scanID, status, result, stderr, command, execTime string) {
	log.Printf("[INFO] Updating ShuffleDNS scan status for %s to %s", scanID, status)
	query := `UPDATE shuffledns_scans SET status = $1, result = $2, stderr = $3, command = $4, execution_time = $5 WHERE scan_id = $6`
	_, err := dbPool.Exec(context.Background(), query, status, result, stderr, command, execTime, scanID)
	if err != nil {
		log.Printf("[ERROR] Failed to update ShuffleDNS scan status for %s: %v", scanID, err)
	} else {
		log.Printf("[INFO] Successfully updated ShuffleDNS scan status for %s", scanID)
	}
}

func getShuffleDNSScanStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scanID := vars["scan_id"]

	var scan ShuffleDNSScanStatus
	query := `SELECT * FROM shuffledns_scans WHERE scan_id = $1`
	err := dbPool.QueryRow(context.Background(), query, scanID).Scan(
		&scan.ID,
		&scan.ScanID,
		&scan.Domain,
		&scan.Status,
		&scan.Result,
		&scan.Error,
		&scan.StdOut,
		&scan.StdErr,
		&scan.Command,
		&scan.ExecTime,
		&scan.CreatedAt,
		&scan.ScopeTargetID,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "Scan not found", http.StatusNotFound)
		} else {
			log.Printf("[ERROR] Failed to get scan status: %v", err)
			http.Error(w, "Failed to get scan status", http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"id":              scan.ID,
		"scan_id":         scan.ScanID,
		"domain":          scan.Domain,
		"status":          scan.Status,
		"result":          nullStringToString(scan.Result),
		"error":           nullStringToString(scan.Error),
		"stdout":          nullStringToString(scan.StdOut),
		"stderr":          nullStringToString(scan.StdErr),
		"command":         nullStringToString(scan.Command),
		"execution_time":  nullStringToString(scan.ExecTime),
		"created_at":      scan.CreatedAt.Format(time.RFC3339),
		"scope_target_id": scan.ScopeTargetID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getShuffleDNSScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]

	if scopeTargetID == "" {
		log.Printf("[ERROR] No scope target ID provided")
		http.Error(w, "No scope target ID provided", http.StatusBadRequest)
		return
	}

	query := `SELECT * FROM shuffledns_scans WHERE scope_target_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to get scans: %v", err)
		http.Error(w, "Failed to get scans", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []map[string]interface{}
	for rows.Next() {
		var scan ShuffleDNSScanStatus
		err := rows.Scan(
			&scan.ID,
			&scan.ScanID,
			&scan.Domain,
			&scan.Status,
			&scan.Result,
			&scan.Error,
			&scan.StdOut,
			&scan.StdErr,
			&scan.Command,
			&scan.ExecTime,
			&scan.CreatedAt,
			&scan.ScopeTargetID,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			continue
		}

		scans = append(scans, map[string]interface{}{
			"id":              scan.ID,
			"scan_id":         scan.ScanID,
			"domain":          scan.Domain,
			"status":          scan.Status,
			"result":          nullStringToString(scan.Result),
			"error":           nullStringToString(scan.Error),
			"stdout":          nullStringToString(scan.StdOut),
			"stderr":          nullStringToString(scan.StdErr),
			"command":         nullStringToString(scan.Command),
			"execution_time":  nullStringToString(scan.ExecTime),
			"created_at":      scan.CreatedAt.Format(time.RFC3339),
			"scope_target_id": scan.ScopeTargetID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scans)
}

func runCeWLScan(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		FQDN string `json:"fqdn" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.FQDN == "" {
		http.Error(w, "Invalid request body. `fqdn` is required.", http.StatusBadRequest)
		return
	}

	domain := payload.FQDN
	wildcardDomain := fmt.Sprintf("*.%s", domain)

	// Get the scope target ID
	query := `SELECT id FROM scope_targets WHERE type = 'Wildcard' AND scope_target = $1`
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(), query, wildcardDomain).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] No matching wildcard scope target found for domain %s", domain)
		http.Error(w, "No matching wildcard scope target found.", http.StatusBadRequest)
		return
	}

	scanID := uuid.New().String()
	insertQuery := `INSERT INTO cewl_scans (scan_id, url, status, scope_target_id) VALUES ($1, $2, $3, $4)`
	_, err = dbPool.Exec(context.Background(), insertQuery, scanID, domain, "pending", scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to create scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}

	go executeAndParseCeWLScan(scanID, domain)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func executeAndParseCeWLScan(scanID, domain string) {
	log.Printf("[DEBUG] ====== Starting CeWL + ShuffleDNS Process ======")
	log.Printf("[DEBUG] ScanID: %s, Domain: %s", scanID, domain)
	startTime := time.Now()

	// First, get all live web servers from the latest httpx scan
	var httpxResults string
	err := dbPool.QueryRow(context.Background(), `
		SELECT result FROM httpx_scans 
		WHERE scope_target_id = (
			SELECT scope_target_id FROM cewl_scans WHERE scan_id = $1
		)
		AND status = 'success'
		ORDER BY created_at DESC 
		LIMIT 1`, scanID).Scan(&httpxResults)

	if err != nil {
		log.Printf("[ERROR] Failed to get httpx results: %v", err)
		updateCeWLScanStatus(scanID, "error", "", "Failed to get httpx results", "", time.Since(startTime).String())
		return
	}

	log.Printf("[DEBUG] Found httpx results length: %d bytes", len(httpxResults))

	// Process each live web server
	urls := strings.Split(httpxResults, "\n")
	log.Printf("[DEBUG] Processing %d URLs from httpx results", len(urls))

	// Create temporary directory for wordlist
	tempDir := "/tmp/cewl-temp"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Printf("[ERROR] Failed to create temp directory: %v", err)
		updateCeWLScanStatus(scanID, "error", "", fmt.Sprintf("Failed to create temp directory: %v", err), "", time.Since(startTime).String())
		return
	}
	defer os.RemoveAll(tempDir)

	// Create temporary file for combined wordlist
	wordlistFile := filepath.Join(tempDir, "combined-wordlist.txt")
	wordSet := make(map[string]bool)

	for _, urlLine := range urls {
		if urlLine == "" {
			continue
		}

		// Parse JSON from httpx output
		var httpxResult struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal([]byte(urlLine), &httpxResult); err != nil {
			log.Printf("[WARN] Failed to parse httpx result line: %v, Line: %s", err, urlLine)
			continue
		}

		if httpxResult.URL == "" {
			continue
		}

		log.Printf("[DEBUG] Running CeWL against URL: %s", httpxResult.URL)

		// Run CeWL against each URL
		cmd := exec.Command(
			"docker", "exec",
			"ars0n-framework-v2-cewl-1",
			"ruby", "/app/cewl.rb",
			httpxResult.URL,
			"-d", "2",
			"-m", "5",
			"-c",
			"--with-numbers",
		)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err = cmd.Run()
		if err != nil {
			log.Printf("[WARN] CeWL scan failed for %s: %v", httpxResult.URL, err)
			log.Printf("[DEBUG] CeWL stderr: %s", stderr.String())
			continue
		}

		log.Printf("[DEBUG] CeWL stdout length for %s: %d bytes", httpxResult.URL, len(stdout.String()))

		// Process CeWL output and add unique words to set
		lines := strings.Split(stdout.String(), "\n")
		log.Printf("[DEBUG] Processing %d lines from CeWL output for %s", len(lines), httpxResult.URL)

		for _, line := range lines[1:] {
			line = strings.TrimSpace(line)
			if line != "" && !strings.Contains(line, "CeWL") && !strings.Contains(line, "Robin Wood") {
				if parts := strings.Split(line, ","); len(parts) > 1 {
					word := strings.TrimSpace(parts[0])
					wordSet[word] = true
				}
			}
		}
	}

	log.Printf("[DEBUG] Total unique words found: %d", len(wordSet))

	// Convert wordset to slice and write to file
	var wordlist []string
	for word := range wordSet {
		wordlist = append(wordlist, word)
	}
	sort.Strings(wordlist)

	// Debug: Print first few words
	if len(wordlist) > 0 {
		previewSize := 10
		if len(wordlist) < previewSize {
			previewSize = len(wordlist)
		}
		log.Printf("[DEBUG] First %d words: %v", previewSize, wordlist[:previewSize])
	}

	if err := os.WriteFile(wordlistFile, []byte(strings.Join(wordlist, "\n")), 0644); err != nil {
		log.Printf("[ERROR] Failed to write combined wordlist: %v", err)
		updateCeWLScanStatus(scanID, "error", "", fmt.Sprintf("Failed to write wordlist: %v", err), "", time.Since(startTime).String())
		return
	}

	log.Printf("[DEBUG] Wordlist file written to: %s", wordlistFile)

	// Debug: Check wordlist file content
	if content, err := os.ReadFile(wordlistFile); err == nil {
		log.Printf("[DEBUG] Wordlist file size: %d bytes", len(content))
	}

	// Copy wordlist to container
	copyCmd := exec.Command(
		"docker", "cp",
		wordlistFile,
		"ars0n-framework-v2-shuffledns-1:/tmp/wordlist.txt")
	if err := copyCmd.Run(); err != nil {
		log.Printf("[ERROR] Failed to copy wordlist to container: %v", err)
		updateCeWLScanStatus(scanID, "error", "", fmt.Sprintf("Failed to copy wordlist to container: %v", err), "", time.Since(startTime).String())
		return
	}

	log.Printf("[DEBUG] Wordlist copied to ShuffleDNS container")

	// Verify file in container
	checkCmd := exec.Command(
		"docker", "exec",
		"ars0n-framework-v2-shuffledns-1",
		"cat", "/tmp/wordlist.txt",
	)
	var checkOutput bytes.Buffer
	checkCmd.Stdout = &checkOutput
	if err := checkCmd.Run(); err == nil {
		log.Printf("[DEBUG] Wordlist in container size: %d bytes", len(checkOutput.String()))
	}

	// Store the wordlist in CeWL results
	updateCeWLScanStatus(scanID, "success", strings.Join(wordlist, "\n"), "", "", time.Since(startTime).String())

	// Start ShuffleDNS custom scan
	shuffleDNSScanID := uuid.New().String()
	log.Printf("[DEBUG] Starting ShuffleDNS custom scan with ID: %s", shuffleDNSScanID)

	// Get scope target ID
	var scopeTargetID string
	err = dbPool.QueryRow(context.Background(),
		`SELECT scope_target_id FROM cewl_scans WHERE scan_id = $1`,
		scanID).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to get scope target ID: %v", err)
		return
	}

	log.Printf("[DEBUG] Found scope target ID: %s", scopeTargetID)

	// Insert ShuffleDNS custom scan record
	_, err = dbPool.Exec(context.Background(),
		`INSERT INTO shufflednscustom_scans (scan_id, domain, status, scope_target_id) VALUES ($1, $2, $3, $4)`,
		shuffleDNSScanID, domain, "pending", scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to create ShuffleDNS custom scan record: %v", err)
		return
	}

	// Debug: Check resolvers file
	resolversCmd := exec.Command(
		"docker", "exec",
		"ars0n-framework-v2-shuffledns-1",
		"cat", "/app/wordlists/resolvers.txt",
	)
	var resolversOutput bytes.Buffer
	resolversCmd.Stdout = &resolversOutput
	if err := resolversCmd.Run(); err == nil {
		log.Printf("[DEBUG] Resolvers file size: %d bytes", len(resolversOutput.String()))
	} else {
		log.Printf("[ERROR] Failed to read resolvers file: %v", err)
	}

	// Run ShuffleDNS with the combined wordlist
	shuffleCmd := exec.Command(
		"docker", "exec",
		"ars0n-framework-v2-shuffledns-1",
		"shuffledns",
		"-d", domain,
		"-w", "/tmp/wordlist.txt",
		"-r", "/app/wordlists/resolvers.txt",
		"-silent",
		"-massdns", "/usr/local/bin/massdns",
		"-mode", "bruteforce",
	)

	var shuffleStdout, shuffleStderr bytes.Buffer
	shuffleCmd.Stdout = &shuffleStdout
	shuffleCmd.Stderr = &shuffleStderr

	log.Printf("[DEBUG] Running ShuffleDNS command: %s", shuffleCmd.String())
	err = shuffleCmd.Run()
	shuffleExecTime := time.Since(startTime).String()

	if err != nil {
		log.Printf("[ERROR] ShuffleDNS custom scan failed: %v", err)
		log.Printf("[DEBUG] ShuffleDNS stderr: %s", shuffleStderr.String())
		log.Printf("[DEBUG] ShuffleDNS stdout: %s", shuffleStdout.String())
		updateShuffleDNSCustomScanStatus(shuffleDNSScanID, "error", "", shuffleStderr.String(), shuffleCmd.String(), shuffleExecTime)
		return
	}

	shuffleResult := shuffleStdout.String()
	log.Printf("[DEBUG] ShuffleDNS stdout length: %d bytes", len(shuffleResult))
	if len(shuffleResult) > 0 {
		log.Printf("[DEBUG] ShuffleDNS results: %s", shuffleResult)
	}

	if shuffleResult == "" {
		log.Printf("[WARN] No results found from ShuffleDNS scan")
		updateShuffleDNSCustomScanStatus(shuffleDNSScanID, "completed", "", "No results found", shuffleCmd.String(), shuffleExecTime)
	} else {
		log.Printf("[INFO] ShuffleDNS found results")
		updateShuffleDNSCustomScanStatus(shuffleDNSScanID, "success", shuffleResult, shuffleStderr.String(), shuffleCmd.String(), shuffleExecTime)
	}

	log.Printf("[DEBUG] ====== Completed CeWL + ShuffleDNS Process ======")
}

func updateShuffleDNSCustomScanStatus(scanID, status, result, stderr, command, execTime string) {
	log.Printf("[INFO] Updating ShuffleDNS custom scan status for %s to %s", scanID, status)
	query := `UPDATE shufflednscustom_scans SET status = $1, result = $2, stderr = $3, command = $4, execution_time = $5 WHERE scan_id = $6`
	_, err := dbPool.Exec(context.Background(), query, status, result, stderr, command, execTime, scanID)
	if err != nil {
		log.Printf("[ERROR] Failed to update ShuffleDNS custom scan status for %s: %v", scanID, err)
	} else {
		log.Printf("[INFO] Successfully updated ShuffleDNS custom scan status for %s", scanID)
	}
}

func getCeWLScanStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scanID := vars["scan_id"]

	var scan CeWLScanStatus
	query := `SELECT * FROM cewl_scans WHERE scan_id = $1`
	err := dbPool.QueryRow(context.Background(), query, scanID).Scan(
		&scan.ID,
		&scan.ScanID,
		&scan.URL,
		&scan.Status,
		&scan.Result,
		&scan.Error,
		&scan.StdOut,
		&scan.StdErr,
		&scan.Command,
		&scan.ExecTime,
		&scan.CreatedAt,
		&scan.ScopeTargetID,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "Scan not found", http.StatusNotFound)
		} else {
			log.Printf("[ERROR] Failed to get scan status: %v", err)
			http.Error(w, "Failed to get scan status", http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"id":              scan.ID,
		"scan_id":         scan.ScanID,
		"url":             scan.URL,
		"status":          scan.Status,
		"result":          nullStringToString(scan.Result),
		"error":           nullStringToString(scan.Error),
		"stdout":          nullStringToString(scan.StdOut),
		"stderr":          nullStringToString(scan.StdErr),
		"command":         nullStringToString(scan.Command),
		"execution_time":  nullStringToString(scan.ExecTime),
		"created_at":      scan.CreatedAt.Format(time.RFC3339),
		"scope_target_id": scan.ScopeTargetID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getCeWLScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]

	if scopeTargetID == "" {
		log.Printf("[ERROR] No scope target ID provided")
		http.Error(w, "No scope target ID provided", http.StatusBadRequest)
		return
	}

	query := `SELECT * FROM cewl_scans WHERE scope_target_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to get scans: %v", err)
		http.Error(w, "Failed to get scans", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []map[string]interface{}
	for rows.Next() {
		var scan CeWLScanStatus
		err := rows.Scan(
			&scan.ID,
			&scan.ScanID,
			&scan.URL,
			&scan.Status,
			&scan.Result,
			&scan.Error,
			&scan.StdOut,
			&scan.StdErr,
			&scan.Command,
			&scan.ExecTime,
			&scan.CreatedAt,
			&scan.ScopeTargetID,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			continue
		}

		scans = append(scans, map[string]interface{}{
			"id":              scan.ID,
			"scan_id":         scan.ScanID,
			"url":             scan.URL,
			"status":          scan.Status,
			"result":          nullStringToString(scan.Result),
			"error":           nullStringToString(scan.Error),
			"stdout":          nullStringToString(scan.StdOut),
			"stderr":          nullStringToString(scan.StdErr),
			"command":         nullStringToString(scan.Command),
			"execution_time":  nullStringToString(scan.ExecTime),
			"created_at":      scan.CreatedAt.Format(time.RFC3339),
			"scope_target_id": scan.ScopeTargetID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scans)
}

func runCeWLScansForUrls(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		URLs []string `json:"urls" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || len(payload.URLs) == 0 {
		http.Error(w, "Invalid request body. `urls` is required and must contain at least one URL.", http.StatusBadRequest)
		return
	}

	scanID := uuid.New().String()
	insertQuery := `INSERT INTO cewl_scans (scan_id, url, status, scope_target_id) VALUES ($1, $2, $3, $4)`
	_, err := dbPool.Exec(context.Background(), insertQuery, scanID, payload.URLs, "pending", nil)
	if err != nil {
		log.Printf("[ERROR] Failed to create scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}

	go executeAndParseCeWLScansForUrls(scanID, payload.URLs)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func executeAndParseCeWLScansForUrls(scanID string, urls []string) {
	log.Printf("[INFO] Starting CeWL scans for URLs (scan ID: %s)", scanID)
	startTime := time.Now()

	for _, url := range urls {
		go executeAndParseCeWLScan(scanID, url)
	}

	execTime := time.Since(startTime).String()
	log.Printf("[INFO] CeWL scans completed in %s", execTime)
}

func runShuffleDNSWithWordlist(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Wordlist string `json:"wordlist" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Wordlist == "" {
		http.Error(w, "Invalid request body. `wordlist` is required.", http.StatusBadRequest)
		return
	}

	scanID := uuid.New().String()
	insertQuery := `INSERT INTO shuffledns_scans (scan_id, domain, status, scope_target_id) VALUES ($1, $2, $3, $4)`
	_, err := dbPool.Exec(context.Background(), insertQuery, scanID, payload.Wordlist, "pending", nil)
	if err != nil {
		log.Printf("[ERROR] Failed to create scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}

	go executeAndParseShuffleDNSWithWordlist(scanID, payload.Wordlist)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func executeAndParseShuffleDNSWithWordlist(scanID, wordlist string) {
	log.Printf("[INFO] Starting ShuffleDNS scan with wordlist (scan ID: %s)", scanID)
	startTime := time.Now()

	// Create temporary directory for wordlist and resolvers
	tempDir := "/tmp/shuffledns-temp"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Printf("[ERROR] Failed to create temp directory: %v", err)
		updateShuffleDNSScanStatus(scanID, "error", "", fmt.Sprintf("Failed to create temp directory: %v", err), "", time.Since(startTime).String())
		return
	}
	defer os.RemoveAll(tempDir)

	// Write wordlist to a temporary file
	wordlistFile := filepath.Join(tempDir, "wordlist.txt")
	if err := os.WriteFile(wordlistFile, []byte(wordlist), 0644); err != nil {
		log.Printf("[ERROR] Failed to write wordlist file: %v", err)
		updateShuffleDNSScanStatus(scanID, "error", "", fmt.Sprintf("Failed to write wordlist file: %v", err), "", time.Since(startTime).String())
		return
	}

	cmd := exec.Command(
		"docker", "exec",
		"ars0n-framework-v2-shuffledns-1",
		"shuffledns",
		"-d", wordlistFile,
		"-w", "/app/wordlists/all.txt",
		"-r", "/app/wordlists/resolvers.txt",
		"-silent",
		"-massdns", "/usr/local/bin/massdns",
		"-mode", "bruteforce",
	)

	log.Printf("[INFO] Executing command: %s", cmd.String())

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	execTime := time.Since(startTime).String()

	if err != nil {
		log.Printf("[ERROR] ShuffleDNS scan failed for wordlist: %v", err)
		log.Printf("[ERROR] stderr output: %s", stderr.String())
		updateShuffleDNSScanStatus(scanID, "error", "", stderr.String(), cmd.String(), execTime)
		return
	}

	result := stdout.String()
	log.Printf("[INFO] ShuffleDNS scan completed in %s for wordlist", execTime)
	log.Printf("[DEBUG] Raw output length: %d bytes", len(result))

	if result == "" {
		log.Printf("[WARN] No output from ShuffleDNS scan")
		updateShuffleDNSScanStatus(scanID, "completed", "", "No results found", cmd.String(), execTime)
	} else {
		log.Printf("[DEBUG] ShuffleDNS output: %s", result)
		updateShuffleDNSScanStatus(scanID, "success", result, stderr.String(), cmd.String(), execTime)
	}

	log.Printf("[INFO] Scan status updated for scan %s", scanID)
}

func updateCeWLScanStatus(scanID, status, result, stderr, command, execTime string) {
	log.Printf("[INFO] Updating CeWL scan status for %s to %s", scanID, status)
	query := `UPDATE cewl_scans SET status = $1, result = $2, stderr = $3, command = $4, execution_time = $5 WHERE scan_id = $6`
	_, err := dbPool.Exec(context.Background(), query, status, result, stderr, command, execTime, scanID)
	if err != nil {
		log.Printf("[ERROR] Failed to update CeWL scan status for %s: %v", scanID, err)
	} else {
		log.Printf("[INFO] Successfully updated CeWL scan status for %s", scanID)
	}
}

func getShuffleDNSCustomScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]

	if scopeTargetID == "" {
		log.Printf("[ERROR] No scope target ID provided")
		http.Error(w, "No scope target ID provided", http.StatusBadRequest)
		return
	}

	query := `SELECT * FROM shufflednscustom_scans WHERE scope_target_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to get scans: %v", err)
		http.Error(w, "Failed to get scans", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []map[string]interface{}
	for rows.Next() {
		var scan ShuffleDNSScanStatus
		err := rows.Scan(
			&scan.ID,
			&scan.ScanID,
			&scan.Domain,
			&scan.Status,
			&scan.Result,
			&scan.Error,
			&scan.StdOut,
			&scan.StdErr,
			&scan.Command,
			&scan.ExecTime,
			&scan.CreatedAt,
			&scan.ScopeTargetID,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			continue
		}

		scans = append(scans, map[string]interface{}{
			"id":              scan.ID,
			"scan_id":         scan.ScanID,
			"domain":          scan.Domain,
			"status":          scan.Status,
			"result":          nullStringToString(scan.Result),
			"error":           nullStringToString(scan.Error),
			"stdout":          nullStringToString(scan.StdOut),
			"stderr":          nullStringToString(scan.StdErr),
			"command":         nullStringToString(scan.Command),
			"execution_time":  nullStringToString(scan.ExecTime),
			"created_at":      scan.CreatedAt.Format(time.RFC3339),
			"scope_target_id": scan.ScopeTargetID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scans)
}

func runGoSpiderScan(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		FQDN string `json:"fqdn" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.FQDN == "" {
		http.Error(w, "Invalid request body. `fqdn` is required.", http.StatusBadRequest)
		return
	}

	domain := payload.FQDN
	wildcardDomain := fmt.Sprintf("*.%s", domain)

	query := `SELECT id FROM scope_targets WHERE type = 'Wildcard' AND scope_target = $1`
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(), query, wildcardDomain).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] No matching wildcard scope target found for domain %s", domain)
		http.Error(w, "No matching wildcard scope target found.", http.StatusBadRequest)
		return
	}

	scanID := uuid.New().String()
	insertQuery := `INSERT INTO gospider_scans (scan_id, domain, status, scope_target_id) VALUES ($1, $2, $3, $4)`
	_, err = dbPool.Exec(context.Background(), insertQuery, scanID, domain, "pending", scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to create scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}

	go executeAndParseGoSpiderScan(scanID, domain)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func executeAndParseGoSpiderScan(scanID, domain string) {
	log.Printf("[INFO] Starting GoSpider scan for domain %s (scan ID: %s)", domain, scanID)
	startTime := time.Now()

	// First, get all live web servers from the latest httpx scan
	var httpxResults string
	err := dbPool.QueryRow(context.Background(), `
		SELECT result FROM httpx_scans 
		WHERE scope_target_id = (
			SELECT scope_target_id FROM gospider_scans WHERE scan_id = $1
		)
		AND status = 'success'
		ORDER BY created_at DESC 
		LIMIT 1`, scanID).Scan(&httpxResults)

	if err != nil {
		log.Printf("[ERROR] Failed to get httpx results: %v", err)
		updateGoSpiderScanStatus(scanID, "error", "", "Failed to get httpx results", "", time.Since(startTime).String(), "")
		return
	}

	log.Printf("[DEBUG] Retrieved httpx results, length: %d bytes", len(httpxResults))

	// Process each live web server
	urls := strings.Split(httpxResults, "\n")
	log.Printf("[INFO] Processing %d URLs from httpx results", len(urls))

	var allSubdomains []string
	seen := make(map[string]bool)
	var allStdout, allStderr bytes.Buffer
	var commands []string

	for _, urlLine := range urls {
		if urlLine == "" {
			continue
		}

		// Parse JSON from httpx output
		var httpxResult struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal([]byte(urlLine), &httpxResult); err != nil {
			log.Printf("[WARN] Failed to parse httpx result line: %v", err)
			continue
		}

		if httpxResult.URL == "" {
			continue
		}

		log.Printf("[INFO] Running GoSpider against URL: %s", httpxResult.URL)
		scanStartTime := time.Now()

		cmd := exec.Command(
			"docker", "exec",
			"ars0n-framework-v2-gospider-1",
			"gospider",
			"-s", httpxResult.URL,
			"-c", "20", // Increased concurrent requests
			"-d", "3", // Increased depth
			"-t", "5", // Timeout in seconds
			"-k", "1", // Delay of 1 second between requests
			"-K", "2", // Random delay of up to 2 seconds
			"-m", "30", // Increased timeout to 30 seconds
			"--blacklist", ".(jpg|jpeg|gif|css|tif|tiff|png|ttf|woff|woff2|ico|svg)", // Skip binary files
			"-a",        // Find URLs from 3rd party sources
			"-w",        // Include subdomains from 3rd party
			"-r",        // Include other source URLs
			"--js",      // Parse JavaScript files
			"--sitemap", // Parse sitemap.xml
			"--robots",  // Parse robots.txt
			"--debug",   // Enable debug mode
			"--json",    // Enable JSON output
			"-v",        // Verbose output
		)

		commands = append(commands, cmd.String())
		log.Printf("[DEBUG] Executing command: %s", cmd.String())

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		scanDuration := time.Since(scanStartTime)
		log.Printf("[DEBUG] GoSpider scan for %s completed in %s", httpxResult.URL, scanDuration)

		if err != nil {
			log.Printf("[WARN] GoSpider scan failed for %s: %v", httpxResult.URL, err)
			log.Printf("[WARN] stderr output: %s", stderr.String())
			continue
		}

		log.Printf("[DEBUG] Raw stdout length for %s: %d bytes", httpxResult.URL, stdout.Len())
		if stdout.Len() == 0 {
			log.Printf("[WARN] No output from GoSpider for %s", httpxResult.URL)
		}

		// Process the results to extract subdomains
		lines := strings.Split(stdout.String(), "\n")
		log.Printf("[DEBUG] Processing %d lines of output for %s", len(lines), httpxResult.URL)
		newSubdomains := 0

		log.Printf("[DEBUG] === Start of detailed output analysis for %s ===", httpxResult.URL)
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			log.Printf("[DEBUG] Line %d: %s", i+1, line)

			// Try to parse the line as a URL
			parsedURL, err := url.Parse(line)
			if err != nil {
				// If it's not a valid URL, try to find URLs in the line
				urlRegex := regexp.MustCompile(`https?://[^\s<>"']+|[^\s<>"']+\.[^\s<>"']+`)
				matches := urlRegex.FindAllString(line, -1)
				if len(matches) > 0 {
					log.Printf("[DEBUG] Found %d URL matches in line using regex", len(matches))
				}
				for _, match := range matches {
					log.Printf("[DEBUG] Processing URL match: %s", match)
					if !strings.HasPrefix(match, "http") {
						match = "https://" + match
						log.Printf("[DEBUG] Added https:// prefix: %s", match)
					}
					if matchURL, err := url.Parse(match); err == nil {
						hostname := matchURL.Hostname()
						log.Printf("[DEBUG] Extracted hostname: %s", hostname)
						if strings.Contains(hostname, domain) {
							if !seen[hostname] {
								log.Printf("[DEBUG] Found new subdomain from URL match: %s", hostname)
								seen[hostname] = true
								allSubdomains = append(allSubdomains, hostname)
								newSubdomains++
							} else {
								log.Printf("[DEBUG] Skipping duplicate subdomain: %s", hostname)
							}
						} else {
							log.Printf("[DEBUG] Hostname %s does not contain domain %s", hostname, domain)
						}
					} else {
						log.Printf("[DEBUG] Failed to parse URL match %s: %v", match, err)
					}
				}
				continue
			}

			// Process the valid URL
			hostname := parsedURL.Hostname()
			log.Printf("[DEBUG] Processing valid URL with hostname: %s", hostname)
			if strings.Contains(hostname, domain) {
				if !seen[hostname] {
					log.Printf("[DEBUG] Found new subdomain from URL: %s", hostname)
					seen[hostname] = true
					allSubdomains = append(allSubdomains, hostname)
					newSubdomains++
				} else {
					log.Printf("[DEBUG] Skipping duplicate subdomain: %s", hostname)
				}
			} else {
				log.Printf("[DEBUG] Hostname %s does not contain domain %s", hostname, domain)
			}

			// Also check for subdomains in the path segments
			pathParts := strings.Split(parsedURL.Path, "/")
			if len(pathParts) > 0 {
				log.Printf("[DEBUG] Checking %d path segments for potential subdomains", len(pathParts))
				for _, part := range pathParts {
					if strings.Contains(part, domain) && strings.Contains(part, ".") {
						cleanPart := strings.Trim(part, ".")
						log.Printf("[DEBUG] Found potential subdomain in path: %s", cleanPart)
						if !seen[cleanPart] {
							log.Printf("[DEBUG] Found new subdomain in path: %s", cleanPart)
							seen[cleanPart] = true
							allSubdomains = append(allSubdomains, cleanPart)
							newSubdomains++
						} else {
							log.Printf("[DEBUG] Skipping duplicate subdomain from path: %s", cleanPart)
						}
					}
				}
			}
		}

		log.Printf("[DEBUG] === End of detailed output analysis ===")
		log.Printf("[DEBUG] Current list of unique subdomains: %v", allSubdomains)
		log.Printf("[INFO] Found %d new unique subdomains from %s", newSubdomains, httpxResult.URL)

		allStdout.WriteString(fmt.Sprintf("\n=== Results for %s (Duration: %s) ===\n", httpxResult.URL, scanDuration))
		allStdout.Write(stdout.Bytes())
		allStderr.WriteString(fmt.Sprintf("\n=== Errors for %s ===\n", httpxResult.URL))
		allStderr.Write(stderr.Bytes())
	}

	// Sort the results
	sort.Strings(allSubdomains)
	result := strings.Join(allSubdomains, "\n")

	execTime := time.Since(startTime).String()
	log.Printf("[INFO] All GoSpider scans completed in %s", execTime)
	log.Printf("[INFO] Found %d total unique subdomains", len(allSubdomains))
	if len(allSubdomains) > 0 {
		log.Printf("[DEBUG] First 10 subdomains found: %v", allSubdomains[:min(10, len(allSubdomains))])
	}

	if result == "" {
		log.Printf("[WARN] No output from any GoSpider scan")
		updateGoSpiderScanStatus(scanID, "completed", "", "No results found", strings.Join(commands, "\n"), execTime, allStdout.String())
	} else {
		updateGoSpiderScanStatus(scanID, "success", result, allStderr.String(), strings.Join(commands, "\n"), execTime, allStdout.String())
	}

	log.Printf("[INFO] Scan status updated for scan %s", scanID)
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func updateGoSpiderScanStatus(scanID, status, result, stderr, command, execTime, stdout string) {
	log.Printf("[INFO] Updating GoSpider scan status for %s to %s", scanID, status)
	query := `UPDATE gospider_scans SET status = $1, result = $2, stderr = $3, command = $4, execution_time = $5, stdout = $6 WHERE scan_id = $7`
	_, err := dbPool.Exec(context.Background(), query, status, result, stderr, command, execTime, stdout, scanID)
	if err != nil {
		log.Printf("[ERROR] Failed to update GoSpider scan status for %s: %v", scanID, err)
	} else {
		log.Printf("[INFO] Successfully updated GoSpider scan status for %s", scanID)
	}
}

func getGoSpiderScanStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scanID := vars["scan_id"]

	var scan GoSpiderScanStatus
	query := `SELECT * FROM gospider_scans WHERE scan_id = $1`
	err := dbPool.QueryRow(context.Background(), query, scanID).Scan(
		&scan.ID,
		&scan.ScanID,
		&scan.Domain,
		&scan.Status,
		&scan.Result,
		&scan.Error,
		&scan.StdOut,
		&scan.StdErr,
		&scan.Command,
		&scan.ExecTime,
		&scan.CreatedAt,
		&scan.ScopeTargetID,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "Scan not found", http.StatusNotFound)
		} else {
			log.Printf("[ERROR] Failed to get scan status: %v", err)
			http.Error(w, "Failed to get scan status", http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"id":              scan.ID,
		"scan_id":         scan.ScanID,
		"domain":          scan.Domain,
		"status":          scan.Status,
		"result":          nullStringToString(scan.Result),
		"error":           nullStringToString(scan.Error),
		"stdout":          nullStringToString(scan.StdOut),
		"stderr":          nullStringToString(scan.StdErr),
		"command":         nullStringToString(scan.Command),
		"execution_time":  nullStringToString(scan.ExecTime),
		"created_at":      scan.CreatedAt.Format(time.RFC3339),
		"scope_target_id": scan.ScopeTargetID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getGoSpiderScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]

	if scopeTargetID == "" {
		log.Printf("[ERROR] No scope target ID provided")
		http.Error(w, "No scope target ID provided", http.StatusBadRequest)
		return
	}

	query := `SELECT * FROM gospider_scans WHERE scope_target_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to get scans: %v", err)
		http.Error(w, "Failed to get scans", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []map[string]interface{}
	for rows.Next() {
		var scan GoSpiderScanStatus
		err := rows.Scan(
			&scan.ID,
			&scan.ScanID,
			&scan.Domain,
			&scan.Status,
			&scan.Result,
			&scan.Error,
			&scan.StdOut,
			&scan.StdErr,
			&scan.Command,
			&scan.ExecTime,
			&scan.CreatedAt,
			&scan.ScopeTargetID,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			continue
		}

		scans = append(scans, map[string]interface{}{
			"id":              scan.ID,
			"scan_id":         scan.ScanID,
			"domain":          scan.Domain,
			"status":          scan.Status,
			"result":          nullStringToString(scan.Result),
			"error":           nullStringToString(scan.Error),
			"stdout":          nullStringToString(scan.StdOut),
			"stderr":          nullStringToString(scan.StdErr),
			"command":         nullStringToString(scan.Command),
			"execution_time":  nullStringToString(scan.ExecTime),
			"created_at":      scan.CreatedAt.Format(time.RFC3339),
			"scope_target_id": scan.ScopeTargetID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scans)
}

func runSubdomainizerScan(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		FQDN string `json:"fqdn" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.FQDN == "" {
		http.Error(w, "Invalid request body. `fqdn` is required.", http.StatusBadRequest)
		return
	}

	domain := payload.FQDN
	wildcardDomain := fmt.Sprintf("*.%s", domain)

	query := `SELECT id FROM scope_targets WHERE type = 'Wildcard' AND scope_target = $1`
	var scopeTargetID string
	err := dbPool.QueryRow(context.Background(), query, wildcardDomain).Scan(&scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] No matching wildcard scope target found for domain %s", domain)
		http.Error(w, "No matching wildcard scope target found.", http.StatusBadRequest)
		return
	}

	scanID := uuid.New().String()
	insertQuery := `INSERT INTO subdomainizer_scans (scan_id, domain, status, scope_target_id) VALUES ($1, $2, $3, $4)`
	_, err = dbPool.Exec(context.Background(), insertQuery, scanID, domain, "pending", scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to create scan record: %v", err)
		http.Error(w, "Failed to create scan record.", http.StatusInternalServerError)
		return
	}

	go executeAndParseSubdomainizerScan(scanID, domain)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"scan_id": scanID})
}

func executeAndParseSubdomainizerScan(scanID, domain string) {
	log.Printf("[INFO] Starting Subdomainizer scan for domain %s (scan ID: %s)", domain, scanID)
	startTime := time.Now()

	// First, get all live web servers from the latest httpx scan
	var httpxResults string
	err := dbPool.QueryRow(context.Background(), `
		SELECT result FROM httpx_scans 
		WHERE scope_target_id = (
			SELECT scope_target_id FROM subdomainizer_scans WHERE scan_id = $1
		)
		AND status = 'success'
		ORDER BY created_at DESC 
		LIMIT 1`, scanID).Scan(&httpxResults)

	if err != nil {
		log.Printf("[ERROR] Failed to get httpx results: %v", err)
		updateSubdomainizerScanStatus(scanID, "error", "", "Failed to get httpx results", "", time.Since(startTime).String(), "")
		return
	}

	// Create mount directory in container
	mkdirCmd := exec.Command(
		"docker", "exec",
		"ars0n-framework-v2-subdomainizer-1",
		"mkdir", "-p", "/tmp/subdomainizer-mounts",
	)
	if err := mkdirCmd.Run(); err != nil {
		log.Printf("[ERROR] Failed to create mount directory in container: %v", err)
		updateSubdomainizerScanStatus(scanID, "error", "", fmt.Sprintf("Failed to create mount directory: %v", err), "", time.Since(startTime).String(), "")
		return
	}

	// Set permissions
	chmodCmd := exec.Command(
		"docker", "exec",
		"ars0n-framework-v2-subdomainizer-1",
		"chmod", "777", "/tmp/subdomainizer-mounts",
	)
	if err := chmodCmd.Run(); err != nil {
		log.Printf("[ERROR] Failed to set permissions on mount directory: %v", err)
		updateSubdomainizerScanStatus(scanID, "error", "", fmt.Sprintf("Failed to set permissions: %v", err), "", time.Since(startTime).String(), "")
		return
	}

	// Process each live web server
	urls := strings.Split(httpxResults, "\n")
	log.Printf("[INFO] Processing %d URLs from httpx results", len(urls))

	var allSubdomains []string
	seen := make(map[string]bool)
	var allStdout, allStderr bytes.Buffer
	var commands []string

	for _, urlLine := range urls {
		if urlLine == "" {
			continue
		}

		// Parse JSON from httpx output
		var httpxResult struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal([]byte(urlLine), &httpxResult); err != nil {
			log.Printf("[WARN] Failed to parse httpx result line: %v", err)
			continue
		}

		if httpxResult.URL == "" {
			continue
		}

		log.Printf("[INFO] Running Subdomainizer against URL: %s", httpxResult.URL)

		cmd := exec.Command(
			"docker", "exec",
			"ars0n-framework-v2-subdomainizer-1",
			"python3", "SubDomainizer.py",
			"-u", httpxResult.URL,
			"-k",
			"-o", "/tmp/subdomainizer-mounts/output.txt",
			"-sop", "/tmp/subdomainizer-mounts/secrets.txt",
		)

		commands = append(commands, cmd.String())
		log.Printf("[INFO] Executing command: %s", cmd.String())

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			log.Printf("[WARN] Subdomainizer scan failed for %s: %v", httpxResult.URL, err)
			log.Printf("[WARN] stderr output: %s", stderr.String())
			continue
		}

		// Read output file from container
		catCmd := exec.Command(
			"docker", "exec",
			"ars0n-framework-v2-subdomainizer-1",
			"cat", "/tmp/subdomainizer-mounts/output.txt",
		)

		var outputContent bytes.Buffer
		catCmd.Stdout = &outputContent
		if err := catCmd.Run(); err != nil {
			log.Printf("[WARN] Failed to read output file for %s: %v", httpxResult.URL, err)
			continue
		}

		// Process the results
		lines := strings.Split(outputContent.String(), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && strings.Contains(line, domain) && !seen[line] {
				seen[line] = true
				allSubdomains = append(allSubdomains, line)
			}
		}

		allStdout.WriteString(fmt.Sprintf("\n=== Results for %s ===\n", httpxResult.URL))
		allStdout.Write(stdout.Bytes())
		allStderr.WriteString(fmt.Sprintf("\n=== Errors for %s ===\n", httpxResult.URL))
		allStderr.Write(stderr.Bytes())
	}

	// Sort the results
	sort.Strings(allSubdomains)
	result := strings.Join(allSubdomains, "\n")

	execTime := time.Since(startTime).String()
	log.Printf("[INFO] All Subdomainizer scans completed in %s", execTime)
	log.Printf("[DEBUG] Found %d unique subdomains", len(allSubdomains))

	if result == "" {
		log.Printf("[WARN] No output from any Subdomainizer scan")
		updateSubdomainizerScanStatus(scanID, "completed", "", "No results found", strings.Join(commands, "\n"), execTime, allStdout.String())
	} else {
		updateSubdomainizerScanStatus(scanID, "success", result, allStderr.String(), strings.Join(commands, "\n"), execTime, allStdout.String())
	}

	// Cleanup files in container
	cleanupCmd := exec.Command(
		"docker", "exec",
		"ars0n-framework-v2-subdomainizer-1",
		"rm", "-rf", "/tmp/subdomainizer-mounts",
	)
	if err := cleanupCmd.Run(); err != nil {
		log.Printf("[WARN] Failed to cleanup files in container: %v", err)
	}

	log.Printf("[INFO] Scan status updated for scan %s", scanID)
}

func updateSubdomainizerScanStatus(scanID, status, result, stderr, command, execTime, stdout string) {
	log.Printf("[INFO] Updating Subdomainizer scan status for %s to %s", scanID, status)
	query := `UPDATE subdomainizer_scans SET status = $1, result = $2, stderr = $3, command = $4, execution_time = $5, stdout = $6 WHERE scan_id = $7`
	_, err := dbPool.Exec(context.Background(), query, status, result, stderr, command, execTime, stdout, scanID)
	if err != nil {
		log.Printf("[ERROR] Failed to update Subdomainizer scan status for %s: %v", scanID, err)
	} else {
		log.Printf("[INFO] Successfully updated Subdomainizer scan status for %s", scanID)
	}
}

func getSubdomainizerScanStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scanID := vars["scan_id"]

	var scan SubdomainizerScanStatus
	query := `SELECT * FROM subdomainizer_scans WHERE scan_id = $1`
	err := dbPool.QueryRow(context.Background(), query, scanID).Scan(
		&scan.ID,
		&scan.ScanID,
		&scan.Domain,
		&scan.Status,
		&scan.Result,
		&scan.Error,
		&scan.StdOut,
		&scan.StdErr,
		&scan.Command,
		&scan.ExecTime,
		&scan.CreatedAt,
		&scan.ScopeTargetID,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "Scan not found", http.StatusNotFound)
		} else {
			log.Printf("[ERROR] Failed to get scan status: %v", err)
			http.Error(w, "Failed to get scan status", http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"id":              scan.ID,
		"scan_id":         scan.ScanID,
		"domain":          scan.Domain,
		"status":          scan.Status,
		"result":          nullStringToString(scan.Result),
		"error":           nullStringToString(scan.Error),
		"stdout":          nullStringToString(scan.StdOut),
		"stderr":          nullStringToString(scan.StdErr),
		"command":         nullStringToString(scan.Command),
		"execution_time":  nullStringToString(scan.ExecTime),
		"created_at":      scan.CreatedAt.Format(time.RFC3339),
		"scope_target_id": scan.ScopeTargetID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getSubdomainizerScansForScopeTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scopeTargetID := vars["id"]

	if scopeTargetID == "" {
		log.Printf("[ERROR] No scope target ID provided")
		http.Error(w, "No scope target ID provided", http.StatusBadRequest)
		return
	}

	query := `SELECT * FROM subdomainizer_scans WHERE scope_target_id = $1 ORDER BY created_at DESC`
	rows, err := dbPool.Query(context.Background(), query, scopeTargetID)
	if err != nil {
		log.Printf("[ERROR] Failed to get scans: %v", err)
		http.Error(w, "Failed to get scans", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scans []map[string]interface{}
	for rows.Next() {
		var scan SubdomainizerScanStatus
		err := rows.Scan(
			&scan.ID,
			&scan.ScanID,
			&scan.Domain,
			&scan.Status,
			&scan.Result,
			&scan.Error,
			&scan.StdOut,
			&scan.StdErr,
			&scan.Command,
			&scan.ExecTime,
			&scan.CreatedAt,
			&scan.ScopeTargetID,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to scan row: %v", err)
			continue
		}

		scans = append(scans, map[string]interface{}{
			"id":              scan.ID,
			"scan_id":         scan.ScanID,
			"domain":          scan.Domain,
			"status":          scan.Status,
			"result":          nullStringToString(scan.Result),
			"error":           nullStringToString(scan.Error),
			"stdout":          nullStringToString(scan.StdOut),
			"stderr":          nullStringToString(scan.StdErr),
			"command":         nullStringToString(scan.Command),
			"execution_time":  nullStringToString(scan.ExecTime),
			"created_at":      scan.CreatedAt.Format(time.RFC3339),
			"scope_target_id": scan.ScopeTargetID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scans)
}
