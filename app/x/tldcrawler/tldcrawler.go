package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jung-kurt/gofpdf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	// Post-quantum ephemeral references (Assumed to be implemented)
	"ghostshell/oqs/oqs_vault"
)

// Constants & Paths
const (
	LogDir         = "ghostshell/logging"
	ReportDir      = "ghostshell/reporting"
	WindowWidth    = 800
	WindowHeight   = 600
	FontSize       = 24
	MaxParticles   = 50
	MaxConcurrency = 100 // Number of concurrent workers for scanning
)

// PortStatus represents the status of a port.
type PortStatus struct {
	Port       int
	Protocol   string
	IsOpen     bool
	IsInsecure bool
}

// InsecurePorts is a predefined list of ports considered insecure.
var InsecurePorts = map[int]bool{
	21:   true, // FTP
	22:   true, // SSH
	23:   true, // Telnet
	25:   true, // SMTP
	53:   true, // DNS
	80:   true, // HTTP
	110:  true, // POP3
	143:  true, // IMAP
	443:  true, // HTTPS
	445:  true, // SMB
	3389: true, // RDP
	5900: true, // VNC
}

// Application encapsulates the main components of the TLD crawler.
type Application struct {
	Logger        *zap.Logger
	ShutdownChan  chan os.Signal
	ReadyChan     chan struct{}
	Enumerated    map[string]bool
	EnumeratedMux sync.Mutex
	Particles     []*Particle
}

// Particle represents a single particle in the UI.
type Particle struct {
	x, y   float32
	dx, dy float32
	color  rl.Color
}

// NewApplication initializes the Application with logging and signal handling.
func NewApplication() (*Application, error) {
	logger, err := setupLogger()
	if err != nil {
		return nil, err
	}

	// Initialize Post-Quantum Vault (Assumed to be implemented)
	if err := oqs_vault.InitEphemeralKey(); err != nil {
		logger.Warn("Failed to initialize Post-Quantum Vault", zap.Error(err))
	}

	return &Application{
		Logger:       logger,
		ShutdownChan: make(chan os.Signal, 1),
		ReadyChan:    make(chan struct{}),
		Enumerated:   make(map[string]bool),
	}, nil
}

// setupLogger initializes a Zap logger with a timestamped log file in ISO8601 format.
func setupLogger() (*zap.Logger, error) {
	if err := os.MkdirAll(LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// ISO8601 timestamp e.g., 2023-10-25T15-30-45Z
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	logFileName := fmt.Sprintf("tldcrawler_log_%s.log", strings.ReplaceAll(timestamp, ":", "-"))
	logFilePath := filepath.Join(LogDir, logFileName)

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logFilePath, "stdout"}

	logger, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}
	return logger, nil
}

// runRaylibVisualization sets up a Raylib window with swirling particles
func (app *Application) runRaylibVisualization() {
	// Initialize Raylib (Assuming Raylib is required; if not, this can be removed)
	// rl.InitWindow(WindowWidth, WindowHeight, "TLD Crawler Visualization - PQ Secure")
	// rl.SetTargetFPS(60)
	// runtime.LockOSThread()

	// app.Particles = generateParticles(MaxParticles)

	// for !rl.WindowShouldClose() {
	// 	rl.BeginDrawing()
	// 	rl.ClearBackground(rl.DarkGray)

	// 	updateParticles(app.Particles)

	// 	rl.DrawText("TLD Crawler Running...", 20, 40, FontSize, rl.RayWhite)
	// 	currentTime := time.Now().Format("15:04:05")
	// 	rl.DrawText(fmt.Sprintf("Current Time: %s", currentTime), 20, 80, FontSize, rl.RayWhite)
	// 	rl.DrawText("Press ESC to Exit", 20, 120, FontSize, rl.Red)

	// 	if rl.IsKeyPressed(rl.KeyEscape) {
	// 		// Signal graceful shutdown
	// 		app.Shutdown()
	// 		break
	// 	}

	// 	rl.EndDrawing()
	// }
	// rl.CloseWindow()
	// app.Logger.Info("Raylib visualization closed")
}

