package nmap

import (
	"bufio"
	"os"
	"strings"
)

// ReadLines reads all lines from a given file and returns them as a slice of strings
func ReadLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// IsValidTarget checks if the given string is a valid Nmap target (IP or hostname)
func IsValidTarget(target string) bool {
	// Basic validation: Ensure target is not empty and doesn't contain invalid characters
	return target != "" && !strings.ContainsAny(target, " \\"")
}