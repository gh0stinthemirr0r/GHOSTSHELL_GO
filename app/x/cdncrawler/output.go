// Package cdnscanner provides output handling for scan results.
package cdncrawler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// OutputHandler manages the output of scan results.
type OutputHandler struct {
	OutputDir string // Directory to save output files
}

// NewOutputHandler initializes and returns a new OutputHandler.
func NewOutputHandler(outputDir string) *OutputHandler {
	// Ensure a default output directory if none is provided
	if outputDir == "" {
		outputDir = "./output"
	}
	return &OutputHandler{OutputDir: outputDir}
}

// SaveAsJSON saves the results to a JSON file.
func (oh *OutputHandler) SaveAsJSON(fileName string, results []Result) error {
	filePath := filepath.Join(oh.OutputDir, fileName+".json")

	// Ensure the output directory exists
	if err := os.MkdirAll(oh.OutputDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create JSON file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		return fmt.Errorf("failed to write results to JSON file: %w", err)
	}

	fmt.Printf("Results saved to %s\n", filePath)
	return nil
}

// SaveAsCSV saves the results to a CSV file.
func (oh *OutputHandler) SaveAsCSV(fileName string, results []Result) error {
	filePath := filepath.Join(oh.OutputDir, fileName+".csv")

	// Ensure the output directory exists
	if err := os.MkdirAll(oh.OutputDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header row
	header := []string{"ASN", "Organization", "IP Range", "Country", "Input", "Timestamp"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, result := range results {
		row := []string{
			result.ASN,
			result.Org,
			formatIPRanges(result.IPRange),
			result.Country,
			result.Input,
			result.Timestamp,
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	fmt.Printf("Results saved to %s\n", filePath)
	return nil
}

// PrintToConsole prints results in a readable format to the console.
func (oh *OutputHandler) PrintToConsole(results []Result) {
	for _, result := range results {
		fmt.Println("===================================")
		fmt.Printf("ASN:       %s\n", result.ASN)
		fmt.Printf("Org:       %s\n", result.Org)
		fmt.Printf("IP Range:  %s\n", formatIPRanges(result.IPRange))
		fmt.Printf("Country:   %s\n", result.Country)
		fmt.Printf("Input:     %s\n", result.Input)
		fmt.Printf("Timestamp: %s\n", result.Timestamp)
		fmt.Println("===================================")
	}
}

// formatIPRanges formats the IP ranges as a comma-separated string.
func formatIPRanges(ipRanges []string) string {
	return strings.Join(ipRanges, ", ")
}
