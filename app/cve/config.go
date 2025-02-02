package cve

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	CVEMap       CVEMapConfig       `json:"cvemap"`
	CloudCrawler CloudCrawlerConfig `json:"cloudcrawler"`
}

type CVEMapConfig struct {
	APIKey   string `json:"api_key"`
	APIURL   string `json:"api_url"`
	Debug    bool   `json:"debug"`
	LogLevel string `json:"log_level"`
}

type CloudCrawlerConfig struct {
	DefaultOutput string   `json:"default_output"`
	Verbose       bool     `json:"verbose"`
	Providers     []string `json:"providers"`
}

// LoadConfig loads the configuration file into the Config struct
func LoadConfig(filePath string) (*Config, error) {
	if filePath == "" {
		filePath = "config.json"
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validate ensures all required configuration values are present
func (c *Config) validate() error {
	if c.CVEMap.APIKey == "" {
		return fmt.Errorf("missing CVEMAP API key")
	}
	if c.CVEMap.APIURL == "" {
		return fmt.Errorf("missing CVEMAP API URL")
	}
	return nil
}
