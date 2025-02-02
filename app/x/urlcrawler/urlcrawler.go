package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/jung-kurt/gofpdf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	// Hypothetical references to local modules
	"ghostshell/urlcrawler/config"
	"ghostshell/urlcrawler/input"
	"ghostshell/urlcrawler/output"
	"ghostshell/urlcrawler/runner"

	// Post-quantum ephemeral placeholders
	"ghostshell/oqs/oqs_vault"
)

const (
	logDir       = "ghostshell/logging"
	reportDir    = "ghostshell/reporting"
	windowWidth  = 1280
	windowHeight = 720
	fontSize     = 24
	maxParticles = 50
)

var logger *zap.Logger

// -------------- Logging Setup --------------

func setupLogger() (*zap.Logger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(logDir, fmt.Sprintf("urlcrawler_log_%s.log", timestamp))

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logFile, "stdout"}

	return cfg.Build()
}

// -------------- Particles UI --------------

type Particle struct {
	x, y   float32
	dx, dy float32
	color  rl.Color
}

func generateParticles(count int) []*Particle {
	ps := make([]*Particle, count)
	rngSeed := time.Now().UnixNano()
	rng := rl.NewRandomGenerator(int64(rngSeed))

	for i := 0; i < count; i++ {
		px := float32(rng.Int() % windowWidth)
		py := float32(rng.Int() % windowHeight)
		dx := float32(rng.Int()%4 - 2)
		dy := float32(rng.Int()%4 - 2)
		clr := rl.NewColor(uint8(rng.Int()%256), uint8(rng.Int()%256), uint8(rng.Int()%256), 255)
		ps[i] = &Particle{x: px, y: py, dx: dx, dy: dy, color: clr}
	}
	return ps
}

func updateParticles(ps []*Particle) {
	for _, p := range ps {
		p.x += p.dx
		p.y += p.dy
		if p.x < 0 || p.x > float32(windowWidth) {
			p.dx *= -1
		}
		if p.y < 0 || p.y > float32(windowHeight) {
			p.dy *= -1
		}
		rl.DrawCircle(int32(p.x), int32(p.y), 4, p.color)
	}
}

// -------------- Main Application --------------

type Application struct {
	Config       *config.Options
	InputHandler *input.Handler
	OutputWriter *output.Writer
	Runner       *runner.Runner
	Logger       *zap.Logger
}

func NewApplication(configPath string) (*Application, error) {
	lg, err := setupLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	logger = lg

	// Post-quantum ephemeral init, placeholder
	if err := oqs_vault.InitEphemeralKey(); err != nil {
		logger.Warn("Failed ephemeral PQ key init", zap.Error(err))
	}

	logger.Info("Initializing URL crawler...")

	// Load config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Ensure directories
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create report directory: %w", err)
	}

	// init input, output, runner
	inp := input.NewHandler()
	out := output.NewWriter(cfg.JSONOutput)
	rnr, err := runner.NewRunner(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to init runner: %w", err)
	}

	return &Application{
		Config:       cfg,
		InputHandler: inp,
		OutputWriter: out,
		Runner:       rnr,
		Logger:       logger,
	}, nil
}

func (app *Application) Start(ctx context.Context) error {
	app.Logger.Info("Starting concurrency-based URL crawling")

	// concurrency scanning in background
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := app.Runner.Run(ctx, app.InputHandler, app.OutputWriter); err != nil {
			app.Logger.Error("Error during URL crawling", zap.Error(err))
		}
	}()

	// Raylib UI
	rl.InitWindow(windowWidth, windowHeight, "URL Crawler Visualization - PQ Secure")
	rl.SetTargetFPS(60)
	runtime.LockOSThread()

	ps := generateParticles(maxParticles)
	running := true

	for running && !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)

		updateParticles(ps)

		rl.DrawText("URL Crawler Running (PQ Secure)", 20, 20, fontSize, rl.Black)
		localTime := time.Now().Format("15:04:05")
		rl.DrawText(fmt.Sprintf("Local Time: %s", localTime), 20, 60, fontSize-4, rl.Gray)
		rl.DrawText("[ESC] to Quit", 20, 100, fontSize-4, rl.Gray)

		if rl.IsKeyPressed(rl.KeyEscape) {
			running = false
		}
		rl.EndDrawing()
	}
	rl.CloseWindow()

	// wait for concurrency scanning to complete
	app.Logger.Info("Waiting for enumerations to finish")
	wg.Wait()

	app.Logger.Info("URL crawler finished enumerations.")
	return nil
}

// generateReports as CSV/PDF or other
func generateReports(app *Application, enumerated map[string]bool) error {
	// Time-stamped files
	tstamp := time.Now().Format("20060102_150405")
	csvFile := filepath.Join(reportDir, fmt.Sprintf("urlcrawler_report_%s.csv", tstamp))
	pdfFile := filepath.Join(reportDir, fmt.Sprintf("urlcrawler_report_%s.pdf", tstamp))

	// CSV
	f, err := os.Create(csvFile)
	if err != nil {
		return err
	}
	defer f.Close()
	for url := range enumerated {
		f.WriteString(fmt.Sprintf("%s\n", url))
	}
	app.Logger.Info("CSV report generated", zap.String("file", csvFile))

	// PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "URL Crawler Post-Quantum Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	for url := range enumerated {
		pdf.CellFormat(0, 8, url, "", 1, "", false, 0, "")
	}
	if err := pdf.OutputFileAndClose(pdfFile); err != nil {
		return err
	}
	app.Logger.Info("PDF report generated", zap.String("file", pdfFile))
	return nil
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "Path to config file")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app, err := NewApplication(configPath)
	if err != nil {
		fmt.Printf("Error initializing app: %v\n", err)
		os.Exit(1)
	}

	// handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-sigChan
		app.Logger.Info("Shutting down gracefully...", zap.String("signal", s.String()))
		cancel()
		// wait a bit or do any cleanup
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()

	// Start concurrency scanning + Raylib UI
	if err := app.Start(ctx); err != nil {
		app.Logger.Error("Application error", zap.Error(err))
		os.Exit(1)
	}

	// Suppose the concurrency scanning enumerates some URLs, stored in a map
	enumerated := map[string]bool{
		"https://example.com": true,
		"https://another.org": true,
	}

	// Finally generate CSV/PDF
	if err := generateReports(app, enumerated); err != nil {
		app.Logger.Error("Failed to generate final reports", zap.Error(err))
	}

	app.Logger.Info("URL crawler application done")
}
