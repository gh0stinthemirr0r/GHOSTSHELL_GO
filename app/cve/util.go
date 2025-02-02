package cve

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// ClearScreen clears the terminal screen
func ClearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	_ = cmd.Run()
}

// CheckFileExists checks if a file exists at the specified path
func CheckFileExists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}

// WriteToFile writes data to the specified file path
func WriteToFile(filePath string, data string) error {
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

// ReadFileContent reads the content of a file
func ReadFileContent(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(data), nil
}
