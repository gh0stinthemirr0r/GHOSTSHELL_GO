package ghostcrawler

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// OutputManager handles writing results to various output formats.
type OutputManager struct {
	OutputFile string
	UseJSON    bool
	mutex      sync.Mutex
}

// NewOutputManager initializes a new OutputManager.
func NewOutputManager(outputFile string, useJSON bool) *OutputManager {
	return &OutputManager{
		OutputFile: outputFile,
		UseJSON:    useJSON,
	}
}

// WriteResult writes a single result to the output, handling both JSON and plain text formats.
func (o *OutputManager) WriteResult(result Result) error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	var output string
	if o.UseJSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal result to JSON: %w", err)
		}
		output = string(data)
	} else {
		output = fmt.Sprintf("Crawler: %s\nData: %v\nError: %v\n", result.CrawlerName, result.Data, result.Error)
	}

	if o.OutputFile != "" {
		file, err := os.OpenFile(o.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open output file: %w", err)
		}
		defer file.Close()

		if _, err := file.WriteString(output + "\n"); err != nil {
			return fmt.Errorf("failed to write to output file: %w", err)
		}
	} else {
		fmt.Println(output)
	}

	return nil
}

// WriteResults writes multiple results to the output file or stdout.
func (o *OutputManager) WriteResults(results []Result) error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.OutputFile != "" {
		file, err := os.OpenFile(o.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open output file: %w", err)
		}
		defer file.Close()

		for _, result := range results {
			var output string
			if o.UseJSON {
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal result to JSON: %w", err)
				}
				output = string(data)
			} else {
				output = fmt.Sprintf("Crawler: %s\nData: %v\nError: %v\n", result.CrawlerName, result.Data, result.Error)
			}
			if _, err := file.WriteString(output + "\n"); err != nil {
				return fmt.Errorf("failed to write to output file: %w", err)
			}
		}
	} else {
		for _, result := range results {
			var output string
			if o.UseJSON {
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal result to JSON: %w", err)
				}
				output = string(data)
			} else {
				output = fmt.Sprintf("Crawler: %s\nData: %v\nError: %v\n", result.CrawlerName, result.Data, result.Error)
			}
			fmt.Println(output)
		}
	}

	return nil
}
