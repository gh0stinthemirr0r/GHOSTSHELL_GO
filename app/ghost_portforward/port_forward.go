package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/csv"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/jung-kurt/gofpdf"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/rand"

	// Hypothetical local module containing quantum-safe logic
	"yourproject/oqs_network"
)

// -------------- Constants --------------

const (
	LogDir       = "ghostshell/logging"
	ReportDir    = "ghostshell/reporting"
	WindowWidth  = 1280
	WindowHeight = 720
	FontSize     = 24
	MaxParticles = 50
)

// -------------- Logging --------------

var logger *zap.Logger

func setupLogger() error {
	if err := os.MkdirAll(LogDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(LogDir, fmt.Sprintf("portforward_log_%s.log", timestamp))

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logFile, "stdout"}

	var err error
	logger, err = cfg.Build()
	if err != nil {
		return fmt.Errorf("failed to build logger: %v", err)
	}
	return nil
}

// -------------- Raylib Particles --------------

type Particle struct {
	x, y   float32
	dx, dy float32
	color  rl.Color
}

func generateParticles(count int) []*Particle {
	rand.Seed(time.Now().UnixNano())
	ps := make([]*Particle, count)
	for i := 0; i < count; i++ {
		ps[i] = &Particle{
			x:  float32(rand.Intn(WindowWidth)),
			y:  float32(rand.Intn(WindowHeight)),
			dx: (rand.Float32()*2 - 1) * 2,
			dy: (rand.Float32()*2 - 1) * 2,
			color: rl.NewColor(
				uint8(rand.Intn(256)),
				uint8(rand.Intn(256)),
				uint8(rand.Intn(256)),
				255,
			),
		}
	}
	return ps
}

func updateParticles(ps []*Particle) {
	for _, p := range ps {
		p.x += p.dx
		p.y += p.dy

		if p.x < 0 || p.x > float32(WindowWidth) {
			p.dx *= -1
		}
		if p.y < 0 || p.y > float32(WindowHeight) {
			p.dy *= -1
		}
		rl.DrawCircle(int32(p.x), int32(p.y), 4, p.color)
	}
}

// -------------- UI Terminal --------------

type Terminal struct {
	font rl.Font
}

// newTerminal loads a custom TTF or fallback
func newTerminal() (*Terminal, error) {
	fontPath := "resources/futuristic_font.ttf"
	font := rl.LoadFontEx(fontPath, FontSize, nil, 0)
	if font.BaseSize == 0 {
		logger.Warn("Failed to load custom font, fallback to default")
		font = rl.GetFontDefault()
	}
	return &Terminal{font: font}, nil
}

// -------------- PortForwarder --------------

type PortForwarder struct {
	sourceAddr      string
	destinationAddr string

	// Post quantum quantum-safe network
	oqsNet *oqs_network.OQSNetwork

	// concurrency
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// metrics
	totalConnections  prometheus.Counter
	activeConnections prometheus.Gauge
	dataTransferred   prometheus.Counter
	mu                sync.Mutex // guard for connections tracking
	connections       []string   // track established connections
	bytesTransferred  float64
}

// NewPortForwarder sets up a quantum-safe port forwarder
func NewPortForwarder(src, dst string, net *oqs_network.OQSNetwork) *PortForwarder {
	pf := &PortForwarder{
		sourceAddr:      src,
		destinationAddr: dst,
		oqsNet:          net,
		connections:     []string{},
	}
	pf.ctx, pf.cancel = context.WithCancel(context.Background())
	pf.initPrometheus()
	return pf
}

func (pf *PortForwarder) initPrometheus() {
	pf.totalConnections = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "port_forwarder_total_connections",
		Help: "Total number of port forwarding connections established.",
	})
	pf.activeConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "port_forwarder_active_connections",
		Help: "Current number of active port forwarding connections.",
	})
	pf.dataTransferred = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "port_forwarder_data_transferred_bytes",
		Help: "Total amount of data transferred through port forwarding in bytes.",
	})

	prometheus.MustRegister(pf.totalConnections, pf.activeConnections, pf.dataTransferred)
}

