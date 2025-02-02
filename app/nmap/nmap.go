package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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

	// Hypothetical quantum-safe module
	"ghostshell/oqs_network"
)

const (
	ScreenWidth   = 1280
	ScreenHeight  = 720
	ParticleCount = 50
	FontPointSize = 24

	LogDir    = "ghostshell/logging"
	ReportDir = "ghostshell/reporting"
)

// -------------- Logging --------------

var logger *zap.Logger

func setupLogger() error {
	if err := os.MkdirAll(LogDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(LogDir, fmt.Sprintf("nmap_log_%s.log", timestamp))

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
			x:  float32(rng.Intn(ScreenWidth)),
			y:  float32(rng.Intn(ScreenHeight)),
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

		if p.x < 0 || p.x > float32(ScreenWidth) {
			p.dx *= -1
		}
		if p.y < 0 || p.y > float32(ScreenHeight) {
			p.dy *= -1
		}
		rl.DrawCircle(int32(p.x), int32(p.y), 4, p.color)
	}
}

// -------------- Terminal --------------

type Terminal struct {
	font rl.Font
}

// newTerminal loads a TTF or fallback
func newTerminal() (*Terminal, error) {
	fontPath := "resources/futuristic_font.ttf"
	font := rl.LoadFontEx(fontPath, FontPointSize, nil, 0)
	if font.BaseSize == 0 {
		logger.Warn("Failed to load custom font, fallback to default", zap.String("fontPath", fontPath))
		font = rl.GetFontDefault()
	}
	return &Terminal{font: font}, nil
}

// -------------- Nmap Concurrency & PQ Security --------------

// NmapResult stores the result of a single scan
type NmapResult struct {
	Host   string
	Status string
	Detail string
}

// NmapManager handles concurrency-based scanning with post-quantum references
type NmapManager struct {
	oqsNet *oqs_network.OQSNetwork // hypothetical PQ network usage

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex

	results []NmapResult
}

// newNmapManager creates a manager with a quantum-safe network
func newNmapManager(net *oqs_network.OQSNetwork) *NmapManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &NmapManager{
		oqsNet:  net,
		ctx:     ctx,
		cancel:  cancel,
		results: []NmapResult{},
	}
}

// Start concurrency-based scanning
func (nm *NmapManager) Start(hosts []string, concurrency int) {
	ch := make(chan string, len(hosts))
	for _, h := range hosts {
		ch <- h
	}
	close(ch)

	for i := 0; i < concurrency; i++ {
		nm.wg.Add(1)
		go nm.worker(ch)
	}
}

// worker processes each host in the channel
func (nm *NmapManager) worker(ch <-chan string) {
	defer nm.wg.Done()
	for host := range ch {
		select {
		case <-nm.ctx.Done():
			return
		default:
		}
		nm.scanOneHost(host)
	}
}

// scanOneHost simulates an Nmap-like scan, with possible quantum-safe connect
func (nm *NmapManager) scanOneHost(host string) {
	start := time.Now()
	// Real logic might do something like:
	// conn, err := nm.oqsNet.Connect(host, "tcp")
	// or a custom Nmap library that references quantum-safe cipher usage
	time.Sleep(time.Duration(rand.Intn(1000)+200) * time.Millisecond)

	success := rand.Float32() < 0.8
	var status string
	var detail string
	if success {
		status = "Open"
		detail = "Ports 80,443 open (placeholder)"
	} else {
		status = "Filtered"
		detail = "No response or blocked"
	}

	dur := time.Since(start)
	logger.Info("Scanned host",
		zap.String("host", host),
		zap.String("status", status),
		zap.Duration("duration", dur),
	)

	nm.mu.Lock()
	nm.results = append(nm.results, NmapResult{Host: host, Status: status, Detail: detail})
	nm.mu.Unlock()
}

// Stop signals concurrency to end
func (nm *NmapManager) Stop() {
	nm.cancel()
	nm.wg.Wait()
}

func (nm *NmapManager) GatherResults() []NmapResult {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	return append([]NmapResult{}, nm.results...)
}

// -------------- CSV/PDF Reporting --------------

