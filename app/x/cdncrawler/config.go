// Package cdnscanner provides configuration management for the CDN scanner.
package cdncrawler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
)

// Config represents the structure of the configuration settings.
type Config struct {
	OutputPath string   `json:"output_path"` // Default path for output files
	CDNList    []string `json:"cdn_list"`    // List of known CDN IP ranges or domains
	LogLevel   string   `json:"log_level"`   // Log level: debug, info, warn, error
	Timeout    int      `json:"timeout"`     // Request timeout in seconds
}

// configInstance holds the singleton instance of the configuration.
var configInstance *Config
var once sync.Once

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		OutputPath: "./output",
		CDNList:    []string{"cloudflare.com", "akamai.com", "fastly.com"},
		LogLevel:   "info",
		Timeout:    10,
	}
}

// LoadConfig reads configuration from a JSON file.
func LoadConfig(filePath string) (*Config, error) {
	configFile, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open configuration file: %w", err)
	}
	defer configFile.Close()

	data, err := ioutil.ReadAll(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration file: %w", err)
	}

	return &config, nil
}

// SaveConfig writes the current configuration to a JSON file.
func (c *Config) SaveConfig(filePath string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to save configuration file: %w", err)
	}

	fmt.Printf("Configuration saved successfully to: %s\n", filePath)
	return nil
}

// GetConfig returns the singleton configuration instance.
// It initializes the configuration with default values if not already loaded.
func GetConfig() *Config {
	once.Do(func() {
		configInstance = DefaultConfig()
	})
	return configInstance
}

// UpdateConfig updates the current configuration values.
func (c *Config) UpdateConfig(newConfig *Config) {
	if newConfig.OutputPath != "" {
		c.OutputPath = newConfig.OutputPath
	}
	if len(newConfig.CDNList) > 0 {
		c.CDNList = newConfig.CDNList
	}
	if newConfig.LogLevel != "" {
		c.LogLevel = newConfig.LogLevel
	}
	if newConfig.Timeout > 0 {
		c.Timeout = newConfig.Timeout
	}
}

// ValidateConfig ensures the configuration is valid.
func (c *Config) ValidateConfig() error {
	if c.OutputPath == "" {
		return errors.New("output path cannot be empty")
	}
	if len(c.CDNList) == 0 {
		return errors.New("CDN list cannot be empty")
	}
	if c.LogLevel == "" {
		return errors.New("log level cannot be empty")
	}
	if c.Timeout <= 0 {
		return errors.New("timeout must be greater than zero")
	}
	return nil
}

// Example usage:
// func main() {
//     config := GetConfig()
//     err := config.ValidateConfig()
//     if err != nil {
//         fmt.Println("Invalid configuration:", err)
//     } else {
//         fmt.Println("Configuration is valid!")
//     }
//
//     config.SaveConfig("config.json")
// }
