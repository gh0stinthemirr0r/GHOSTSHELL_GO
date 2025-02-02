package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	// GhostShell core
	"ghostshell/agents"
	"ghostshell/ai"
	"ghostshell/config"
	"ghostshell/metrics"
	"ghostshell/oqs"
	"ghostshell/storage"
	"ghostshell/ui"

	// Logging
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Application encapsulates the entire GhostShell system.
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

	// Metrics manager or overlay
	Metrics *metrics.MetricsManager
	Overlay *metrics.MetricsOverlay // Additional overlay for system usage, AI usage

	// Agent manager
	AgentsManager *agents.AgentManager

	// Channels
	ShutdownChan chan os.Signal
	ReadyChan    chan struct{}
}

// NewApplication loads configs, sets up logging, returns an Application.
func NewApplication() (*Application, error) {
	// 1) Load main config (config.yaml)
	mainCfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to load main config: %w", err)
	}

	// 2) Create a dynamic Zap logger
	currentTime := time.Now().UTC().Format("20060102_150405")
	logFileName := fmt.Sprintf("ai_log_%s.log", currentTime)

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

// main is the primary entry point for GhostShell
func main() {
	// Instantiate app
	app, err := NewApplication()
	if err != nil {
		fmt.Printf("Application initialization failed: %v\n", err)
		os.Exit(1)
	}
	defer app.Cleanup()

	// Handle signals
	signal.Notify(app.ShutdownChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Initialize components
	wg.Add(1)
	go func() {
		defer wg.Done()
		if initErr := app.InitializeComponents(); initErr != nil {
			app.Logger.Error("Failed to initialize components", zap.Error(initErr))
		}
		close(app.ReadyChan)
	}()

	// Start server
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-app.ReadyChan
		server.StartServer(app.Config, app.ModelLoader, app.Logger)
	}()

	// Example: Use AI agent for a task
	err = app.AgentsManager.ExecuteAgent("DynamicTaskAgent", "Analyze AI model performance")
	if err != nil {
		app.Logger.Error("Failed to execute agent task", zap.Error(err))
	}

	// Wait for shutdown signal
	<-app.ShutdownChan
	app.Logger.Info("Received shutdown signal")

	// Wait for all goroutines
	wg.Wait()
	app.Logger.Info("All goroutines completed, exiting")
}

// InitializeComponents sets up everything needed to run
func (app *Application) InitializeComponents() error {
	var errs []error

	// 1) Initialize ephemeral PQ memory if needed
	if err := oqs.InitializeSecureMemory(); err != nil {
		app.Logger.Warn("Failed ephemeral PQ memory init", zap.Error(err))
	}

	// 2) Initialize metrics
	mmCfg := metrics.Config{
		Port: "9090",
		Path: "/metrics",
	}
	mm, err := metrics.NewMetricsManager(app.Logger, mmCfg)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to init metrics manager: %w", err))
	} else {
		app.Metrics = mm
		go mm.RunWithGracefulShutdown(mmCfg)
	}

	// 3) Initialize overlay for system usage (CPU, net, AI usage)
	overlay := metrics.NewMetricsOverlay(app.Logger)
	app.Overlay = overlay

	// 4) Initialize vault with post-quantum logic
	if vaultErr := app.initializeVault(); vaultErr != nil {
		errs = append(errs, vaultErr)
	}

	// 5) Initialize storage system
	if stErr := storage.InitializeStorage(); stErr != nil {
		errs = append(errs, fmt.Errorf("failed to initialize storage: %w", stErr))
	}

	// 6) Initialize Ghostshell UI
	app.Ghostshell = ui.NewGhostshell()
	if app.Ghostshell == nil {
		errs = append(errs, fmt.Errorf("failed to init Ghostshell UI"))
	}

	// 7) Load sub-configs (AI, system, themes)
	if aiErr := app.loadAIConfig("ghostshell/config/ai.yaml"); aiErr != nil {
		errs = append(errs, aiErr)
	}
	if sysErr := app.loadSystemConfig("ghostshell/config/system.yaml"); sysErr != nil {
		errs = append(errs, sysErr)
	}
	if thErr := app.loadThemesConfig("ghostshell/config/themes.yaml"); thErr != nil {
		errs = append(errs, thErr)
	}

	// 8) Initialize AI model loader
	if app.AIConfig != nil && len(errs) == 0 {
		mdl, err := ai.NewModelLoader(app.AIConfig, app.Logger)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to create AI model loader: %w", err))
		} else {
			app.ModelLoader = mdl
			if loadErr := app.ModelLoader.LoadModel(); loadErr != nil {
				errs = append(errs, fmt.Errorf("failed to auto-load AI model: %w", loadErr))
			}
		}
	}

	// 9) Initialize Agent Manager
	app.Logger.Info("Initializing Agent Manager")
	agentManager := agents.NewAgentManager()
	netAgent := agents.NewNetworkAgent("NetMonitor")
	taskAgent := agents.NewTaskSchedulerAgent("TaskScheduler")
	agentManager.RegisterAgent(netAgent)
	agentManager.RegisterAgent(taskAgent)

	app.AgentsManager = agentManager
	app.Logger.Info("Agent Manager initialized successfully")

	// Start agent tasks
	go app.AgentsManager.ExecuteAgent("NetMonitor", []string{"1.1.1.1", "8.8.8.8", "9.9.9.9"})
	go app.AgentsManager.ExecuteAgent("TaskScheduler", "Cleanup Temp Files")

	// Periodic reporting
	go func() {
		for {
			time.Sleep(10 * time.Second)
			app.Logger.Info("Agent Reports", zap.String("reports", app.AgentsManager.ReportAll()))
		}
	}()

	if len(errs) > 0 {
		return fmt.Errorf("initialization errors: %v", errs)
	}

	app.Logger.Info("All components initialized successfully")
	return nil
}

// initializeVault sets up a post-quantum secure vault
func (app *Application) initializeVault() error {
	app.Logger.Info("Initializing vault with post-quantum security")
	encryptionKey, err := oqs.GenerateRandomBytes(32)
	if err != nil {
		return fmt.Errorf("failed to generate encryption key: %w", err)
	}
	v, err := storage.NewVault(app.Config.Storage.VaultPath, encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to init vault: %w", err)
	}
	app.Vault = v
	app.Logger.Info("Vault initialized successfully")
	return nil
}

// Cleanup finalizes all resources
func (app *Application) Cleanup() {
	if app.Logger != nil {
		_ = app.Logger.Sync()
	}
	if app.Vault != nil {
		app.Vault.Close()
	}
	if app.Ghostshell != nil {
		app.Ghostshell.Close()
	}
	if app.AgentsManager != nil {
		app.Logger.Info("Terminating agents...")
		for name, agent := range app.AgentsManager.Agents() {
			app.Logger.Info("Terminating agent", zap.String("agent", name))
			agent.Terminate()
		}
	}
	app.Logger.Info("Application cleaned up and shut down gracefully")
}
