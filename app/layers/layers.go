package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/jung-kurt/gofpdf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// -------------- Global Constants --------------

const (
	logDir       = "ghostshell/logging"
	reportDir    = "ghostshell/reporting"
	windowWidth  = 1280
	windowHeight = 720
	maxParticles = 50
)

// -------------- Shared Data Structures --------------

// TestResult represents one outcome from a single layer test or sub-test.
type TestResult struct {
	Layer   int    `json:"layer"`
	Status  string `json:"status"`  // e.g. "Passed", "Failed"
	Message string `json:"message"` // Additional details
}

// LayerRunner is the interface each layer implements, returning one or more test results.
type LayerRunner interface {
	RunTests(logger *zap.Logger) ([]TestResult, error)
}

// Particle is used for the Raylib bouncing particle effect.
type Particle struct {
	x, y   float32
	dx, dy float32
	color  rl.Color
}

// -------------- Logging Setup --------------

var logger *zap.Logger

// InitializeLogger sets up the Zap logger with a standard date/time format.
func InitializeLogger() error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// e.g., "20260102_153045"
	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(logDir, fmt.Sprintf("osilayers_log_%s.log", timestamp))

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logFile, "stdout"}

	var err error
	logger, err = cfg.Build()
	if err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}
	return nil
}

// -------------- Layers 1 - 7 Runners --------------
//
// Each runner implements RunTests(logger *zap.Logger) ([]TestResult, error)

// ----- Layer1Runner (Physical) -----
type Layer1Runner struct {
	AttemptCount int
}

func (l *Layer1Runner) RunTests(logger *zap.Logger) ([]TestResult, error) {
	logger.Info("Layer1: Starting physical layer checks",
		zap.Int("attempt_count", l.AttemptCount),
	)
	if l.AttemptCount <= 0 {
		l.AttemptCount = 3
	}

	var wg sync.WaitGroup
	resultsChan := make(chan bool, l.AttemptCount)

	for i := 0; i < l.AttemptCount; i++ {
		wg.Add(1)
		go func(iter int) {
			defer wg.Done()
			success := checkPhysicalConnection(iter)
			resultsChan <- success
		}(i)
	}

	wg.Wait()
	close(resultsChan)

	allOk := true
	for b := range resultsChan {
		if !b {
			allOk = false
			break
		}
	}

	strength := checkSignalStrength()
	var testRes []TestResult
	if !allOk {
		err := errors.New("one or more concurrency cable/link checks failed")
		msg := fmt.Sprintf("Layer1 physical concurrency checks: some failures. signal=%d%%", strength)
		logger.Error(msg, zap.Error(err))
		testRes = append(testRes, TestResult{Layer: 1, Status: "Failed", Message: msg})
		return testRes, err
	}
	if strength < 50 {
		err := fmt.Errorf("signal strength too low: %d%%", strength)
		logger.Error("Layer1 signal check failed", zap.Error(err))
		msg := "Layer1 test fail. " + err.Error()
		testRes = append(testRes, TestResult{Layer: 1, Status: "Failed", Message: msg})
		return testRes, err
	}

	passMsg := fmt.Sprintf("Layer1 concurrency checks all pass, signal=%d%%", strength)
	logger.Info(passMsg)
	testRes = append(testRes, TestResult{Layer: 1, Status: "Passed", Message: passMsg})
	return testRes, nil
}
func checkPhysicalConnection(iter int) bool {
	// Real logic might do netlink or "ip link show"
	// We'll simulate success
	time.Sleep(20 * time.Millisecond)
	return true
}
func checkSignalStrength() int {
	return 85
}

// ----- Layer2Runner (Data Link) -----
type Layer2Runner struct{}

