package riskmatrix

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ParseMarkdown reads and processes a Markdown file to extract relevant data
func ParseMarkdown(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Markdown file: %w", err)
	}
	defer file.Close()

	var extractedData []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			extractedData = append(extractedData, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading Markdown file: %w", err)
	}

	return extractedData, nil
}
