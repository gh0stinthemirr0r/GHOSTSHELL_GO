// settings.go

// Package storage provides functionalities for managing application-wide configuration settings.
// It allows loading, saving, retrieving, and updating settings with thread-safe operations.
// It integrates with Prometheus for metrics, supports structured logging, and includes
// comprehensive error handling.
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// Settings represents application-wide configuration settings.
type Settings struct {
	Theme        string                 `json:"theme"`
	Language     string                 `json:"language"`
	AutoUpdate   bool                   `json:"auto_update"`
	MaxRetries   int                    `json:"max_retries"`
	OtherConfigs map[string]interface{} `json:"other_configs"`
}

// SettingsManagerConfig holds configuration parameters for SettingsManager.
type SettingsManagerConfig struct {
	// FilePath is the path to the JSON file storing settings.
	FilePath string
	// Logger is used for logging settings activities and errors.
	Logger *logrus.Logger
	// PrometheusMetrics enables Prometheus metrics collection.
	PrometheusMetrics bool
}

// SettingsManager handles the storage and retrieval of application settings.
type SettingsManager struct {
	config   SettingsManagerConfig
	settings Settings
	filePath string
	lock     sync.RWMutex

	// Prometheus metrics
	totalLoads        prometheus.Counter
	totalSaves        prometheus.Counter
	saveFailures      prometheus.Counter
	loadFailures      prometheus.Counter
	settingUpdates    prometheus.Counter
	settingFetches    prometheus.Counter
	settingFetchFails prometheus.Counter
}

// NewSettingsManager initializes and returns a new SettingsManager instance.
func NewSettingsManager(cfg SettingsManagerConfig) (*SettingsManager, error) {
	if cfg.FilePath == "" {
		return nil, errors.New("file path cannot be empty")
	}
	if cfg.Logger == nil {
		cfg.Logger = logrus.New()
		cfg.Logger.SetFormatter(&logrus.JSONFormatter{})
		cfg.Logger.SetLevel(logrus.InfoLevel)
	}

	sm := &SettingsManager{
		config:   cfg,
		filePath: cfg.FilePath,
	}

	// Initialize Prometheus metrics if enabled
	if cfg.PrometheusMetrics {
		sm.initPrometheusMetrics()
	}

	// Load existing settings from file
	if err := sm.Load(); err != nil {
		cfg.Logger.Errorf("Failed to load settings: %v", err)
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}

	return sm, nil
}

// initPrometheusMetrics initializes Prometheus metrics for SettingsManager.
func (sm *SettingsManager) initPrometheusMetrics() {
	sm.totalLoads = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "settings_total_loads",
		Help: "Total number of settings load operations.",
	})
	sm.totalSaves = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "settings_total_saves",
		Help: "Total number of settings save operations.",
	})
	sm.saveFailures = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "settings_save_failures_total",
		Help: "Total number of failed settings save operations.",
	})
	sm.loadFailures = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "settings_load_failures_total",
		Help: "Total number of failed settings load operations.",
	})
	sm.settingUpdates = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "settings_updates_total",
		Help: "Total number of settings updates.",
	})
	sm.settingFetches = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "settings_fetches_total",
		Help: "Total number of settings fetch attempts.",
	})
	sm.settingFetchFails = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "settings_fetch_failures_total",
		Help: "Total number of failed settings fetch attempts.",
	})

	// Register metrics
	prometheus.MustRegister(
		sm.totalLoads,
		sm.totalSaves,
		sm.saveFailures,
		sm.loadFailures,
		sm.settingUpdates,
		sm.settingFetches,
		sm.settingFetchFails,
	)
}

// Load loads settings from the JSON file into memory.
func (sm *SettingsManager) Load() error {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	sm.totalLoads.Inc()

	file, err := os.Open(sm.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			sm.config.Logger.Infof("Settings file does not exist. Initializing with default settings.")
			sm.settings = Settings{
				Theme:        "default",
				Language:     "en",
				AutoUpdate:   true,
				MaxRetries:   3,
				OtherConfigs: make(map[string]interface{}),
			}
			return sm.Save()
		}
		sm.loadFailures.Inc()
		return err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		sm.loadFailures.Inc()
		return err
	}

	if len(data) == 0 {
		sm.config.Logger.Infof("Settings file is empty. Initializing with default settings.")
		sm.settings = Settings{
			Theme:        "default",
			Language:     "en",
			AutoUpdate:   true,
			MaxRetries:   3,
			OtherConfigs: make(map[string]interface{}),
		}
		return sm.Save()
	}

	if err := json.Unmarshal(data, &sm.settings); err != nil {
		sm.loadFailures.Inc()
		return err
	}

	sm.config.Logger.Infof("Loaded settings from %s", sm.filePath)
	return nil
}

