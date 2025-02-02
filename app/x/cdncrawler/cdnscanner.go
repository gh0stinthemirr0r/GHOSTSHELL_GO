package cdncrawler

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/jung-kurt/gofpdf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"ghostshell/cdncrawler/options"
	"ghostshell/oqs"        // Hypothetical post-quantum security package
	"ghostshell/securedata" // Hypothetical secure data handling package
)

// -------------- Constants & Paths --------------

const (
	ScreenWidth   = 1280
	ScreenHeight  = 720
	ParticleCount = 50
	FontPointSize = 30

	// For reporting
	ReportDir     = "ghostshell/reporting"
	LogDir        = "ghostshell/logging"
	SecureDataDir = "ghostshell/secure_data"
)

// -------------- Data Structures --------------

// Particle represents a small colored square/bubble in the background.
type Particle struct {
	x, y   float32
	dx, dy float32
	color  rl.Color
}

// MenuManager manages a simple text-based menu in the Raylib UI.
type MenuManager struct {
	items       []string
	currentItem int
	terminal    *Terminal
}

// Terminal holds the main application state: UI elements, scanning results, etc.
type Terminal struct {
	font             rl.Font
	particles        []*Particle
	menuManager      *MenuManager
	gracefulShutdown chan os.Signal

	isScanning bool
	mu         sync.Mutex // guards access to isScanning + any concurrency changes

	// Results from scanning
	cdnResults []string

	// The logger
	logger *zap.Logger

	// Secure Vault for Post-Quantum Security
	vault *securedata.Vault

	// CLI/parsed Options
	options *options.Options
}

// -------------- Logging Setup --------------

// setupLogger initializes a Zap logger with a timestamped log file in ISO8601 format.
func setupLogger() (*zap.Logger, error) {
	if err := os.MkdirAll(LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	// ISO8601 timestamp e.g., 2023-10-25T15-30-45Z
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	logFileName := fmt.Sprintf("cdnscanner_log_%s.log", strings.ReplaceAll(timestamp, ":", "-"))
	logFilePath := filepath.Join(LogDir, logFileName)

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logFilePath, "stdout"}

	return cfg.Build()
}

// -------------- Secure Vault Initialization --------------

// initializeVault sets up the secure vault using post-quantum cryptography.
func (t *Terminal) initializeVault() error {
	t.logger.Info("Initializing post-quantum secure vault")

	// Ensure the secure data directory exists
	if err := os.MkdirAll(SecureDataDir, 0700); err != nil {
		t.logger.Error("Failed to create secure data directory", zap.Error(err))
		return fmt.Errorf("failed to create secure data directory: %w", err)
	}

	// Generate or load a post-quantum encryption key
	encryptionKeyPath := filepath.Join(SecureDataDir, "encryption_key.key")
	var encryptionKey []byte
	if _, err := os.Stat(encryptionKeyPath); os.IsNotExist(err) {
		// Generate a new encryption key
		key, err := oqs.GenerateRandomBytes(32) // Assuming 256-bit key
		if err != nil {
			t.logger.Error("Failed to generate encryption key", zap.Error(err))
			return fmt.Errorf("failed to generate encryption key: %w", err)
		}
		encryptionKey = key

		// Save the encryption key securely
		if err := oqs.SaveKey(encryptionKeyPath, encryptionKey); err != nil {
			t.logger.Error("Failed to save encryption key", zap.Error(err))
			return fmt.Errorf("failed to save encryption key: %w", err)
		}
		t.logger.Info("Generated and saved new encryption key", zap.String("path", encryptionKeyPath))
	} else {
		// Load existing encryption key
		key, err := oqs.LoadKey(encryptionKeyPath)
		if err != nil {
			t.logger.Error("Failed to load existing encryption key", zap.Error(err))
			return fmt.Errorf("failed to load existing encryption key: %w", err)
		}
		encryptionKey = key
		t.logger.Info("Loaded existing encryption key", zap.String("path", encryptionKeyPath))
	}

	// Initialize the secure vault
	vault, err := securedata.NewVault(encryptionKey)
	if err != nil {
		t.logger.Error("Failed to initialize secure vault", zap.Error(err))
		return fmt.Errorf("failed to initialize secure vault: %w", err)
	}
	t.vault = vault
	t.logger.Info("Secure vault initialized successfully")

	return nil
}

// -------------- Terminal Constructor --------------

