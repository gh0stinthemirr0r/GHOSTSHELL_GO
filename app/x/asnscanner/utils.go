// Package scanner contains shared utilities for the ASN scanner.
package asnscanner

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// Result represents the structure of data processed by the ASN scanner.
type Result struct {
	Timestamp string   // The timestamp of the result
	Input     string   // The input provided (e.g., IP or domain)
	ASN       string   // The Autonomous System Number
	Org       string   // The organization associated with the ASN
	Country   string   // The country of the ASN
	IPRange   []string // The IP range associated with the ASN
}

// IsValidIP checks if a string is a valid IP address.
func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// IsValidDomain checks if a string is a valid domain name.
func IsValidDomain(domain string) bool {
	return strings.Contains(domain, ".") && !strings.Contains(domain, " ")
}

// IsValidASN checks if a string is a valid ASN.
func IsValidASN(asn string) bool {
	return strings.HasPrefix(strings.ToUpper(asn), "AS")
}

// WriteJSONToFile writes the provided result to a JSON file at the specified path.
func WriteJSONToFile(result interface{}, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to write JSON to file: %w", err)
	}

	return nil
}

// WriteCSVToFile writes the provided results to a CSV file at the specified path.
func WriteCSVToFile(results []*Result, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header row
	if err := writer.Write([]string{"Timestamp", "Input", "ASN", "Organization", "Country", "IP Range"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, result := range results {
		row := []string{
			result.Timestamp,
			result.Input,
			result.ASN,
			result.Org,
			result.Country,
			strings.Join(result.IPRange, ", "),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// SanitizeInput removes potentially harmful characters from a string.
func SanitizeInput(input string) string {
	// Remove any characters that could be used maliciously.
	cleanInput := strings.ReplaceAll(input, ";", "")
	cleanInput = strings.ReplaceAll(cleanInput, "&", "")
	cleanInput = strings.ReplaceAll(cleanInput, "|", "")
	cleanInput = strings.ReplaceAll(cleanInput, "`", "")
	return cleanInput
}

// FormatTimestamp returns the current timestamp in RFC3339 format.
func FormatTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

// PrintError prints a formatted error message.
func PrintError(err error) {
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}
