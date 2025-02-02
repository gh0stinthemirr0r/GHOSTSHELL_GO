package ghostcrawler

import (
	"bufio"
	"os"
	"strings"
)

// ReadLines reads all lines from a file and returns them as a slice of strings.
func ReadLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// FileExists checks if a given file exists and is not a directory.
func FileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// UniqueStrings removes duplicate strings from a slice.
func UniqueStrings(input []string) []string {
	uniqueMap := make(map[string]bool)
	for _, item := range input {
		uniqueMap[item] = true
	}

	var uniqueSlice []string
	for item := range uniqueMap {
		uniqueSlice = append(uniqueSlice, item)
	}

	return uniqueSlice
}

// NormalizeURL ensures a URL is in a standardized format.
func NormalizeURL(rawURL string) (string, error) {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return "http://" + rawURL, nil
	}
	return rawURL, nil
}
