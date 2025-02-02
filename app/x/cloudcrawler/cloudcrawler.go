package cloudcrawler

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/jung-kurt/gofpdf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"ghostshell/cloudcrawler/options"
	"ghostshell/oqs"        // Hypothetical post-quantum security package
	"ghostshell/securedata" // Hypothetical secure data handling package
)

const (
	ScreenWidth   = 1280
	ScreenHeight  = 720
	ParticleCount = 50
	FontPointSize = 24

	LogDir        = "ghostshell/logging"
	ReportDir     = "ghostshell/reporting"
	SecureDataDir = "ghostshell/secure_data"
)

// Particle is a small, moving background element.
type Particle struct {
	x, y   float32
	dx, dy float32
	color  rl.Color
}

// MenuManager handles a simple text-based menu in the Raylib UI.
type MenuManager struct {
	items       []string
	currentItem int
	crawler     *Terminal
}

// Terminal represents the main Cloud Crawler application state.
type Terminal struct {
	font        rl.Font
	particles   []*Particle
	menuManager *MenuManager
	logger      *zap.Logger

	scanning  bool
	mu        sync.Mutex
	scanItems []string // results from scanning

	// Secure Vault for Post-Quantum Security
	vault *securedata.Vault

	// We store the context or channel for graceful shutdown
	shutdownChan chan os.Signal
	options      *options.Options
}

// -------------- Logging Setup --------------

