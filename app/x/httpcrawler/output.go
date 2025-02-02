package httpcrawler

import (
	"fmt"
	"os"
)

type Result struct {
	URL    string
	Status int
	Body   string
}

// writeResults writes the results to a file or stdout
func writeResults(results <-chan Result, outputFile string) error {
	var file *os.File
	var err error

	// Open file for writing if outputFile is specified
	if outputFile != "" {
		file, err = os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
	}

	// Write results
	for result := range results {
		output := fmt.Sprintf("URL: %s | Status: %d\n", result.URL, result.Status)
		if file != nil {
			if _, err := file.WriteString(output); err != nil {
				return fmt.Errorf("failed to write to output file: %w", err)
			}
		} else {
			fmt.Print(output)
		}
	}

	return nil
}