func generateReports(results []NmapResult) error {
	if len(results) == 0 {
		logger.Warn("No Nmap results to report on, skipping generation")
		return nil
	}

	reportDir := ReportDir
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %v", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	csvFile := filepath.Join(reportDir, fmt.Sprintf("nmap_report_%s.csv", timestamp))
	pdfFile := filepath.Join(reportDir, fmt.Sprintf("nmap_report_%s.pdf", timestamp))

	// CSV
	f, err := os.Create(csvFile)
	if err != nil {
		logger.Error("Failed to create CSV file", zap.Error(err))
		return err
	}
	defer f.Close()

	// Write header
	fmt.Fprintln(f, "Host,Status,Detail")
	for _, r := range results {
		line := fmt.Sprintf("%s,%s,%s", r.Host, r.Status, r.Detail)
		fmt.Fprintln(f, line)
	}
	logger.Info("CSV report generated", zap.String("file", csvFile))

	// PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "Nmap (Quantum-Safe) Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	for _, r := range results {
		line := fmt.Sprintf("Host: %s, Status: %s, Detail: %s", r.Host, r.Status, r.Detail)
		pdf.MultiCell(190, 6, line, "", "", false)
	}
	if err := pdf.OutputFileAndClose(pdfFile); err != nil {
		logger.Error("Failed to write PDF", zap.Error(err))
		return err
	}
	logger.Info("PDF report generated", zap.String("file", pdfFile))
	return nil
}

// -------------- Main & Raylib UI --------------

func main() {
	// Setup logger
	if err := setupLogger(); err != nil {
		fmt.Printf("Logger setup error: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Suppose we have a PQ network
	certMgr := &MyCertManager{} // placeholder
	oqsNet, err := oqs_network.NewOQSNetwork(certMgr)
	if err != nil {
		logger.Fatal("Failed to init OQSNetwork for Nmap", zap.Error(err))
	}

	// Create a concurrency manager
	manager := newNmapManager(oqsNet)
	hosts := []string{"10.0.0.1", "192.168.1.10", "some-site.org"}
	concurrency := 3
	manager.Start(hosts, concurrency)

	// Setup Raylib
	rl.InitWindow(ScreenWidth, ScreenHeight, "Nmap (Quantum-Safe)")
	rl.SetTargetFPS(60)
	runtime.LockOSThread()

	t, err := newTerminal()
	if err != nil {
		logger.Fatal("Failed to create Terminal UI", zap.Error(err))
	}
	ps := generateParticles(ParticleCount)

	// Graceful signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// main loop
	mainCtx, mainCancel := context.WithCancel(context.Background())
	for !rl.WindowShouldClose() && mainCtx.Err() == nil {
		select {
		case <-sigChan:
			logger.Info("Got shutdown signal")
			mainCancel()
		default:
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.DarkBlue)

		for _, p := range ps {
			p.x += p.dx
			p.y += p.dy
			if p.x < 0 || p.x > float32(ScreenWidth) {
				p.dx *= -1
			}
			if p.y < 0 || p.y > float32(ScreenHeight) {
				p.dy *= -1
			}
			rl.DrawCircle(int32(p.x), int32(p.y), 4, p.color)
		}

		rl.DrawTextEx(t.font, "Nmap - Post Quantum", rl.NewVector2(40, 40), float32(FontPointSize), 2, rl.White)
		localTime := time.Now().Format("15:04:05")
		rl.DrawTextEx(t.font, fmt.Sprintf("Local Time: %s", localTime), rl.NewVector2(40, 80), 20, 2, rl.LightGray)
		rl.DrawTextEx(t.font, "Press ESC to exit", rl.NewVector2(40, 110), 20, 2, rl.LightGray)

		if rl.IsKeyPressed(rl.KeyEscape) {
			mainCancel()
		}

		rl.EndDrawing()
	}

	// Cleanup concurrency
	manager.Stop()
	results := manager.GatherResults()
	if err := generateReports(results); err != nil {
		logger.Error("Failed to generate final reports", zap.Error(err))
	}

	// Close window
	rl.CloseWindow()
	logger.Info("Application shutting down gracefully.")
}

// MyCertManager is a placeholder for your quantum-safe cert usage
type MyCertManager struct{}

func (mgr *MyCertManager) LoadClientCert() (tls.Certificate, error) {
	// stub
	return tls.Certificate{}, nil
}
func (mgr *MyCertManager) LoadRootCAs() (*x509.CertPool, error) {
	// stub
	return nil, nil
}