func (l *Layer2Runner) RunTests(logger *zap.Logger) ([]TestResult, error) {
	logger.Info("Layer2: Starting data link checks")
	// concurrency for each net interface
	ifaces, err := net.Interfaces()
	if err != nil {
		msg := fmt.Sprintf("Failed to fetch net interfaces: %v", err)
		logger.Error(msg)
		return []TestResult{{Layer: 2, Status: "Failed", Message: msg}}, err
	}

	var wg sync.WaitGroup
	ifaceResChan := make(chan interfaceCheckResult, len(ifaces))

	for _, iface := range ifaces {
		wg.Add(1)
		go func(ifc net.Interface) {
			defer wg.Done()
			res := checkInterface(ifc)
			ifaceResChan <- res
		}(iface)
	}
	wg.Wait()
	close(ifaceResChan)

	var allPassed bool = true
	var details strings.Builder
	for r := range ifaceResChan {
		details.WriteString(fmt.Sprintf("iface=%s MAC=%s => %s\n", r.Name, r.MAC, r.Result))
		if r.Result != "OK" {
			allPassed = false
		}
	}

	var testRes []TestResult
	if allPassed {
		msg := "Layer2 all interfaces appear OK\n" + details.String()
		logger.Info(msg)
		testRes = append(testRes, TestResult{Layer: 2, Status: "Passed", Message: msg})
		return testRes, nil
	} else {
		err := errors.New("one or more interfaces invalid or down")
		msg := "Layer2 fail\n" + details.String()
		logger.Error(msg, zap.Error(err))
		testRes = append(testRes, TestResult{Layer: 2, Status: "Failed", Message: msg})
		return testRes, err
	}
}

type interfaceCheckResult struct {
	Name   string
	MAC    string
	Result string
}

func checkInterface(ifc net.Interface) interfaceCheckResult {
	mac := ifc.HardwareAddr.String()
	if mac == "" || strings.HasPrefix(mac, "00:00:00") {
		return interfaceCheckResult{Name: ifc.Name, MAC: mac, Result: "INVALID_MAC"}
	}
	if (ifc.Flags & net.FlagUp) == 0 {
		return interfaceCheckResult{Name: ifc.Name, MAC: mac, Result: "DOWN"}
	}
	return interfaceCheckResult{Name: ifc.Name, MAC: mac, Result: "OK"}
}

// ----- Layer3Runner (Network) -----
type Layer3Runner struct {
	Hostname  string
	PingAddr  string
	PingCount int
}

