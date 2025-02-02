package burp

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BIID        string `yaml:"biid"`
	Verbose     bool   `yaml:"verbose"`
	Silent      bool   `yaml:"silent"`
	Version     bool   `yaml:"version"`
	Interval    int    `yaml:"interval"`
	HTTPMessage string `yaml:"http_message"`
	DNSMessage  string `yaml:"dns_message"`
	CLIMessage  string `yaml:"cli_message"`
	SMTPMessage string `yaml:"smtp_message"`
}

// LoadConfig reads and parses a YAML configuration file
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
	if config.Interval == 0 {
		config.Interval = 5 // Default polling interval
	}
	if config.HTTPMessage == "" {
		config.HTTPMessage = "Default HTTP Message"
	}
	if config.DNSMessage == "" {
		config.DNSMessage = "Default DNS Message"
	}
	if config.SMTPMessage == "" {
		config.SMTPMessage = "Default SMTP Message"
	}
	if config.CLIMessage == "" {
		config.CLIMessage = "Default CLI Message"
	}

	return &config, nil
}