func NewTerminal(parsedOptions *options.Options) (*Terminal, error) {
	// Set up logging
	logger, err := setupLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to setup logger: %v", err)
	}

	// Attempt to load a custom font
	fontPath := "resources/futuristic_font.ttf"
	font := rl.LoadFontEx(fontPath, FontPointSize, nil, 0)
	if font.BaseSize == 0 {
		logger.Warn("Failed to load custom font, fallback to default", zap.String("fontPath", fontPath))
		font = rl.GetFontDefault()
	}

	// Create a background particle system
	rand.Seed(time.Now().UnixNano())
	particles := generateParticles(ParticleCount)

	// Initialize a menu with some items
	menuItems := []string{"Scan CDN", "Generate Report", "Exit"}
	menu := &MenuManager{
		items:       menuItems,
		currentItem: 0,
	}

	t := &Terminal{
		font:             font,
		particles:        particles,
		menuManager:      menu,
		gracefulShutdown: make(chan os.Signal, 1),
		isScanning:       false,
		cdnResults:       []string{},
		logger:           logger,
		options:          parsedOptions,
	}
	menu.terminal = t

	// Initialize post-quantum secure vault
	if err := t.initializeVault(); err != nil {
		logger.Error("Failed to initialize secure vault", zap.Error(err))
		return nil, err
	}

	// Start metrics server if needed (optional)
	// go startMetricsServer(logger)

	logger.Info("CDN Crawler terminal created successfully")
	return t, nil
}

// -------------- Particle Generation --------------

func generateParticles(count int) []*Particle {
	pts := make([]*Particle, count)
	for i := 0; i < count; i++ {
		p := &Particle{
			x:  float32(rand.Intn(ScreenWidth)),
			y:  float32(rand.Intn(ScreenHeight)),
			dx: (rand.Float32()*2 - 1) * 2,
			dy: (rand.Float32()*2 - 1) * 2,
			color: rl.Color{
				R: uint8(rand.Intn(256)),
				G: uint8(rand.Intn(256)),
				B: uint8(rand.Intn(256)),
				A: 255,
			},
		}
		pts[i] = p
	}
	return pts
}

// -------------- Terminal Methods --------------

func (t *Terminal) Update() {
	// Update particles
	if !t.isScanning {
		for _, p := range t.particles {
			p.x += p.dx
			p.y += p.dy
			if p.x < 0 || p.x > ScreenWidth {
				p.dx *= -1
			}
			if p.y < 0 || p.y > ScreenHeight {
				p.dy *= -1
			}
		}
	}
	// Update menu
	t.menuManager.Update()
}

func (t *Terminal) Draw() {
	rl.BeginDrawing()
	rl.ClearBackground(rl.DarkBlue)

	// Draw particles
	for _, p := range t.particles {
		rl.DrawRectangle(int32(p.x), int32(p.y), 5, 5, p.color)
	}

	// Title
	rl.DrawTextEx(t.font, "CDN Crawler", rl.NewVector2(20, 40), float32(FontPointSize), 2, rl.White)

	// If scanning is in progress
	if t.isScanning {
		rl.DrawTextEx(t.font, "Scanning in progress...", rl.NewVector2(20, 80), 24, 2, rl.Green)
	}

	// Render menu
	t.menuManager.Draw()

	rl.EndDrawing()
}

func (t *Terminal) Shutdown() {
	t.logger.Info("CDNCrawler shutting down")

	// Close the secure vault
	if t.vault != nil {
		if err := t.vault.Close(); err != nil {
			t.logger.Error("Failed to close secure vault", zap.Error(err))
		} else {
			t.logger.Info("Secure vault closed successfully")
		}
	}

	// Sync logger
	if t.logger != nil {
		_ = t.logger.Sync()
	}

	rl.CloseWindow()
	os.Exit(0)
}

// -------------- Scanning Logic --------------

