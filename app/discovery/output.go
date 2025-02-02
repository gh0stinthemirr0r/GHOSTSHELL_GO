package discovery

import (
	"fmt"
	"os"
)

type Result struct {
	Source string
	IP     string
	Port   int
	Host   string
	Url    string
}

// writeResults writes results to the console and a file if specified
func writeResults(results []Result, outputFile string) error {
	// Open the file for writing if an output file is provided
	var file *os.File
	var err error
	if outputFile != "" {
		file, err = os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
	}

	// Write results to console and file
	for _, result := range results {
		output := fmt.Sprintf("Source: %s | IP: %s | Port: %d | Host: %s | URL: %s\n",
			result.Source, result.IP, result.Port, result.Host, result.Url)
		fmt.Print(output)

		if file != nil {
			if _, err := file.WriteString(output); err != nil {
				return fmt.Errorf("failed to write to output file: %w", err)
			}
		}
	}

	return nil
}
