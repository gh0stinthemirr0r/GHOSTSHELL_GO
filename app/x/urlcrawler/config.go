package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Options defines the configuration structure for the URL crawler.
type Options struct {
	InputFile  string `yaml:"input_file"`  // Path to the input file
	OutputFile string `yaml:"output_file"` // Path to the output file
	JSONOutput bool   `yaml:"json_output"` // Output results in JSON format
	Threads    int    `yaml:"threads"`     // Number of threads to use
}

// LoadConfig loads configuration from a YAML file.
func LoadConfig(filePath string) (*Options, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config Options
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to a YAML file.
func SaveConfig(filePath string, config *Options) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	defer encoder.Close()

	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode config to file: %w", err)
	}

	return nil
}
