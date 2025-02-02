package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/csv"
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
	"golang.org/x/exp/rand"

	// Hypothetical local modules for quantum-safe usage
	"ghostshell/oqs_network"
)

const (
	windowWidth  = 1280
	windowHeight = 720
	fontSize     = 24
	maxParticles = 50
	logDir       = "ghostshell/logging"
	reportDir    = "ghostshell/reporting"
)

// -------------- Logging --------------

var logger *zap.Logger

func setupLogger() error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(logDir, fmt.Sprintf("httpcrawler_log_%s.log", timestamp))

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
		rl.DrawCircle(int32(p.x), int32(p.y), 5, p.color)
	}
}

// -------------- Terminal UI --------------

type Terminal struct {
	font rl.Font
}

func newTerminal() (*Terminal, error) {
	fontPath := "resources/futuristic_font.ttf"
	font := rl.LoadFontEx(fontPath, fontSize, nil, 0)
	if font.BaseSize == 0 {
		logger.Warn("Failed to load custom TTF font, fallback to default")
		font = rl.GetFontDefault()
	}
	return &Terminal{font: font}, nil
}

// -------------- HTTP Crawler with PQ Security --------------

type ProbeResult struct {
	URL        string
	StatusCode int
	Duration   time.Duration
	Err        error
}

// HTTPCrawler manages concurrency-based scanning with quantum-safe channels
type HTTPCrawler struct {
	oqsNet  *oqs_network.OQSNetwork // hypothetical post-quantum net usage
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.Mutex
	results map[string]ProbeResult
}

// newHTTPCrawler initializes the crawler with a quantum-safe network
func newHTTPCrawler(net *oqs_network.OQSNetwork) *HTTPCrawler {
	ctx, cancel := context.WithCancel(context.Background())
	return &HTTPCrawler{
		oqsNet:  net,
		ctx:     ctx,
		cancel:  cancel,
		results: make(map[string]ProbeResult),
	}
}

// Start begins concurrency-based scanning of multiple URLs
func (hc *HTTPCrawler) Start(urls []string, concurrency int) {
	ch := make(chan string, len(urls))
	for _, u := range urls {
		ch <- u
	}
	close(ch)

	for i := 0; i < concurrency; i++ {
		hc.wg.Add(1)
		go hc.worker(ch)
	}
}

// worker processes incoming URLs from the channel
func (hc *HTTPCrawler) worker(ch <-chan string) {
	defer hc.wg.Done()
	for url := range ch {
		select {
		case <-hc.ctx.Done():
			return
		default:
		}
		hc.probeOneURL(url)
	}
}

// probeOneURL does a quantum-safe GET request (placeholder logic)
func (hc *HTTPCrawler) probeOneURL(url string) {
	start := time.Now()

	// Real logic might do:
	// conn, err := hc.oqsNet.Connect(url, "tcp")
	// send HTTP GET manually...
	// or use a custom RoundTripper that references the OQSNetwork for TLS
	time.Sleep(time.Duration(rand.Intn(500)+100) * time.Millisecond)

	success := rand.Float32() < 0.7 // 70% success
	var code int
	var err error
	if success {
		codes := []int{200, 302, 404}
		code = codes[rand.Intn(len(codes))]
	} else {
		err = fmt.Errorf("random simulated error")
	}
	dur := time.Since(start)

	hc.mu.Lock()
	hc.results[url] = ProbeResult{
		URL:        url,
		StatusCode: code,
		Duration:   dur,
		Err:        err,
	}
	hc.mu.Unlock()
}

// Stop signals the concurrency to end
func (hc *HTTPCrawler) Stop() {
	hc.cancel()
	hc.wg.Wait()
}

// GatherResults returns a slice of slice (URL, StatusCode, Duration, Error) for CSV/PDF
func (hc *HTTPCrawler) GatherResults() [][]string {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	rows := [][]string{{"URL", "Status", "Duration(s)", "Error"}}
	for _, r := range hc.results {
		rows = append(rows, []string{
			r.URL,
			fmt.Sprintf("%d", r.StatusCode),
			fmt.Sprintf("%.3f", r.Duration.Seconds()),
			fmt.Sprintf("%v", r.Err),
		})
	}
	return rows
}

