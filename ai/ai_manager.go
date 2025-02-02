package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	// Core app imports
	"ghostshell/ai"
	"ghostshell/config"
	"ghostshell/metrics"
	"ghostshell/storage"
	"ghostshell/ui"

	// Post-quantum ephemeral references

	// Logging
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Application struct {
	// Main app config
	Config *config.Config

	// Additional sub-configs
	AIConfig     *ai.Config
	SystemConfig *system.Config
	ThemesConfig *themes.Config

	// Logger
	Logger *zap.Logger

	// AI model loader
	ModelLoader *ai.ModelLoader

	// Ghostshell UI
	Ghostshell *ui.Ghostshell

	// Secure Vault
	Vault *storage.Vault

	// Metrics manager and overlay
	Metrics *metrics.MetricsManager
	Overlay *metrics.MetricsOverlay

	// Channels for synchronization
	ShutdownChan chan os.Signal
	ReadyChan    chan struct{}
}

func NewApplication() (*Application, error) {
	// 1. Load main application config (config.yaml)
	mainCfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to load main config: %w", err)
	}

	// 2. Initialize Zap logger with dynamic timestamped filename
	currentTime := time.Now().UTC().Format("20060102_150405")
	logFileName := fmt.Sprintf("ghostshell_log_%s.log", currentTime)

	loggerConfig := zap.NewProductionConfig()
	loggerConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	// Log to both file and stdout
	loggerConfig.OutputPaths = []string{logFileName, "stdout"}
	unifiedLogger, err := loggerConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	app := &Application{
		Config:       mainCfg,
		Logger:       unifiedLogger,
		ShutdownChan: make(chan os.Signal, 1),
		ReadyChan:    make(chan struct{}),
	}

	return app, nil
}

func main() {
	// Instantiate the main application
	app, err := NewApplication()
	if err != nil {
		fmt.Printf("Application initialization failed: %v\n", err)
		os.Exit(1)
	}
	defer app.Cleanup()

	// Handle shutdown signals
	signal.Notify(app.ShutdownChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	// 1. Initialize components in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		if initErr := app.InitializeComponents(); initErr != nil {
			app.Logger.Error("Failed to initialize components", zap.Error(initErr))
			os.Exit(1) // Exit if critical components fail to initialize
		}
		close(app.ReadyChan) // Signal that components have initialized
	}()

	// 2. Start the server in a separate goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-app.ReadyChan // Ensure components are ready
		server.StartServer(app.Config, app.ModelLoader, app.Logger, app.Overlay)
	}()

	// 3. Start the Ghostshell UI in a separate goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-app.ReadyChan // Ensure components are ready
		if err := app.Ghostshell.Run(); err != nil {
			app.Logger.Error("Ghostshell UI encountered an error", zap.Error(err))
		}
	}()

	// 4. Start system metrics collection
	wg.Add(1)
	go func() {
		defer wg.Done()
		if app.Overlay != nil {
			app.Overlay.RunWithGracefulShutdown("8081") // Example port for dashboards
		}
	}()

	// 5. Start Prometheus metrics server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if app.Metrics != nil {
			app.Metrics.RunWithGracefulShutdown(app.Config.Metrics)
		}
	}()

	// Wait for shutdown signal
	<-app.ShutdownChan
	app.Logger.Info("Received shutdown signal")

	// Initiate graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop all components
	app.StopComponents(shutdownCtx)

	// Wait for all goroutines to finish
	wg.Wait()
	app.Logger.Info("All components have shut down gracefully")
}

