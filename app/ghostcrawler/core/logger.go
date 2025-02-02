package ghostcrawler

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger handles logging for the ghostcrawler module.
type Logger struct {
	logger *zap.Logger
}

var moduleLogger *Logger

// InitializeLogger sets up the logger for the ghostcrawler module.
func InitializeLogger(baseDir, logType, logLevel string) (*Logger, error) {
	logFilePath, err := initializeLogFile(baseDir, logType)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize log file: %w", err)
	}

	// Set log level
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(logLevel)); err != nil {
		level = zapcore.InfoLevel // Default to InfoLevel
	}

	// Encoder configuration for structured logging
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		MessageKey:     "message",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create the logger
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(&zapcore.FileWriteSyncer{Filename: logFilePath}),
		level,
	)
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	moduleLogger = &Logger{logger: zapLogger}
	zapLogger.Info("Logger initialized successfully for ghostcrawler",
		zap.String("log_file", logFilePath),
	)

	return moduleLogger, nil
}

// GetLogger returns the singleton logger instance for ghostcrawler.
func GetLogger() *Logger {
	if moduleLogger == nil {
		_, _ = InitializeLogger("ghostshell/logging", "ghostcrawler_log", "info")
	}
	return moduleLogger
}

// LogInfo logs an informational message.
func (l *Logger) LogInfo(message string, fields ...zap.Field) {
	l.logger.Info(message, fields...)
}

// LogError logs an error message.
func (l *Logger) LogError(message string, err error, fields ...zap.Field) {
	l.logger.Error(message, append(fields, zap.Error(err))...)
}

// initializeLogFile ensures the directory and log file exist.
func initializeLogFile(baseDir, logType string) (string, error) {
	today := time.Now().Format("2006-01-02")
	logDir := filepath.Join(baseDir, logType)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}
	return filepath.Join(logDir, fmt.Sprintf("%s_%s.log", logType, today)), nil
}
