package layers

import (
	"encoding/csv"
	"fmt"
	"net/http"
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

// TestResult represents the result of a layer test.
type TestResult struct {
	Layer   int
	Status  string
	Message string
}

// LayerRunner is an interface if you want to integrate multiple layers in the same code base.
type LayerRunner interface {
	RunTests(logger *zap.Logger) ([]TestResult, error)
}

// -------------- Constants --------------

var (
	logDir    = "ghostshell/logging"
	reportDir = "ghostshell/reporting"
)

// -------------- Logging Setup --------------

// InitializeLogger sets up a concurrency-friendly Zap logger with standard date/time format.
func InitializeLogger() (*zap.Logger, string, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create log directory: %w", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(logDir, fmt.Sprintf("osilayers_log_%s.log", timestamp))

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logFile, "stdout"}

	logger, err := cfg.Build()
	if err != nil {
		return nil, "", fmt.Errorf("failed to initialize logger: %w", err)
	}

	return logger, logFile, nil
}

// -------------- Layer7Runner (Application Layer) --------------

// Layer7Runner runs concurrency-based HTTP GET requests to multiple endpoints
// to verify that the application-level protocol is working.
type Layer7Runner struct {
	Endpoints []string      // e.g. ["https://jsonplaceholder.typicode.com/posts/1", "https://httpbin.org/get"]
	Timeout   time.Duration // e.g. 5 seconds
}

// NewLayer7Runner returns a runner with default sample endpoints if none are provided.
func NewLayer7Runner() *Layer7Runner {
	return &Layer7Runner{
		Endpoints: []string{
			"https://jsonplaceholder.typicode.com/posts/1",
			"https://jsonplaceholder.typicode.com/posts/2",
		},
		Timeout: 5 * time.Second,
	}
}

// RunTests spawns concurrency for all endpoints, collecting pass/fail info.
func (l *Layer7Runner) RunTests(logger *zap.Logger) ([]TestResult, error) {
	logger.Info("Starting Layer7 (Application Layer) tests",
		zap.Strings("endpoints", l.Endpoints),
		zap.Duration("timeout", l.Timeout),
	)

	if len(l.Endpoints) == 0 {
		err := fmt.Errorf("no endpoints provided for Layer7Runner")
		logger.Error("Aborting Layer7 tests", zap.Error(err))
		return nil, err
	}

	// concurrency for each endpoint
	var wg sync.WaitGroup
	resultChan := make(chan TestResult, len(l.Endpoints))

	for _, endpoint := range l.Endpoints {
		wg.Add(1)
		go func(ep string) {
			defer wg.Done()
			res := l.checkEndpoint(ep, logger)
			resultChan <- res
		}(endpoint)
	}

	wg.Wait()
	close(resultChan)

	var allResults []TestResult
	failCount := 0
	for r := range resultChan {
		if r.Status == "Failed" {
			failCount++
		}
		allResults = append(allResults, r)
	}

	if failCount == len(allResults) {
		err := fmt.Errorf("all application-layer endpoints failed")
		logger.Error(err.Error())
		return allResults, err
	}

	logger.Info("Layer7 concurrency checks complete",
		zap.Int("total_endpoints", len(allResults)),
		zap.Int("failures", failCount),
	)
	return allResults, nil
}

// checkEndpoint does a GET request with a custom http.Client, checks for 2xx status
func (l *Layer7Runner) checkEndpoint(endpoint string, logger *zap.Logger) TestResult {
	layerNum := 7
	client := &http.Client{Timeout: l.Timeout}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		msg := fmt.Sprintf("Failed to create request for %s: %v", endpoint, err)
		logger.Error(msg)
		return TestResult{Layer: layerNum, Status: "Failed", Message: msg}
	}

	resp, err := client.Do(req)
	if err != nil {
		msg := fmt.Sprintf("HTTP GET failed for %s: %v", endpoint, err)
		logger.Error(msg)
		return TestResult{Layer: layerNum, Status: "Failed", Message: msg}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := fmt.Sprintf("HTTP request to %s returned status %d", endpoint, resp.StatusCode)
		logger.Error(msg)
		return TestResult{Layer: layerNum, Status: "Failed", Message: msg}
	}

	logger.Info("Layer7 check success",
		zap.String("endpoint", endpoint),
		zap.Int("status_code", resp.StatusCode),
	)
	return TestResult{
		Layer:   layerNum,
		Status:  "Passed",
		Message: fmt.Sprintf("HTTP GET %s => %d OK", endpoint, resp.StatusCode),
	}
}

// -------------- Reporting --------------

func GenerateReport(results []TestResult) error {
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	csvPath := filepath.Join(reportDir, fmt.Sprintf("layer7_report_%s.csv", timestamp))
	pdfPath := filepath.Join(reportDir, fmt.Sprintf("layer7_report_%s.pdf", timestamp))

	if err := writeCSVReport(results, csvPath); err != nil {
		return err
	}
	if err := writePDFReport(results, pdfPath); err != nil {
		return err
	}
	return nil
}

func writeCSVReport(results []TestResult, csvPath string) error {
	f, err := os.Create(csvPath)
	if err != nil {
		return fmt.Errorf("failed to create csv: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// header
	if err := w.Write([]string{"Layer", "Status", "Message"}); err != nil {
		return err
	}

	for _, r := range results {
		row := []string{
			strconv.Itoa(r.Layer),
			r.Status,
			r.Message,
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func writePDFReport(results []TestResult, pdfPath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "OSI Layer 7 - Application Layer Report")
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
		// Move cursor to next line
		pdf.SetXY(x+20+30+140, y)
		pdf.Ln(8)
	}

	if err := pdf.OutputFileAndClose(pdfPath); err != nil {
		return fmt.Errorf("failed to write pdf: %w", err)
	}
	return nil
}

// -------------- Raylib Visualization --------------

func DisplayInterface(results []TestResult) {
	rl.InitWindow(1280, 720, "Layer 7 (Application) Test Results")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	// Attempt to load a custom TTF
	font := rl.LoadFontEx("resources/futuristic_font.ttf", 24, nil, 0)
	if font.BaseSize == 0 {
		font = rl.GetFontDefault()
	}
	defer rl.UnloadFont(font)

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.DarkBlue)

		title := "Application Layer (Layer 7) Test Results"
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

// -------------- MAIN (Demo) --------------

func main() {
	// 1) Initialize logger
	logger, logFile, err := InitializeLogger()
	if err != nil {
		fmt.Printf("Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	logger.Info("Logger initialized", zap.String("log_file", logFile))

	// 2) Create the runner
	runner := NewLayer7Runner()
	// Optionally customize:
	// runner.Endpoints = []string{"https://jsonplaceholder.typicode.com/posts/1", "https://httpbin.org/get"}
	// runner.Timeout = 3 * time.Second

	// 3) Run tests
	results, err := runner.RunTests(logger)
	if err != nil {
		logger.Warn("Some or all application checks failed", zap.Error(err))
	}

	// 4) Generate CSV/PDF
	if err := GenerateReport(results); err != nil {
		logger.Error("Report generation failure", zap.Error(err))
	} else {
		logger.Info("Report generation success")
	}

	// 5) Show a Raylib UI
	DisplayInterface(results)
	logger.Info("All done. Exiting.")
}
