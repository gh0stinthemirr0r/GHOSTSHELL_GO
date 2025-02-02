package webcrawler

import (
	"encoding/json"
	"fmt"
	"os"
)

type CrawlResult struct {
	Target string   `json:"target"`
	Status string   `json:"status"`
	Links  []string `json:"links"`
}

// writeResults writes the crawl results to a file or stdout
func writeResults(results []CrawlResult, outputFile string, format string) error {
	var output string

	if format == "json" {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal results to JSON: %w", err)
		}
		output = string(data)
	} else {
		for _, result := range results {
			output += fmt.Sprintf("Target: %s\nStatus: %s\nLinks:\n", result.Target, result.Status)
			for _, link := range result.Links {
				output += fmt.Sprintf("  - %s\n", link)
			}
		}
	}

	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()

		_, err = file.WriteString(output)
		if err != nil {
			return fmt.Errorf("failed to write to output file: %w", err)
		}
		return nil
	}

	// Print to stdout
	fmt.Println(output)
	return nil
}
