package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// SanitizeDomain validates and sanitizes a domain name to prevent command injection.
// It checks if the domain is a valid FQDN and contains no malicious characters.
func SanitizeDomain(domain string) (string, error) {
	// Trim whitespace
	domain = strings.TrimSpace(domain)

	// FQDN validation regex
	fqdnRegex := `^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,6}$`
	if !regexp.MustCompile(fqdnRegex).MatchString(domain) {
		return "", fmt.Errorf("invalid domain name format: %s", domain)
	}

	// Check for command injection characters
	if strings.ContainsAny(domain, ";|&`$()<>\\") {
		return "", fmt.Errorf("domain contains malicious characters: %s", domain)
	}

	return domain, nil
}

// SanitizeURL validates and sanitizes a URL to prevent command injection.
func SanitizeURL(rawURL string) (string, error) {
	// Trim whitespace
	rawURL = strings.TrimSpace(rawURL)

	// Check for command injection characters
	if strings.ContainsAny(rawURL, ";|&`$()<>\\") {
		return "", fmt.Errorf("url contains malicious characters: %s", rawURL)
	}

	return rawURL, nil
}

// IsValidUUID checks if a string is a valid UUID.
func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}
