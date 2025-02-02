package util

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

// HandleShutdown gracefully handles termination signals to ensure cleanup.
func HandleShutdown(cancelFunc context.CancelFunc, logger *zap.Logger) {
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-shutdownChan
		logger.Info("Received shutdown signal, initiating cleanup.")
		cancelFunc() // Call the context cancel function to terminate ongoing operations
		os.Exit(0)
	}()
}

// FileExists checks if the given file path exists and is not a directory.
func FileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// CreateDirectory ensures that the specified directory exists, creating it if necessary.
func CreateDirectory(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}
