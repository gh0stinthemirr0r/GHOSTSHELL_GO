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

// TestResult holds the result of a layer test.
type TestResult struct {
	Layer   int
	Status  string
	Message string
}

// LayerRunner is an interface for OSI layer tests.
type LayerRunner interface {
	RunTests(logger *zap.Logger) (TestResult, error)
}

// -------------- Layer4Runner --------------

// Layer4Runner is responsible for testing the Transport Layer (OSI Layer 4).
// We demonstrate concurrency-based TCP checks for multiple ports, plus a sample UDP check.
type Layer4Runner struct {
	TCPAddresses []string // e.g. ["8.8.8.8:53", "1.1.1.1:53"]
	UDPAddress   string   // e.g. "8.8.8.8:53"
	Timeout      time.Duration
}

// NewLayer4Runner with default addresses if none are provided
func NewLayer4Runner() *Layer4Runner {
	return &Layer4Runner{
		TCPAddresses: []string{"8.8.8.8:53", "8.8.4.4:53", "1.1.1.1:53"},
		UDPAddress:   "8.8.8.8:53",
		Timeout:      5 * time.Second,
	}
}

// RunTests performs concurrency-based TCP checks and a single UDP check.
func (r *Layer4Runner) RunTests(logger *zap.Logger) (TestResult, error) {
	logger.Info("Starting Layer 4 (Transport) tests",
		zap.Strings("tcp_addresses", r.TCPAddresses),
		zap.String("udp_address", r.UDPAddress),
	)

	// 1) Concurrency-based TCP checks
	var wg sync.WaitGroup
	tcpResults := make(chan tcpCheckResult, len(r.TCPAddresses))

	for _, addr := range r.TCPAddresses {
		wg.Add(1)
		go func(a string) {
			defer wg.Done()
			ok, errMsg := checkTCPConnection(a, r.Timeout)
			tcpResults <- tcpCheckResult{
				Address: a,
				Success: ok,
				ErrMsg:  errMsg,
			}
		}(addr)
	}

	wg.Wait()
	close(tcpResults)

	var tcpFailures []string
	var details strings.Builder
	details.WriteString("TCP Checks:\n")

	for tr := range tcpResults {
		if tr.Success {
			logger.Info("TCP connection successful", zap.String("address", tr.Address))
			details.WriteString(fmt.Sprintf("  - %s: OK\n", tr.Address))
		} else {
			logger.Error("TCP connection failed", zap.String("address", tr.Address), zap.String("err", tr.ErrMsg))
			details.WriteString(fmt.Sprintf("  - %s: FAIL (%s)\n", tr.Address, tr.ErrMsg))
			tcpFailures = append(tcpFailures, tr.Address)
		}
	}

	// If all TCP checks fail, or some fail, we consider partial success
	if len(tcpFailures) > 0 && len(tcpFailures) == len(r.TCPAddresses) {
		err := errors.New("all TCP connections failed")
		msg := fmt.Sprintf("Layer4 test fails. See details:\n%s", details.String())
		logger.Error(msg, zap.Error(err))
		return TestResult{Layer: 4, Status: "Failed", Message: msg}, err
	}

	// 2) Single UDP check
	udpOK, udpErr := checkUDPConnection(r.UDPAddress, r.Timeout)
	if !udpOK {
		// We still log but consider partial pass if TCP is partially passing
		logger.Error("UDP check failed", zap.String("address", r.UDPAddress), zap.String("err", udpErr))
		details.WriteString(fmt.Sprintf("\nUDP Check: %s => FAIL (%s)\n", r.UDPAddress, udpErr))
		// We treat this as a fail for the entire test. Or partial success if you prefer.
		err := fmt.Errorf("UDP connection to %s failed: %s", r.UDPAddress, udpErr)
		msg := fmt.Sprintf("Transport check partial fail. Details:\n%s", details.String())
		return TestResult{Layer: 4, Status: "Failed", Message: msg}, err
	}
	details.WriteString(fmt.Sprintf("\nUDP Check: %s => OK\n", r.UDPAddress))

	// All is well
	logger.Info("Layer4 test success", zap.String("details", details.String()))
	return TestResult{
		Layer:   4,
		Status:  "Passed",
		Message: details.String(),
	}, nil
}

