package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/jung-kurt/gofpdf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	windowWidth  = 1280
	windowHeight = 720
	fontSize     = 24
	maxParticles = 50
	reportDir    = "ghostshell/reporting"
	logDir       = "ghostshell/logging"
)

// -------------- Logging --------------

var logger *zap.Logger

// setupLogger configures a standard date/time-based Zap logger
func setupLogger() error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(logDir, fmt.Sprintf("dnscrawler_log_%s.log", timestamp))

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logFile, "stdout"}

	var err error
	logger, err = cfg.Build()
	if err != nil {
		return fmt.Errorf("failed to build logger: %v", err)
	}
	return nil
}

// -------------- Particles --------------

type Particle struct {
	x, y   float32
	dx, dy float32
	color  rl.Color
}

// -------------- Terminal --------------

type Terminal struct {
	font rl.Font
}

// newTerminal loads a custom or fallback font
func newTerminal() (*Terminal, error) {
	fontPath := "resources/futuristic_font.ttf"
	font := rl.LoadFontEx(fontPath, fontSize, nil, 0)
	if font.BaseSize == 0 {
		logger.Warn("Failed to load custom font, using default", zap.String("font", fontPath))
		font = rl.GetFontDefault()
	}
	return &Terminal{font: font}, nil
}

// Shutdown cleans up terminal resources
func (t *Terminal) Shutdown() {
	if t.font.BaseSize != 0 {
		rl.UnloadFont(t.font)
	}
	rl.CloseWindow()
	logger.Info("Terminal shut down successfully")
}

// -------------- Particles Generation --------------

func generateParticles(count int) []*Particle {
	rand.Seed(time.Now().UnixNano())
	ps := make([]*Particle, count)
	for i := 0; i < count; i++ {
		ps[i] = &Particle{
			x:  float32(rand.Intn(windowWidth)),
			y:  float32(rand.Intn(windowHeight)),
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
	return ps
}

// updateParticles updates positions and draws them
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

// -------------- DNS Scanning --------------

// simulateDNSCrawl is a stub concurrency-based DNS scanning
func simulateDNSCrawl(ctx context.Context, wg *sync.WaitGroup, results *[]string, mu *sync.Mutex) {
	defer wg.Done()
	logger.Info("DNS crawler started")

	endpoints := []string{
		"google.com", "example.com", "microsoft.com",
		"ubuntu.com", "archlinux.org", "kubernetes.io",
	}

	idx := 0
	for {
		select {
		case <-ctx.Done():
			logger.Info("DNS crawler context canceled")
			return
		default:
		}
		if idx >= len(endpoints) {
			idx = 0
		}
		ep := endpoints[idx]
		// random success/fail
		time.Sleep(800 * time.Millisecond)
		if rand.Float32() < 0.4 {
			line := fmt.Sprintf("%s => FAIL (no record found)", ep)
			logger.Warn("DNS lookup fail", zap.String("endpoint", ep))
			mu.Lock()
			*results = append(*results, line)
			mu.Unlock()
		} else {
			line := fmt.Sprintf("%s => OK (some IP addresses found)", ep)
			logger.Info("DNS lookup success", zap.String("endpoint", ep))
			mu.Lock()
			*results = append(*results, line)
			mu.Unlock()
		}
		idx++
	}
}

// -------------- Reports --------------

// generateReports writes the DNS crawler results to CSV & PDF
func generateReports(data []string) error {
	if len(data) == 0 {
		logger.Warn("No DNS crawler results to report on")
		return nil
	}
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		logger.Error("Failed to create report directory", zap.Error(err))
		return err
	}
	timestamp := time.Now().Format("20060102_150405")
	csvFile := filepath.Join(reportDir, fmt.Sprintf("dnscrawler_report_%s.csv", timestamp))
	pdfFile := filepath.Join(reportDir, fmt.Sprintf("dnscrawler_report_%s.pdf", timestamp))

	// CSV
	if err := writeCSV(csvFile, data); err != nil {
		return err
	}

	// PDF
	if err := writePDF(pdfFile, data); err != nil {
		return err
	}
	return nil
}

func writeCSV(path string, data []string) error {
	f, err := os.Create(path)
	if err != nil {
		logger.Error("Failed to create CSV file", zap.String("file", path), zap.Error(err))
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Write header
	if err := w.Write([]string{"DNS Endpoint => Status"}); err != nil {
		return err
	}
	for _, line := range data {
		if err := w.Write([]string{line}); err != nil {
			return err
		}
	}
	logger.Info("CSV report generated", zap.String("file", path))
	return nil
}

func writePDF(path string, data []string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "DNS Crawler Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	for _, line := range data {
		pdf.MultiCell(190, 8, line, "", "", false)
	}
	if err := pdf.OutputFileAndClose(path); err != nil {
		logger.Error("Failed to write PDF", zap.String("file", path), zap.Error(err))
		return err
	}
	logger.Info("PDF report generated", zap.String("file", path))
	return nil
}

// -------------- main --------------

func main() {
	// setup logging
	if err := setupLogger(); err != nil {
		fmt.Printf("Failed to setup logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	logger.Info("Logger initialized")

	// set up the Raylib window
	rl.InitWindow(windowWidth, windowHeight, "DNS Crawler")
	rl.SetTargetFPS(60)
	runtime.LockOSThread() // Raylib requires main thread for rendering

	// create Terminal
	term, err := newTerminal()
	if err != nil {
		logger.Fatal("Failed to create terminal", zap.Error(err))
	}
	defer term.Shutdown()

	// create particles
	particles := generateParticles(maxParticles)

	// concurrency scanning
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	results := make([]string, 0)
	mu := sync.Mutex{}

	wg.Add(1)
	go simulateDNSCrawl(ctx, &wg, &results, &mu)

	// graceful shutdown if the user closes window or hits ESC
	// or we can handle OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Received shutdown signal")
		cancel()
	}()

	// main loop
	for !rl.WindowShouldClose() && ctx.Err() == nil {
		rl.BeginDrawing()
		rl.ClearBackground(rl.DarkBlue)

		updateParticles(particles)

		// draw text
		rl.DrawTextEx(term.font, "DNS Crawler", rl.NewVector2(40, 40), float32(fontSize), 2, rl.White)
		localTime := time.Now().Format("15:04:05")
		rl.DrawTextEx(term.font, fmt.Sprintf("Local Time: %s", localTime), rl.NewVector2(40, 80), 20, 2, rl.White)
		rl.DrawTextEx(term.font, "Press ESC to Exit", rl.NewVector2(40, 110), 20, 2, rl.LightGray)

		// if user hits ESC
		if rl.IsKeyPressed(rl.KeyEscape) {
			cancel()
		}

		rl.EndDrawing()
	}

	// wait for concurrency to complete
	wg.Wait()

	// generate reports
	if err := generateReports(results); err != nil {
		logger.Error("Failed to generate reports", zap.Error(err))
	}

	logger.Info("Exiting application gracefully")
}
