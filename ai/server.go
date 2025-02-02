package ai

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// Global logger instance
var logger *zap.SugaredLogger

// init initializes the Zap logger with dynamic file naming
func init() {
	currentTime := time.Now().UTC().Format("20060102_150405")
	logFileName := fmt.Sprintf("ai_log_%s.log", currentTime)

	// Configure Zap logger
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{
		logFileName,
		"stdout", // Write logs to console as well
	}

	log, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}

	logger = log.Sugar()
	logger.Infof("Logger initialized with file: %s", logFileName)
}

// StartServer initializes and starts the server with AI functionality.
func StartServer(configPath string) {
	// Ensure logs are flushed on exit
	defer logger.Sync()

	// 1. Load the configuration file
	config, err := LoadConfig(configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// 2. Validate the configuration
	if err := config.Validate(); err != nil {
		logger.Fatal("Invalid configuration", zap.Error(err))
	}

	// 3. Initialize the model loader
	loader, err := NewModelLoader(config)
	if err != nil {
		logger.Fatal("Failed to initialize model loader", zap.Error(err))
	}

	// 4. Create a new Fiber app with additional configurations
	app := fiber.New(fiber.Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	})

	// 5. Setup routes for the app
	//    We'll pass in the global logger (zap.SugaredLogger) and the loader
	SetupRoutes(app, loader, logger)

	// Graceful shutdown handling
	go func() {
		if err := app.Listen(":8080"); err != nil {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	logger.Info("Server started successfully on :8080")

	// Handle OS signals for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill)
	<-quit

	logger.Info("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		logger.Error("Error during server shutdown", zap.Error(err))
	}

	logger.Info("Server stopped gracefully")
}