func (pf *PortForwarder) Start() error {
	// Listen on source
	ln, err := net.Listen("tcp", pf.sourceAddr)
	if err != nil {
		logger.Error("Failed to listen on source", zap.String("address", pf.sourceAddr), zap.Error(err))
		return err
	}
	logger.Info("Listening on source address", zap.String("address", pf.sourceAddr))
	pf.wg.Add(1)
	go pf.acceptLoop(ln)
	return nil
}

func (pf *PortForwarder) acceptLoop(ln net.Listener) {
	defer pf.wg.Done()
	for {
		select {
		case <-pf.ctx.Done():
			logger.Info("Stop accepting new connections (context canceled)")
			ln.Close()
			return
		default:
		}
		ln.SetDeadline(time.Now().Add(1 * time.Second)) // short accept timeout
		conn, err := ln.Accept()
		if err != nil {
			// if it's a timeout, continue
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			logger.Warn("Accept error", zap.Error(err))
			continue
		}
		pf.wg.Add(1)
		pf.totalConnections.Inc()
		pf.activeConnections.Inc()

		// add to connections
		pf.mu.Lock()
		pf.connections = append(pf.connections, conn.RemoteAddr().String())
		pf.mu.Unlock()

		// handle in goroutine
		go pf.handleConnection(conn)
	}
}

func (pf *PortForwarder) handleConnection(srcConn net.Conn) {
	defer pf.wg.Done()
	defer pf.activeConnections.Dec()

	// create a quantum-safe connection to the destination
	// The OQSNetwork uses e.g. protocol "tcp"
	dstConn, err := pf.oqsNet.Connect(pf.destinationAddr, "tcp")
	if err != nil {
		logger.Error("Failed to connect to destination with OQS", zap.String("destination", pf.destinationAddr), zap.Error(err))
		srcConn.Close()
		return
	}

	netDstConn, ok := dstConn.(net.Conn)
	if !ok {
		logger.Error("Destination connection is not a net.Conn type, ignoring", zap.String("destination", pf.destinationAddr))
		srcConn.Close()
		return
	}

	// concurrency: forward data from src -> dst and from dst -> src
	var wgLocal sync.WaitGroup
	wgLocal.Add(2)

	// forward src -> dst
	go pf.forwardTraffic(srcConn, netDstConn, &wgLocal)
	// forward dst -> src
	go pf.forwardTraffic(netDstConn, srcConn, &wgLocal)

	wgLocal.Wait()
	netDstConn.Close()
	srcConn.Close()
	logger.Info("Connection closed gracefully", zap.String("remote", srcConn.RemoteAddr().String()))
}

// forwardTraffic copies data from inConn to outConn, tracking data transferred
func (pf *PortForwarder) forwardTraffic(inConn net.Conn, outConn net.Conn, wgLocal *sync.WaitGroup) {
	defer wgLocal.Done()
	buffer := make([]byte, 4096)

	for {
		inConn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, err := inConn.Read(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// check for context done
				select {
				case <-pf.ctx.Done():
					return
				default:
					continue // read again
				}
			}
			// some other error or EOF
			return
		}
		if n == 0 {
			return
		}
		w, werr := outConn.Write(buffer[:n])
		if werr != nil {
			logger.Warn("Write error", zap.Error(werr))
			return
		}
		pf.mu.Lock()
		pf.bytesTransferred += float64(w)
		pf.mu.Unlock()
		pf.dataTransferred.Add(float64(w))
	}
}

// Stop signals the forwarder to stop accepting connections, closes existing connections gracefully
func (pf *PortForwarder) Stop() {
	pf.cancel()
	pf.wg.Wait()
}

// -------------- Raylib UI & Main --------------

