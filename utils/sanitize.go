// Package sanitize provides functionality to clean and validate user inputs.
package utils

import (
	"strings"
	"unicode"
	"errors"
)

// SanitizeInput removes potentially harmful characters from the input string.
// It ensures the input string is safe for processing.
func SanitizeInput(input string) string {
	// Trim leading and trailing whitespaces
	cleanedInput := strings.TrimSpace(input)

	// Replace potentially harmful characters
	replacer := strings.NewReplacer(
		";", "",
		"&", "",
		"|", "",
		"`", "",
		">", "",
		"<", "",
		"$", "",
		"\"", "",
		"\'", "",
	)
	cleanedInput = replacer.Replace(cleanedInput)

	return cleanedInput
}

// ValidateASN ensures the input is a valid ASN format (e.g., "AS1234").
func ValidateASN(asn string) error {
	if !strings.HasPrefix(strings.ToUpper(asn), "AS") {
		return errors.New("ASN must start with 'AS' prefix")
	}
	for _, r := range asn[2:] {
		if !unicode.IsDigit(r) {
			return errors.New("ASN contains invalid characters")
		}
	}
	return nil
}

// ValidateIP ensures the input is a valid IPv4 or IPv6 address.
func ValidateIP(ip string) error {
	// Placeholder for IP validation logic. For a robust implementation,
	// you may use libraries like `net` package's `ParseIP`.
	if strings.Count(ip, ".") == 3 || strings.Contains(ip, ":") {
		return nil
	}
	return errors.New("invalid IP address format")
}

// ValidateDomain ensures the input is a valid domain name.
func ValidateDomain(domain string) error {
	// Basic domain validation. You may extend this using regex for stricter validation.
	if len(domain) < 1 || strings.ContainsAny(domain, " \\/:*?\"<>|") {
		return errors.New("invalid domain format")
	}
	return nil
}