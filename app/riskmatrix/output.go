package riskmatrix

import (
	"encoding/json"
	"fmt"
	"os"
)

// Output handles the display and export of risk matrix results
func writeJSONOutput(filePath string, data interface{}) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode data to JSON: %w", err)
	}

	fmt.Printf("Results successfully written to %s\n", filePath)
	return nil
}

// writeConsoleOutput prints risk matrix results to the console
func writeConsoleOutput(data interface{}) {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("failed to marshal data for console output: %v\n", err)
		return
	}

	fmt.Println(string(output))
}
