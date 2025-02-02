package cloudcrawler

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Config holds the global application configuration
type Config struct {
	Providers map[string]ProviderConfig `json:"providers"` // Configurations for each provider
}

// ProviderConfig represents the configuration for a single provider
type ProviderConfig struct {
	APIKey    string `json:"api_key"`    // API key for authentication
	SecretKey string `json:"secret_key"` // Secret key for authentication
	Region    string `json:"region"`     // Region (optional, provider-specific)
}

// LoadConfig loads the configuration file into a Config struct
func LoadConfig() (*Config, error) {
	// Default config file path
	const defaultConfigPath = "config.json"

	// Open config file
	file, err := os.Open(defaultConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %v", err)
	}
	defer file.Close()

	// Parse the file
	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Validate the config
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validate checks that the required fields are present in the config
func (c *Config) validate() error {
	if len(c.Providers) == 0 {
		return errors.New("no providers configured")
	}

	for name, provider := range c.Providers {
		if provider.APIKey == "" {
			return fmt.Errorf("missing API key for provider: %s", name)
		}
		if provider.SecretKey == "" {
			return fmt.Errorf("missing secret key for provider: %s", name)
		}
	}
	return nil
}
