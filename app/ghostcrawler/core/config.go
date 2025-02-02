package ghostcrawler

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config represents the configuration for ghostcrawler.
type Config struct {
	Concurrency int    `yaml:"concurrency" json:"concurrency"`
	Timeout     int    `yaml:"timeout" json:"timeout"`
	UserAgent   string `yaml:"user_agent" json:"user_agent"`
	OutputDir   string `yaml:"output_dir" json:"output_dir"`
}

var (
	config     *Config
	configOnce sync.Once
)

// LoadConfig loads the configuration from a YAML or JSON file.
func LoadConfig(filePath string) (*Config, error) {
	var cfg Config

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	switch ext := getFileExtension(filePath); ext {
	case "yaml", "yml":
		if err := yaml.NewDecoder(file).Decode(&cfg); err != nil {
			return nil, fmt.Errorf("failed to decode YAML config: %w", err)
		}
	case "json":
		if err := json.NewDecoder(file).Decode(&cfg); err != nil {
			return nil, fmt.Errorf("failed to decode JSON config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	applyDefaults(&cfg)
	return &cfg, nil
}

// GetConfig returns a singleton instance of the loaded configuration.
func GetConfig(filePath string) (*Config, error) {
	var err error
	configOnce.Do(func() {
		config, err = LoadConfig(filePath)
	})
	return config, err
}

// applyDefaults applies default values to the configuration if not set.
func applyDefaults(cfg *Config) {
	if cfg.Concurrency == 0 {
		cfg.Concurrency = 5
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 // seconds
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "GhostCrawler/1.0"
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "ghostshell/reporting"
	}
}

// getFileExtension extracts the file extension from a file path.
func getFileExtension(filePath string) string {
	if len(filePath) > 0 {
		parts := []rune(filePath)
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i] == '.' {
				return string(parts[i+1:])
			}
		}
	}
	return ""
}
