package dnscrawler

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Resolvers []string `json:"resolvers"`
	Retries   int      `json:"retries"`
}

// loadConfig reads configuration from a JSON file
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

	// Apply default resolvers if none are provided
	if len(cfg.Resolvers) == 0 {
		cfg.Resolvers = []string{
			"1.1.1.1:53", // Cloudflare
			"8.8.8.8:53", // Google
			"9.9.9.9:53", // Quad9
		}
	}

	// Apply default retries if not set
	if cfg.Retries == 0 {
		cfg.Retries = 3
	}

	return &cfg, nil
}
