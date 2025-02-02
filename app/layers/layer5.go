package layers

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/jung-kurt/gofpdf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// -------------- Constants & Paths --------------
const (
	reportingDir = "ghostshell/reporting"
	loggingDir   = "ghostshell/logging"
)

// -------------- Data Structures --------------

// TestResult represents the outcome for a single layer test
type TestResult struct {
	Layer   int
	Status  string // "Passed" or "Failed"
	Message string
}

// LayerRunner is an interface for any OSI layer runner
type LayerRunner interface {
	RunTests(logger *zap.Logger) ([]TestResult, error)
}

// -------------- Session Layer 5 Runner --------------

// Layer5Runner is responsible for testing Session layer logic.
// For demonstration, we do concurrency-based session checks with multiple hosts or paths.
type Layer5Runner struct {
	Targets []string      // e.g. ["example.com:80", "some-other-host:443"]
	Timeout time.Duration // e.g. 5 * time.Second
}

// NewLayer5Runner with defaults if none provided
func NewLayer5Runner() *Layer5Runner {
	return &Layer5Runner{
		Targets: []string{"example.com:80", "example.net:80"},
		Timeout: 5 * time.Second,
	}
}

// RunTests spawns concurrency to create TCP sessions & exchange minimal data
func (l *Layer5Runner) RunTests(logger *zap.Logger) ([]TestResult, error) {
	logger.Info("Starting Layer 5 (Session) tests",
		zap.Strings("targets", l.Targets),
		zap.Duration("timeout", l.Timeout),
	)

	if len(l.Targets) == 0 {
		err := errors.New("no session targets provided")
		logger.Error("Layer5 test aborted", zap.Error(err))
		return nil, err
	}

	var wg sync.WaitGroup
	resultsChan := make(chan TestResult, len(l.Targets))

	for _, t := range l.Targets {
		wg.Add(1)
		go func(target string) {
			defer wg.Done()
			res := checkSession(target, l.Timeout, logger)
			resultsChan <- res
		}(t)
	}

	wg.Wait()
	close(resultsChan)

	var allResults []TestResult
	var failCount int
	for res := range resultsChan {
		if res.Status == "Failed" {
			failCount++
		}
		allResults = append(allResults, res)
	}

	if failCount == len(allResults) {
		// if all concurrency checks failed, we consider overall fail
		err := errors.New("all concurrency session attempts failed")
		logger.Error(err.Error())
		return allResults, err
	}

	logger.Info("Layer 5 concurrency checks complete", zap.Int("total", len(allResults)), zap.Int("failures", failCount))
	return allResults, nil
}

// checkSession attempts to open a TCP session, send minimal data (HTTP GET), read response
func checkSession(target string, timeout time.Duration, logger *zap.Logger) TestResult {
	layerNumber := 5
	// e.g. "example.com:80"

	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		msg := fmt.Sprintf("Failed to establish session with %s: %v", target, err)
		logger.Error(msg)
		return TestResult{Layer: layerNumber, Status: "Failed", Message: msg}
	}
	defer conn.Close()

	// minimal request
	req := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", strings.Split(target, ":")[0])
	_, wErr := conn.Write([]byte(req))
	if wErr != nil {
		msg := fmt.Sprintf("Failed sending session data to %s: %v", target, wErr)
		logger.Error(msg)
		return TestResult{Layer: layerNumber, Status: "Failed", Message: msg}
	}
	logger.Info("Session data sent", zap.String("target", target))

	// read response
	buf := make([]byte, 2048)
	n, rErr := conn.Read(buf)
	if rErr != nil && rErr != io.EOF {
		msg := fmt.Sprintf("Failed reading session response from %s: %v", target, rErr)
		logger.Error(msg)
		return TestResult{Layer: layerNumber, Status: "Failed", Message: msg}
	}

	msg := fmt.Sprintf("Session with %s established. Received %d bytes.\n", target, n)
	logger.Info("Session success", zap.String("target", target), zap.Int("bytes_received", n))
	return TestResult{Layer: layerNumber, Status: "Passed", Message: msg}
}

// -------------- Logging Setup --------------

