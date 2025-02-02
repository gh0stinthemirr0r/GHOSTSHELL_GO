package riskmatrix

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// ParseJSONL reads and parses a JSONL (JSON Lines) file
func ParseJSONL(filePath string) ([]map[string]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open JSONL file: %w", err)
	}
	defer file.Close()

	var records []map[string]interface{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		var record map[string]interface{}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, fmt.Errorf("failed to parse JSONL line: %w", err)
		}
		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading JSONL file: %w", err)
	}

	return records, nil
}
