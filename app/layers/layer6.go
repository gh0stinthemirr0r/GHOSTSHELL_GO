package layers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
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

// TestResult holds the result of a single layer test.
type TestResult struct {
	Layer   int
	Status  string
	Message string
}

// LayerRunner interface can be extended for all layers.
type LayerRunner interface {
	RunTests(logger *zap.Logger) ([]TestResult, error)
}

// -------------- Logging Setup --------------

var (
	loggingDir   = "ghostshell/logging"
	reportingDir = "ghostshell/reporting"
)

// InitializeLogger sets up a dynamic logger with a standard date/time format.
func InitializeLogger() (*zap.Logger, string, error) {
	if err := os.MkdirAll(loggingDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create log directory: %w", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(loggingDir, fmt.Sprintf("layer6_log_%s.log", timestamp))

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logFile, "stdout"}

	logger, err := cfg.Build()
	if err != nil {
		return nil, "", fmt.Errorf("failed to initialize logger: %w", err)
	}
	return logger, logFile, nil
}

// -------------- Layer6Runner (Presentation Layer) --------------

// Layer6Runner tests serialization/deserialization concurrency for multiple data sets or formats.
type Layer6Runner struct {
	// concurrency-based data sets to test
	DataSets []map[string]string
	// possible expansions: other encodings like "json", "xml", "base64", etc.
	Format string
}

// NewLayer6Runner returns a default runner with some sample data sets.
func NewLayer6Runner() *Layer6Runner {
	return &Layer6Runner{
		DataSets: []map[string]string{
			{"message": "Hello OSI L6 - Test 1", "status": "ok"},
			{"message": "Hello OSI L6 - Test 2", "status": "ok2"},
		},
		Format: "json",
	}
}

// RunTests concurrency-based encoding/decoding checks for multiple data sets
func (l *Layer6Runner) RunTests(logger *zap.Logger) ([]TestResult, error) {
	logger.Info("Starting Layer 6 (Presentation) tests",
		zap.Int("dataset_count", len(l.DataSets)),
		zap.String("format", l.Format),
	)

	var wg sync.WaitGroup
	resultChan := make(chan TestResult, len(l.DataSets))

	for i, ds := range l.DataSets {
		wg.Add(1)
		go func(idx int, data map[string]string) {
			defer wg.Done()
			res := l.checkEncodingDecoding(idx, data, logger)
			resultChan <- res
		}(i, ds)
	}

	wg.Wait()
	close(resultChan)

	var allResults []TestResult
	var failCount int
	for r := range resultChan {
		if r.Status == "Failed" {
			failCount++
		}
		allResults = append(allResults, r)
	}

	if failCount == len(allResults) {
		// If all concurrency checks fail, consider overall fail
		err := fmt.Errorf("all concurrency presentation checks failed")
		logger.Error(err.Error())
		return allResults, err
	}

	logger.Info("Layer6 concurrency checks complete",
		zap.Int("total", len(allResults)),
		zap.Int("failures", failCount),
	)
	return allResults, nil
}

// checkEncodingDecoding encodes the data map, then decodes, verifying no mismatch
func (l *Layer6Runner) checkEncodingDecoding(idx int, data map[string]string, logger *zap.Logger) TestResult {
	layer := 6

	// For demonstration, we handle only JSON.
	// You could add "xml" or other formats if l.Format == "xml" ...
	encoded, err := json.Marshal(data)
	if err != nil {
		msg := fmt.Sprintf("Data set %d: failed to encode to JSON: %v", idx, err)
		logger.Error(msg)
		return TestResult{Layer: layer, Status: "Failed", Message: msg}
	}
	logger.Debug("Data encoded", zap.Int("dataset_index", idx), zap.String("encoded", string(encoded)))

	var decoded map[string]string
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		msg := fmt.Sprintf("Data set %d: failed to decode JSON: %v", idx, err)
		logger.Error(msg)
		return TestResult{Layer: layer, Status: "Failed", Message: msg}
	}

	// Validate
	if !compareMaps(data, decoded) {
		msg := fmt.Sprintf("Data set %d: mismatch after encode/decode. original=%v decoded=%v", idx, data, decoded)
		logger.Error(msg)
		return TestResult{Layer: layer, Status: "Failed", Message: msg}
	}

	msg := fmt.Sprintf("Data set %d: successfully encoded & decoded. original=%v", idx, data)
	logger.Info("Layer6 encode/decode success", zap.Int("dataset_index", idx))
	return TestResult{Layer: layer, Status: "Passed", Message: msg}
}

// compareMaps checks if two string maps are identical
func compareMaps(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
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
		return fmt.Errorf("failed to write csv header: %w", err)
	}

	for _, r := range results {
		row := []string{
			strconv.Itoa(r.Layer),
			r.Status,
			r.Message,
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write csv row: %w", err)
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

	// headers
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(20, 8, "Layer")
	pdf.Cell(30, 8, "Status")
	pdf.Cell(140, 8, "Message")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 12)
	for _, r := range results {
		layerStr := strconv.Itoa(r.Layer)
		pdf.Cell(20, 8, layerStr)
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

// -------------- Visualization --------------

func DisplayInterface(results []TestResult) {
	rl.InitWindow(1280, 720, "Layer 6 (Presentation) Test Results")
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

		title := "Presentation Layer (Layer 6) Test Results"
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

// -------------- MAIN DEMO --------------

func main() {
	logger, logFile, err := InitializeLogger()
	if err != nil {
		fmt.Printf("Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Logger initialized", zap.String("log_file", logFile))

	runner := NewLayer6Runner()
	// Optionally customize runner fields:
	// runner.DataSets = append(runner.DataSets, map[string]string{"message": "AnotherTest", "status": "example"})
	// runner.Format = "json"

	results, err := runner.RunTests(logger)
	if err != nil {
		logger.Warn("Some or all presentation checks failed", zap.Error(err))
	}

	// Generate CSV/PDF
	if err := GenerateReport(results); err != nil {
		logger.Error("Report generation failure", zap.Error(err))
	} else {
		logger.Info("Report generation success")
	}

	// Show UI
	DisplayInterface(results)
	logger.Info("Presentation layer tests complete. Exiting.")
}
