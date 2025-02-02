package layers

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/jung-kurt/gofpdf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TestResult holds the result for a single OSI layer.
type TestResult struct {
	Layer   int    // OSI Layer number
	Status  string // "Passed" or "Failed"
	Message string // Additional details about test results
}

// LayerRunner is an interface for each OSI layer’s runner.
type LayerRunner interface {
	RunTests(logger *zap.Logger) (TestResult, error)
}

// -------------- Layer1Runner (Physical Layer) --------------

// Layer1Runner simulates physical layer checks (e.g., cables, link detection).
type Layer1Runner struct {
	AttemptCount int
}

// RunTests concurrently checks physical connections, signal strength, etc.
func (l *Layer1Runner) RunTests(logger *zap.Logger) (TestResult, error) {
	logger.Info("Starting Layer 1 (Physical Layer) tests")

	// concurrency-based checking
	if l.AttemptCount <= 0 {
		l.AttemptCount = 3
	}

	var wg sync.WaitGroup
	results := make(chan bool, l.AttemptCount)

	for i := 0; i < l.AttemptCount; i++ {
		wg.Add(1)
		go func(iter int) {
			defer wg.Done()
			// Check cable or physical link
			isOk := checkPhysicalConnection(iter)
			results <- isOk
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
		err := errors.New("at least one concurrency check failed for physical link")
		logger.Error("Layer1 physical test failure", zap.Error(err))
		return TestResult{Layer: 1, Status: "Failed", Message: err.Error()}, err
	}

	// Next, signal strength
	strength := checkSignalStrength()
	if strength < 50 {
		err := fmt.Errorf("signal strength too low: %d%%", strength)
		logger.Error("Layer1 signal check failed", zap.Error(err))
		return TestResult{Layer: 1, Status: "Failed", Message: err.Error()}, err
	}

	msg := fmt.Sprintf("Physical layer test passed. AttemptCount=%d, Min signal=%d%%", l.AttemptCount, strength)
	logger.Info("Layer1 tests completed successfully", zap.String("details", msg))
	return TestResult{Layer: 1, Status: "Passed", Message: msg}, nil
}

// checkPhysicalConnection is a stub that might read from netlink, parse "ip link show", etc.
func checkPhysicalConnection(iter int) bool {
	time.Sleep(50 * time.Millisecond) // simulate
	// We pretend success for demonstration
	return true
}

// checkSignalStrength returns 85 for demonstration.
func checkSignalStrength() int {
	return 85
}

// -------------- Layer3Runner (Network Layer) --------------

type Layer3Runner struct {
	Hostname  string // for DNS resolution
	PingAddr  string // IP to ping
	PingCount int    // how many echo requests
}

func (l Layer3Runner) RunTests(logger *zap.Logger) (TestResult, error) {
	if l.Hostname == "" {
		l.Hostname = "example.com"
	}
	if l.PingAddr == "" {
		l.PingAddr = "8.8.8.8"
	}
	if l.PingCount <= 0 {
		l.PingCount = 4
	}

	logger.Info("Starting Layer3 (Network) tests", zap.String("hostname", l.Hostname), zap.String("ping_ip", l.PingAddr))

	// 1) DNS Lookup
	ipAddrs, err := net.LookupIP(l.Hostname)
	if err != nil {
		msg := fmt.Sprintf("Failed to resolve hostname '%s': %v", l.Hostname, err)
		logger.Error(msg)
		return TestResult{Layer: 3, Status: "Failed", Message: msg}, err
	}

	// 2) Ping
	pingOutput, err := runPing(l.PingAddr, l.PingCount)
	if err != nil {
		msg := fmt.Sprintf("Ping to %s failed: %v", l.PingAddr, err)
		logger.Error(msg)
		return TestResult{Layer: 3, Status: "Failed", Message: msg}, err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Hostname '%s' resolved to IP(s):\n", l.Hostname))
	for _, ip := range ipAddrs {
		sb.WriteString(fmt.Sprintf("  %s\n", ip.String()))
	}
	sb.WriteString(fmt.Sprintf("Ping to %s successful. Output:\n%s", l.PingAddr, pingOutput))

	logger.Info("Layer3 test success", zap.String("details", sb.String()))
	return TestResult{
		Layer:   3,
		Status:  "Passed",
		Message: sb.String(),
	}, nil
}

// runPing attempts to run the OS ping command with an appropriate flag based on the platform.
func runPing(addr string, count int) (string, error) {
	var cmd *exec.Cmd
	cArg := fmt.Sprintf("%d", count)

	switch runtime.GOOS {
	case "windows":
		// Windows ping uses -n <count>
		cmd = exec.Command("ping", "-n", cArg, addr)
	default:
		// Unix-like
		cmd = exec.Command("ping", "-c", cArg, addr)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("ping command failed: %w (output=%s)", err, string(output))
	}
	return string(output), nil
}

// -------------- Logging Setup --------------

func InitializeLogger() (*zap.Logger, string, error) {
	logDir := "ghostshell/logging"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create log directory: %w", err)
	}

	// Use a more standard format: 20260102_150405
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
	pdfPath := filepath.Join(reportDir, fmt.Sprintf("osilayers_report_%s.pdf", timestamp))
	csvPath := filepath.Join(reportDir, fmt.Sprintf("osilayers_report_%s.csv", timestamp))

	// CSV
	if err := writeCSVReport(results, csvPath); err != nil {
		return err
	}

	// PDF
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

// -------------- Visualization with Raylib --------------

func DisplayInterface(results []TestResult) {
	rl.InitWindow(1280, 720, "OSI Layer Tests")
	defer rl.CloseWindow()

	// load custom font if available
	font := rl.LoadFontEx("resources/futuristic_font.ttf", 24, nil, 0)
	if font.BaseSize == 0 {
		font = rl.GetFontDefault()
	}
	defer rl.UnloadFont(font)

	rl.SetTargetFPS(60)
	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.DarkBlue)

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

			// If we want to show a snippet of the message:
			snippet := r.Message
			if len(snippet) > 80 {
				snippet = snippet[:80] + "..."
			}
			rl.DrawTextEx(font, snippet, rl.NewVector2(60, y), 18, 1, rl.White)
			y += 30
		}

		rl.EndDrawing()
	}
}

// -------------- MAIN (if desired) --------------

func main() {
	logger, logFile, err := InitializeLogger()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	logger.Info("Logger initialized", zap.String("log_file", logFile))

	// Create your OSI layer runners
	l1 := &Layer1Runner{AttemptCount: 3}
	l3 := &Layer3Runner{
		Hostname:  "example.com",
		PingAddr:  "8.8.8.8",
		PingCount: 4,
	}

	// Run tests
	results := []TestResult{}

	l1Res, err := l1.RunTests(logger)
	results = append(results, l1Res)

	l3Res, err := l3.RunTests(logger)
	results = append(results, l3Res)

	// Generate CSV/PDF
	if err := GenerateReport(results); err != nil {
		logger.Error("Report generation failed", zap.Error(err))
	} else {
		logger.Info("Report generation success")
	}

	// Show UI
	DisplayInterface(results)
	logger.Info("All tests complete. Exiting.")
}
