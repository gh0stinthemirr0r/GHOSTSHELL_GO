package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
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
)

// -------------- Constants --------------

const (
	defaultBaseURL = "https://cve.mitre.org/api/v3/cves"

	windowWidth  = 1280
	windowHeight = 720
	fontSize     = 24
	maxParticles = 50

	logDir    = "ghostshell/logging"
	reportDir = "ghostshell/reporting"
)

// -------------- Logging --------------

var logger *zap.Logger

func setupLogger() error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(logDir, fmt.Sprintf("discovery_log_%s.log", timestamp))

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logFile, "stdout"}

	var err error
	logger, err = cfg.Build()
	if err != nil {
		return fmt.Errorf("failed to init logger: %v", err)
	}
	return nil
}

// -------------- CLI Flags --------------

type Options struct {
	Debug   bool
	Queries string
}

// parseFlags collects user arguments from CLI
func parseFlags() (*Options, error) {
	var debug bool
	var queries string

	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.StringVar(&queries, "q", "", "Comma-separated queries for CVE search")
	flag.Parse()

	return &Options{
		Debug:   debug,
		Queries: queries,
	}, nil
}

// -------------- Particles --------------

type Particle struct {
	x, y, dx, dy float32
	color        rl.Color
}

// -------------- Terminal --------------

type Terminal struct {
	font rl.Font
}

// -------------- Entry Instantiation --------------

func NewTerminal(opts *Options) (*Terminal, error) {
	// Attempt to load a custom font or fallback to default
	fontPath := "resources/futuristic_font.ttf"
	font := rl.LoadFontEx(fontPath, fontSize, nil, 0)
	if font.BaseSize == 0 {
		logger.Warn("Failed to load custom font, fallback to default", zap.String("fontPath", fontPath))
		font = rl.GetFontDefault()
	}
	if opts.Debug {
		logger.Info("Debug mode enabled")
	}
	return &Terminal{font: font}, nil
}

// -------------- Particle Generation --------------

