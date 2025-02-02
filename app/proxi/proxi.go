package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/jung-kurt/gofpdf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	// Hypothetical local PQ modules
	"ghostshell/oqs_network"
)

const (
	logDir       = "ghostshell/logging"
	reportDir    = "ghostshell/reporting"
	windowWidth  = 1280
	windowHeight = 720
	fontSize     = 24
)

// -------------- Logging --------------

var logger *zap.Logger

func setupLogger() error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(logDir, fmt.Sprintf("proxi_log_%s.log", timestamp))

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logFile, "stdout"}

	var err error
	logger, err = cfg.Build()
	if err != nil {
		return fmt.Errorf("failed to create logger: %v", err)
	}
	return nil
}

// -------------- CLI / Options --------------

type Options struct {
	ListenAddress string
	Port          int
}

func parseInput() (*Options, error) {
	// For demonstration, we hardcode or parse from environment/args
	return &Options{
		ListenAddress: "127.0.0.1",
		Port:          8080,
	}, nil
}

// -------------- Raylib UI --------------

type Terminal struct {
	font rl.Font
}

func newTerminal() (*Terminal, error) {
	fontPath := "resources/roboto.ttf"
	font := rl.LoadFontEx(fontPath, fontSize, nil, 0)
	if font.BaseSize == 0 {
		logger.Warn("Failed to load TTF font, fallback to default", zap.String("fontPath", fontPath))
		font = rl.GetFontDefault()
	}
	return &Terminal{font: font}, nil
}

func (t *Terminal) Shutdown() {
	rl.UnloadFont(t.font)
	rl.CloseWindow()
	logger.Info("Raylib terminal shut down.")
}

// -------------- PQ TLS Config (Placeholder) --------------

func GenerateTLSConfig(logger *zap.Logger) (interface{}, error) {
	// Hypothetical quantum-safe config
	logger.Info("Generating quantum-safe TLS configuration")
	// e.g., we create an ephemeral key from oqs_vault, or do something in oqs_network
	// Return a placeholder
	return &oqs_network.PQTLSConfig{}, nil
}

// -------------- MITM Proxy --------------

type MITMProxy struct {
	listenAddress string
	port          int
	tlsConfig     interface{}
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewMITMProxy(logger *zap.Logger, tlsConfig interface{}, addr string, port int) (*MITMProxy, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &MITMProxy{
		listenAddress: addr,
		port:          port,
		tlsConfig:     tlsConfig,
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

// Start the proxy
func (p *MITMProxy) Start() error {
	// Placeholder for actually listening & intercepting
	logger.Info("Starting PQ MITM Proxy",
		zap.String("address", p.listenAddress),
		zap.Int("port", p.port),
	)
	// For demonstration, just simulate a run:
	go func() {
		// e.g. net.Listen + handle connections
		time.Sleep(time.Hour)
	}()
	return nil
}

// Shutdown the proxy gracefully
func (p *MITMProxy) Shutdown() {
	p.cancel()
	logger.Info("Proxy shutdown completed.")
}

// -------------- Reporting (CSV/PDF) --------------

func createReport(logger *zap.Logger) {
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		logger.Error("Failed to create reporting directory", zap.Error(err))
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	pdfFile := filepath.Join(reportDir, fmt.Sprintf("proxi_report_%s.pdf", timestamp))
	csvFile := filepath.Join(reportDir, fmt.Sprintf("proxi_report_%s.csv", timestamp))

	// CSV
	if err := writeCSV(csvFile, logger); err != nil {
		logger.Error("Error writing CSV", zap.Error(err))
	} else {
		logger.Info("CSV report generated", zap.String("file", csvFile))
	}

	// PDF
	if err := writePDF(pdfFile, logger); err != nil {
		logger.Error("Error writing PDF", zap.Error(err))
	} else {
		logger.Info("PDF report generated", zap.String("file", pdfFile))
	}
}

func writeCSV(path string, logger *zap.Logger) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	f.WriteString("URL,Status,Details\n")
	f.WriteString("example.com,200,OK\n")
	return nil
}

func writePDF(path string, logger *zap.Logger) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "Proxi Post Quantum Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	pdf.MultiCell(190, 6, "Placeholder for real proxy interactions", "", "", false)

	if err := pdf.OutputFileAndClose(path); err != nil {
		return err
	}
	return nil
}

// -------------- Main --------------

func main() {
	// Setup logger
	if err := setupLogger(); err != nil {
		fmt.Printf("Error initializing logging: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	logger.Info("Proxi starting")

	// Parse options
	opts, err := parseInput()
	if err != nil {
		logger.Fatal("Error parsing input", zap.Error(err))
	}

	// Create TLS config (Quantum-Safe)
	tlsConfig, err := GenerateTLSConfig(logger)
	if err != nil {
		logger.Fatal("Failed to generate PQ TLS config", zap.Error(err))
	}

	// Create the PQ MITM Proxy
	proxy, err := NewMITMProxy(logger, tlsConfig, opts.ListenAddress, opts.Port)
	if err != nil {
		logger.Fatal("Failed to initialize MITM proxy", zap.Error(err))
	}

	// Start the proxy
	if err := proxy.Start(); err != nil {
		logger.Fatal("Error starting proxy", zap.Error(err))
	}

	// Setup graceful signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Shutting down Proxi gracefully...")
		proxy.Shutdown()
		os.Exit(0)
	}()

	// Raylib UI
	rl.InitWindow(windowWidth, windowHeight, "Proxi - PQ Secure Proxy")
	rl.SetTargetFPS(60)
	runtime.LockOSThread()

	term, err := newTerminal()
	if err != nil {
		logger.Fatal("Failed to init Terminal", zap.Error(err))
	}
	defer term.Shutdown()

	// Generate a report in the background for demonstration
	go createReport(logger)

	// Main loop
	for !rl.WindowShouldClose() {
		if rl.IsKeyPressed(rl.KeyEscape) {
			proxy.Shutdown()
			break
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.DarkGray)

		// Basic text
		x, y := float32(20), float32(40)
		rl.DrawTextEx(term.font, "Proxi - Post Quantum Secure MITM Proxy", rl.NewVector2{x, y}, fontSize, 2, rl.White)
		y += 40
		rl.DrawTextEx(term.font, fmt.Sprintf("Listening on %s:%d", opts.ListenAddress, opts.Port), rl.NewVector2{x, y}, fontSize-4, 2, rl.Green)
		y += 30
		localTime := time.Now().Format("15:04:05")
		rl.DrawTextEx(term.font, fmt.Sprintf("Local Time: %s", localTime), rl.NewVector2{x, y}, fontSize-4, 2, rl.LightGray)
		y += 30
		rl.DrawTextEx(term.font, "Press ESC to Quit", rl.NewVector2{x, y}, fontSize-4, 2, rl.LightGray)

		rl.EndDrawing()
	}

	rl.CloseWindow()
	logger.Info("Application shutdown completed")
}
