package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// clearScreen clears the terminal screen
func clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	_ = cmd.Run()
}

// checkFileExists checks if a file exists at the specified path
func checkFileExists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}

// writeToFile writes data to the specified file path
func writeToFile(filePath string, data string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(data)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// readFileContent reads the content of a file
func readFileContent(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(data), nil
}
