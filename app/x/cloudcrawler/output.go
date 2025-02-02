package cloudcrawler

import (
	"fmt"
	"os"
)

type Result struct {
	Provider string
	Resource string
}

// Display formats and outputs the results to the console and optionally to a file
func Display(results []Result, outputFile string) {
	// Print results to the console
	fmt.Println("Discovered Resources:")
	for _, result := range results {
		fmt.Printf("[%s] %s\n", result.Provider, result.Resource)
	}

	// Save results to the specified output file if provided
	if outputFile != "" {
		if err := saveToFile(results, outputFile); err != nil {
			fmt.Printf("Failed to save results to file: %v\n", err)
		} else {
			fmt.Printf("Results saved to %s\n", outputFile)
		}
	}
}

// saveToFile writes the results to the specified file
func saveToFile(results []Result, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, result := range results {
		_, err := file.WriteString(fmt.Sprintf("[%s] %s\n", result.Provider, result.Resource))
		if err != nil {
			return err
		}
	}
	return nil
}