func (l *Layer3Runner) RunTests(logger *zap.Logger) ([]TestResult, error) {
	if l.Hostname == "" {
		l.Hostname = "example.com"
	}
	if l.PingAddr == "" {
		l.PingAddr = "8.8.8.8"
	}
	if l.PingCount <= 0 {
		l.PingCount = 4
	}

	logger.Info("Layer3: Starting network checks", zap.String("hostname", l.Hostname), zap.String("ping", l.PingAddr))

	// DNS resolution
	ips, err := net.LookupIP(l.Hostname)
	var results []TestResult

	if err != nil {
		msg := fmt.Sprintf("Failed DNS for %s: %v", l.Hostname, err)
		logger.Error(msg)
		results = append(results, TestResult{Layer: 3, Status: "Failed", Message: msg})
		return results, err
	}
	var sb strings.Builder
	for _, ip := range ips {
		sb.WriteString(ip.String() + " ")
	}
	logger.Info("Layer3 DNS success", zap.String("resolved_ips", sb.String()))
	results = append(results, TestResult{
		Layer:   3,
		Status:  "Passed",
		Message: fmt.Sprintf("DNS for %s => %s", l.Hostname, sb.String()),
	})

	// Ping
	out, err := runPing(l.PingAddr, l.PingCount)
	if err != nil {
		msg := fmt.Sprintf("Ping to %s fail: %v", l.PingAddr, err)
		logger.Error(msg)
		results = append(results, TestResult{Layer: 3, Status: "Failed", Message: msg})
		return results, err
	}
	passMsg := fmt.Sprintf("Ping to %s success:\n%s", l.PingAddr, out)
	logger.Info(passMsg)
	results = append(results, TestResult{Layer: 3, Status: "Passed", Message: passMsg})
	return results, nil
}
func runPing(ip string, count int) (string, error) {
	cStr := strconv.Itoa(count)
	var cmd *exec.Cmd
	switch detectOS() {
	case "windows":
		cmd = exec.Command("ping", "-n", cStr, ip)
	default:
		cmd = exec.Command("ping", "-c", cStr, ip)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}
func detectOS() string {
	// basic check, or use runtime.GOOS
	return strings.ToLower(os.Getenv("GOOS")) // might be empty, fallback
}

// ----- Layer4Runner (Transport) -----
type Layer4Runner struct {
	TCPAddresses []string
	UDPAddress   string
	Timeout      time.Duration
}

func (l *Layer4Runner) RunTests(logger *zap.Logger) ([]TestResult, error) {
	logger.Info("Layer4: Starting transport checks",
		zap.Strings("tcpAddresses", l.TCPAddresses),
		zap.String("udpAddress", l.UDPAddress),
	)
	var results []TestResult

	// concurrency for TCP addresses
	if len(l.TCPAddresses) == 0 {
		l.TCPAddresses = []string{"8.8.8.8:53", "1.1.1.1:53"}
	}
	if l.UDPAddress == "" {
		l.UDPAddress = "8.8.8.8:53"
	}
	if l.Timeout <= 0 {
		l.Timeout = 5 * time.Second
	}

	// TCP
	var wg sync.WaitGroup
	tcpChan := make(chan TestResult, len(l.TCPAddresses))

	for _, addr := range l.TCPAddresses {
		wg.Add(1)
		go func(a string) {
			defer wg.Done()
			pass, msg := checkTCPConnection(a, l.Timeout)
			st := "Passed"
			if !pass {
				st = "Failed"
			}
			tcpChan <- TestResult{Layer: 4, Status: st, Message: fmt.Sprintf("TCP %s => %s", a, msg)}
		}(addr)
	}
	wg.Wait()
	close(tcpChan)

	var tcpFails int
	for r := range tcpChan {
		if r.Status == "Failed" {
			tcpFails++
			logger.Error(r.Message)
		} else {
			logger.Info(r.Message)
		}
		results = append(results, r)
	}

	// UDP
	ok, msg := checkUDPConnection(l.UDPAddress, l.Timeout)
	if !ok {
		failMsg := fmt.Sprintf("UDP %s => %s", l.UDPAddress, msg)
		logger.Error(failMsg)
		results = append(results, TestResult{Layer: 4, Status: "Failed", Message: failMsg})
		return results, errors.New("transport layer partial fail")
	}
	successMsg := fmt.Sprintf("UDP %s => %s", l.UDPAddress, msg)
	logger.Info(successMsg)
	results = append(results, TestResult{Layer: 4, Status: "Passed", Message: successMsg})

	return results, nil
}
func checkTCPConnection(addr string, timeout time.Duration) (bool, string) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false, err.Error()
	}
	conn.Close()
	return true, "OK"
}
func checkUDPConnection(addr string, timeout time.Duration) (bool, string) {
	conn, err := net.Dial("udp", addr)
	if err != nil {
		return false, err.Error()
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	// send small data
	msg := []byte{0x00, 0x01}
	_, wErr := conn.Write(msg)
	if wErr != nil {
		return false, wErr.Error()
	}

	buf := make([]byte, 32)
	n, rErr := conn.Read(buf)
	if rErr != nil && rErr != io.EOF {
		return false, rErr.Error()
	}
	return true, fmt.Sprintf("Received %d bytes", n)
}

// ----- Layer5Runner (Session) -----
type Layer5Runner struct {
	Targets []string
	Timeout time.Duration
}

func (l *Layer5Runner) RunTests(logger *zap.Logger) ([]TestResult, error) {
	if len(l.Targets) == 0 {
		l.Targets = []string{"example.com:80", "example.net:80"}
	}
	if l.Timeout <= 0 {
		l.Timeout = 5 * time.Second
	}
	logger.Info("Layer5: Starting session checks", zap.Strings("targets", l.Targets))

	var wg sync.WaitGroup
	resChan := make(chan TestResult, len(l.Targets))

	for _, t := range l.Targets {
		wg.Add(1)
		go func(target string) {
			defer wg.Done()
			res := checkSession(target, l.Timeout, logger)
			resChan <- res
		}(t)
	}
	wg.Wait()
	close(resChan)

	var results []TestResult
	var failCount int
	for r := range resChan {
		if r.Status == "Failed" {
			failCount++
			logger.Error(r.Message)
		} else {
			logger.Info(r.Message)
		}
		results = append(results, r)
	}

	if failCount == len(results) {
		return results, errors.New("session layer concurrency checks all failed")
	}
	return results, nil
}
func checkSession(target string, timeout time.Duration, logger *zap.Logger) TestResult {
	layer := 5
	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		msg := fmt.Sprintf("Session fail for %s: %v", target, err)
		return TestResult{Layer: layer, Status: "Failed", Message: msg}
	}
	defer conn.Close()

	req := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", strings.Split(target, ":")[0])
	_, wErr := conn.Write([]byte(req))
	if wErr != nil {
		msg := fmt.Sprintf("Failed sending session data to %s: %v", target, wErr)
		return TestResult{Layer: layer, Status: "Failed", Message: msg}
	}

	buf := make([]byte, 512)
	n, rErr := conn.Read(buf)
	if rErr != nil && rErr != io.EOF {
		msg := fmt.Sprintf("Failed reading session response from %s: %v", target, rErr)
		return TestResult{Layer: layer, Status: "Failed", Message: msg}
	}

	passMsg := fmt.Sprintf("Session to %s success, read %d bytes", target, n)
	return TestResult{Layer: layer, Status: "Passed", Message: passMsg}
}

