package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LoggerManager manages Zap loggers and integrates agent error response logging.
type LoggerManager struct {
	loggers map[string]*zap.Logger
	baseDir string
}

// NewLoggerManager initializes and returns a LoggerManager.
func NewLoggerManager(baseDir string) *LoggerManager {
	return &LoggerManager{
		loggers: make(map[string]*zap.Logger),
		baseDir: baseDir,
	}
}

// GetLogger retrieves or creates a logger for the specified log type.
func (lm *LoggerManager) GetLogger(logType string, logLevel string) (*zap.Logger, error) {
	if logger, exists := lm.loggers[logType]; exists {
		return logger, nil
	}

	logFilePath, err := lm.initializeLogFile(logType)
	if err != nil {
		return nil, err
	}

	// Set log level
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(logLevel)); err != nil {
		level = zapcore.InfoLevel
	}

	// Configure lumberjack for log rotation
	lumberjackLogger := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    5,    // Max size of 5 MB
		MaxBackups: 10,   // Maximum of 10 backup files
		MaxAge:     30,   // Retention for 30 days
		Compress:   true, // Compress rotated logs
	}

	// Encoder configuration for structured logging
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
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
		zapcore.AddSync(lumberjackLogger),
		level,
	)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	lm.loggers[logType] = logger
	logger.Info("Logger initialized successfully",
		zap.String("log_file", logFilePath),
		zap.Int("max_size_mb", lumberjackLogger.MaxSize),
		zap.Int("max_backups", lumberjackLogger.MaxBackups),
		zap.Int("max_age_days", lumberjackLogger.MaxAge),
	)

	return logger, nil
}

// initializeLogFile ensures the directory and log file exist.
func (lm *LoggerManager) initializeLogFile(logType string) (string, error) {
	today := time.Now().Format("2006-01-02")
	logDir := filepath.Join(lm.baseDir, logType)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}
	return filepath.Join(logDir, fmt.Sprintf("%s_%s.log", logType, today)), nil
}

// LogAgentError logs an error and provides an AI-assisted response.
func (lm *LoggerManager) LogAgentError(logType, context, message string) {
	logger, err := lm.GetLogger(logType, "error")
	if err != nil {
		fmt.Printf("Failed to get logger for %s: %v\n", logType, err)
		return
	}

	logger.Error("Agent error occurred",
		zap.String("context", context),
		zap.String("message", message),
	)

	// Simulated AI response integration
	aiResponse := generateAIResponse(context, message)
	logger.Info("AI Response Generated",
		zap.String("context", context),
		zap.String("ai_response", aiResponse),
	)
}

// generateAIResponse simulates an AI-generated response for the given context and message.
func generateAIResponse(context, message string) string {
	return fmt.Sprintf("Based on the context '%s', the issue '%s' suggests checking network connectivity and firewall rules.", context, message)
}