// setupLogger initializes a Zap logger with a timestamped log file in ISO8601 format.
func setupLogger() (*zap.Logger, error) {
	if err := os.MkdirAll(LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	// ISO8601 timestamp e.g., 2023-10-25T15:30:45Z
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	logFileName := fmt.Sprintf("cloudcrawler_log_%s.log", strings.ReplaceAll(timestamp, ":", "-"))
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

// -------------- Terminal: Constructor --------------

func NewTerminal(parsedOpts *options.Options) (*Terminal, error) {
	logger, err := setupLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to setup logger: %v", err)
	}

	// load font
	fontPath := "resources/futuristic_font.ttf"
	font := rl.LoadFontEx(fontPath, FontPointSize, nil, 0)
	if font.BaseSize == 0 {
		logger.Warn("Failed to load custom font, fallback to default",
			zap.String("fontPath", fontPath))
		font = rl.GetFontDefault()
	}

	// random background particles
	rand.Seed(time.Now().UnixNano())
	parts := make([]*Particle, ParticleCount)
	for i := 0; i < ParticleCount; i++ {
		parts[i] = &Particle{
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
	}

	// menu items
	menuItems := []string{"Scan Cloud", "Generate Report", "Exit"}
	menu := &MenuManager{
		items:       menuItems,
		currentItem: 0,
	}

	t := &Terminal{
		font:         font,
		particles:    parts,
		menuManager:  menu,
		logger:       logger,
		scanning:     false,
		scanItems:    []string{},
		shutdownChan: make(chan os.Signal, 1),
		options:      parsedOpts,
	}

	// link them
	menu.crawler = t

	// Initialize post-quantum secure vault
	if err := t.initializeVault(); err != nil {
		logger.Error("Failed to initialize secure vault", zap.Error(err))
		return nil, err
	}

	logger.Info("Cloud Crawler terminal created successfully")
	return t, nil
}

// -------------- MenuManager Methods --------------

func (m *MenuManager) Update() {
	if rl.IsKeyPressed(rl.KeyDown) {
		m.currentItem++
		if m.currentItem >= len(m.items) {
			m.currentItem = 0
		}
	}
	if rl.IsKeyPressed(rl.KeyUp) {
		m.currentItem--
		if m.currentItem < 0 {
			m.currentItem = len(m.items) - 1
		}
	}
	if rl.IsKeyPressed(rl.KeyEnter) {
		m.executeItem()
	}
}

func (m *MenuManager) Draw() {
	baseY := float32(100)
	spacing := float32(30)

	for i, item := range m.items {
		color := rl.White
		if i == m.currentItem {
			color = rl.Yellow
		}
		rl.DrawText(item, 50, int32(baseY+float32(i)*spacing), 24, color)
	}
}

func (m *MenuManager) executeItem() {
	selection := m.items[m.currentItem]
	switch selection {
	case "Scan Cloud":
		m.crawler.StartScan()
	case "Generate Report":
		if err := m.crawler.GenerateReports(); err != nil {
			m.crawler.logger.Error("Report generation error", zap.Error(err))
		}
	case "Exit":
		m.crawler.Shutdown()
	default:
		m.crawler.logger.Warn("Unknown menu item", zap.String("item", selection))
	}
}

// -------------- Terminal Methods --------------

func (t *Terminal) Update() {
	// If not scanning, update particle movement
	if !t.scanning {
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
		rl.DrawCircle(int32(p.x), int32(p.y), 3, p.color)
	}

	// Title
	rl.DrawTextEx(t.font, "Cloud Crawler", rl.NewVector2(40, 40), float32(FontPointSize), 2, rl.White)

	// If scanning is in progress
	if t.scanning {
		rl.DrawTextEx(t.font, "Scanning in progress...", rl.NewVector2(40, 90), 24, 2, rl.Green)
	}

	// Render menu
	t.menuManager.Draw()

	// Show current time or some status
	localTime := time.Now().Format("15:04:05")
	status := fmt.Sprintf("Local Time: %s", localTime)
	rl.DrawTextEx(t.font, status, rl.NewVector2(40, 130), 20, 1, rl.White)

	rl.EndDrawing()
}

// Shutdown flushes logs, closes Raylib, and exits the application gracefully.
func (t *Terminal) Shutdown() {
	t.logger.Info("Cloud Crawler shutting down")

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

// StartScan initiates the Cloud scanning process with concurrency and secure data handling.
func (t *Terminal) StartScan() {
	t.mu.Lock()
	if t.scanning {
		t.logger.Warn("Scan already in progress")
		t.mu.Unlock()
		return
	}
	t.scanning = true
	t.scanItems = []string{}
	t.mu.Unlock()

	t.logger.Info("Starting concurrency-based cloud scan...")

	go func() {
		start := time.Now()
		cloudEndpoints := []string{"aws.amazon.com", "cloud.google.com", "azure.microsoft.com"}
		var wg sync.WaitGroup

		for _, ce := range cloudEndpoints {
			wg.Add(1)
			go func(endpoint string) {
				defer wg.Done()
				scanStart := time.Now()
				success, msg := performCloudCheck(endpoint)
				duration := time.Since(scanStart).Seconds()

				line := fmt.Sprintf("%s => %s", endpoint, msg)

				t.mu.Lock()
				t.scanItems = append(t.scanItems, line)
				t.mu.Unlock()

				if success {
					t.logger.Info("Cloud scan successful", zap.String("endpoint", endpoint), zap.Float64("duration_sec", duration))
					// Encrypt and store the scan result securely
					if err := t.vault.StoreEncryptedData(endpoint, line); err != nil {
						t.logger.Error("Failed to store encrypted scan result", zap.String("endpoint", endpoint), zap.Error(err))
					}
				} else {
					t.logger.Warn("Cloud scan failed", zap.String("endpoint", endpoint), zap.String("error", msg))
				}
			}(ce)
		}

		wg.Wait()

		duration := time.Since(start).Seconds()
		t.logger.Info("Cloud scan completed", zap.Int("total_endpoints", len(cloudEndpoints)), zap.Float64("total_duration_sec", duration))

		t.mu.Lock()
		t.scanning = false
		t.mu.Unlock()
	}()
}

// performCloudCheck simulates a cloud endpoint check with random success/failure.
func performCloudCheck(endpoint string) (bool, string) {
	// Simulate a scan delay
	time.Sleep(time.Duration(rand.Intn(700)) * time.Millisecond)
	r := rand.Float32()
	if r < 0.5 { // 50% chance of success
		return true, "OK"
	}
	return false, "Request timeout/Invalid response"
}

// GenerateReports writes the encrypted scan results to CSV and PDF.
func (t *Terminal) GenerateReports() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.scanItems) == 0 {
		t.logger.Warn("No scan results to report on")
		return nil
	}

	if err := os.MkdirAll(ReportDir, 0755); err != nil {
		t.logger.Error("Failed to create report directory", zap.Error(err))
		return err
	}
	timestamp := time.Now().Format("20060102_150405")
	csvPath := filepath.Join(ReportDir, fmt.Sprintf("cloudcrawler_report_%s.csv", timestamp))
	pdfPath := filepath.Join(ReportDir, fmt.Sprintf("cloudcrawler_report_%s.pdf", timestamp))

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

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"Cloud Endpoint", "Scan Result"}); err != nil {
		t.logger.Error("Failed to write CSV header", zap.Error(err))
		return err
	}

	// Write data
	for _, line := range t.scanItems {
		tokens := strings.SplitN(line, " => ", 2)
		if len(tokens) < 2 {
			tokens = append(tokens, "")
		}
		if err := writer.Write(tokens); err != nil {
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
	pdf.Cell(40, 10, "Cloud Crawler Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(60, 8, "Cloud Endpoint")
	pdf.Cell(120, 8, "Scan Result")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 12)
	for _, line := range t.scanItems {
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

// -------------- Main --------------

func MainCloudCrawler(parsedOpts *options.Options) {
	t, err := NewTerminal(parsedOpts)
	if err != nil {
		fmt.Printf("Error initializing terminal: %v\n", err)
		os.Exit(1)
	}
	// Start Raylib
	rl.InitWindow(ScreenWidth, ScreenHeight, "CloudCrawler")
	rl.SetTargetFPS(60)
	defer rl.CloseWindow()

	// Setup graceful shutdown
	signal.Notify(t.shutdownChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-t.shutdownChan
		t.logger.Info("Received shutdown signal, shutting down gracefully...")
		t.Shutdown()
	}()

	// Main loop
	for !rl.WindowShouldClose() {
		t.Update()
		t.Draw()
	}

	t.Shutdown()
}
