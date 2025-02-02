package subcrawler

import (
	"fmt"
	"os"
)

// SaveResults writes unique subdomains to a file or prints them to the console
func SaveResults(results map[string]bool, outputFile string) error {
	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()

		for subdomain := range results {
			if _, err := file.WriteString(subdomain + "\n"); err != nil {
				return fmt.Errorf("failed to write to output file: %w", err)
			}
		}

		fmt.Printf("Results saved to %s\n", outputFile)
	} else {
		for subdomain := range results {
			fmt.Println(subdomain)
		}
	}

	return nil
}
