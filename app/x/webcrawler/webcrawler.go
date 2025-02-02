package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/jung-kurt/gofpdf"
	"github.com/projectdiscovery/subfinder/v2/pkg/runner"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	// Hypothetical references for your local modules

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

// -------------- Logging --------------

var logger *zap.Logger

func setupLogger() error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(logDir, fmt.Sprintf("webcrawler_log_%s.log", timestamp))

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logFile, "stdout"}

	var err error
	logger, err = cfg.Build()
	if err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}
	return nil
}

// -------------- Particles --------------

type Particle struct {
	x, y   float32
	dx, dy float32
	color  rl.Color
}

func generateParticles(count int) []*Particle {
	rngSeed := time.Now().UnixNano()
	ps := make([]*Particle, count)
	for i := 0; i < count; i++ {
		px := float32(raylibRandom(0, windowWidth))
		py := float32(raylibRandom(0, windowHeight))
		dx := float32(raylibRandom(-2, 2))
		dy := float32(raylibRandom(-2, 2))
		clr := rl.NewColor(
			uint8(raylibRandom(50, 255)),
			uint8(raylibRandom(50, 255)),
			uint8(raylibRandom(50, 255)),
			255,
		)
		ps[i] = &Particle{x: px, y: py, dx: dx, dy: dy, color: clr}
	}
	return ps
}

func raylibRandom(min, max int) int {
	return rl.GetRandomValue(int32(min), int32(max))
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

// -------------- Config & Options placeholders --------------

type Config struct{}
type Options struct {
	ConfigFile  string
	Targets     []string
	Concurrency int
}

func loadConfig(path string) (*Config, error) {
	// placeholder
	return &Config{}, nil
}

func parseInput() (*Options, error) {
	var cfgFile string
	var concurrency int
	var targetsStr string
	flag.StringVar(&cfgFile, "config", "webcrawler_config.yaml", "Path to config file")
	flag.IntVar(&concurrency, "concurrency", 5, "Number of concurrency workers")
	flag.StringVar(&targetsStr, "targets", "https://example.com,https://test.com", "Comma-separated list of target URLs")
	flag.Parse()

	if cfgFile == "" {
		return nil, errors.New("no config file specified")
	}
	targets := strings.Split(targetsStr, ",")
	for i := range targets {
		targets[i] = strings.TrimSpace(targets[i])
	}

	return &Options{
		ConfigFile:  cfgFile,
		Targets:     targets,
		Concurrency: concurrency,
	}, nil
}

// -------------- The Main Application --------------

type Application struct {
	Config  *Config
	Options *Options
	Runner  *runner.Runner // concurrency scanning logic
	Logger  *zap.Logger
}

func NewApplication(opts *Options) (*Application, error) {
	// post-quantum ephemeral usage
	if err := oqs_vault.InitEphemeralKey(); err != nil {
		logger.Warn("Failed ephemeral key init", zap.Error(err))
	}

	// load config
	cfg, err := loadConfig(opts.ConfigFile)
	if err != nil {
		return nil, err
	}

	// init runner
	runr, err := runner.NewRunner(cfg, opts.Concurrency)
	if err != nil {
		return nil, fmt.Errorf("failed to init runner: %w", err)
	}

	return &Application{
		Config:  cfg,
		Options: opts,
		Runner:  runr,
		Logger:  logger,
	}, nil
}

func (app *Application) Start(ctx context.Context) error {
	app.Logger.Info("Starting concurrency-based web crawling",
		zap.Strings("targets", app.Options.Targets),
		zap.Int("concurrency", app.Options.Concurrency),
	)

	// concurrency scanning
	var wg sync.WaitGroup
	resultsChan := make(chan string, 1000)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := app.Runner.RunCrawl(ctx, app.Options.Targets, resultsChan); err != nil {
			app.Logger.Error("Error during concurrency crawling", zap.Error(err))
		}
		close(resultsChan)
	}()

	// read results
	enumerated := make(map[string]bool)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for r := range resultsChan {
			enumerated[r] = true
			app.Logger.Info("Crawled result", zap.String("url", r))
		}
	}()

	// Raylib UI
	go runRaylibUI()

	wg.Wait()

	// final reports
	if err := generateReports(enumerated); err != nil {
		app.Logger.Error("Failed to generate final reports", zap.Error(err))
	}

	app.Logger.Info("All crawling tasks finished successfully.")
	return nil
}

// -------------- Raylib UI --------------

func runRaylibUI() {
	rl.InitWindow(windowWidth, windowHeight, "WebCrawler - PQ Secure")
	rl.SetTargetFPS(60)
	runtime.LockOSThread()

	ps := generateParticles(maxParticles)
	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.DarkBlue)

		updateParticles(ps)

		rl.DrawText("WebCrawler - Post Quantum", 20, 20, fontSize, rl.White)
		localTime := time.Now().Format("15:04:05")
		rl.DrawText(fmt.Sprintf("Local Time: %s", localTime), 20, 60, fontSize-4, rl.LightGray)
		rl.DrawText("[ESC] to exit", 20, 100, fontSize-4, rl.LightGray)

		if rl.IsKeyPressed(rl.KeyEscape) {
			break
		}
		rl.EndDrawing()
	}
	rl.CloseWindow()
}

// -------------- Reporting --------------

func generateReports(enumerated map[string]bool) error {
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		logger.Error("Failed to create report dir", zap.Error(err))
		return err
	}
	tstamp := time.Now().Format("20060102_150405")
	csvFile := filepath.Join(reportDir, fmt.Sprintf("webcrawler_report_%s.csv", tstamp))
	pdfFile := filepath.Join(reportDir, fmt.Sprintf("webcrawler_report_%s.pdf", tstamp))

	// CSV
	if err := writeCSV(csvFile, enumerated); err != nil {
		return err
	}
	logger.Info("CSV report generated", zap.String("file", csvFile))

	// PDF
	if err := writePDF(pdfFile, enumerated); err != nil {
		return err
	}
	logger.Info("PDF report generated", zap.String("file", pdfFile))

	return nil
}

func writeCSV(path string, enumerated map[string]bool) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for url := range enumerated {
		f.WriteString(fmt.Sprintf("%s\n", url))
	}
	return nil
}

func writePDF(path string, enumerated map[string]bool) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "WebCrawler Post-Quantum Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	for url := range enumerated {
		pdf.Cell(0, 8, url)
		pdf.Ln(8)
	}
	return pdf.OutputFileAndClose(path)
}

// -------------- Main --------------

func main() {
	if err := setupLogger(); err != nil {
		fmt.Printf("Error setting up logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Webcrawler starting...")

	// parse flags
	opts, err := parseInput()
	if err != nil {
		logger.Fatal("Error parsing input", zap.Error(err))
	}

	// create app
	app, err := NewApplication(opts)
	if err != nil {
		logger.Fatal("Error creating app", zap.Error(err))
	}

	// handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		s := <-sigChan
		logger.Info("Shutting down gracefully...", zap.String("signal", s.String()))
		cancel()
		// wait or do final tasks
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()

	// start concurrency scanning + raylib UI
	if err := app.Start(ctx); err != nil {
		logger.Fatal("Error during crawling", zap.Error(err))
	}

	logger.Info("Web crawling completed successfully")
}
