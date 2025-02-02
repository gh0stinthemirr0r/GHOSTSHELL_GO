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
type LayerRunner interface {
	RunTests(logger *zap.Logger) (TestResult, error)
}

// --------------------- Layer1Runner (Physical) ---------------------

type Layer1Runner struct {
	AttemptCount int
}

// NewLayer1Runner returns a concurrency-based approach for checking physical connections (e.g. cables).
func NewLayer1Runner(attemptCount int) *Layer1Runner {
	if attemptCount <= 0 {
		attemptCount = 3
	}
	return &Layer1Runner{AttemptCount: attemptCount}
}

// RunTests spawns concurrency for cable checks, then checks signal strength.
func (l *Layer1Runner) RunTests(logger *zap.Logger) (TestResult, error) {
	logger.Info("Starting Layer 1 (Physical) tests",
		zap.Int("attempt_count", l.AttemptCount),
	)

	var wg sync.WaitGroup
	results := make(chan bool, l.AttemptCount)

	// concurrency-based check for physical cable or link
	for i := 0; i < l.AttemptCount; i++ {
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
	for res := range results {
		if !res {
			allOk = false
			break
		}
	}

	if !allOk {
		err := errors.New("some concurrency checks for physical cable/link failed")
		logger.Error("Layer1 physical checks failed", zap.Error(err))
		return TestResult{Layer: 1, Status: "Failed", Message: err.Error()}, err
	}

	// Then check signal strength
	strength := checkSignalStrength()
	if strength < 50 {
		err := fmt.Errorf("signal strength too low: %d%%", strength)
		logger.Error("Layer1 signal check failed", zap.Error(err))
		return TestResult{Layer: 1, Status: "Failed", Message: err.Error()}, err
	}

	msg := fmt.Sprintf("Layer1 test success. AttemptCount=%d, signal=%d%%", l.AttemptCount, strength)
	logger.Info(msg)
	return TestResult{
		Layer:   1,
		Status:  "Passed",
		Message: msg,
	}, nil
}

// checkPhysicalConnection is a stub. Real logic might query netlink, device driver, etc.
func checkPhysicalConnection(iter int) bool {
	time.Sleep(30 * time.Millisecond) // simulate
	// We'll pretend success for demonstration
	return true
}

// checkSignalStrength is a stub returning 85 for demonstration.
func checkSignalStrength() int {
	return 85
}

// --------------------- Layer2Runner (Data Link) ---------------------

type Layer2Runner struct {
	CheckAllInterfaces bool
}

// RunTests fetches all network interfaces, checks if they are up, and logs their MAC addresses.
func (l *Layer2Runner) RunTests(logger *zap.Logger) (TestResult, error) {
	logger.Info("Starting Layer 2 (Data Link) tests...")

	interfaces, err := net.Interfaces()
	if err != nil {
		msg := "Unable to fetch network interfaces"
		logger.Error(msg, zap.Error(err))
		return TestResult{Layer: 2, Status: "Failed", Message: msg}, err
	}

	// We'll do concurrency for each interface check
	var wg sync.WaitGroup
	ifaceResults := make(chan interfaceCheckResult, len(interfaces))

	for _, iface := range interfaces {
		wg.Add(1)
		go func(ifc net.Interface) {
			defer wg.Done()
			r := checkInterface(ifc)
			ifaceResults <- r
		}(iface)
	}

	wg.Wait()
	close(ifaceResults)

	var details strings.Builder
	passAll := true
	for r := range ifaceResults {
		details.WriteString(fmt.Sprintf("Interface: %s, MAC: %s, Status: %s\n",
			r.Name, r.MAC, r.Result))
		if r.Result != "OK" {
			passAll = false
		}
	}

	if passAll {
		msg := "Layer2 test success. Interfaces:\n" + details.String()
		logger.Info(msg)
		return TestResult{Layer: 2, Status: "Passed", Message: msg}, nil
	} else {
		err := errors.New("one or more interfaces failed checks (either down or bad MAC)")
		logger.Error(err.Error())
		return TestResult{Layer: 2, Status: "Failed", Message: details.String()}, err
	}
}

// interfaceCheckResult is a small struct to hold concurrency results for one interface.
type interfaceCheckResult struct {
	Name   string
	MAC    string
	Result string // e.g. "OK" or "DOWN"/"INVALID_MAC"
}

// checkInterface checks if an interface is up and has a valid MAC address, etc.
func checkInterface(ifc net.Interface) interfaceCheckResult {
	mac := ifc.HardwareAddr.String()
	// Basic MAC check: if it's empty or "00:00:00:00:00:00" => fail
	if mac == "" || strings.HasPrefix(mac, "00:00:00:00:00:00") {
		return interfaceCheckResult{
			Name:   ifc.Name,
			MAC:    mac,
			Result: "INVALID_MAC",
		}
	}

	// Check if interface is up
	if (ifc.Flags & net.FlagUp) == 0 {
		return interfaceCheckResult{
			Name:   ifc.Name,
			MAC:    mac,
			Result: "DOWN",
		}
	}

	// Possibly check net.FlagLoopback to exclude loopbacks?
	// For demonstration, we'll allow them.

	return interfaceCheckResult{
		Name:   ifc.Name,
		MAC:    mac,
		Result: "OK",
	}
}

// --------------------- Logging Setup ---------------------

func InitializeLogger() (*zap.Logger, string, error) {
	logDir := "ghostshell/logging"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create log directory: %w", err)
	}

	// Use a standard format: 20260102_150405
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

// --------------------- Reporting ---------------------

func GenerateReport(results []TestResult) error {
	reportDir := "ghostshell/reporting"
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	pdfPath := filepath.Join(reportDir, fmt.Sprintf("osilayers_report_%s.pdf", timestamp))
	csvPath := filepath.Join(reportDir, fmt.Sprintf("osilayers_report_%s.csv", timestamp))

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

		// For multiline message, use MultiCell
		x := pdf.GetX()
		y := pdf.GetY()
		pdf.MultiCell(140, 8, r.Message, "", "", false)
		// Move cursor to next line
		pdf.SetXY(x+20+30+140, y)
		pdf.Ln(8)
	}

	if err := pdf.OutputFileAndClose(path); err != nil {
		return fmt.Errorf("failed to write pdf: %w", err)
	}
	return nil
}