func generateParticles(count int) []*Particle {
	rand.Seed(time.Now().UnixNano())
	ps := make([]*Particle, count)
	for i := 0; i < count; i++ {
		ps[i] = &Particle{
			x:  float32(rand.Intn(windowWidth)),
			y:  float32(rand.Intn(windowHeight)),
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
	return ps
}

// -------------- CVE Searching --------------

// doRequest is a stub for sending an HTTP GET, with placeholders for real logic
func doRequest(url string) ([]byte, error) {
	// Real logic might handle timeouts with a custom client
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("non-2xx code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func fetchCVE(query string, logger *zap.Logger, results *[]string, mu *sync.Mutex) {
	// Stub: half the time it fails
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
	if rand.Float32() < 0.5 {
		logger.Warn("Random fail in fetching CVE", zap.String("query", query))
		mu.Lock()
		*results = append(*results, fmt.Sprintf("%s => FAIL", query))
		mu.Unlock()
		return
	}
	// Otherwise success
	// Real code might do e.g. doRequest(fmt.Sprintf("%s?keyword=%s", defaultBaseURL, query))
	// parse JSON, etc.
	desc := "Description for " + query
	logger.Info("Fetched CVE data", zap.String("query", query), zap.String("desc", desc))

	mu.Lock()
	*results = append(*results, fmt.Sprintf("%s => OK", query))
	mu.Unlock()
}

// -------------- Reporting --------------

func writeReports(data []string, logger *zap.Logger) error {
	if len(data) == 0 {
		logger.Warn("No data to report, skipping report generation")
		return nil
	}
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		logger.Error("Failed to create report directory", zap.Error(err))
		return err
	}
	timestamp := time.Now().Format("20060102_150405")

	csvFile := filepath.Join(reportDir, fmt.Sprintf("discovery_report_%s.csv", timestamp))
	pdfFile := filepath.Join(reportDir, fmt.Sprintf("discovery_report_%s.pdf", timestamp))

	// CSV
	if err := writeCSV(csvFile, data, logger); err != nil {
		return err
	}

	// PDF
	if err := writePDF(pdfFile, data, logger); err != nil {
		return err
	}

	return nil
}

func writeCSV(path string, data []string, logger *zap.Logger) error {
	f, err := os.Create(path)
	if err != nil {
		logger.Error("Failed to create CSV file", zap.Error(err))
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// example: single column
	if err := w.Write([]string{"Query => Result"}); err != nil {
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

func writePDF(path string, data []string, logger *zap.Logger) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Discovery CVE Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	for _, line := range data {
		pdf.MultiCell(190, 8, line, "", "", false)
	}
	if err := pdf.OutputFileAndClose(path); err != nil {
		logger.Error("Failed to write PDF file", zap.String("file", path), zap.Error(err))
		return err
	}
	logger.Info("PDF report generated", zap.String("file", path))
	return nil
}

// -------------- Main Program --------------

func main() {
	// 1) Parse CLI flags
	opts, err := parseFlags()
	if err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// 2) Setup logging
	if err := setupLogger(); err != nil {
		fmt.Printf("Failed to setup logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	logger.Info("Logger initialized")

	// 3) Build the Raylib window + set up the UI
	rl.InitWindow(int32(windowWidth), int32(windowHeight), "Discovery")
	rl.SetTargetFPS(60)
	runtime.LockOSThread() // Raylib requires main thread for drawing

	// 4) Create terminal
	term, err := NewTerminal(opts)
	if err != nil {
		logger.Fatal("Failed to init terminal", zap.Error(err))
	}
	defer term.Shutdown()

	// 5) Particle background
	particles := generateParticles(maxParticles)

	// 6) Graceful shutdown via signals
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Received shutdown signal")
		cancel()
		rl.CloseWindow()
	}()

	// 7) If user specified queries, do concurrency fetch
	var results []string
	var mu sync.Mutex
	queries := strings.Split(opts.Queries, ",")
	if len(queries) == 1 && queries[0] == "" {
		queries = []string{}
	}

	if len(queries) > 0 {
		logger.Info("Starting concurrency for queries", zap.Strings("queries", queries))
		var wg sync.WaitGroup
		for _, q := range queries {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			wg.Add(1)
			go func(query string) {
				defer wg.Done()
				fetchCVE(query, logger, &results, &mu)
			}(q)
		}
		wg.Wait()
		logger.Info("All queries fetched", zap.Int("count", len(results)))

		// Generate reports
		if err := writeReports(results, logger); err != nil {
			logger.Error("Failed to generate reports", zap.Error(err))
		}
	} else {
		logger.Warn("No queries provided, skipping concurrency fetch.")
	}

	// 8) Main loop
	for !rl.WindowShouldClose() && ctx.Err() == nil {
		// update
		for _, p := range particles {
			p.x += p.dx
			p.y += p.dy
			if p.x < 0 || p.x > float32(windowWidth) {
				p.dx *= -1
			}
			if p.y < 0 || p.y > float32(windowHeight) {
				p.dy *= -1
			}
		}
		// check ESC
		if rl.IsKeyPressed(rl.KeyEscape) {
			rl.CloseWindow()
		}

		// draw
		rl.BeginDrawing()
		rl.ClearBackground(rl.DarkBlue)

		// draw particles
		for _, p := range particles {
			rl.DrawCircle(int32(p.x), int32(p.y), 3, p.color)
		}

		// Title & local time
		rl.DrawTextEx(term.font, "Discovery", rl.NewVector2(40, 40), float32(fontSize), 2, rl.White)
		localTime := time.Now().Format("15:04:05")
		rl.DrawTextEx(term.font, "Press ESC to exit", rl.NewVector2(40, 80), 20, 2, rl.LightGray)
		rl.DrawTextEx(term.font, fmt.Sprintf("Local Time: %s", localTime), rl.NewVector2(40, 110), 20, 2, rl.White)

		rl.EndDrawing()
	}

	// exit
	logger.Info("Exiting application gracefully")
}
