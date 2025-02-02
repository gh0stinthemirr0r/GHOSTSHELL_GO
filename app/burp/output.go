package burp

import (
	"fmt"
	"os"
)

// writeOutput writes the output message to a file or stdout
func writeOutput(output string, outputFile string) error {
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

	// Write to stdout
	fmt.Println(output)
	return nil
}
