package ai

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"oqs"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ModelLoader manages the loading/unloading of AI models and configuration with post-quantum security.
type ModelLoader struct {
	config       *Config
	model        *LanguageModel
	secureMemory []byte
	mutex        sync.Mutex
	logger       *zap.Logger
}

// NewModelLoader initializes a new ModelLoader instance with a separate dynamic logger.
func NewModelLoader(config *Config) (*ModelLoader, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create a dynamic logger with date/time
	currentTime := time.Now().UTC().Format("20060102_150405")
	logFileName := fmt.Sprintf("ai_log_%s.log", currentTime)

	// Zap production config for this logger
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	loggerConfig.OutputPaths = []string{logFileName, "stdout"}

	logger, err := loggerConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize loader logger: %w", err)
	}

	// Ensure config file exists, create if missing
	configFilePath := GetDefaultConfigPath()
	if err := ensureConfigFileExists(config, configFilePath, logger.Sugar()); err != nil {
		return nil, err
	}

	return &ModelLoader{
		config: config,
		logger: logger,
	}, nil
}

// ensureConfigFileExists checks if config file exists; creates it if not.
func ensureConfigFileExists(config *Config, path string, slog *zap.SugaredLogger) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		slog.Infof("Creating default config file at %s", path)
		if err := SaveConfig(config, path); err != nil {
			return fmt.Errorf("failed to create default config file: %w", err)
		}
	}
	return nil
}

// LoadModel loads the AI model based on config.ModelPath and allocates secure memory.
func (loader *ModelLoader) LoadModel() error {
	loader.mutex.Lock()
	defer loader.mutex.Unlock()

	if loader.model != nil {
		return errors.New("a model is already loaded")
	}

	loader.logger.Info("Loading model", zap.String("modelPath", loader.config.ModelPath))
	if err := checkModelFileExists(loader.config.ModelPath); err != nil {
		return err
	}

	// Allocate secure memory for the model
	secureMemory, err := oqs.AllocateSecureMemory(1024 * 1024) // Allocate 1MB secure memory
	if err != nil {
		return fmt.Errorf("failed to allocate secure memory: %w", err)
	}
	loader.secureMemory = secureMemory

	// Load the model
	lm, err := NewLanguageModel(loader.config.ModelPath)
	if err != nil {
		return fmt.Errorf("failed to create LanguageModel: %w", err)
	}

	if err := lm.LoadModel(); err != nil {
		return fmt.Errorf("failed to initialize model: %w", err)
	}

	loader.model = lm
	loader.logger.Info("Model loaded successfully with secure memory", zap.String("modelPath", loader.config.ModelPath))
	return nil
}

// UnloadModel frees resources from the currently loaded model and zeroizes memory.
func (loader *ModelLoader) UnloadModel() error {
	loader.mutex.Lock()
	defer loader.mutex.Unlock()

	if loader.model == nil {
		return errors.New("no model currently loaded")
	}

	loader.logger.Info("Unloading model and zeroizing memory...")

	// Zeroize secure memory
	if loader.secureMemory != nil {
		if err := oqs.ZeroizeMemory(loader.secureMemory); err != nil {
			return fmt.Errorf("failed to zeroize secure memory: %w", err)
		}
		loader.secureMemory = nil
	}

	// Unload model
	loader.model = nil
	loader.logger.Info("Model unloaded securely")
	return nil
}

// Eject unloads the model and optionally does other teardown steps.
func (loader *ModelLoader) Eject() error {
	loader.logger.Info("Ejecting model (unload + teardown)")
	if err := loader.UnloadModel(); err != nil {
		return err
	}
	// Additional teardown logic if needed
	loader.logger.Info("Eject complete")
	return nil
}

// ReloadModel unloads and re-loads the current model path from config.
func (loader *ModelLoader) ReloadModel() error {
	if err := loader.UnloadModel(); err != nil {
		return err
	}
	return loader.LoadModel()
}

// LoadSelectedModel updates config.ModelPath, saves, and loads the new model.
func (loader *ModelLoader) LoadSelectedModel(newPath string) error {
	loader.mutex.Lock()
	defer loader.mutex.Unlock()

	// Unload if there's a currently loaded model
	if loader.model != nil {
		if err := loader.UnloadModel(); err != nil {
			return err
		}
	}

	loader.config.ModelPath = newPath
	cfgPath := GetDefaultConfigPath()
	if err := SaveConfig(loader.config, cfgPath); err != nil {
		return fmt.Errorf("failed to save updated config file: %w", err)
	}

	loader.logger.Info("Updated config with new model path", zap.String("modelPath", newPath))
	// Now load the newly selected model
	return loader.LoadModel()
}

// SetControlParameters updates control params in config, saves them, and optionally reloads.
func (loader *ModelLoader) SetControlParameters(params ControlParameters, reload bool) error {
	loader.mutex.Lock()
	defer loader.mutex.Unlock()

	loader.config.ControlParams = params
	cfgPath := GetDefaultConfigPath()
	if err := SaveConfig(loader.config, cfgPath); err != nil {
		return fmt.Errorf("failed to save updated control parameters: %w", err)
	}

	loader.logger.Info("Updated control parameters", zap.Any("params", params))
	if reload && loader.model != nil {
		loader.logger.Info("Reloading model to apply new parameters")
		if err := loader.ReloadModel(); err != nil {
			return err
		}
	}
	return nil
}

// checkModelFileExists ensures the model file is actually present on disk.
func checkModelFileExists(modelPath string) error {
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("model file does not exist: %s", modelPath)
	}
	return nil
}