// Save saves the current settings to the JSON file.
func (sm *SettingsManager) Save() error {
	sm.lock.RLock()
	defer sm.lock.RUnlock()

	sm.totalSaves.Inc()

	data, err := json.MarshalIndent(sm.settings, "", "  ")
	if err != nil {
		sm.saveFailures.Inc()
		return err
	}

	err = ioutil.WriteFile(sm.filePath, data, 0644)
	if err != nil {
		sm.saveFailures.Inc()
		return err
	}

	sm.config.Logger.Infof("Saved settings to %s", sm.filePath)
	return nil
}

// GetSettings retrieves the current settings.
func (sm *SettingsManager) GetSettings() Settings {
	sm.lock.RLock()
	defer sm.lock.RUnlock()

	sm.settingFetches.Inc()

	sm.config.Logger.Infof("Fetched current settings")
	return sm.settings
}

// UpdateSettings updates the settings with new values and saves them to the file.
func (sm *SettingsManager) UpdateSettings(newSettings Settings) error {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	sm.settings = newSettings
	sm.settingUpdates.Inc()

	sm.config.Logger.Infof("Updated settings")

	if err := sm.Save(); err != nil {
		sm.config.Logger.Errorf("Failed to save settings after update: %v", err)
		return fmt.Errorf("failed to save settings: %w", err)
	}

	return nil
}

// GetSetting retrieves a specific setting value by key.
// Returns an error if the setting does not exist.
func (sm *SettingsManager) GetSetting(key string) (interface{}, error) {
	sm.lock.RLock()
	defer sm.lock.RUnlock()

	sm.settingFetches.Inc()

	switch key {
	case "theme":
		sm.config.Logger.Infof("Fetched setting: theme")
		return sm.settings.Theme, nil
	case "language":
		sm.config.Logger.Infof("Fetched setting: language")
		return sm.settings.Language, nil
	case "auto_update":
		sm.config.Logger.Infof("Fetched setting: auto_update")
		return sm.settings.AutoUpdate, nil
	case "max_retries":
		sm.config.Logger.Infof("Fetched setting: max_retries")
		return sm.settings.MaxRetries, nil
	default:
		value, exists := sm.settings.OtherConfigs[key]
		if !exists {
			sm.settingFetchFails.Inc()
			sm.config.Logger.Warnf("Attempted to fetch non-existent setting: %s", key)
			return nil, errors.New("setting not found")
		}
		sm.config.Logger.Infof("Fetched setting: %s", key)
		return value, nil
	}
}

// SetSetting sets a specific setting value by key and saves the settings to the file.
// Returns an error if the key is invalid or the value type is incorrect.
func (sm *SettingsManager) SetSetting(key string, value interface{}) error {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	sm.settingUpdates.Inc()

	switch key {
	case "theme":
		v, ok := value.(string)
		if !ok {
			sm.config.Logger.Warnf("Invalid type for setting 'theme': %T", value)
			return errors.New("invalid type for setting 'theme'")
		}
		sm.settings.Theme = v
		sm.config.Logger.Infof("Set setting 'theme' to '%s'", v)
	case "language":
		v, ok := value.(string)
		if !ok {
			sm.config.Logger.Warnf("Invalid type for setting 'language': %T", value)
			return errors.New("invalid type for setting 'language'")
		}
		sm.settings.Language = v
		sm.config.Logger.Infof("Set setting 'language' to '%s'", v)
	case "auto_update":
		v, ok := value.(bool)
		if !ok {
			sm.config.Logger.Warnf("Invalid type for setting 'auto_update': %T", value)
			return errors.New("invalid type for setting 'auto_update'")
		}
		sm.settings.AutoUpdate = v
		sm.config.Logger.Infof("Set setting 'auto_update' to '%v'", v)
	case "max_retries":
		v, ok := value.(int)
		if !ok {
			sm.config.Logger.Warnf("Invalid type for setting 'max_retries': %T", value)
			return errors.New("invalid type for setting 'max_retries'")
		}
		sm.settings.MaxRetries = v
		sm.config.Logger.Infof("Set setting 'max_retries' to '%d'", v)
	default:
		sm.settings.OtherConfigs[key] = value
		sm.config.Logger.Infof("Set setting '%s' to '%v'", key, value)
	}

	if err := sm.Save(); err != nil {
		sm.config.Logger.Errorf("Failed to save settings after setting '%s': %v", key, err)
		return fmt.Errorf("failed to save settings: %w", err)
	}

	return nil
}
