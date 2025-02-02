package webcrawler

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Targets      []string `yaml:"targets"`
	Concurrency  int      `yaml:"concurrency"`
	UserAgent    string   `yaml:"user_agent"`
	MaxDepth     int      `yaml:"max_depth"`
	OutputFormat string   `yaml:"output_format"`
}

// LoadConfig loads and parses a YAML configuration file
func LoadConfig(filePath string) (*Config, error) {
	if filePath == "" {
		return nil, fmt.Errorf("configuration file path is required")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	// Apply defaults if necessary
	if config.Concurrency == 0 {
		config.Concurrency = 5 // Default concurrency level
	}
	if config.UserAgent == "" {
		config.UserAgent = "WebCrawler/1.0"
	}
	if config.MaxDepth == 0 {
		config.MaxDepth = 3 // Default crawl depth
	}
	if config.OutputFormat == "" {
		config.OutputFormat = "json" // Default output format
	}

	return &config, nil
}