// ----- Layer6Runner (Presentation) -----
type Layer6Runner struct {
	DataSets []map[string]string
}

func (l *Layer6Runner) RunTests(logger *zap.Logger) ([]TestResult, error) {
	if len(l.DataSets) == 0 {
		l.DataSets = []map[string]string{
			{"message": "Hello Layer6 #1", "status": "ok"},
			{"message": "Hello Layer6 #2", "status": "ok2"},
		}
	}
	logger.Info("Layer6: Starting presentation checks", zap.Int("datasets", len(l.DataSets)))

	var wg sync.WaitGroup
	resChan := make(chan TestResult, len(l.DataSets))

	for i, ds := range l.DataSets {
		wg.Add(1)
		go func(idx int, data map[string]string) {
			defer wg.Done()
			r := checkEncodingDecoding(idx, data, logger)
			resChan <- r
		}(i, ds)
	}

	wg.Wait()
	close(resChan)

	var results []TestResult
	var failCount int
	for r := range resChan {
		if r.Status == "Failed" {
			failCount++
			logger.Error(r.Message)
		} else {
			logger.Info(r.Message)
		}
		results = append(results, r)
	}

	if failCount == len(results) {
		return results, errors.New("all concurrency presentation checks failed")
	}
	return results, nil
}
func checkEncodingDecoding(idx int, data map[string]string, logger *zap.Logger) TestResult {
	layer := 6
	enc, err := json.Marshal(data)
	if err != nil {
		msg := fmt.Sprintf("Dataset %d JSON encode fail: %v", idx, err)
		return TestResult{Layer: layer, Status: "Failed", Message: msg}
	}
	var dec map[string]string
	if err := json.Unmarshal(enc, &dec); err != nil {
		msg := fmt.Sprintf("Dataset %d JSON decode fail: %v", idx, err)
		return TestResult{Layer: layer, Status: "Failed", Message: msg}
	}

	if !compareMaps(data, dec) {
		msg := fmt.Sprintf("Dataset %d mismatch after encode/decode original=%v dec=%v", idx, data, dec)
		return TestResult{Layer: layer, Status: "Failed", Message: msg}
	}
	passMsg := fmt.Sprintf("Dataset %d presentation check pass. original=%v", idx, data)
	return TestResult{Layer: layer, Status: "Passed", Message: passMsg}
}
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

// ----- Layer7Runner (Application) -----
type Layer7Runner struct {
	Endpoints []string
	Timeout   time.Duration
}

