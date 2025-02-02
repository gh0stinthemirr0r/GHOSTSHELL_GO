package proxi

import (
	"fmt"
	"os"
)

// writeToFile writes the given data to a specified file
func writeToFile(filePath string, data string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(data)
	if err != nil {
		return fmt.Errorf("failed to write data to file: %w", err)
	}

	fmt.Printf("Data successfully written to %s\n", filePath)
	return nil
}

// writeToConsole writes the given data to the console
func writeToConsole(data string) {
	fmt.Println(data)
}