func (app *Application) InitializeComponents() error {
	var errs []error

	// ------------------------
	// 1. Initialize Post-Quantum Secure Memory
	// ------------------------
	if err := oqs.InitializeSecureMemory(); err != nil {
		app.Logger.Warn("Failed to initialize post-quantum secure memory", zap.Error(err))
		// Continue even if PQ secure memory fails; depends on requirements
	}

	// ------------------------
	// 2. Initialize Metrics Manager
	// ------------------------
	mmCfg := app.Config.Metrics
	mm, err := metrics.NewMetricsManager(app.Logger, mmCfg)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to initialize metrics manager: %w", err))
	} else {
		app.Metrics = mm
		go mm.RunWithGracefulShutdown(mmCfg)
	}

	// ------------------------
	// 3. Initialize Metrics Overlay
	// ------------------------
	overlay := metrics.NewMetricsOverlay(app.Logger)
	app.Overlay = overlay

	// Optionally, integrate system metrics with the overlay
	// e.g., periodically update overlay with system metrics
	go app.collectSystemMetrics()

	// ------------------------
	// 4. Initialize Vault with Post-Quantum Security
	// ------------------------
	if err := app.initializeVault(); err != nil {
		errs = append(errs, err)
	}

	// ------------------------
	// 5. Initialize Storage
	// ------------------------
	if err := storage.InitializeStorage(app.Config.Storage); err != nil {
		errs = append(errs, fmt.Errorf("failed to initialize storage: %w", err))
	}

	// ------------------------
	// 6. Initialize Ghostshell UI with Theme Management
	// ------------------------
	uiCfg := app.Config.UI
	ghostshellUI, err := ui.NewGhostshell(uiCfg, app.Logger, app.Overlay)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to initialize Ghostshell UI: %w", err))
	} else {
		app.Ghostshell = ghostshellUI
	}

	// ------------------------
	// 7. Load AI, System, and Themes Configurations
	// ------------------------
	if err := app.loadAIConfig("ghostshell/config/ai_config.yaml"); err != nil {
		errs = append(errs, err)
	}
	if err := app.loadSystemConfig("ghostshell/config/system_config.yaml"); err != nil {
		errs = append(errs, err)
	}
	if err := app.loadThemesConfig("ghostshell/config/themes.yaml"); err != nil {
		errs = append(errs, err)
	}

	// ------------------------
	// 8. Initialize AI Model Loader
	// ------------------------
	if app.AIConfig != nil && len(errs) == 0 {
		loader, err := ai.NewModelLoader(app.AIConfig, app.Logger)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to create AI model loader: %w", err))
		} else {
			app.ModelLoader = loader
			// Optionally auto-load the model
			if loadErr := app.ModelLoader.LoadModel(); loadErr != nil {
				app.Logger.Warn("Failed to auto-load AI model", zap.Error(loadErr))
			}
		}
	}

	// ------------------------
	// 9. Initialize Application Registry
	// ------------------------
	if app.Overlay != nil {
		app.Overlay.InitializeDashboards()
		go app.Overlay.StartDashboards("8081") // Example port for dashboards
	}

	// ------------------------
	// 10. Register Applications in the Registry
	// ------------------------
	if app.Overlay != nil {
		registry := app.Overlay.GetRegistry()
		if registry != nil {
			err := registry.RegisterApps()
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to register applications: %w", err))
			}
		}
	}

	// ------------------------
	// 11. Initialize Theme Management
	// ------------------------
	if app.ThemesConfig != nil && app.Ghostshell != nil {
		if err := app.Ghostshell.ApplyTheme(app.ThemesConfig.DefaultTheme); err != nil {
			errs = append(errs, fmt.Errorf("failed to apply default theme: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("initialization errors: %v", errs)
	}

	app.Logger.Info("All components initialized successfully")
	return nil
}

// collectSystemMetrics periodically collects system metrics and updates the overlay
func (app *Application) collectSystemMetrics() {
	ticker := time.NewTicker(app.Config.SystemMetrics.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-app.ShutdownChan:
			app.Logger.Info("Stopping system metrics collection")
			return
		case <-ticker.C:
			// Collect CPU and Memory usage
			cpuUsage, memUsage, err := system.GetSystemUsage()
			if err != nil {
				app.Logger.Error("Failed to collect system usage", zap.Error(err))
				continue
			}

			// Collect Network addresses and throughput
			netAddrs, netStats, err := system.GetNetworkStats()
			if err != nil {
				app.Logger.Error("Failed to collect network stats", zap.Error(err))
				continue
			}

			// Collect AI model usage
			aiStats, err := app.ModelLoader.GetAIUsage()
			if err != nil {
				app.Logger.Error("Failed to collect AI usage stats", zap.Error(err))
				continue
			}

			// Update metrics overlay
			app.Overlay.UpdateSystemMetrics(cpuUsage, memUsage, netAddrs, netStats, aiStats)
		}
	}
}

// initializeVault sets up the secure vault using Post-Quantum features
func (app *Application) initializeVault() error {
	app.Logger.Info("Initializing vault with post-quantum security")

	// Generate a secure encryption key
	encryptionKey, err := oqs.GenerateRandomBytes(32)
	if err != nil {
		return fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Allocate secure memory for the key
	keyMemory, err := oqs.AllocateMemory(len(encryptionKey))
	if err != nil {
		return fmt.Errorf("failed to allocate secure memory for encryption key: %w", err)
	}

	// Copy the key into secure memory
	copy((*[1 << 30]byte)(keyMemory)[:len(encryptionKey)], encryptionKey)

	// Initialize the vault with the encryption key
	vault, err := storage.NewVault(app.Config.Storage.VaultPath, encryptionKey)
	if err != nil {
		oqs.FreeMemory((*[1 << 30]byte)(keyMemory)[:len(encryptionKey)])
		return fmt.Errorf("failed to initialize vault: %w", err)
	}
	app.Vault = vault

	// Free the secure memory after initializing the vault
	oqs.FreeMemory((*[1 << 30]byte)(keyMemory)[:len(encryptionKey)])

	app.Logger.Info("Vault initialized successfully")
	return nil
}

// loadAIConfig loads the AI configuration from the specified path or creates defaults
func (app *Application) loadAIConfig(path string) error {
	aiCfg, err := ai.LoadConfig(path)
	if err != nil {
		app.Logger.Warn("No existing AI config found or error reading it. Creating defaults.",
			zap.String("path", path),
			zap.Error(err),
		)
		aiCfg = &ai.Config{
			ModelPath: "ai/models/default.gguf",
			ControlParams: ai.ControlParameters{
				Temperature: 0.7,
				MaxTokens:   512,
			},
		}
		if saveErr := ai.SaveConfig(aiCfg, path); saveErr != nil {
			return fmt.Errorf("failed to create default AI config: %w", saveErr)
		}
	}
	app.AIConfig = aiCfg
	return nil
}

// loadSystemConfig loads the system configuration from the specified path or creates defaults
func (app *Application) loadSystemConfig(path string) error {
	sysCfg, err := system.LoadConfig(path)
	if err != nil {
		app.Logger.Warn("No existing system config found or error reading it. Creating defaults.",
			zap.String("path", path),
			zap.Error(err),
		)
		sysCfg = &system.Config{
			SystemName:         "GhostshellSystem",
			Description:        "Default system configuration",
			CollectionInterval: 10 * time.Second,
			DiskPaths:          []string{"/"},
			EnableNetwork:      true,
			EnableEncryption:   true,
			EncryptionKey:      make([]byte, 32), // Placeholder; should be securely generated
			PrometheusPort:     "9090",
		}
		if saveErr := system.SaveConfig(sysCfg, path); saveErr != nil {
			return fmt.Errorf("failed to create default system config: %w", saveErr)
		}
	}
	app.SystemConfig = sysCfg
	return nil
}

// loadThemesConfig loads the themes configuration from the specified path or creates defaults
func (app *Application) loadThemesConfig(path string) error {
	thCfg, err := themes.LoadConfig(path)
	if err != nil {
		app.Logger.Warn("No existing themes config found or error reading it. Creating defaults.",
			zap.String("path", path),
			zap.Error(err),
		)
		thCfg = &themes.Config{
			DefaultTheme:    "dark",
			AvailableThemes: []string{"dark", "light"},
		}
		if saveErr := themes.SaveConfig(thCfg, path); saveErr != nil {
			return fmt.Errorf("failed to create default themes config: %w", saveErr)
		}
	}
	app.ThemesConfig = thCfg
	return nil
}

// StopComponents gracefully shuts down all components
func (app *Application) StopComponents(ctx context.Context) {
	// 1. Stop Metrics Manager
	if app.Metrics != nil {
		if err := app.Metrics.StopMetricsServer(ctx); err != nil {
			app.Logger.Error("Error stopping metrics manager", zap.Error(err))
		}
	}

	// 2. Stop Metrics Overlay
	if app.Overlay != nil {
		// Assuming overlay has a Stop method
		if err := app.Overlay.Stop(); err != nil {
			app.Logger.Error("Error stopping metrics overlay", zap.Error(err))
		}
	}

	// 3. Stop Vault
	if app.Vault != nil {
		if err := app.Vault.Close(); err != nil {
			app.Logger.Error("Error closing vault", zap.Error(err))
		}
	}

	// 4. Stop Ghostshell UI
	if app.Ghostshell != nil {
		if err := app.Ghostshell.Close(); err != nil {
			app.Logger.Error("Error closing Ghostshell UI", zap.Error(err))
		}
	}

	// 5. Shutdown OQS Secure Memory
	if err := oqs.ShutdownSecureMemory(); err != nil {
		app.Logger.Warn("Failed to shutdown secure memory", zap.Error(err))
	}

	// Cancel any ongoing operations
	close(app.ShutdownChan)
}

// Cleanup releases all resources and performs a graceful shutdown.
func (app *Application) Cleanup() {
	if app.Logger != nil {
		_ = app.Logger.Sync()
	}
	app.Logger.Info("Application shut down gracefully.")
}