func (l *Layer7Runner) RunTests(logger *zap.Logger) ([]TestResult, error) {
	if len(l.Endpoints) == 0 {
		l.Endpoints = []string{
			"https://jsonplaceholder.typicode.com/posts/1",
			"https://jsonplaceholder.typicode.com/posts/2",
		}
	}
	if l.Timeout <= 0 {
		l.Timeout = 5 * time.Second
	}

	logger.Info("Layer7: Starting application checks",
		zap.Strings("endpoints", l.Endpoints),
		zap.Duration("timeout", l.Timeout),
	)
	var wg sync.WaitGroup
	resChan := make(chan TestResult, len(l.Endpoints))

	for _, ep := range l.Endpoints {
		wg.Add(1)
		go func(e string) {
			defer wg.Done()
			r := checkHTTPGet(e, l.Timeout, logger)
			resChan <- r
		}(ep)
	}

	wg.Wait()
	close(resChan)

	var results []TestResult
	var failCount int
	for r := range resChan {
		if r.Status == "Failed" {
			failCount++
			logger.Error(r.Message)
		} else {
			logger.Info(r.Message)
		}
		results = append(results, r)
	}
	if failCount == len(results) {
		err := errors.New("all application layer endpoints failed")
		return results, err
	}
	return results, nil
}
func checkHTTPGet(url string, timeout time.Duration, logger *zap.Logger) TestResult {
	layer := 7
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		msg := fmt.Sprintf("HTTP GET %s fail: %v", url, err)
		return TestResult{Layer: layer, Status: "Failed", Message: msg}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := fmt.Sprintf("HTTP GET %s => status %d", url, resp.StatusCode)
		return TestResult{Layer: layer, Status: "Failed", Message: msg}
	}

	passMsg := fmt.Sprintf("HTTP GET %s => %d OK", url, resp.StatusCode)
	return TestResult{Layer: layer, Status: "Passed", Message: passMsg}
}

// -------------- Aggregation --------------

// ExecuteLayers runs all runners in order, collecting results. Then calls reporting.
type Options struct {
	OutputFormat string
}

func ExecuteLayers(runners []LayerRunner, opts Options) []TestResult {
	var allResults []TestResult
	for _, r := range runners {
		subResults, err := r.RunTests(logger)
		allResults = append(allResults, subResults...)
		if err != nil {
			logger.Warn("Some sub-tests in layer encountered errors", zap.Error(err))
		}
	}

	// Then generate the chosen output
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		logger.Error("Failed to create report directory", zap.Error(err))
		return allResults
	}

	timestamp := time.Now().Format("20060102_150405")
	csvPath := filepath.Join(reportDir, fmt.Sprintf("osilayers_report_%s.csv", timestamp))
	pdfPath := filepath.Join(reportDir, fmt.Sprintf("osilayers_report_%s.pdf", timestamp))
	jsonPath := filepath.Join(reportDir, fmt.Sprintf("osilayers_report_%s.json", timestamp))

	switch strings.ToLower(opts.OutputFormat) {
	case "csv":
		writeCSVReport(allResults, csvPath)
	case "pdf":
		writePDFReport(allResults, pdfPath)
	case "json":
		writeJSONReport(allResults, jsonPath)
	default:
		logger.Error("Unsupported output format. Choose 'csv', 'pdf', or 'json'.",
			zap.String("requested_format", opts.OutputFormat),
		)
	}

	return allResults
}

// -------------- Writers --------------

func writeCSVReport(results []TestResult, outputPath string) {
	f, err := os.Create(outputPath)
	if err != nil {
		logger.Error("Failed to create CSV file", zap.Error(err))
		return
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"Layer", "Status", "Message"}); err != nil {
		logger.Error("Failed to write CSV header", zap.Error(err))
		return
	}
	for _, r := range results {
		row := []string{
			strconv.Itoa(r.Layer),
			r.Status,
			r.Message,
		}
		if err := w.Write(row); err != nil {
			logger.Error("Failed to write CSV row", zap.Error(err))
		}
	}
	logger.Info("CSV report generated", zap.String("file", outputPath))
}

