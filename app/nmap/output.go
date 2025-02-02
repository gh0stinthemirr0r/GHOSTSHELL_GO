package nmap

import (
	"fmt"
	"os"
)

// writeResults writes scan results to a file or prints them to stdout
func writeResults(results []byte, outputFile string) error {
	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()

		_, err = file.Write(results)
		if err != nil {
			return fmt.Errorf("failed to write results to file: %w", err)
		}
		fmt.Printf("Results written to %s\n", outputFile)
	} else {
		fmt.Println(string(results))
	}

	return nil
}
