package httpcrawler

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Resolvers []string `json:"resolvers"`
	Retries   int      `json:"retries"`
	Timeout   int      `json:"timeout"`
	UserAgent string   `json:"user_agent"`
}

// DefaultConfig provides default settings for the application
var DefaultConfig = Config{
	Resolvers: []string{
		"1.1.1.1", // Cloudflare
		"8.8.8.8", // Google
		"9.9.9.9", // Quad9
	},
	Retries:   3,
	Timeout:   10,
	UserAgent: "HttpCrawler/1.0",
}

// LoadConfig reads configuration from a JSON file
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

	// Apply defaults if necessary
	if len(cfg.Resolvers) == 0 {
		cfg.Resolvers = DefaultConfig.Resolvers
	}
	if cfg.Retries == 0 {
		cfg.Retries = DefaultConfig.Retries
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultConfig.Timeout
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultConfig.UserAgent
	}

	return &cfg, nil
}