func InitializeLogger() (*zap.Logger, string, error) {
	if err := os.MkdirAll(loggingDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create log directory: %w", err)
	}

	// e.g. "20260907_153012"
	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(loggingDir, fmt.Sprintf("osilayers_log_%s.log", timestamp))

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logFile, "stdout"}

	logger, err := cfg.Build()
	if err != nil {
		return nil, "", fmt.Errorf("failed to initialize logger: %w", err)
	}

	return logger, logFile, nil
}

// -------------- Reporting --------------

func GenerateReport(results []TestResult) error {
	if err := os.MkdirAll(reportingDir, 0755); err != nil {
		return fmt.Errorf("failed to create reporting directory: %w", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	csvPath := filepath.Join(reportingDir, fmt.Sprintf("osilayers_report_%s.csv", timestamp))
	pdfPath := filepath.Join(reportingDir, fmt.Sprintf("osilayers_report_%s.pdf", timestamp))

	if err := writeCSVReport(results, csvPath); err != nil {
		return err
	}
	if err := writePDFReport(results, pdfPath); err != nil {
		return err
	}
	return nil
}

func writeCSVReport(results []TestResult, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create csv file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// header
	if err := w.Write([]string{"Layer", "Status", "Message"}); err != nil {
		return err
	}
	for _, r := range results {
		row := []string{strconv.Itoa(r.Layer), r.Status, r.Message}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func writePDFReport(results []TestResult, path string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(40, 10, "OSI Layer Test Report")
	pdf.Ln(12)

	// Table header
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(20, 8, "Layer")
	pdf.Cell(30, 8, "Status")
	pdf.Cell(140, 8, "Message")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 12)
	for _, r := range results {
		pdf.Cell(20, 8, strconv.Itoa(r.Layer))
		pdf.Cell(30, 8, r.Status)

		x := pdf.GetX()
		y := pdf.GetY()
		pdf.MultiCell(140, 8, r.Message, "", "", false)
		// Move to next line
		pdf.SetXY(x+20+30+140, y)
		pdf.Ln(8)
	}

	if err := pdf.OutputFileAndClose(path); err != nil {
		return fmt.Errorf("failed to write pdf: %w", err)
	}
	return nil
}

// -------------- Visualization with Raylib --------------

func DisplayInterface(results []TestResult) {
	rl.InitWindow(1280, 720, "OSI Layer 5 (Session) Test Results")
	defer rl.CloseWindow()

	// Attempt custom font
	font := rl.LoadFontEx("resources/futuristic_font.ttf", 24, nil, 0)
	if font.BaseSize == 0 {
		font = rl.GetFontDefault()
	}
	defer rl.UnloadFont(font)

	rl.SetTargetFPS(60)
	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.DarkBlue)

		title := "Session Layer (Layer 5) Test Results"
		rl.DrawTextEx(font, title, rl.NewVector2(40, 20), 30, 2, rl.White)

		y := float32(80)
		for _, r := range results {
			color := rl.Green
			if strings.ToLower(r.Status) == "failed" {
				color = rl.Red
			}
			text := fmt.Sprintf("Layer %d: %s", r.Layer, r.Status)
			rl.DrawTextEx(font, text, rl.NewVector2(40, y), 24, 2, color)
			y += 30

			snippet := r.Message
			if len(snippet) > 120 {
				snippet = snippet[:120] + "..."
			}
			rl.DrawTextEx(font, snippet, rl.NewVector2(60, y), 20, 1, rl.White)
			y += 40
		}

		rl.EndDrawing()
	}
}

// -------------- MAIN --------------

func main() {
	logger, logFile, err := InitializeLogger()
	if err != nil {
		fmt.Printf("Logger init error: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	logger.Info("Logger initialized", zap.String("log_file", logFile))

	runner := NewLayer5Runner()
	// e.g. customizing: runner.Targets = []string{"example.com:80", "api.example.org:443"}
	// runner.Timeout = 5 * time.Second

	results, err := runner.RunTests(logger)
	if err != nil {
		logger.Warn("Some or all session checks failed", zap.Error(err))
	}

	// Generate CSV/PDF
	if err := GenerateReport(results); err != nil {
		logger.Error("Report generation failure", zap.Error(err))
	} else {
		logger.Info("Report generation success")
	}

	// Raylib UI
	DisplayInterface(results)

	logger.Info("Session layer tests complete. Exiting.")
}