// FindOpenPorts identifies open ports within the specified range and protocol.
// It returns a sorted list of PortStatus, prioritizing insecure ports first.
func (app *Application) FindOpenPorts(ctx context.Context, startPort, endPort int, protocol string) ([]PortStatus, error) {
	app.Logger.Info("Starting port scan",
		zap.Int("startPort", startPort),
		zap.Int("endPort", endPort),
		zap.String("protocol", protocol),
	)

	var (
		openPorts   []PortStatus
		mu          sync.Mutex
		wg          sync.WaitGroup
		portChan    = make(chan int, MaxConcurrency)
		resultsChan = make(chan PortStatus, MaxConcurrency)
	)

	// Worker function to scan ports
	worker := func() {
		defer wg.Done()
		for port := range portChan {
			select {
			case <-ctx.Done():
				return
			default:
				address := fmt.Sprintf("127.0.0.1:%d", port)
				conn, err := net.DialTimeout(protocol, address, 500*time.Millisecond)
				if err == nil {
					conn.Close()
					isInsecure := InsecurePorts[port]
					portStatus := PortStatus{
						Port:       port,
						Protocol:   protocol,
						IsOpen:     true,
						IsInsecure: isInsecure,
					}
					resultsChan <- portStatus
					app.Logger.Debug("Port is open",
						zap.Int("port", port),
						zap.String("protocol", protocol),
						zap.Bool("isInsecure", isInsecure),
					)
				} else {
					app.Logger.Debug("Port is closed",
						zap.Int("port", port),
						zap.String("protocol", protocol),
						zap.Error(err),
					)
				}
			}
		}
	}

	// Start worker pool
	for i := 0; i < MaxConcurrency; i++ {
		wg.Add(1)
		go worker()
	}

	// Send ports to be scanned
	go func() {
		for port := startPort; port <= endPort; port++ {
			select {
			case <-ctx.Done():
				break
			case portChan <- port:
			}
		}
		close(portChan)
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for portStatus := range resultsChan {
		mu.Lock()
		openPorts = append(openPorts, portStatus)
		mu.Unlock()
	}

	// Sort open ports: insecure ports first, then ascending order
	sort.Slice(openPorts, func(i, j int) bool {
		if openPorts[i].IsInsecure && !openPorts[j].IsInsecure {
			return true
		}
		if openPorts[j].IsInsecure && !openPorts[i].IsInsecure {
			return false
		}
		return openPorts[i].Port < openPorts[j].Port
	})

	app.Logger.Info("Port scan completed", zap.Int("openPortsFound", len(openPorts)))

	return openPorts, nil
}

// EnumerateDomains performs native subdomain enumeration within the specified base domain.
// It uses a simple wordlist for demonstration purposes.
func (app *Application) EnumerateDomains(ctx context.Context, baseDomain string, wordlist []string) error {
	app.Logger.Info("Starting domain enumeration", zap.String("baseDomain", baseDomain))

	var wg sync.WaitGroup
	domainChan := make(chan string, MaxConcurrency)
	resultsChan := make(chan string, MaxConcurrency)

	// Worker function to perform DNS lookup
	worker := func() {
		defer wg.Done()
		for subdomain := range domainChan {
			select {
			case <-ctx.Done():
				return
			default:
				fqdn := fmt.Sprintf("%s.%s", subdomain, baseDomain)
				_, err := net.LookupHost(fqdn)
				if err == nil {
					resultsChan <- fqdn
					app.Logger.Info("Subdomain found", zap.String("subdomain", fqdn))
				} else {
					app.Logger.Debug("Subdomain not found", zap.String("subdomain", fqdn), zap.Error(err))
				}
			}
		}
	}

	// Start worker pool
	for i := 0; i < MaxConcurrency; i++ {
		wg.Add(1)
		go worker()
	}

	// Send subdomains to be scanned
	go func() {
		for _, word := range wordlist {
			select {
			case <-ctx.Done():
				break
			case domainChan <- word:
			}
		}
		close(domainChan)
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for fqdn := range resultsChan {
		app.EnumeratedMux.Lock()
		app.Enumerated[fqdn] = true
		app.EnumeratedMux.Unlock()
	}

	app.Logger.Info("Domain enumeration completed", zap.Int("domainsFound", len(app.Enumerated)))

	return nil
}

// generateWordlist generates a simple list of subdomains for enumeration.
// In practice, use a comprehensive wordlist or integrate with a subdomain enumeration tool.
func generateWordlist() []string {
	return []string{
		"www",
		"mail",
		"ftp",
		"test",
		"dev",
		"admin",
		"ns1",
		"ns2",
		"api",
		"blog",
		"shop",
		"secure",
	}
}

// DisplayOpenPorts prints the open ports, listing insecure ports first.
func DisplayOpenPorts(openPorts []PortStatus) {
	fmt.Println("Open Ports:")
	fmt.Println("------------")
	fmt.Printf("%-10s %-10s %-10s\n", "Port", "Protocol", "Insecure")
	fmt.Printf("%-10s %-10s %-10s\n", "----", "--------", "---------")
	for _, port := range openPorts {
		insecure := "No"
		if port.IsInsecure {
			insecure = "Yes"
		}
		fmt.Printf("%-10d %-10s %-10s\n", port.Port, port.Protocol, insecure)
	}
}

// GenerateReports creates CSV and PDF reports of the enumerated domains.
func (app *Application) GenerateReports() error {
	app.Logger.Info("Generating reports")

	// Ensure report directory exists
	if err := os.MkdirAll(ReportDir, 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	timestamp := time.Now().Format("20060102T150405Z")
	csvFilePath := filepath.Join(ReportDir, fmt.Sprintf("tldcrawler_report_%s.csv", timestamp))
	pdfFilePath := filepath.Join(ReportDir, fmt.Sprintf("tldcrawler_report_%s.pdf", timestamp))

	// Write CSV
	csvFile, err := os.Create(csvFilePath)
	if err != nil {
		return fmt.Errorf("failed to create CSV report: %w", err)
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"Domain"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write domains
	app.EnumeratedMux.Lock()
	for domain := range app.Enumerated {
		if err := writer.Write([]string{domain}); err != nil {
			app.Logger.Error("Failed to write domain to CSV", zap.String("domain", domain), zap.Error(err))
		}
	}
	app.EnumeratedMux.Unlock()

	app.Logger.Info("CSV report generated", zap.String("file", csvFilePath))

	// Write PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "TLD Crawler Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	app.EnumeratedMux.Lock()
	for domain := range app.Enumerated {
		pdf.Cell(40, 8, domain)
		pdf.Ln(8)
	}
	app.EnumeratedMux.Unlock()

	if err := pdf.OutputFileAndClose(pdfFilePath); err != nil {
		return fmt.Errorf("failed to write PDF report: %w", err)
	}

	app.Logger.Info("PDF report generated", zap.String("file", pdfFilePath))
	return nil
}

// Shutdown gracefully shuts down the application, ensuring all logs are flushed.
func (app *Application) Shutdown() {
	app.Logger.Info("Shutting down application...")
	_ = app.Logger.Sync()
	os.Exit(0)
}

// main is the entry point of the application.
func main() {
	// Initialize Application
	app, err := NewApplication()
	if err != nil {
		fmt.Printf("Error initializing application: %v\n", err)
		os.Exit(1)
	}
	defer app.Shutdown()

	// Handle graceful shutdown
	signal.Notify(app.ShutdownChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-app.ShutdownChan
		app.Logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
		app.Shutdown()
	}()

	// Parse command-line arguments
	baseDomain := flag.String("domain", "", "Base domain to enumerate subdomains for (e.g., example.com)")
	startPort := flag.Int("start-port", 1, "Start port for port scanning")
	endPort := flag.Int("end-port", 1024, "End port for port scanning")
	protocol := flag.String("protocol", "tcp", "Protocol for port scanning (tcp/udp)")
	flag.Parse()

	if *baseDomain == "" {
		app.Logger.Error("Base domain is required")
		fmt.Println("Usage: tldcrawler -domain example.com")
		os.Exit(1)
	}

	// Generate wordlist for subdomain enumeration
	wordlist := generateWordlist()

	// Create a context that is canceled on shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start domain enumeration
	go func() {
		if err := app.EnumerateDomains(ctx, *baseDomain, wordlist); err != nil {
			app.Logger.Error("Error during domain enumeration", zap.Error(err))
		}
	}()

	// Start port scanning after enumeration is done
	// For simplicity, wait for enumeration to finish (could be optimized)
	time.Sleep(2 * time.Second) // Replace with proper synchronization

	openPorts, err := app.FindOpenPorts(ctx, *startPort, *endPort, *protocol)
	if err != nil {
		app.Logger.Error("Error scanning ports", zap.Error(err))
	}

	// Display open ports
	DisplayOpenPorts(openPorts)

	// Generate reports
	if err := app.GenerateReports(); err != nil {
		app.Logger.Error("Failed to generate reports", zap.Error(err))
	}

	app.Logger.Info("TLD Crawler application completed successfully")
}
