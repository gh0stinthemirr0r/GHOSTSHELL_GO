package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// Profile represents a user profile with authentication and custom settings.
type Profile struct {
	Name     string                 `json:"name"`
	Username string                 `json:"username"`
	Password string                 `json:"password"` // Consider hashing passwords securely.
	Settings map[string]interface{} `json:"settings"`
}

// ProfileManagerConfig holds configuration parameters for ProfileManager.
type ProfileManagerConfig struct {
	FilePath          string // Path to the JSON file storing profiles.
	PrometheusMetrics bool   // Enable Prometheus metrics collection.
}

// ProfileManager manages user profiles with thread-safe operations.
type ProfileManager struct {
	config   ProfileManagerConfig
	profiles map[string]Profile
	mutex    sync.RWMutex

	// Prometheus metrics
	totalProfiles     prometheus.Gauge
	profileAdditions  prometheus.Counter
	profileRemovals   prometheus.Counter
	profileFetches    prometheus.Counter
	profileFetchFails prometheus.Counter
}

// Initialize logger with dynamic file naming
var logger *zap.SugaredLogger

func init() {
	currentTime := time.Now().UTC().Format("20060102_150405")
	logFileName1 := fmt.Sprintf("profiles_log_%s.log", currentTime)

	// Configure zap to write to both log files and stdout
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{
		logFileName1,
		"stdout", // Also write logs to the console
	}

	// Build the logger
	log, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}

	// Set the global logger
	logger = log.Sugar()

	// Log initialization information
	logger.Infof("Logger initialized with files: %s and %s", logFileName1)
}

// NewProfileManager initializes and returns a new ProfileManager instance.
func NewProfileManager(cfg ProfileManagerConfig, logger *zap.Logger) (*ProfileManager, error) {
	if cfg.FilePath == "" {
		return nil, errors.New("file path cannot be empty")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}

	pm := &ProfileManager{
		config:   cfg,
		profiles: make(map[string]Profile),
		logger:   logger,
	}

	// Initialize Prometheus metrics if enabled
	if cfg.PrometheusMetrics {
		pm.initPrometheusMetrics()
	}

	// Load existing profiles from file
	if err := pm.LoadProfiles(); err != nil {
		logger.Error("Failed to load profiles", zap.Error(err))
		return nil, fmt.Errorf("failed to load profiles: %w", err)
	}

	return pm, nil
}

// initPrometheusMetrics initializes Prometheus metrics for ProfileManager.
func (pm *ProfileManager) initPrometheusMetrics() {
	pm.totalProfiles = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "profiles_total",
		Help: "Total number of profiles loaded.",
	})
	pm.profileAdditions = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "profiles_additions_total",
		Help: "Total number of profiles added.",
	})
	pm.profileRemovals = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "profiles_removals_total",
		Help: "Total number of profiles removed.",
	})
	pm.profileFetches = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "profiles_fetches_total",
		Help: "Total number of profile fetch attempts.",
	})
	pm.profileFetchFails = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "profiles_fetch_failures_total",
		Help: "Total number of failed profile fetch attempts.",
	})

	// Register metrics
	prometheus.MustRegister(pm.totalProfiles, pm.profileAdditions, pm.profileRemovals, pm.profileFetches, pm.profileFetchFails)
}

// LoadProfiles loads profiles from the JSON file into memory.
func (pm *ProfileManager) LoadProfiles() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	file, err := os.Open(pm.config.FilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			pm.logger.Info("Profile file does not exist. Starting with empty profiles.")
			return nil // No profiles to load initially
		}
		return err
	}
	defer file.Close()

	data, err := os.ReadFile(pm.config.FilePath)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		pm.logger.Info("Profile file is empty. Starting with empty profiles.")
		return nil
	}

	if err := json.Unmarshal(data, &pm.profiles); err != nil {
		return err
	}

	pm.totalProfiles.Set(float64(len(pm.profiles)))
	pm.logger.Info("Loaded profiles", zap.Int("count", len(pm.profiles)), zap.String("file", pm.config.FilePath))
	return nil
}

// SaveProfiles saves the current profiles to the JSON file.
func (pm *ProfileManager) SaveProfiles() error {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	data, err := json.MarshalIndent(pm.profiles, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(pm.config.FilePath, data, 0644); err != nil {
		return err
	}

	pm.logger.Info("Saved profiles", zap.Int("count", len(pm.profiles)), zap.String("file", pm.config.FilePath))
	return nil
}

// AddProfile adds a new profile. Returns an error if the profile already exists.
func (pm *ProfileManager) AddProfile(profile Profile) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if _, exists := pm.profiles[profile.Name]; exists {
		pm.logger.Warn("Attempted to add duplicate profile", zap.String("profile", profile.Name))
		return errors.New("profile already exists")
	}

	pm.profiles[profile.Name] = profile
	pm.profileAdditions.Inc()
	pm.totalProfiles.Inc()

	if err := pm.SaveProfiles(); err != nil {
		pm.logger.Error("Failed to save profiles after adding", zap.Error(err))
		return fmt.Errorf("failed to save profiles: %w", err)
	}

	pm.logger.Info("Added new profile", zap.String("profile", profile.Name))
	return nil
}

// RemoveProfile removes an existing profile by name. Returns an error if the profile does not exist.
func (pm *ProfileManager) RemoveProfile(name string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if _, exists := pm.profiles[name]; !exists {
		pm.logger.Warn("Attempted to remove non-existent profile", zap.String("profile", name))
		return errors.New("profile not found")
	}

	delete(pm.profiles, name)
	pm.profileRemovals.Inc()
	pm.totalProfiles.Dec()

	if err := pm.SaveProfiles(); err != nil {
		pm.logger.Error("Failed to save profiles after removing", zap.Error(err))
		return fmt.Errorf("failed to save profiles: %w", err)
	}

	pm.logger.Info("Removed profile", zap.String("profile", name))
	return nil
}

// GetProfile retrieves a profile by name. Returns an error if the profile does not exist.
func (pm *ProfileManager) GetProfile(name string) (Profile, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	pm.profileFetches.Inc()

	profile, exists := pm.profiles[name]
	if !exists {
		pm.profileFetchFails.Inc()
		pm.logger.Warn("Profile not found", zap.String("profile", name))
		return Profile{}, errors.New("profile not found")
	}

	pm.logger.Info("Fetched profile", zap.String("profile", name))
	return profile, nil
}

// ListProfiles returns a slice of all profile names.
func (pm *ProfileManager) ListProfiles() []string {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	names := make([]string, 0, len(pm.profiles))
	for name := range pm.profiles {
		names = append(names, name)
	}

	pm.logger.Info("Listing profiles", zap.Int("count", len(names)))
	return names
}
