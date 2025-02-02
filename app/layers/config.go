package layers

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the structure for application configuration.
type Config struct {
	OutputFormat string `json:"output_format"` // Output format: "csv", "pdf", or "json"
	OutputPath   string `json:"output_path"`   // Path for saving the output
	LogLevel     string `json:"log_level"`     // Log level: "info", "debug", or "error"
}

// LoadConfig reads the configuration from a JSON file.
func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			fmt.Printf("Warning: failed to close config file: %v\n", cerr)
		}
	}()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// validateConfig ensures that the configuration values are valid.
func validateConfig(config *Config) error {
	validOutputFormats := map[string]struct{}{
		"csv":  {},
		"pdf":  {},
		"json": {},
	}
	if _, valid := validOutputFormats[config.OutputFormat]; !valid {
		return fmt.Errorf("invalid output format: %s. Allowed formats: csv, pdf, json", config.OutputFormat)
	}
	if config.OutputPath == "" {
		return fmt.Errorf("output path cannot be empty")
	}
	validLogLevels := map[string]struct{}{
		"info":  {},
		"debug": {},
		"error": {},
	}
	if _, valid := validLogLevels[config.LogLevel]; !valid {
		return fmt.Errorf("invalid log level: %s. Allowed levels: info, debug, error", config.LogLevel)
	}
	return nil
}

// PrintConfig displays the configuration values.
func PrintConfig(config *Config) {
	fmt.Println("Configuration:")
	fmt.Printf("  Output Format: %s\n", config.OutputFormat)
	fmt.Printf("  Output Path: %s\n", config.OutputPath)
	fmt.Printf("  Log Level: %s\n", config.LogLevel)
}
