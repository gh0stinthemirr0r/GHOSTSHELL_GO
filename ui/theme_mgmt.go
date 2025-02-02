package graphics

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"

	"ghostshell/oqs/vault"
)

// ------------------------------------------------------
// 1) Define sub-structs mirroring your expanded YAML
// ------------------------------------------------------

// FontConfig holds data about font family, size, weight, smoothing, etc.
type FontConfig struct {
	Family        string   `json:"family"`
	Size          int      `json:"size"`
	Weight        string   `json:"weight"`
	Smoothing     bool     `json:"smoothing"`
	FallbackFonts []string `json:"fallback_fonts,omitempty"`
}

// AnsiColors holds your 16 ANSI color codes (plus any extended).
type AnsiColors struct {
	Black         string `json:"black"`
	Red           string `json:"red"`
	Green         string `json:"green"`
	Yellow        string `json:"yellow"`
	Blue          string `json:"blue"`
	Magenta       string `json:"magenta"`
	Cyan          string `json:"cyan"`
	White         string `json:"white"`
	BrightBlack   string `json:"bright_black"`
	BrightRed     string `json:"bright_red"`
	BrightGreen   string `json:"bright_green"`
	BrightYellow  string `json:"bright_yellow"`
	BrightBlue    string `json:"bright_blue"`
	BrightMagenta string `json:"bright_magenta"`
	BrightCyan    string `json:"bright_cyan"`
	BrightWhite   string `json:"bright_white"`
}

// ColorsConfig corresponds to the `colors` section in YAML.
type ColorsConfig struct {
	Background             string     `json:"background"`
	Foreground             string     `json:"foreground"`
	Cursor                 string     `json:"cursor"`
	SelectionBackground    string     `json:"selection_background"`
	SelectionForeground    string     `json:"selection_foreground"`
	BackgroundTransparency float64    `json:"background_transparency"`
	AnsiColors             AnsiColors `json:"ansi_colors"`
	NeonGlowColor          string     `json:"neon_glow_color"`
}

// ------------------------------------------------------
// FullTheme and DefaultTheme Definitions
// ------------------------------------------------------
type FullTheme struct {
	Font   FontConfig   `json:"font"`
	Colors ColorsConfig `json:"colors"`
	// Additional sections omitted for brevity
}

var DefaultTheme = FullTheme{
	Font: FontConfig{
		Family:    "JetBrains Mono",
		Size:      14,
		Weight:    "normal",
		Smoothing: true,
		FallbackFonts: []string{
			"Noto Sans",
			"DejaVu Sans Mono",
		},
	},
	Colors: ColorsConfig{
		Background: "#1E1E2E",
		Foreground: "#A6ACCD",
		Cursor:     "#F7768E",
		// Omitted additional default values
	},
}

// ------------------------------------------------------
// ThemeManager Logic
// ------------------------------------------------------

type ThemeManager struct {
	currentTheme    FullTheme
	availableThemes map[string]FullTheme

	mutex      sync.RWMutex
	logger     *zap.Logger
	watcher    *fsnotify.Watcher
	configPath string
	vault      *vault.Vault

	subscribers map[chan FullTheme]struct{}
	subMutex    sync.Mutex
}

func NewThemeManager(configPath string, logger *zap.Logger) (*ThemeManager, error) {
	if configPath == "" {
		return nil, errors.New("configPath cannot be empty")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}

	// Initialize secure memory with OQS Vault
	if err := vault.InitializeSecureMemory(); err != nil {
		logger.Warn("Failed to initialize secure memory", zap.Error(err))
	}

	v, err := vault.NewVault([]byte("theme-manager-key-32bytes-exactly!!"))
	if err != nil {
		return nil, fmt.Errorf("failed to create vault: %w", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	tm := &ThemeManager{
		currentTheme:    DefaultTheme,
		availableThemes: make(map[string]FullTheme),
		logger:          logger,
		watcher:         watcher,
		configPath:      configPath,
		vault:           v,
		subscribers:     make(map[chan FullTheme]struct{}),
	}

	if err := tm.loadThemes(); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to load themes: %w", err)
	}

	go tm.watchConfigChanges()
	return tm, nil
}

func (tm *ThemeManager) loadThemes() error {
	// Simplified implementation of loadThemes
	data, err := os.ReadFile(tm.configPath)
	if err != nil {
		tm.logger.Warn("Error reading theme config file", zap.Error(err))
		tm.currentTheme = DefaultTheme
		return nil
	}

	decrypted, err := tm.vault.Decrypt(data)
	if err != nil {
		tm.logger.Warn("Error decrypting theme config", zap.Error(err))
		return nil
	}

	var themeMap map[string]FullTheme
	if err := json.Unmarshal(decrypted, &themeMap); err != nil {
		return fmt.Errorf("failed to parse theme JSON: %w", err)
	}

	tm.availableThemes = themeMap
	if defaultTheme, ok := themeMap["default"]; ok {
		tm.currentTheme = defaultTheme
	}

	return nil
}

func (tm *ThemeManager) watchConfigChanges() {
	// Watch for file changes and reload themes
	for {
		select {
		case event := <-tm.watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				tm.logger.Info("Theme config file modified. Reloading themes.")
				tm.loadThemes()
			}
		case err := <-tm.watcher.Errors:
			tm.logger.Error("Watcher error", zap.Error(err))
		}
	}
}

func (tm *ThemeManager) SetTheme(themeName string) error {
	// Activate a theme by name
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	th, exists := tm.availableThemes[themeName]
	if !exists {
		return fmt.Errorf("theme '%s' does not exist", themeName)
	}

	tm.currentTheme = th
	tm.notifySubscribers()
	return tm.saveThemes()
}

func (tm *ThemeManager) notifySubscribers() {
	tm.subMutex.Lock()
	defer tm.subMutex.Unlock()

	for ch := range tm.subscribers {
		ch <- tm.currentTheme
	}
}

func (tm *ThemeManager) saveThemes() error {
	// Persist themes to file
	data, err := json.Marshal(tm.availableThemes)
	if err != nil {
		return fmt.Errorf("failed to serialize themes: %w", err)
	}

	encrypted, err := tm.vault.Encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt themes: %w", err)
	}

	if err := os.WriteFile(tm.configPath, encrypted, 0644); err != nil {
		return fmt.Errorf("failed to write theme config: %w", err)
	}

	return nil
}

// Close shuts down the watcher and cleans up resources
func (tm *ThemeManager) Close() error {
	tm.logger.Info("Closing ThemeManager")
	tm.watcher.Close()
	return tm.vault.Close()
}
