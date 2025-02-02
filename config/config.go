package config

import (
	"fmt"

	"github.com/cristalhq/aconfig"
	"go.uber.org/zap"
)

type LoggerConfig struct {
	LogLevel    string `yaml:"log_level" env:"LOG_LEVEL"`
	LogFilePath string `yaml:"log_file_path" env:"LOG_FILE_PATH"`
}

type AIConfig struct {
	ModelPath  string `yaml:"model_path"`
	NumThreads int    `yaml:"num_threads"`
	Backend    string `yaml:"backend"`
}

type ThemeConfig struct {
	Font struct {
		Family string `yaml:"family"`
		Size   int    `yaml:"size"`
		Weight string `yaml:"weight"`
	} `yaml:"font"`

	Colors struct {
		Background          string            `yaml:"background"`
		Foreground          string            `yaml:"foreground"`
		Cursor              string            `yaml:"cursor"`
		SelectionBackground string            `yaml:"selection_background"`
		SelectionForeground string            `yaml:"selection_foreground"`
		AnsiColors          map[string]string `yaml:"ansi_colors"`
	} `yaml:"colors"`

	Animations struct {
		EnableParticleEffects bool    `yaml:"enable_particle_effects"`
		ParticleCount         int     `yaml:"particle_count"`
		ParticleSpeed         float64 `yaml:"particle_speed"`
		EnableGlowEffect      bool    `yaml:"enable_glow_effect"`
	} `yaml:"animations"`

	Layout struct {
		PromptPosition string  `yaml:"prompt_position"`
		LineSpacing    float64 `yaml:"line_spacing"`
		Padding        struct {
			Top    int `yaml:"top"`
			Bottom int `yaml:"bottom"`
			Left   int `yaml:"left"`
			Right  int `yaml:"right"`
		} `yaml:"padding"`
	} `yaml:"layout"`

	Widgets struct {
		Clock         bool `yaml:"clock"`
		NetworkStatus bool `yaml:"network_status"`
		CPUUsage      bool `yaml:"cpu_usage"`
		MemoryUsage   bool `yaml:"memory_usage"`
	} `yaml:"widgets"`

	Shadows struct {
		Enabled    bool    `yaml:"enabled"`
		Intensity  float64 `yaml:"intensity"`
		BlurRadius int     `yaml:"blur_radius"`
	} `yaml:"shadows"`

	Borders struct {
		Radius    int    `yaml:"radius"`
		Thickness int    `yaml:"thickness"`
		Color     string `yaml:"color"`
	} `yaml:"borders"`
}

type AppConfig struct {
	Logger LoggerConfig `yaml:"logger"`
	AI     AIConfig     `yaml:"ai"`
}

type Theme struct {
	Theme ThemeConfig `yaml:"theme"`
}

// LoadConfig loads the system configuration from YAML files.
func LoadConfig(filePath string) (*AppConfig, error) {
	var cfg AppConfig
	loader := aconfig.LoaderFor(&cfg, aconfig.Config{
		Files: []string{filePath},
	})
	if err := loader.Load(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &cfg, nil
}

// LoadTheme loads the theme configuration from YAML files.
func LoadTheme(filePath string) (*Theme, error) {
	var theme Theme
	loader := aconfig.LoaderFor(&theme, aconfig.Config{
		Files: []string{filePath},
	})
	if err := loader.Load(); err != nil {
		return nil, fmt.Errorf("failed to load theme: %w", err)
	}
	return &theme, nil
}

// NewLogger initializes a zap logger based on the configuration.
func NewLogger(cfg LoggerConfig) (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{cfg.LogFilePath}
	config.Level = zap.NewAtomicLevelAt(parseLogLevel(cfg.LogLevel))
	return config.Build()
}

func parseLogLevel(level string) zap.AtomicLevel {
	switch level {
	case "debug":
		return zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		return zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		return zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		return zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		return zap.NewAtomicLevelAt(zap.InfoLevel)
	}
}
