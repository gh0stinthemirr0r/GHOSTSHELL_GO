package asnscanner

import (
	"context"
	"encoding/csv"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/jung-kurt/gofpdf"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"ghostshell/options"
	"ghostshell/oqs"     // Hypothetical post-quantum security package
	"ghostshell/storage" // Hypothetical storage package for secure vault
)

// -------------- Constants & Paths --------------

const (
	ScreenWidth    = 1280
	ScreenHeight   = 720
	ParticleCount  = 50
	FontPointSize  = 30
	MaxLogMessages = 100
	ReportDir      = "ghostshell/reporting"
	LogDir         = "ghostshell/logging"
	SecureDataDir  = "ghostshell/secure_data"
)

// -------------- Prometheus Metrics --------------

var (
	scannedTargets = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scanned_targets_total",
			Help: "Total number of targets scanned successfully by asnscanner.",
		},
		[]string{"target"},
	)
	scanDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "scan_duration_seconds",
			Help:    "Duration of scans in seconds.",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0},
		},
		[]string{"target"},
	)
	invalidTargets = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "invalid_targets_total",
			Help: "Total number of invalid or unreachable targets encountered.",
		},
	)
)

func init() {
	prometheus.MustRegister(scannedTargets, scanDuration, invalidTargets)
}

// -------------- Data Structures --------------

type Terminal struct {
	font             rl.Font
	particles        []*Particle
	menuManager      *MenuManager
	logger           *zap.Logger
	gracefulShutdown chan os.Signal

	// Options from CLI
	options *options.Options

	// State
	isScanning  bool
	scanResults []string // each line: "target => result"

	// Post-Quantum Secure Vault
	vault *storage.Vault

	// Mutex for thread-safe operations
	mutex sync.Mutex
}

type Particle struct {
	x, y   float32
	dx, dy float32
	color  rl.Color
}

// MenuManager handles a simple text-based menu in the Raylib UI.
type MenuManager struct {
	active      bool
	items       []string
	currentItem int
	terminal    *Terminal
}

// -------------- Logging Setup --------------

func setupLogger() (*zap.Logger, error) {
	if err := os.MkdirAll(LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	// ISO8601 timestamp e.g., 2023-10-25T15:30:45Z
	timestamp := time.Now().UTC().Format(time.RFC3339)
	logFileName := fmt.Sprintf("ai_log_%s.log", strings.ReplaceAll(timestamp, ":", "-"))
	logFilePath := filepath.Join(LogDir, logFileName)

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logFilePath, "stdout"}

	return cfg.Build()
}

// -------------- Secure Vault Initialization --------------

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
	vault, err := storage.NewVault(encryptionKey)
	if err != nil {
		t.logger.Error("Failed to initialize secure vault", zap.Error(err))
		return fmt.Errorf("failed to initialize secure vault: %w", err)
	}
	t.vault = vault
	t.logger.Info("Secure vault initialized successfully")

	return nil
}

// -------------- Terminal / Setup --------------

func NewTerminal(parsedOptions *options.Options) (*Terminal, error) {
	// Set up logging
	logger, err := setupLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %v", err)
	}

	// Load custom font or fallback
	fontPath := "resources/futuristic_font.ttf"
	font := rl.LoadFontEx(fontPath, FontPointSize, nil, 0)
	if font.BaseSize == 0 {
		logger.Warn("Failed to load custom font, fallback to default", zap.String("fontPath", fontPath))
		font = rl.GetFontDefault()
	}

	particles := generateParticles(ParticleCount)
	menuItems := []string{"Scan Network", "Generate Report", "Exit"}
	menu := &MenuManager{
		items:       menuItems,
		currentItem: 0,
	}

	t := &Terminal{
		font:             font,
		particles:        particles,
		menuManager:      menu,
		logger:           logger,
		gracefulShutdown: make(chan os.Signal, 1),
		options:          parsedOptions,
		isScanning:       false,
		scanResults:      []string{},
	}
	menu.terminal = t

	// Initialize post-quantum secure vault
	if err := t.initializeVault(); err != nil {
		logger.Error("Failed to initialize secure vault", zap.Error(err))
		return nil, err
	}

	// Start metrics server
	go startMetricsServer(logger)

	logger.Info("ASN Scanner Terminal initialized successfully")
	return t, nil
}

// -------------- Particles --------------

func generateParticles(count int) []*Particle {
	rand.Seed(time.Now().UnixNano())
	pts := make([]*Particle, count)
	for i := 0; i < count; i++ {
		pts[i] = &Particle{
			x:  float32(rand.Intn(ScreenWidth)),
			y:  float32(rand.Intn(ScreenHeight)),
			dx: (rand.Float32()*2 - 1) * 2,
			dy: (rand.Float32()*2 - 1) * 2,
			color: rl.NewColor(
				uint8(rand.Intn(256)),
				uint8(rand.Intn(256)),
				uint8(rand.Intn(256)),
				255,
			),
		}
	}
	return pts
}

// -------------- Metrics Server --------------

func startMetricsServer(logger *zap.Logger) {
	http.Handle("/metrics", promhttp.Handler())
	logger.Info("Starting Prometheus metrics server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatal("Failed to start Prometheus metrics server", zap.Error(err))
	}
}

// -------------- Terminal Methods --------------

