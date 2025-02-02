package utils

import (
	"errors"
	"net"
	"strings"
)

// IsIPv4 checks if the input string is a valid IPv4 address.
func IsIPv4(address string) bool {
	parsedIP := net.ParseIP(address)
	return parsedIP != nil && parsedIP.To4() != nil
}

// IsIPv6 checks if the input string is a valid IPv6 address.
func IsIPv6(address string) bool {
	parsedIP := net.ParseIP(address)
	return parsedIP != nil && parsedIP.To16() != nil && parsedIP.To4() == nil
}

// IsASN checks if the input string is a valid ASN (Autonomous System Number).
func IsASN(input string) bool {
	if !strings.HasPrefix(strings.ToUpper(input), "AS") {
		return false
	}
	// Ensure the rest of the string is numeric
	asn := strings.TrimPrefix(strings.ToUpper(input), "AS")
	for _, char := range asn {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// ValidateDomain ensures the input string is a valid domain name.
func ValidateDomain(domain string) error {
	if len(domain) < 1 || len(domain) > 253 {
		return errors.New("domain length must be between 1 and 253 characters")
	}

	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return errors.New("domain must have at least one dot")
	}

	for _, part := range parts {
		if len(part) < 1 || len(part) > 63 {
			return errors.New("each domain segment must be between 1 and 63 characters")
		}
		if !isAlphanumeric(part) {
			return errors.New("domain segments must be alphanumeric")
		}
	}

	return nil
}

// isAlphanumeric checks if a string contains only alphanumeric characters and hyphens.
func isAlphanumeric(segment string) bool {
	for i, char := range segment {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' {
			return false
		}
		// Hyphen cannot be the first or last character
		if (i == 0 || i == len(segment)-1) && char == '-' {
			return false
		}
	}
	return true
}

// GetNormalizedDomain ensures the domain is in a standardized format.
func GetNormalizedDomain(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}
