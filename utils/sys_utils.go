package utils

import (
	"encoding/base64"
	"encoding/hex"
	"io/ioutil"
	"os"
	"runtime/debug"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// UtilsConfig holds configuration parameters for all utilities.
type UtilsConfig struct {
	Logger              *zap.SugaredLogger
	PrometheusMetrics   bool
	ExitOnCriticalError bool
}

// SysUtils combines encoding, error handling, and file operations.
type SysUtils struct {
	config UtilsConfig

	// Prometheus metrics
	hexEncodeCount    prometheus.Counter
	hexDecodeCount    prometheus.Counter
	base64EncodeCount prometheus.Counter
	base64DecodeCount prometheus.Counter
	fileExistsCount   prometheus.Counter
	dirExistsCount    prometheus.Counter
	readFileCount     prometheus.Counter
	writeFileCount    prometheus.Counter
	deleteFileCount   prometheus.Counter

	// Mutex to ensure thread-safe operations
	mu sync.Mutex
}

// NewSysUtils initializes and returns a new SysUtils instance.
func NewSysUtils(cfg UtilsConfig) (*SysUtils, error) {
	if cfg.Logger == nil {
		logger, err := zap.NewProduction()
		if err != nil {
			return nil, err
		}
		cfg.Logger = logger.Sugar()
	}

	su := &SysUtils{
		config: cfg,
	}

	if cfg.PrometheusMetrics {
		su.initPrometheusMetrics()
	}

	return su, nil
}

// initPrometheusMetrics initializes Prometheus metrics.
func (su *SysUtils) initPrometheusMetrics() {
	su.hexEncodeCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "utils_hex_encode_total",
		Help: "Total number of Hex encoding operations.",
	})
	su.hexDecodeCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "utils_hex_decode_total",
		Help: "Total number of Hex decoding operations.",
	})
	su.base64EncodeCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "utils_base64_encode_total",
		Help: "Total number of Base64 encoding operations.",
	})
	su.base64DecodeCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "utils_base64_decode_total",
		Help: "Total number of Base64 decoding operations.",
	})
	su.fileExistsCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "utils_file_exists_total",
		Help: "Total number of FileExists checks performed.",
	})
	su.dirExistsCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "utils_dir_exists_total",
		Help: "Total number of DirectoryExists checks performed.",
	})
	su.readFileCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "utils_read_file_total",
		Help: "Total number of ReadFile operations performed.",
	})
	su.writeFileCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "utils_write_file_total",
		Help: "Total number of WriteFile operations performed.",
	})
	su.deleteFileCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "utils_delete_file_total",
		Help: "Total number of DeleteFile operations performed.",
	})

	prometheus.MustRegister(
		su.hexEncodeCount,
		su.hexDecodeCount,
		su.base64EncodeCount,
		su.base64DecodeCount,
		su.fileExistsCount,
		su.dirExistsCount,
		su.readFileCount,
		su.writeFileCount,
		su.deleteFileCount,
	)
}

// EncodeToHex encodes data to a hexadecimal string.
func (su *SysUtils) EncodeToHex(data []byte) string {
	su.mu.Lock()
	defer su.mu.Unlock()

	su.hexEncodeCount.Inc()
	su.config.Logger.Debugf("Encoding data to Hex: %x", data)
	return hex.EncodeToString(data)
}

// DecodeFromHex decodes a hexadecimal string to bytes.
func (su *SysUtils) DecodeFromHex(hexString string) ([]byte, error) {
	su.mu.Lock()
	defer su.mu.Unlock()

	su.hexDecodeCount.Inc()
	decoded, err := hex.DecodeString(hexString)
	if err != nil {
		su.config.Logger.Errorf("Failed to decode Hex: %v", err)
		return nil, err
	}
	return decoded, nil
}

// EncodeToBase64 encodes data to a Base64 string.
func (su *SysUtils) EncodeToBase64(data []byte) string {
	su.mu.Lock()
	defer su.mu.Unlock()

	su.base64EncodeCount.Inc()
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeFromBase64 decodes a Base64 string to bytes.
func (su *SysUtils) DecodeFromBase64(base64String string) ([]byte, error) {
	su.mu.Lock()
	defer su.mu.Unlock()

	su.base64DecodeCount.Inc()
	decoded, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		su.config.Logger.Errorf("Failed to decode Base64: %v", err)
		return nil, err
	}
	return decoded, nil
}

// FileExists checks if a file exists.
func (su *SysUtils) FileExists(path string) bool {
	su.mu.Lock()
	defer su.mu.Unlock()

	su.fileExistsCount.Inc()
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// DirectoryExists checks if a directory exists.
func (su *SysUtils) DirectoryExists(path string) bool {
	su.mu.Lock()
	defer su.mu.Unlock()

	su.dirExistsCount.Inc()
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// ReadFile reads content from a file.
func (su *SysUtils) ReadFile(path string) (string, error) {
	su.mu.Lock()
	defer su.mu.Unlock()

	su.readFileCount.Inc()
	content, err := ioutil.ReadFile(path)
	if err != nil {
		su.config.Logger.Errorf("Failed to read file: %v", err)
		return "", err
	}
	return string(content), nil
}

// WriteFile writes content to a file.
func (su *SysUtils) WriteFile(path string, content string) error {
	su.mu.Lock()
	defer su.mu.Unlock()

	su.writeFileCount.Inc()
	return ioutil.WriteFile(path, []byte(content), 0644)
}

// DeleteFile deletes a file.
func (su *SysUtils) DeleteFile(path string) error {
	su.mu.Lock()
	defer su.mu.Unlock()

	su.deleteFileCount.Inc()
	return os.Remove(path)
}

// RecoverAndLog handles panics and logs the stack trace.
func (su *SysUtils) RecoverAndLog() {
	if r := recover(); r != nil {
		su.config.Logger.Errorf("Recovered from panic: %v", r)
		su.config.Logger.Debugf("Stack trace:\n%s", debug.Stack())
		if su.config.ExitOnCriticalError {
			os.Exit(1)
		}
	}
}
