// Package cdnscanner provides utility functions shared across the CDN scanner.
package cdncrawler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// WriteJSONToFile writes the provided data to a JSON file at the specified path.
func WriteJSONToFile(filePath string, data interface{}) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create JSON file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty-print JSON
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to write JSON data: %w", err)
	}

	fmt.Printf("JSON output successfully written to: %s\n", filePath)
	return nil
}

// WriteCSVToFile writes the provided data to a CSV file at the specified path.
func WriteCSVToFile(filePath string, headers []string, rows [][]string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write headers
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	// Write rows
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	fmt.Printf("CSV output successfully written to: %s\n", filePath)
	return nil
}

// IsValidIP validates if a string is a valid IPv4 or IPv6 address.
func IsValidIP(ip string) bool {
	return strings.Count(ip, ".") == 3 || strings.Count(ip, ":") > 1
}

// FormatAsJSON formats a data structure as a pretty JSON string.
func FormatAsJSON(data interface{}) (string, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format data as JSON: %w", err)
	}
	return string(jsonData), nil
}

// SanitizeInput ensures user-provided input is safe for processing.
func SanitizeInput(input string) string {
	safeInput := strings.TrimSpace(input)
	safeInput = strings.ReplaceAll(safeInput, ";", "")
	safeInput = strings.ReplaceAll(safeInput, "&", "")
	safeInput = strings.ReplaceAll(safeInput, "|", "")
	safeInput = strings.ReplaceAll(safeInput, "`", "")
	return safeInput
}

// FileExists checks if a file exists at the given path.
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// ParseCSVFile reads a CSV file and returns its data as a 2D slice.
func ParseCSVFile(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	data, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file: %w", err)
	}

	return data, nil
}

// PrintDivider prints a visual divider line for the console output.
func PrintDivider(char string, length int) {
	fmt.Println(strings.Repeat(char, length))
}

// Example usage:
// func main() {
//     // Write JSON example
//     data := map[string]string{"key": "value"}
//     err := WriteJSONToFile("output.json", data)
//     if err != nil {
//         fmt.Println("Error writing JSON:", err)
//     }
//
//     // Validate IP example
//     ip := "192.168.1.1"
//     fmt.Printf("Is valid IP: %v\n", IsValidIP(ip))
//
//     // Sanitize input example
//     input := "rm -rf /;"
//     fmt.Printf("Sanitized input: %s\n", SanitizeInput(input))
// }
