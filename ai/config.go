package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// ControlParameters define user-tunable generation parameters, etc.
type ControlParameters struct {
	Temperature float64 `yaml:"temperature"`
	MaxTokens   int     `yaml:"max_tokens"`
	TopP        float64 `yaml:"top_p"`
	TopK        int     `yaml:"top_k"`
	// Add more parameters as needed
}

// Config represents the overall AI configuration.
type Config struct {
	ModelPath     string            `yaml:"model_path"`
	ControlParams ControlParameters `yaml:"control_params"`
	LogLevel      string            `yaml:"log_level"`
	CacheEnabled  bool              `yaml:"cache_enabled"`
	CachePath     string            `yaml:"cache_path"`
	// Add more configuration fields as needed
}

// Validate ensures the config is logically valid.
func (c *Config) Validate() error {
	if c.ModelPath == "" {
		return fmt.Errorf("model_path cannot be empty")
	}
	if c.ControlParams.Temperature < 0.0 || c.ControlParams.Temperature > 1.0 {
		return fmt.Errorf("temperature must be between 0.0 and 1.0")
	}
	if c.ControlParams.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be greater than zero")
	}
	if c.ControlParams.TopP < 0.0 || c.ControlParams.TopP > 1.0 {
		return fmt.Errorf("top_p must be between 0.0 and 1.0")
	}
	if c.ControlParams.TopK < 0 {
		return fmt.Errorf("top_k must be non-negative")
	}
	if c.CacheEnabled && c.CachePath == "" {
		return fmt.Errorf("cache_path must be set if caching is enabled")
	}
	// Add more validation rules as needed
	return nil
}

// GetDefaultConfigPath returns the default path for ai_config.yaml
func GetDefaultConfigPath() string {
	return filepath.Join("ghostshell", "config", "ai_config.yaml")
}

// SaveConfig marshals the Config into YAML and writes it to disk.
// It logs the process using the provided Zap logger.
func SaveConfig(config *Config, filePath string, logger *zap.Logger) error {
	logger.Info("Saving AI configuration", zap.String("file_path", filePath))
	data, err := yaml.Marshal(config)
	if err != nil {
		logger.Error("Failed to marshal config to YAML", zap.Error(err))
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	// Ensure the directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Error("Failed to create config directory", zap.String("directory", dir), zap.Error(err))
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write the YAML data to the file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		logger.Error("Failed to write config file", zap.String("file_path", filePath), zap.Error(err))
		return fmt.Errorf("failed to write config file: %w", err)
	}

	logger.Info("AI configuration saved successfully", zap.String("file_path", filePath))
	return nil
}

// LoadConfig reads YAML from disk into a Config struct.
// It logs the process using the provided Zap logger.
func LoadConfig(filePath string, logger *zap.Logger) (*Config, error) {
	logger.Info("Loading AI configuration", zap.String("file_path", filePath))

	// Check if the config file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logger.Warn("Config file does not exist. Creating default configuration.", zap.String("file_path", filePath))
		defaultConfig := GetDefaultConfig()
		if err := SaveConfig(defaultConfig, filePath, logger); err != nil {
			logger.Error("Failed to save default config", zap.Error(err))
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
		return defaultConfig, nil
	} else if err != nil {
		logger.Error("Error checking config file existence", zap.String("file_path", filePath), zap.Error(err))
		return nil, fmt.Errorf("error checking config file: %w", err)
	}

	// Read the config file
	data, err := os.ReadFile(filePath)
	if err != nil {
		logger.Error("Failed to read config file", zap.String("file_path", filePath), zap.Error(err))
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal YAML into Config struct
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		logger.Error("Failed to unmarshal config YAML", zap.Error(err))
		return nil, fmt.Errorf("failed to unmarshal config YAML: %w", err)
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		logger.Error("Configuration validation failed", zap.Error(err))
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	logger.Info("AI configuration loaded successfully", zap.String("file_path", filePath))
	return &cfg, nil
}

// GetDefaultConfig returns a Config struct with default values.
func GetDefaultConfig() *Config {
	return &Config{
		ModelPath: "ai/models/default.gguf",
		ControlParams: ControlParameters{
			Temperature: 0.7,
			MaxTokens:   512,
			TopP:        0.9,
			TopK:        40,
		},
		LogLevel:     "info",
		CacheEnabled: true,
		CachePath:    "ai/cache",
		// Initialize other default fields as needed
	}
}

// LoadOrCreateConfig attempts to load the configuration from the given path.
// If the configuration file does not exist, it creates a default config and saves it.
// It logs all steps using the provided Zap logger.
func LoadOrCreateConfig(filePath string, logger *zap.Logger) (*Config, error) {
	cfg, err := LoadConfig(filePath, logger)
	if err != nil {
		logger.Error("Failed to load or create AI configuration", zap.Error(err))
		return nil, err
	}
	return cfg, nil
}

// Singleton pattern for configuration management (optional)
// Ensures that the configuration is loaded only once and is thread-safe.
type ConfigManager struct {
	config *Config
	once   sync.Once
	err    error
}

// GetConfigManager returns a singleton instance of ConfigManager.
func GetConfigManager(logger *zap.Logger) *ConfigManager {
	return &ConfigManager{}
}

// LoadConfigSingleton loads the configuration using the singleton pattern.
// Subsequent calls return the already loaded configuration.
func (cm *ConfigManager) LoadConfigSingleton(filePath string, logger *zap.Logger) (*Config, error) {
	cm.once.Do(func() {
		cm.config, cm.err = LoadOrCreateConfig(filePath, logger)
	})
	return cm.config, cm.err
}