// tcpCheckResult is a concurrency container for each TCP address check
type tcpCheckResult struct {
	Address string
	Success bool
	ErrMsg  string
}

// checkTCPConnection attempts a DialTimeout to confirm connectivity
func checkTCPConnection(addr string, timeout time.Duration) (bool, string) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false, err.Error()
	}
	_ = conn.Close()
	return true, ""
}

// checkUDPConnection tries to connect with a UDP dial, optionally sending data
func checkUDPConnection(addr string, timeout time.Duration) (bool, string) {
	udpConn, err := net.Dial("udp", addr)
	if err != nil {
		return false, err.Error()
	}
	defer udpConn.Close()

	// set read/write deadlines
	_ = udpConn.SetDeadline(time.Now().Add(timeout))

	// Optionally send some data (like a DNS query)
	// For demonstration, let's just send empty data
	msg := []byte{0x00, 0x01, 0x02}
	_, wErr := udpConn.Write(msg)
	if wErr != nil {
		return false, wErr.Error()
	}

	buf := make([]byte, 64)
	n, rErr := udpConn.Read(buf)
	if rErr != nil {
		// Often we get no response for arbitrary data. We'll treat no response as fail
		return false, rErr.Error()
	}
	// If we read something, success
	if n > 0 {
		return true, ""
	}
	return false, "no data received"
}

// -------------- Logging Setup --------------

func InitializeLogger() (*zap.Logger, string, error) {
	logDir := "ghostshell/logging"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create log directory: %w", err)
	}

	// Standard format: 20260102_150405
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
		pdf.Cell(20, 8, strconv.Itoa(r.Layer))
		pdf.Cell(30, 8, r.Status)

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

// -------------- Visualization --------------

func DisplayInterface(results []TestResult) {
	rl.InitWindow(1280, 720, "Transport Layer (Layer4) Test Results")
	defer rl.CloseWindow()

	// Try to load a custom TTF
	font := rl.LoadFontEx("resources/futuristic_font.ttf", 24, nil, 0)
	if font.BaseSize == 0 {
		font = rl.GetFontDefault()
	}
	defer rl.UnloadFont(font)

	rl.SetTargetFPS(60)

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.DarkBlue)

		title := "Layer 4 - Transport Layer Tests"
		rl.DrawTextEx(font, title, rl.NewVector2(50, 20), 30, 2, rl.White)

		y := float32(80)
		for _, r := range results {
			color := rl.Green
			if strings.ToLower(r.Status) == "failed" {
				color = rl.Red
			}
			text := fmt.Sprintf("Layer %d: %s", r.Layer, r.Status)
			rl.DrawTextEx(font, text, rl.NewVector2(50, y), 24, 2, color)
			y += 30

			snippet := r.Message
			if len(snippet) > 120 {
				snippet = snippet[:120] + "..."
			}
			rl.DrawTextEx(font, snippet, rl.NewVector2(70, y), 20, 1, rl.White)
			y += 40
		}

		rl.EndDrawing()
	}
}

// -------------- MAIN (Demo) --------------

func main() {
	logger, logFile, err := InitializeLogger()
	if err != nil {
		fmt.Println("Failed to init logger:", err)
		os.Exit(1)
	}
	defer logger.Sync()
	logger.Info("Logger initialized", zap.String("log_file", logFile))

	// Instantiate a Layer4Runner
	runner := NewLayer4Runner()
	// Optionally customize:
	// runner.TCPAddresses = []string{"8.8.8.8:53", "8.8.4.4:53"}
	// runner.UDPAddress = "1.1.1.1:53"
	// runner.Timeout = 3 * time.Second

	// Run the test
	res, err := runner.RunTests(logger)
	if err != nil {
		logger.Error("Layer4 test encountered error", zap.Error(err))
	}

	// Build final results slice
	results := []TestResult{res}

	// Generate CSV/PDF
	if err := GenerateReport(results); err != nil {
		logger.Error("Report generation failure", zap.Error(err))
	} else {
		logger.Info("Report generation success")
	}

	// Show UI
	DisplayInterface(results)
	logger.Info("Done. Exiting.")
}
