package httpcrawler

import (
	"bufio"
	"os"
	"strings"
)

// ReadLines reads all lines from a file and returns them as a slice of strings
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

// IsValidURL checks if the given string is a valid URL
func IsValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}
