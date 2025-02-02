package layers

import (
	"encoding/csv"
	"errors"
	"fmt"
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

// TestResult holds the result for a single layer.
type TestResult struct {
	Layer   int
	Status  string
	Message string
}

// LayerRunner is an interface for each OSI layer’s runner.
// In your code, you can define RunTests(logger *zap.Logger) for each layer.
type LayerRunner interface {
	RunTests(logger *zap.Logger) (TestResult, error)
}

// -------------- Layer 1 Runner (Physical Layer) --------------

type Layer1Runner struct {
	// Additional fields like concurrency, attempt counts, or hardware checks
	attemptCount int
}

// NewLayer1Runner returns a default runner for layer1.
func NewLayer1Runner(attemptCount int) *Layer1Runner {
	if attemptCount <= 0 {
		attemptCount = 3
	}
	return &Layer1Runner{attemptCount: attemptCount}
}

// RunTests for Layer1 tries to detect physical connections, signal strength, etc.
func (l *Layer1Runner) RunTests(logger *zap.Logger) (TestResult, error) {
	logger.Info("Starting Layer 1 (Physical Layer) tests...")

	// Example: Checking concurrency for multiple cables or interfaces
	// We'll just replicate a "physical check" a few times.
	var wg sync.WaitGroup
	results := make(chan bool, l.attemptCount)

	for i := 0; i < l.attemptCount; i++ {
		wg.Add(1)
		go func(iter int) {
			defer wg.Done()
			ok := checkPhysicalConnection(iter)
			results <- ok
		}(i)
	}

	wg.Wait()
	close(results)

	allOk := true
	for r := range results {
		if !r {
			allOk = false
			break
		}
	}
	if !allOk {
		err := errors.New("physical cable or connection not detected on at least one attempt")
		logger.Error("Layer 1 test failed", zap.Error(err))
		return TestResult{Layer: 1, Status: "Failed", Message: err.Error()}, err
	}

	// Next, check signal strength
	strength := checkSignalStrength()
	if strength < 50 {
		err := fmt.Errorf("signal strength too low at %d%%", strength)
		logger.Error("Layer 1 test failed", zap.Error(err))
		return TestResult{Layer: 1, Status: "Failed", Message: err.Error()}, err
	}

	msg := fmt.Sprintf("Layer 1 test successful. Physical checks: %d attempts, min signal: %d%%", l.attemptCount, strength)
	logger.Info(msg)
	return TestResult{Layer: 1, Status: "Passed", Message: msg}, nil
}

// checkPhysicalConnection is a stub for real hardware cable detection or link status read
func checkPhysicalConnection(attempt int) bool {
	// Randomly always returning true for demonstration
	// Real logic might read from e.g. "ip link show" or netlink on Linux
	time.Sleep(20 * time.Millisecond) // simulate some I/O
	return true
}

// checkSignalStrength is a stub that returns 85
func checkSignalStrength() int {
	return 85
}

// -------------- Layer 2 Runner (Data Link Layer) --------------

type Layer2Runner struct{}

func (l Layer2Runner) RunTests(logger *zap.Logger) (TestResult, error) {
	logger.Info("Starting Layer 2 (Data Link) tests...")

	// Example test: fetch network interfaces for MAC addresses
	interfaces, err := net.Interfaces()
	if err != nil {
		errMsg := "Unable to fetch network interfaces"
		logger.Error(errMsg, zap.Error(err))
		return TestResult{Layer: 2, Status: "Failed", Message: errMsg}, err
	}

	var details strings.Builder
	for _, iface := range interfaces {
		details.WriteString(fmt.Sprintf("Interface: %s, MAC: %s\n", iface.Name, iface.HardwareAddr.String()))
	}

	msg := "Layer 2 Test successful. Details:\n" + details.String()
	logger.Info(msg)
	return TestResult{Layer: 2, Status: "Passed", Message: msg}, nil
}

// -------------- Logging Setup --------------

func InitializeLogger() (*zap.Logger, string, error) {
	logDir := "ghostshell/logging"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create log directory: %w", err)
	}

	// Use a more standard format: 20230102_150405
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

// -------------- Reporting --------------

func GenerateReport(results []TestResult) error {
	reportDir := "ghostshell/reporting"
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	// dynamic filenames
	timestamp := time.Now().Format("20060102_150405")
	pdfFile := filepath.Join(reportDir, fmt.Sprintf("osilayers_report_%s.pdf", timestamp))
	csvFile := filepath.Join(reportDir, fmt.Sprintf("osilayers_report_%s.csv", timestamp))

	// CSV
	if err := writeCSVReport(results, csvFile); err != nil {
		return err
	}

	// PDF
	if err := writePDFReport(results, pdfFile); err != nil {
		return err
	}
	return nil
}

func writeCSVReport(results []TestResult, csvPath string) error {
	f, err := os.Create(csvPath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	if err := writer.Write([]string{"Layer", "Status", "Message"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}
	for _, r := range results {
		row := []string{strconv.Itoa(r.Layer), r.Status, r.Message}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}
	return nil
}

func writePDFReport(results []TestResult, pdfPath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(40, 10, "OSI Layer Test Report")
	pdf.Ln(12)

	// Table header
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(30, 10, "Layer")
	pdf.Cell(40, 10, "Status")
	pdf.Cell(120, 10, "Message")
	pdf.Ln(10)

	// Table rows
	pdf.SetFont("Arial", "", 12)
	for _, r := range results {
		pdf.Cell(30, 10, strconv.Itoa(r.Layer))
		pdf.Cell(40, 10, r.Status)
		// Use MultiCell for the message in case it's multiline
		x := pdf.GetX()
		y := pdf.GetY()
		pdf.MultiCell(120, 10, r.Message, "", "", false)
		pdf.SetXY(x+30+40+120, y)
		pdf.Ln(10)
	}

	if err := pdf.OutputFileAndClose(pdfPath); err != nil {
		return fmt.Errorf("failed to create PDF report: %w", err)
	}
	return nil
}

// -------------- Visualization --------------

func DisplayInterface(results []TestResult) {
	rl.InitWindow(1280, 720, "OSI Layer Tests")
	defer rl.CloseWindow()

	// Try to load a custom TTF
	font := rl.LoadFontEx("resources/futuristic_font.ttf", 24, nil, 0)
	if font.BaseSize == 0 {
		// fallback if custom font fails
		font = rl.GetFontDefault()
	}
	defer rl.UnloadFont(font)

	rl.SetTargetFPS(60)

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		y := float32(40)
		headerText := "OSI Layers Test Results"
		rl.DrawTextEx(font, headerText, rl.NewVector2(40, 10), 28, 2, rl.RayWhite)

		// For each layer result, display
		for _, r := range results {
			statusColor := rl.Green
			if r.Status == "Failed" {
				statusColor = rl.Red
			}
			msg := fmt.Sprintf("Layer %d: %s", r.Layer, r.Status)
			rl.DrawTextEx(font, msg, rl.NewVector2(40, y), 24, 1, statusColor)

			// If you want to display part of the message or multiline, do so here
			// or do a line break with y += some spacing
			y += 30

			// Possibly limit how big we get if results are too many
			if y > float32(rl.GetScreenHeight()-40) {
				break
			}
		}

		rl.EndDrawing()
	}
}

// -------------- MAIN --------------

func main() {
	// 1) Initialize logging
	logger, logFile, err := InitializeLogger()
	if err != nil {
		fmt.Println("Error initializing logger:", err)
		os.Exit(1)
	}
	defer logger.Sync()
	logger.Info("Logger initialized", zap.String("log_file", logFile))

	// 2) Create layer runners
	// e.g., we have a Layer1Runner and a Layer2Runner
	layer1 := NewLayer1Runner(3) // 3 attempts
	layer2 := Layer2Runner{}

	// 3) Run tests
	runners := []LayerRunner{layer1, layer2}
	var results []TestResult
	for _, runner := range runners {
		r, err := runner.RunTests(logger)
		results = append(results, r)
		if err != nil {
			logger.Warn("A layer test encountered errors", zap.Error(err))
		}
	}

	// 4) Generate reports
	if err := GenerateReport(results); err != nil {
		logger.Warn("Failed to generate reports", zap.Error(err))
	} else {
		logger.Info("Reports generated successfully")
	}

	// 5) Display an optional UI
	DisplayInterface(results)

	logger.Info("All done. Exiting now.")
}
