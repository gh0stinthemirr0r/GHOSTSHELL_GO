package openport

import (
	"fmt"
	"os"
)

// writeResults writes the list of open ports to a file or stdout
func writeResults(openPorts []int, outputFile string) error {
	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()

		for _, port := range openPorts {
			if _, err := file.WriteString(fmt.Sprintf("Port %d is open\n", port)); err != nil {
				return fmt.Errorf("failed to write to output file: %w", err)
			}
		}

		fmt.Printf("Results written to %s\n", outputFile)
	} else {
		for _, port := range openPorts {
			fmt.Printf("Port %d is open\n", port)
		}
	}

	return nil
}