// --------------------- Visualization with Raylib ---------------------

func DisplayInterface(results []TestResult) {
	rl.InitWindow(1280, 720, "OSI Layer Tests")
	defer rl.CloseWindow()

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

		// Title
		title := "OSI Layer Test Results"
		rl.DrawTextEx(font, title, rl.NewVector2(40, 20), 32, 2, rl.White)

		y := float32(80)
		for _, r := range results {
			color := rl.Green
			if strings.ToLower(r.Status) == "failed" {
				color = rl.Red
			}
			text := fmt.Sprintf("Layer %d - %s", r.Layer, r.Status)
			rl.DrawTextEx(font, text, rl.NewVector2(40, y), 24, 2, color)
			y += 30

			// If you want to show a snippet of the message:
			snippet := r.Message
			if len(snippet) > 100 {
				snippet = snippet[:100] + "..."
			}
			rl.DrawTextEx(font, snippet, rl.NewVector2(60, y), 18, 1, rl.White)
			y += 30
		}

		rl.EndDrawing()
	}
}

// --------------------- MAIN ---------------------

func main() {
	// 1) Initialize logging
	logger, logFile, err := InitializeLogger()
	if err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	logger.Info("Logger initialized", zap.String("log_file", logFile))

	// 2) Instantiate layer runners
	layer1 := NewLayer1Runner(3) // e.g. 3 concurrency attempts
	layer2 := Layer2Runner{}     // concurrency checks for each interface

	// 3) Run tests for each layer
	runners := []LayerRunner{layer1, layer2}
	var results []TestResult
	for _, runner := range runners {
		r, err := runner.RunTests(logger)
		results = append(results, r)
		if err != nil {
			logger.Warn("Layer test encountered error", zap.Int("layer", r.Layer), zap.Error(err))
		}
	}

	// 4) Generate CSV/PDF reports
	if err := GenerateReport(results); err != nil {
		logger.Error("Report generation failed", zap.Error(err))
	} else {
		logger.Info("Report generation success")
	}

	// 5) Display a Raylib UI
	DisplayInterface(results)

	logger.Info("All layers tested. Exiting.")
}
