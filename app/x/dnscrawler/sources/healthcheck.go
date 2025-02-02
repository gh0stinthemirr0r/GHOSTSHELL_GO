package sources

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Constants & Paths
const (
	LogDir = "ghostshell/logging"
)

// DNSRecord represents a DNS record with host, type, and value.
type DNSRecord struct {
	Host  string
	Type  string
	Value string
}

// DoHealthCheck performs a system and network health check using native methods and Zap logger.
func DoHealthCheck(configFilePath string) string {
	// Initialize Zap logger
	logger, err := setupLogger()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	var result strings.Builder

	logger.Info("Starting system and network health check")

	// Operating System
	osName := runtime.GOOS
	result.WriteString(fmt.Sprintf("Operating System: %s\n", osName))
	logger.Info("Operating System", zap.String("OS", osName))

	// Architecture
	arch := runtime.GOARCH
	result.WriteString(fmt.Sprintf("Architecture: %s\n", arch))
	logger.Info("Architecture", zap.String("Architecture", arch))

	// Go Version
	goVersion := runtime.Version()
	result.WriteString(fmt.Sprintf("Go Version: %s\n", goVersion))
	logger.Info("Go Version", zap.String("GoVersion", goVersion))

	// Check if the configuration file is readable
	readable, err := isReadable(configFilePath)
	if readable {
		msg := fmt.Sprintf("Config file \"%s\" Read => OK\n", configFilePath)
		result.WriteString(msg)
		logger.Info("Config file readable", zap.String("file", configFilePath))
	} else {
		msg := fmt.Sprintf("Config file \"%s\" Read => ERROR (%s)\n", configFilePath, err.Error())
		result.WriteString(msg)
		logger.Error("Config file not readable", zap.String("file", configFilePath), zap.Error(err))
	}

	// Check if the configuration file is writable
	writable, err := isWritable(configFilePath)
	if writable {
		msg := fmt.Sprintf("Config file \"%s\" Write => OK\n", configFilePath)
		result.WriteString(msg)
		logger.Info("Config file writable", zap.String("file", configFilePath))
	} else {
		msg := fmt.Sprintf("Config file \"%s\" Write => ERROR (%s)\n", configFilePath, err.Error())
		result.WriteString(msg)
		logger.Error("Config file not writable", zap.String("file", configFilePath), zap.Error(err))
	}

	// Check IPv4 connectivity
	ipv4Conn, err := net.DialTimeout("tcp4", "scanme.sh:80", 5*time.Second)
	if ipv4Conn != nil {
		result.WriteString("IPv4 connectivity => OK\n")
		logger.Info("IPv4 connectivity", zap.String("address", "scanme.sh:80"))
		ipv4Conn.Close()
	} else {
		msg := fmt.Sprintf("IPv4 connectivity => ERROR (%s)\n", err.Error())
		result.WriteString(msg)
		logger.Error("IPv4 connectivity failed", zap.String("address", "scanme.sh:80"), zap.Error(err))
	}

	// Check IPv6 connectivity
	ipv6Conn, err := net.DialTimeout("tcp6", "scanme.sh:80", 5*time.Second)
	if ipv6Conn != nil {
		result.WriteString("IPv6 connectivity => OK\n")
		logger.Info("IPv6 connectivity", zap.String("address", "scanme.sh:80"))
		ipv6Conn.Close()
	} else {
		msg := fmt.Sprintf("IPv6 connectivity => ERROR (%s)\n", err.Error())
		result.WriteString(msg)
		logger.Error("IPv6 connectivity failed", zap.String("address", "scanme.sh:80"), zap.Error(err))
	}

	// Check UDP connectivity
	udpConn, err := net.DialTimeout("udp", "scanme.sh:53", 5*time.Second)
	if udpConn != nil {
		result.WriteString("UDP connectivity => OK\n")
		logger.Info("UDP connectivity", zap.String("address", "scanme.sh:53"))
		udpConn.Close()
	} else {
		msg := fmt.Sprintf("UDP connectivity => ERROR (%s)\n", err.Error())
		result.WriteString(msg)
		logger.Error("UDP connectivity failed", zap.String("address", "scanme.sh:53"), zap.Error(err))
	}

	return result.String()
}

// isReadable checks if the file at the given path is readable.
func isReadable(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	return true, nil
}

// isWritable checks if the file at the given path is writable.
func isWritable(path string) (bool, error) {
	// Attempt to open the file in append mode
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return false, err
	}
	defer file.Close()
	return true, nil
}

// setupLogger initializes a Zap logger with a timestamped log file in ISO8601 format.
func setupLogger() (*zap.Logger, error) {
	if err := os.MkdirAll(LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	// ISO8601 timestamp e.g., 2023-10-25T15-30-45Z
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	logFileName := fmt.Sprintf("dnscrawler_log_%s.log", strings.ReplaceAll(timestamp, ":", "-"))
	logFilePath := filepath.Join(LogDir, logFileName)

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logFilePath, "stdout"}

	logger, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %v", err)
	}
	return logger, nil
}
