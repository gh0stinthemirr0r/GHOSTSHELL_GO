package discovery

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	APIKeys   map[string]string `json:"api_keys"`
	RateLimit int               `json:"rate_limit"`
	Timeout   int               `json:"timeout"`
}

// loadConfig loads the configuration file into the Config struct
func loadConfig(filePath string) (*Config, error) {
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
	if len(c.APIKeys) == 0 {
		return fmt.Errorf("missing API keys in configuration")
	}
	if c.RateLimit <= 0 {
		return fmt.Errorf("invalid rate limit value")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("invalid timeout value")
	}
	return nil
}
