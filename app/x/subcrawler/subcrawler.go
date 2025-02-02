package main

import (
	"context"
	"encoding/csv"
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
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/rand"

	// Hypothetical references to your local modules
	"ghostshell/app_suite/subcrawler/options"
	"ghostshell/app_suite/subcrawler/output"
	"ghostshell/app_suite/subcrawler/passive"

	// Post-quantum placeholders
	"ghostshell/ghostshell/oqs/oqs_vault"
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
		return fmt.Errorf("failed to create log directory: %v", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(logDir, fmt.Sprintf("subcrawler_log_%s.log", timestamp))

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

// -------------- Raylib UI: Particles + Terminal --------------

type Particle struct {
	x, y   float32
	dx, dy float32
	color  rl.Color
}

func generateParticles(count int) []*Particle {
	rng := rand.New(rand.NewSource(uint64(time.Now().UnixNano())))
	ps := make([]*Particle, count)
	for i := 0; i < count; i++ {
		ps[i] = &Particle{
			x:  float32(rng.Intn(windowWidth)),
			y:  float32(rng.Intn(windowHeight)),
			dx: (rng.Float32()*2 - 1) * 2,
			dy: (rng.Float32()*2 - 1) * 2,
			color: rl.NewColor(
				uint8(rng.Intn(256)),
				uint8(rng.Intn(256)),
				uint8(rng.Intn(256)),
				255,
			),
		}
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

// -------------- Concurrency Subdomain Enumeration --------------

func enumerateAllSources(options *options.Options, sources []passive.Source, results chan<- string) error {
	var wg sync.WaitGroup

	for _, src := range sources {
		wg.Add(1)
		go func(s passive.Source) {
			defer wg.Done()
			logger.Info("Enumerating source", zap.String("source", s.Name()))
			if err := s.Enumerate(options.Domain, results); err != nil {
				logger.Warn("Error enumerating source", zap.String("source", s.Name()), zap.Error(err))
			}
		}(src)
	}

	// Wait for all
	wg.Wait()
	close(results)
	return nil
}

// -------------- Post-Quantum Ephemeral Encryption (Placeholder) --------------

func ephemeralEncryptSubdomains(subdomains []string) ([]byte, error) {
	// For example, ephemeral key from `oqs_vault.InitEphemeralKey()`
	// Then do `oqs_network.EncryptEphemeral` or something similar
	data := []byte(strings.Join(subdomains, "\n"))
	// placeholder
	return data, nil
}

// -------------- CSV/PDF Reporting --------------

func generateCSVReport(path string, subdomains []string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// header
	if err := w.Write([]string{"Subdomain"}); err != nil {
		return err
	}
	for _, s := range subdomains {
		if err := w.Write([]string{s}); err != nil {
			return err
		}
	}
	return nil
}

func generatePDFReport(path string, subdomains []string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Subcrawler Post-Quantum Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	for _, s := range subdomains {
		pdf.Cell(0, 8, s)
		pdf.Ln(8)
	}
	return pdf.OutputFileAndClose(path)
}

// -------------- Main --------------

func main() {
	// 1) Setup logger
	if err := setupLogger(); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	logger.Info("Subcrawler starting...")

	// 2) Parse options
	opts, err := options.ParseOptions()
	if err != nil {
		logger.Fatal("Error parsing options", zap.Error(err))
	}
	logger.Info("Parsed options", zap.String("domain", opts.Domain), zap.String("outputFile", opts.OutputFile))

	// 3) Post-quantum ephemeral init (placeholder)
	if err := oqs_vault.InitEphemeralKey(); err != nil {
		logger.Warn("Failed ephemeral key init", zap.Error(err))
	}

	// 4) Initialize Raylib
	rl.InitWindow(windowWidth, windowHeight, "Subcrawler - PQ Secure")
	rl.SetTargetFPS(60)
	runtime.LockOSThread()

	font := rl.LoadFontEx("resources/futuristic_font.ttf", fontSize, nil, 0)
	if font.BaseSize == 0 {
		logger.Fatal("Failed to load Raylib font")
	}
	defer func() {
		rl.UnloadFont(font)
		rl.CloseWindow()
	}()

	ps := generateParticles(maxParticles)

	// 5) Initialize sources
	sources, err := passive.InitializeSources()
	if err != nil {
		logger.Fatal("Error initializing sources", zap.Error(err))
	}

	// concurrency scanning
	resultsChan := make(chan string, 1000)
	go func() {
		if err := enumerateAllSources(opts, sources, resultsChan); err != nil {
			logger.Warn("Error enumerating sources", zap.Error(err))
		}
	}()

	uniqueResults := make(map[string]bool)
	go func() {
		// collect results
		for r := range resultsChan {
			// deduplicate
			uniqueResults[r] = true
			logger.Info("Subdomain found", zap.String("subdomain", r))
		}
	}()

	// 6) Setup graceful signals
	doneChan := make(chan os.Signal, 1)
	signal.Notify(doneChan, os.Interrupt, syscall.SIGTERM)

	mainCtx, mainCancel := context.WithCancel(context.Background())

	// 7) Main loop
	for !rl.WindowShouldClose() && mainCtx.Err() == nil {
		select {
		case <-doneChan:
			logger.Info("Received shutdown signal")
			mainCancel()
		default:
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.DarkBlue)

		// Particle swirl
		updateParticles(ps)

		// Some text
		rl.DrawTextEx(font, "Subcrawler - Post Quantum", rl.NewVector2(20, 40), float32(fontSize), 2, rl.White)
		localTime := time.Now().Format("15:04:05")
		rl.DrawTextEx(font, fmt.Sprintf("Local Time: %s", localTime), rl.NewVector2(20, 80), float32(fontSize-4), 2, rl.LightGray)
		rl.DrawTextEx(font, "Press ESC to quit", rl.NewVector2(20, 120), float32(fontSize-4), 2, rl.LightGray)

		if rl.IsKeyPressed(rl.KeyEscape) {
			mainCancel()
		}

		rl.EndDrawing()
	}

	// finalize
	subList := make([]string, 0, len(uniqueResults))
	for k := range uniqueResults {
		subList = append(subList, k)
	}
	// ephemeral encryption (placeholder)
	encrypted, err := ephemeralEncryptSubdomains(subList)
	if err != nil {
		logger.Warn("Failed ephemeral encryption of subdomains", zap.Error(err))
	} else {
		logger.Info("Ephemeral encryption success", zap.Int("bytes", len(encrypted)))
	}

	// 8) Save final results
	if err := output.SaveResults(uniqueResults, opts.OutputFile, reportDir); err != nil {
		logger.Error("Error saving results with output package", zap.Error(err))
	}

	// 9) Generate CSV/PDF
	timestamp := time.Now().Format("20060102_150405")
	csvPath := filepath.Join(reportDir, fmt.Sprintf("subcrawler_report_%s.csv", timestamp))
	pdfPath := filepath.Join(reportDir, fmt.Sprintf("subcrawler_report_%s.pdf", timestamp))

	if err := generateCSVReport(csvPath, subList); err != nil {
		logger.Error("Failed to generate CSV report", zap.Error(err))
	} else {
		logger.Info("CSV report generated", zap.String("file", csvPath))
	}

	if err := generatePDFReport(pdfPath, subList); err != nil {
		logger.Error("Failed to generate PDF report", zap.Error(err))
	} else {
		logger.Info("PDF report generated", zap.String("file", pdfPath))
	}

	logger.Info("Subdomain enumeration completed.")
	logger.Info("Exiting application")
}
