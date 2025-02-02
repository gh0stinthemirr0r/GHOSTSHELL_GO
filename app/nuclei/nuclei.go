package nuclei

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/jung-kurt/gofpdf"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"golang.org/x/exp/rand"
)

// Constants for Raylib interface and particle effects.
const (
	windowWidth  = 800
	windowHeight = 600
	maxParticles = 100
	fontSize     = 24
)

// Particle represents a visual particle in the Raylib interface.
type Particle struct {
	x, y, dx, dy float32
	color        rl.Color
}

// NucleiScannerConfig holds the configuration for the scanner.
type NucleiScannerConfig struct {
	TemplatesDir      string
	OutputDir         string
	Timeout           time.Duration
	Retries           int
	RetryInterval     time.Duration
	Concurrency       int
	Logger            *zap.Logger
	PrometheusMetrics bool
	ShowUI            bool
}

// NucleiScanner represents the scanner with all functionality.
type NucleiScanner struct {
	config       NucleiScannerConfig
	totalScans   prometheus.Counter
	successScans prometheus.Counter
	failedScans  prometheus.Counter
	scanDuration prometheus.Histogram
	particles    []*Particle
	logger       *zap.Logger
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewNucleiScanner initializes a new NucleiScanner with the given configuration.
func NewNucleiScanner(cfg NucleiScannerConfig) (*NucleiScanner, error) {
	timestamp := time.Now().Format("01022006_1504")
	logDir := "ghostshell/logging"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logFile := fmt.Sprintf("%s/nuclei_log_%s.log", logDir, timestamp)
	loggerCfg := zap.NewProductionConfig()
	loggerCfg.OutputPaths = []string{logFile, "stdout"}
	logger, err := loggerCfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	ns := &NucleiScanner{
		config:    cfg,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
		particles: make([]*Particle, maxParticles),
	}

	for i := range ns.particles {
		ns.particles[i] = &Particle{
			x:     float32(windowWidth / 2),
			y:     float32(windowHeight / 2),
			dx:    (rand.Float32()*2 - 1) * 4,
			dy:    (rand.Float32()*2 - 1) * 4,
			color: rl.NewColor(uint8(rand.Intn(255)), uint8(rand.Intn(255)), 255, 255),
		}
	}

	if cfg.PrometheusMetrics {
		ns.initPrometheusMetrics()
	}

	return ns, nil
}

// initPrometheusMetrics initializes Prometheus metrics.
func (ns *NucleiScanner) initPrometheusMetrics() {
	ns.totalScans = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "nuclei_total_scans",
		Help: "Total number of scans performed.",
	})
	ns.successScans = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "nuclei_successful_scans",
		Help: "Total number of successful scans.",
	})
	ns.failedScans = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "nuclei_failed_scans",
		Help: "Total number of failed scans.",
	})
	ns.scanDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "nuclei_scan_duration_seconds",
		Help:    "Duration of scans in seconds.",
		Buckets: prometheus.DefBuckets,
	})
	prometheus.MustRegister(ns.totalScans, ns.successScans, ns.failedScans, ns.scanDuration)
}

// Scan simulates a scanning process and reports metrics.
func (ns *NucleiScanner) Scan(targets []string) {
	for _, target := range targets {
		start := time.Now()
		// Simulate a successful scan
		ns.logger.Info("Scanning target", zap.String("target", target))
		time.Sleep(2 * time.Second) // Simulated scan duration
		ns.totalScans.Inc()
		ns.successScans.Inc()
		ns.scanDuration.Observe(time.Since(start).Seconds())
		ns.logger.Info("Scan completed", zap.String("target", target))
	}
}

// SaveReport generates PDF and CSV reports.
func (ns *NucleiScanner) SaveReport(targets []string) error {
	timestamp := time.Now().Format("01022006_1504")
	reportDir := "ghostshell/reporting"
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	pdfFile := fmt.Sprintf("%s/nuclei_report_%s.pdf", reportDir, timestamp)
	csvFile := fmt.Sprintf("%s/nuclei_report_%s.csv", reportDir, timestamp)

	// Generate CSV report
	csvF, err := os.Create(csvFile)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer csvF.Close()
	writer := csv.NewWriter(csvF)
	defer writer.Flush()
	writer.Write([]string{"Target", "Status"})
	for _, target := range targets {
		writer.Write([]string{target, "Success"})
	}
	ns.logger.Info("CSV report generated", zap.String("file", csvFile))

	// Generate PDF report
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Nuclei Scan Report")
	pdf.Ln(12)
	pdf.SetFont("Arial", "", 12)
	for _, target := range targets {
		pdf.Cell(40, 10, fmt.Sprintf("Target: %s - Status: Success", target))
		pdf.Ln(8)
	}
	if err := pdf.OutputFileAndClose(pdfFile); err != nil {
		return fmt.Errorf("failed to create PDF report: %w", err)
	}
	ns.logger.Info("PDF report generated", zap.String("file", pdfFile))
	return nil
}

// Close cleans up resources used by the NucleiScanner.
func (ns *NucleiScanner) Close() {
	ns.cancel()
	ns.logger.Sync()
	if ns.config.ShowUI {
		rl.CloseWindow()
	}
	ns.logger.Info("Scanner shut down successfully")
}

// main function demonstrates scanner usage.
func main() {
	targets := []string{"example.com", "test.com"}
	config := NucleiScannerConfig{
		TemplatesDir:      "templates",
		OutputDir:         "output",
		Retries:           3,
		RetryInterval:     5 * time.Second,
		Concurrency:       5,
		PrometheusMetrics: true,
		ShowUI:            true,
	}

	ns, err := NewNucleiScanner(config)
	if err != nil {
		log.Fatalf("Failed to initialize scanner: %v", err)
	}
	defer ns.Close()

	ns.Scan(targets)
	if err := ns.SaveReport(targets); err != nil {
		log.Fatalf("Failed to save report: %v", err)
	}
}