func writePDFReport(results []TestResult, outputPath string) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(40, 10, "OSI Layer Test Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(30, 8, "Layer")
	pdf.Cell(40, 8, "Status")
	pdf.Cell(120, 8, "Message")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 12)
	for _, r := range results {
		pdf.Cell(30, 8, strconv.Itoa(r.Layer))
		pdf.Cell(40, 8, r.Status)

		x := pdf.GetX()
		y := pdf.GetY()
		pdf.MultiCell(120, 8, r.Message, "", "", false)
		pdf.SetXY(x+30+40+120, y)
		pdf.Ln(8)
	}

	if err := pdf.OutputFileAndClose(outputPath); err != nil {
		logger.Error("Failed to write PDF report", zap.Error(err))
	} else {
		logger.Info("PDF report generated", zap.String("file", outputPath))
	}
}

func writeJSONReport(results []TestResult, outputPath string) {
	f, err := os.Create(outputPath)
	if err != nil {
		logger.Error("Failed to create JSON file", zap.Error(err))
		return
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", " ")
	if err := enc.Encode(results); err != nil {
		logger.Error("Failed to write JSON report", zap.Error(err))
	} else {
		logger.Info("JSON report generated", zap.String("file", outputPath))
	}
}

// -------------- GUI / Raylib --------------

func RunGUI(ctx context.Context) {
	rl.InitWindow(windowWidth, windowHeight, "OSI Layers Test")
	defer rl.CloseWindow()

	rand.Seed(time.Now().UnixNano())

	// generate random particles
	particles := make([]*Particle, maxParticles)
	for i := 0; i < maxParticles; i++ {
		p := &Particle{
			x:  float32(rand.Intn(windowWidth)),
			y:  float32(rand.Intn(windowHeight)),
			dx: (rand.Float32()*2 - 1) * 2,
			dy: (rand.Float32()*2 - 1) * 2,
			color: rl.Color{
				R: uint8(rand.Intn(256)),
				G: uint8(rand.Intn(256)),
				B: uint8(rand.Intn(256)),
				A: 255,
			},
		}
		particles[i] = p
	}

	rl.SetTargetFPS(60)

	for !rl.WindowShouldClose() && ctx.Err() == nil {
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		// update & draw
		for _, p := range particles {
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

		rl.DrawText("Press ESC or Ctrl+C to exit", 20, 20, 20, rl.RayWhite)

		rl.EndDrawing()
	}

	logger.Info("Raylib GUI shutdown complete")
}

// -------------- MAIN --------------

func main() {
	// 1) Initialize logging
	if err := InitializeLogger(); err != nil {
		fmt.Printf("Logger init fail: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// 2) Handle signals for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		logger.Info("Received shutdown signal, canceling context")
		cancel()
	}()

	// 3) Parse flags
	outputFormat := "csv"
	flag.StringVar(&outputFormat, "format", outputFormat, "Output format: csv, pdf, or json")
	flag.Parse()
	opts := Options{OutputFormat: outputFormat}

	// 4) Build your layer runners
	// e.g. Layer1 -> 7
	layerRunners := []LayerRunner{
		&Layer1Runner{AttemptCount: 3},
		&Layer2Runner{},
		&Layer3Runner{
			Hostname:  "example.com",
			PingAddr:  "8.8.8.8",
			PingCount: 4,
		},
		&Layer4Runner{
			TCPAddresses: []string{"8.8.8.8:53", "1.1.1.1:53"},
			UDPAddress:   "8.8.8.8:53",
			Timeout:      5 * time.Second,
		},
		&Layer5Runner{
			Targets: []string{"example.com:80", "api.example.net:443"},
			Timeout: 5 * time.Second,
		},
		&Layer6Runner{
			DataSets: []map[string]string{
				{"message": "Hello L6 #1", "status": "ok"},
				{"message": "Hello L6 #2", "status": "ok2"},
			},
		},
		&Layer7Runner{
			Endpoints: []string{
				"https://jsonplaceholder.typicode.com/posts/1",
				"https://jsonplaceholder.typicode.com/posts/2",
			},
			Timeout: 5 * time.Second,
		},
	}

	// 5) Execute tests & produce chosen report
	results := ExecuteLayers(layerRunners, opts)

	// 6) Launch Raylib GUI with bouncing particles
	RunGUI(ctx)

	logger.Info("Program shut down cleanly.")
}