// -------------- CSV/PDF Reporting --------------

func generateReports(data [][]string) error {
	if len(data) <= 1 {
		logger.Warn("No data to report or only header row, skipping generation")
		return nil
	}
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		logger.Error("Failed to create report dir", zap.Error(err))
		return err
	}
	timestamp := time.Now().Format("20060102_150405")
	csvFile := filepath.Join(reportDir, fmt.Sprintf("httpcrawler_report_%s.csv", timestamp))
	pdfFile := filepath.Join(reportDir, fmt.Sprintf("httpcrawler_report_%s.pdf", timestamp))

	// CSV
	f, err := os.Create(csvFile)
	if err != nil {
		logger.Error("Failed to create CSV file", zap.Error(err))
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	for _, row := range data {
		if err := w.Write(row); err != nil {
			return err
		}
	}
	logger.Info("CSV report generated", zap.String("file", csvFile))

	// PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "HTTP Crawler Post-Quantum Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	// data[0] is the header
	for i, row := range data {
		if i == 0 {
			continue // skip the header in the main content
		}
		line := fmt.Sprintf("%s - code: %s, dur: %ss, err: %s", row[0], row[1], row[2], row[3])
		pdf.MultiCell(190, 6, line, "", "", false)
	}
	if err := pdf.OutputFileAndClose(pdfFile); err != nil {
		logger.Error("Failed to write PDF file", zap.Error(err))
		return err
	}
	logger.Info("PDF report generated", zap.String("file", pdfFile))
	return nil
}

// -------------- main & Raylib UI --------------

func main() {
	// Setup logging
	if err := setupLogger(); err != nil {
		fmt.Printf("Logger setup failed: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Suppose we have a PQ network
	certMgr := &MyCertManager{} // placeholder
	oqsNet, err := oqs_network.NewOQSNetwork(certMgr)
	if err != nil {
		logger.Fatal("Failed to init OQSNetwork", zap.Error(err))
	}

	// Create the crawler
	crawler := newHTTPCrawler(oqsNet)
	urls := []string{"https://example.com", "https://some-other.org", "https://fail.com"}
	concurrency := 3
	crawler.Start(urls, concurrency)

	// Raylib init
	rl.InitWindow(windowWidth, windowHeight, "HTTP Crawler (Quantum-Safe)")
	rl.SetTargetFPS(60)
	runtime.LockOSThread()

	// Create UI
	t, err := newTerminal()
	if err != nil {
		logger.Fatal("Failed to create Terminal", zap.Error(err))
	}
	// Particles
	ps := generateParticles(maxParticles)

	// Graceful signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	mainCtx, mainCancel := context.WithCancel(context.Background())

	// main loop
	for !rl.WindowShouldClose() && mainCtx.Err() == nil {
		select {
		case <-sigChan:
			logger.Info("Got shutdown signal")
			mainCancel()
		default:
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.DarkBlue)

		updateParticles(ps)

		// Draw text
		rl.DrawTextEx(t.font, "HTTP Crawler - PQ Secure", rl.NewVector2(40, 40), float32(fontSize), 2, rl.White)
		localTime := time.Now().Format("15:04:05")
		rl.DrawTextEx(t.font, fmt.Sprintf("Local Time: %s", localTime), rl.NewVector2(40, 80), 20, 2, rl.LightGray)
		rl.DrawTextEx(t.font, "Press ESC to exit", rl.NewVector2(40, 110), 20, 2, rl.LightGray)

		if rl.IsKeyPressed(rl.KeyEscape) {
			mainCancel()
		}

		rl.EndDrawing()
	}

	// Cleanup
	crawler.Stop()
	results := crawler.GatherResults()
	if err := generateReports(results); err != nil {
		logger.Error("Failed to generate final reports", zap.Error(err))
	}

	rl.CloseWindow()
	logger.Info("Shutting down application gracefully")
}

// MyCertManager is a placeholder for real PQ cert usage
type MyCertManager struct{}

func (mgr *MyCertManager) LoadClientCert() (tls.Certificate, error) {
	// stub
	return tls.Certificate{}, nil
}
func (mgr *MyCertManager) LoadRootCAs() (*x509.CertPool, error) {
	// stub
	return nil, nil
}
