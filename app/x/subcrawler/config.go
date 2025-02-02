package subcrawler

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the configuration settings for Subcrawler
// Includes API keys and source-specific configurations
type Config struct {
	APIKeys map[string]string `json:"api_keys"`
	Timeout int               `json:"timeout"`
	Retries int               `json:"retries"`
}

// LoadConfig loads configuration settings from a JSON file
func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return &config, nil
}