func main() {
	if err := setupLogger(); err != nil {
		fmt.Printf("Failed to setup logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// if we want to expose Prometheus metrics
	// We'll host them on :8080/metrics
	go func() {
		if err := startPrometheus(); err != nil {
			logger.Warn("Prometheus server error", zap.Error(err))
		}
	}()

	// init OQS network (hypothetical usage)
	certMgr := &MyCertManager{} // you would define this
	oqsNet, err := oqs_network.NewOQSNetwork(certMgr)
	if err != nil {
		logger.Fatal("Failed to init OQS network", zap.Error(err))
	}

	// create the forwarder
	pf := NewPortForwarder("127.0.0.1:9000", "10.0.0.1:80", oqsNet)
	if err := pf.Start(); err != nil {
		logger.Fatal("Failed to start port forwarder", zap.Error(err))
	}

	// init Raylib
	rl.InitWindow(WindowWidth, WindowHeight, "Port Forwarder - OQS Secured")
	rl.SetTargetFPS(60)
	runtime.LockOSThread()

	t, err := newTerminal()
	if err != nil {
		logger.Fatal("Failed to init terminal UI", zap.Error(err))
	}
	defer t.Shutdown()

	// create particles
	particles := generateParticles(MaxParticles)

	// handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// main loop
	for !rl.WindowShouldClose() {
		select {
		case <-sigChan:
			logger.Info("Received shutdown signal")
			pf.Stop()
			goto cleanup
		default:
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.DarkBlue)

		updateParticles(particles)

		// draw text
		rl.DrawTextEx(t.font, "Port Forwarder (Post-Quantum)", rl.NewVector2(40, 40), float32(FontSize), 2, rl.White)
		localTime := time.Now().Format("15:04:05")
		rl.DrawTextEx(t.font, fmt.Sprintf("Local Time: %s", localTime), rl.NewVector2(40, 80), 20, 2, rl.White)
		rl.DrawTextEx(t.font, "Press ESC to Exit", rl.NewVector2(40, 110), 20, 2, rl.LightGray)

		if rl.IsKeyPressed(rl.KeyEscape) {
			pf.Stop()
			goto cleanup
		}

		rl.EndDrawing()
	}

cleanup:
	// after we exit the loop
	pf.Stop()
	// generate CSV/PDF
	pf.mu.Lock()
	connections := append([]string{}, pf.connections...)
	bytesTransferred := pf.bytesTransferred
	pf.mu.Unlock()

	if err := generateReports(connections, bytesTransferred); err != nil {
		logger.Error("Failed to generate reports", zap.Error(err))
	}

	rl.CloseWindow()
	logger.Info("Application shutting down gracefully")
}

// startPrometheus is a helper to host the metrics on :8080
func startPrometheus() error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(":8080", nil)
}

// generateReports writes the CSV & PDF
func generateReports(connections []string, bytesTransferred float64) error {
	if err := os.MkdirAll(ReportDir, 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %v", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	csvFile := filepath.Join(ReportDir, fmt.Sprintf("portforward_report_%s.csv", timestamp))
	pdfFile := filepath.Join(ReportDir, fmt.Sprintf("portforward_report_%s.pdf", timestamp))

	// CSV
	f, err := os.Create(csvFile)
	if err != nil {
		logger.Error("Failed to create CSV file", zap.Error(err))
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"Connection", "BytesTransferred"}); err != nil {
		return err
	}
	for _, c := range connections {
		if err := w.Write([]string{c, fmt.Sprintf("%.2f", bytesTransferred)}); err != nil {
			return err
		}
	}
	logger.Info("CSV report generated", zap.String("file", csvFile))

	// PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "Port Forwarder Report (Quantum-Safe)")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	for _, c := range connections {
		pdf.Cell(0, 8, fmt.Sprintf("Connection: %s", c))
		pdf.Ln(8)
	}
	pdf.Cell(0, 8, fmt.Sprintf("Total bytes transferred: %.2f", bytesTransferred))
	pdf.Ln(8)

	if err := pdf.OutputFileAndClose(pdfFile); err != nil {
		logger.Error("Failed to write PDF file", zap.Error(err))
		return err
	}
	logger.Info("PDF report generated", zap.String("file", pdfFile))
	return nil
}

// MyCertManager is a placeholder CertManager for the OQSNetwork. Implement as needed.
type MyCertManager struct{}

func (mgr *MyCertManager) LoadClientCert() (tls.Certificate, error) {
	// stub
	return tls.Certificate{}, nil
}

func (mgr *MyCertManager) LoadRootCAs() (*x509.CertPool, error) {
	// stub
	return nil, nil
}
