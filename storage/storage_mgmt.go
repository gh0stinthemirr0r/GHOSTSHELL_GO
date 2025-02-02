// storage_manager.go

package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	oqs_vault "ghostshell/oqs/vault"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// Initialize logger with dynamic file naming
var logger *zap.SugaredLogger

func init() {
	currentTime := time.Now().UTC().Format("20060102_150405")
	logFileName := fmt.Sprintf("profiles_log_%s.log", currentTime)

	// Configure zap to write to both log files and stdout
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{
		logFileName,
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
	logger.Infof("Logger initialized with file: %s", logFileName)
}

// StorageManagerConfig holds configuration parameters for StorageManager.
type StorageManagerConfig struct {
	PrometheusMetrics   bool
	MaintenanceInterval time.Duration
}

// StorageManager manages storage paths with post-quantum security.
type StorageManager struct {
	config StorageManagerConfig
	paths  map[string][]byte // Encrypted paths using OQS Vault
	mu     sync.RWMutex

	// Prometheus metrics
	totalPaths        prometheus.Counter
	pathAdditions     prometheus.Counter
	pathRemovals      prometheus.Counter
	maintenanceRuns   prometheus.Counter
	maintenanceErrors prometheus.Counter

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewStorageManager initializes and returns a new instance of StorageManager.
func NewStorageManager(cfg StorageManagerConfig) (*StorageManager, error) {
	sm := &StorageManager{
		config: cfg,
		paths:  make(map[string][]byte),
	}

	// Initialize Prometheus metrics if enabled
	if cfg.PrometheusMetrics {
		sm.initPrometheusMetrics()
	}

	// Initialize context for graceful shutdown
	sm.ctx, sm.cancel = context.WithCancel(context.Background())

	return sm, nil
}

// initPrometheusMetrics initializes Prometheus metrics for StorageManager.
func (sm *StorageManager) initPrometheusMetrics() {
	sm.totalPaths = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "storage_total_paths",
		Help: "Total number of storage paths managed.",
	})
	sm.pathAdditions = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "storage_path_additions_total",
		Help: "Total number of storage paths added.",
	})
	sm.pathRemovals = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "storage_path_removals_total",
		Help: "Total number of storage paths removed.",
	})
	sm.maintenanceRuns = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "storage_maintenance_runs_total",
		Help: "Total number of storage maintenance runs.",
	})
	sm.maintenanceErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "storage_maintenance_errors_total",
		Help: "Total number of errors during storage maintenance.",
	})

	prometheus.MustRegister(
		sm.totalPaths,
		sm.pathAdditions,
		sm.pathRemovals,
		sm.maintenanceRuns,
		sm.maintenanceErrors,
	)
}

// AddPath adds a storage path to the manager using post-quantum encryption.
func (sm *StorageManager) AddPath(key, path string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.paths[key]; exists {
		logger.Warnf("Duplicate storage path attempt: %s", key)
		return errors.New("storage path already exists")
	}

	// Encrypt the path using OQS Vault
	encryptedPath, err := oqs_vault.Vault.VaultEncrypt([]byte(path))
	if err != nil {
		logger.Errorf("Failed to encrypt storage path %s: %v", path, err)
		return fmt.Errorf("failed to encrypt storage path %s: %w", path, err)
	}

	sm.paths[key] = encryptedPath
	sm.pathAdditions.Inc()
	sm.totalPaths.Inc()
	logger.Infof("Encrypted storage path added: %s", key)

	return nil
}

// RemovePath removes a storage path from the manager by key.
func (sm *StorageManager) RemovePath(key string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	encryptedPath, exists := sm.paths[key]
	if !exists {
		logger.Warnf("Attempt to remove non-existent storage path: %s", key)
		return errors.New("storage path not found")
	}

	// Overwrite and remove the encrypted path
	delete(sm.paths, key)
	sm.pathRemovals.Inc()
	sm.totalPaths.Dec()

	logger.Infof("Storage path removed: %s", key)
	return nil
}

// GetPath retrieves and decrypts a storage path by key.
func (sm *StorageManager) GetPath(key string) (string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	encryptedPath, exists := sm.paths[key]
	if !exists {
		return "", fmt.Errorf("storage path not found for key: %s", key)
	}

	decryptedPath, err := oqs_vault.Vault.VaultDecrypt(encryptedPath)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt storage path for key %s: %w", key, err)
	}

	return string(decryptedPath), nil
}

// PerformStorageMaintenance runs periodic tasks with post-quantum security.
func (sm *StorageManager) PerformStorageMaintenance() {
	sm.wg.Add(1)
	defer sm.wg.Done()
	ticker := time.NewTicker(sm.config.MaintenanceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			logger.Info("Storage maintenance stopped.")
			return
		case <-ticker.C:
			sm.runMaintenance()
		}
	}
}

// runMaintenance performs secure cleanup and maintenance tasks.
func (sm *StorageManager) runMaintenance() {
	sm.maintenanceRuns.Inc()
	logger.Info("Starting secure storage maintenance...")

	sm.mu.RLock()
	pathsCopy := make(map[string][]byte)
	for key, path := range sm.paths {
		pathsCopy[key] = path
	}
	sm.mu.RUnlock()

	for key, encPath := range pathsCopy {
		decryptedPath, err := oqs_vault.Vault.VaultDecrypt(encPath)
		if err != nil {
			sm.maintenanceErrors.Inc()
			logger.Errorf("Failed to decrypt path during maintenance for key %s: %v", key, err)
			continue
		}

		// Secure cleanup of temporary files
		err = filepath.Walk(string(decryptedPath), func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				logger.Errorf("Error accessing file %s: %v", filePath, err)
				return nil
			}
			if filepath.Ext(filePath) == ".tmp" {
				logger.Infof("Securely removing temp file: %s", filePath)
				if err := os.Remove(filePath); err != nil {
					logger.Errorf("Failed to remove file %s: %v", filePath, err)
				}
			}
			return nil
		})
		if err != nil {
			sm.maintenanceErrors.Inc()
			logger.Errorf("Error during cleanup for path %s: %v", string(decryptedPath), err)
		}
	}

	logger.Info("Secure storage maintenance completed.")
}