func (t *Terminal) Update() {
	// Update particles
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
	// Update menu
	t.menuManager.Update()
}

func (t *Terminal) Draw() {
	rl.BeginDrawing()
	rl.ClearBackground(rl.DarkBlue)

	for _, p := range t.particles {
		rl.DrawCircle(int32(p.x), int32(p.y), 3, p.color)
	}

	// Title
	title := "ASN Scanner"
	rl.DrawTextEx(t.font, title, rl.Vector2{X: 20, Y: 40}, FontPointSize, 2, rl.White)

	// If scanning
	if t.isScanning {
		rl.DrawTextEx(t.font, "Scanning in progress...", rl.NewVector2(20, 100), 24, 2, rl.Green)
	}

	// Render menu
	t.menuManager.Draw()

	rl.EndDrawing()
}

func (t *Terminal) Shutdown() {
	t.logger.Info("Shutting down ASN Scanner...")
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

func (t *Terminal) ScanNetwork() {
	// For demonstration, let's pick some random "targets"
	targets := []string{"8.8.8.8", "1.1.1.1", "192.168.0.1"}
	t.scanResults = []string{}
	t.isScanning = true

	var wg sync.WaitGroup
	for _, target := range targets {
		wg.Add(1)
		go func(tg string) {
			defer wg.Done()
			scanStart := time.Now()
			success, errMsg := doScan(tg)
			duration := time.Since(scanStart).Seconds()

			if success {
				scannedTargets.WithLabelValues(tg).Inc()
				scanDuration.WithLabelValues(tg).Observe(duration)
				line := fmt.Sprintf("%s => SUCCESS (%.2fs)", tg, duration)
				t.logger.Info("Scan success", zap.String("target", tg), zap.Float64("duration", duration))
				t.scanResults = append(t.scanResults, line)

				// Encrypt and store the scan result securely
				if err := t.vault.StoreEncryptedData(tg, line); err != nil {
					t.logger.Error("Failed to store encrypted scan result", zap.String("target", tg), zap.Error(err))
				}
			} else {
				invalidTargets.Inc()
				line := fmt.Sprintf("%s => FAIL (%s)", tg, errMsg)
				t.logger.Error("Scan fail", zap.String("target", tg), zap.String("error", errMsg))
				t.scanResults = append(t.scanResults, line)
			}
		}(target)
	}
	wg.Wait()

	t.isScanning = false
	t.logger.Info("Scanning completed", zap.Int("targets", len(targets)))
}

func doScan(target string) (bool, string) {
	// A stub: pretend half the time we succeed, half fail
	r := rand.Float32()
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	if r < 0.5 {
		return false, "Timeout/No route"
	}
	return true, ""
}

// -------------- Reporting --------------

func (t *Terminal) GenerateReport() error {
	timestamp := time.Now().Format("20060102_150405")
	csvPath := filepath.Join(ReportDir, fmt.Sprintf("asnscanner_report_%s.csv", timestamp))
	pdfPath := filepath.Join(ReportDir, fmt.Sprintf("asnscanner_report_%s.pdf", timestamp))

	if err := os.MkdirAll(ReportDir, 0755); err != nil {
		t.logger.Error("Failed to create report directory", zap.Error(err))
		return err
	}

	// CSV
	if err := t.writeCSVReport(csvPath); err != nil {
		return err
	}
	// PDF
	if err := t.writePDFReport(pdfPath); err != nil {
		return err
	}

	return nil
}

func (t *Terminal) writeCSVReport(outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		t.logger.Error("Failed to create CSV", zap.Error(err))
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"Target", "Scan Result"}); err != nil {
		t.logger.Error("Failed to write CSV header", zap.Error(err))
		return err
	}
	for _, r := range t.scanResults {
		// e.g.: "8.8.8.8 => SUCCESS (0.12s)"
		tokens := strings.SplitN(r, " => ", 2)
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

func (t *Terminal) writePDFReport(outputPath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "ASN Scanner Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(60, 8, "Target")
	pdf.Cell(120, 8, "Result")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 12)
	for _, line := range t.scanResults {
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

// -------------- MenuManager --------------

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
		rl.DrawText(item, 50, int32(baseY+spacing*float32(i)), 20, color)
	}
}

func (m *MenuManager) executeItem() {
	item := m.items[m.currentItem]
	switch item {
	case "Scan Network":
		m.terminal.ScanNetwork()
	case "Generate Report":
		if err := m.terminal.GenerateReport(); err != nil {
			m.terminal.logger.Error("Report generation error", zap.Error(err))
		}
	case "Exit":
		m.terminal.Shutdown()
	}
}

// -------------- Main Execution --------------

func MainASNScanner(parsedOptions *options.Options) {
	// Initialize the Raylib window
	rl.InitWindow(ScreenWidth, ScreenHeight, "ASN Scanner")
	defer rl.CloseWindow()

	terminal, err := NewTerminal(parsedOptions)
	if err != nil {
		fmt.Printf("Failed to init Terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Shutdown()

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		terminal.logger.Info("Received shutdown signal, closing gracefully...")
		cancel()
	}()

	rl.SetTargetFPS(60)
	for !rl.WindowShouldClose() && ctx.Err() == nil {
		terminal.Update()
		terminal.Draw()
	}
	// Exit
	terminal.Shutdown()
}
