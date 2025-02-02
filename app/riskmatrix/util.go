package riskmatrix

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

// Contains checks if a slice of strings contains a specific string
func Contains(slice []string, item string) bool {
	for _, element := range slice {
		if element == item {
			return true
		}
	}
	return false
}