// StartScan initiates the CDN scanning process with concurrency and secure data handling.
func (t *Terminal) StartScan() {
	t.mu.Lock()
	if t.isScanning {
		t.logger.Warn("Scan already in progress")
		t.mu.Unlock()
		return
	}
	t.isScanning = true
	t.cdnResults = []string{}
	t.mu.Unlock()

	t.logger.Info("Starting CDN scan...")

	go func() {
		// Define CDN targets to scan
		cdnTargets := []string{"cdn1.example.com", "cdn2.example.net", "assets.example.org"}
		var wg sync.WaitGroup

		for _, cdn := range cdnTargets {
			wg.Add(1)
			go func(target string) {
				defer wg.Done()
				scanStart := time.Now()
				success, msg := performCDNCheck(target)
				duration := time.Since(scanStart).Seconds()

				line := fmt.Sprintf("%s => %s", target, msg)
				t.mu.Lock()
				t.cdnResults = append(t.cdnResults, line)
				t.mu.Unlock()

				if success {
					t.logger.Info("CDN scan successful", zap.String("target", target), zap.Float64("duration_sec", duration))
					// Encrypt and store the scan result securely
					if err := t.vault.StoreEncryptedData(target, line); err != nil {
						t.logger.Error("Failed to store encrypted scan result", zap.String("target", target), zap.Error(err))
					}
				} else {
					t.logger.Warn("CDN scan failed", zap.String("target", target), zap.String("error", msg))
				}
			}(cdn)
		}

		wg.Wait()

		t.mu.Lock()
		t.isScanning = false
		t.mu.Unlock()
		t.logger.Info("CDN scan completed", zap.Int("total_targets", len(cdnTargets)))
	}()
}

// performCDNCheck simulates a CDN check with random success/failure.
func performCDNCheck(target string) (bool, string) {
	// Simulate a scan delay
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	r := rand.Float32()
	if r < 0.6 { // 60% chance of success
		return true, "OK"
	}
	return false, "Request timeout/Invalid response"
}

// ShowSettings can open a sub-menu or prompt for options
func (t *Terminal) ShowSettings() {
	t.logger.Info("Opening settings (placeholder)")
	// Future logic e.g., UI for configuring scan parameters
}

// -------------- Reporting --------------

// GenerateReports writes the encrypted scan results to CSV and PDF.
func (t *Terminal) GenerateReports() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.cdnResults) == 0 {
		t.logger.Warn("No CDN results to report on")
		return nil
	}

	if err := os.MkdirAll(ReportDir, 0755); err != nil {
		t.logger.Error("Failed to create report directory", zap.Error(err))
		return err
	}

	timestamp := time.Now().Format("20060102_150405")
	csvPath := filepath.Join(ReportDir, fmt.Sprintf("cdncrawler_report_%s.csv", timestamp))
	pdfPath := filepath.Join(ReportDir, fmt.Sprintf("cdncrawler_report_%s.pdf", timestamp))

	// Write CSV report
	if err := t.writeCSVReport(csvPath); err != nil {
		return err
	}

	// Write PDF report
	if err := t.writePDFReport(pdfPath); err != nil {
		return err
	}

	t.logger.Info("Reports generated successfully", zap.String("csv", csvPath), zap.String("pdf", pdfPath))
	return nil
}

// writeCSVReport writes the scan results to a CSV file.
func (t *Terminal) writeCSVReport(outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		t.logger.Error("Failed to create CSV report", zap.Error(err))
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Write header
	if err := w.Write([]string{"CDN", "Scan Result"}); err != nil {
		t.logger.Error("Failed to write CSV header", zap.Error(err))
		return err
	}

	// Write data
	for _, line := range t.cdnResults {
		tokens := strings.SplitN(line, " => ", 2)
		if len(tokens) < 2 {
			tokens = append(tokens, "")
		}
		if err := w.Write(tokens); err != nil {
			t.logger.Error("Failed to write CSV row", zap.Error(err))
		}
	}

	t.logger.Info("CSV report generated", zap.String("path", outputPath))
	return nil
}

// writePDFReport writes the scan results to a PDF file.
func (t *Terminal) writePDFReport(outputPath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "CDN Crawler Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(60, 8, "CDN")
	pdf.Cell(120, 8, "Result")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 12)
	for _, line := range t.cdnResults {
		tokens := strings.SplitN(line, " => ", 2)
		if len(tokens) < 2 {
			tokens = append(tokens, "")
		}
		pdf.Cell(60, 8, tokens[0])
		x := pdf.GetX()
		y := pdf.GetY()
		pdf.MultiCell(120, 8, tokens[1], "", "", false)
		pdf.SetXY(x+60+120, y)
		pdf.Ln(8)
	}

	if err := pdf.OutputFileAndClose(outputPath); err != nil {
		t.logger.Error("Failed to write PDF report", zap.Error(err))
		return err
	}
	t.logger.Info("PDF report generated", zap.String("path", outputPath))
	return nil
}
