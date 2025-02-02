package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
)

const (
	logDir       = "ghostshell/logging"
	reportDir    = "ghostshell/reporting"
	windowWidth  = 1280
	windowHeight = 720
	fontSize     = 24
	maxParticles = 50
)

// -------------- Prometheus Metrics --------------

var (
	trafficGenerated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "traffic_generated_total",
			Help: "Total number of traffic packets generated.",
		},
		[]string{"destination"},
	)
	enumeratedHosts = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "enumerated_hosts",
			Help: "Number of hosts enumerated in a CIDR or single IP domain.",
		},
		[]string{"destination"},
	)
	packetLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "packet_latency_ms",
			Help:    "Latency of generated packets in milliseconds.",
			Buckets: prometheus.LinearBuckets(10, 10, 10),
		},
		[]string{"destination"},
	)
)

// -------------- Global Vars --------------

var (
	logger      *zap.Logger
	metricsPort string
)

// -------------- Logging Initialization --------------

func setupLogger() error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(logDir, fmt.Sprintf("surveyor_log_%s.log", timestamp))

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

// -------------- Main --------------

func main() {
	// Register custom Prometheus metrics
	prometheus.MustRegister(trafficGenerated, enumeratedHosts, packetLatency)

	// 1) Setup logger
	if err := setupLogger(); err != nil {
		fmt.Printf("Error setting up logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Surveyor starting...")

	// 2) Parse flags or env
	dest := flag.String("dest", "", "Target IP/CIDR to scan (e.g., 192.168.1.0/24 or 8.8.8.8)")
	mp := flag.String("metrics-port", "8080", "Port for Prometheus metrics server")
	flag.Parse()

	if *dest == "" {
		// fallback
		*dest = "192.168.1.0/24"
	}
	metricsPort = *mp
	logger.Info("Surveyor config",
		zap.String("destination", *dest),
		zap.String("metricsPort", metricsPort),
	)

	// 3) Start Prometheus metrics server
	srv := startMetricsServer(metricsPort)

	// 4) Enumerate endpoints: hosts in a single IP or entire CIDR
	hosts, err := enumerateEndpoints(*dest)
	if err != nil {
		logger.Fatal("Error enumerating endpoints", zap.Error(err))
	}
	enumeratedHosts.WithLabelValues(*dest).Set(float64(len(hosts)))

	// 5) Collect additional info: resolve hostnames, OS guess, open ports
	enrichedHosts := collectHostDetails(*dest, hosts)

	// 6) Start concurrency traffic generation
	generateTrafficConcurrently(*dest, enrichedHosts)

	// 7) Launch Raylib interface for visualizing metrics
	go visualizeInterface(*dest, len(enrichedHosts))

	// 8) Generate final PDF/CSV
	if err := generateReport(*dest, enrichedHosts); err != nil {
		logger.Fatal("Failed to generate final report", zap.Error(err))
	}

	logger.Info("Surveying completed. Awaiting shutdown signal...")

	// 9) Wait for OS signals to gracefully shut down
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	s := <-sigChan
	logger.Info("Received shutdown signal", zap.String("signal", s.String()))

	// Gracefully shut down Prometheus server
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Warn("Prometheus server shutdown error", zap.Error(err))
	}

	logger.Info("Surveyor exited cleanly")
}

// -------------- Start the Prometheus server --------------

func startMetricsServer(port string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		logger.Info("Starting metrics server", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Metrics server listen failed", zap.Error(err))
		}
	}()
	return srv
}

// -------------- Enumerate Endpoints --------------

func enumerateEndpoints(destination string) ([]string, error) {
	logger.Info("Enumerating endpoints", zap.String("destination", destination))

	// If it contains "/", we assume it's a CIDR. Otherwise, single IP.
	if strings.Contains(destination, "/") {
		// parse as CIDR
		_, ipnet, err := net.ParseCIDR(destination)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR: %w", err)
		}
		var hosts []string
		for ip := ipnet.IP.Mask(ipnet.Mask); ipnet.Contains(ip); incIP(ip) {
			hosts = append(hosts, ip.String())
		}
		// remove network and broadcast if needed
		if len(hosts) > 2 {
			hosts = hosts[1 : len(hosts)-1]
		}
		logger.Info("CIDR enumeration complete", zap.Int("host_count", len(hosts)))
		return hosts, nil
	} else {
		// single IP or domain
		return []string{destination}, nil
	}
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] != 0 {
			break
		}
	}
}

// -------------- Collect Host Details (Hostname, OS, Open Ports) --------------

func collectHostDetails(destination string, hosts []string) []string {
	logger.Info("Collecting host details", zap.Int("count", len(hosts)))
	var wg sync.WaitGroup
	results := make([]string, len(hosts))
	wg.Add(len(hosts))

	for i, h := range hosts {
		go func(i int, host string) {
			defer wg.Done()
			// do DNS reverse lookup for hostname
			// do OS guess (placeholder)
			// do port scanning (placeholder)
			time.Sleep(50 * time.Millisecond) // simulate

			results[i] = fmt.Sprintf("%s|SomeHostname|Linux|Ports:80,443", host)
		}(i, h)
	}

	wg.Wait()
	logger.Info("Host detail collection complete")
	return results
}

// -------------- Traffic Generation --------------

func generateTrafficConcurrently(destination string, hosts []string) {
	logger.Info("Starting traffic generation", zap.String("destination", destination), zap.Int("host_count", len(hosts)))

	numWorkers := 3
	jobs := make(chan string, len(hosts))
	for _, h := range hosts {
		jobs <- h
	}
	close(jobs)

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(id int) {
			defer wg.Done()
			for host := range jobs {
				generateTrafficForHost(destination, host)
			}
		}(i)
	}
	wg.Wait()
	logger.Info("Traffic generation completed")
}

func generateTrafficForHost(destination, host string) {
	// We'll pretend we send 10 packets
	for i := 0; i < 10; i++ {
		start := time.Now()
		// simulate a ping or request
		time.Sleep(time.Duration(rand.Intn(50)+10) * time.Millisecond)
		latency := float64(time.Since(start).Milliseconds())

		// Update Prometheus
		trafficGenerated.WithLabelValues(destination).Inc()
		packetLatency.WithLabelValues(destination).Observe(latency)

		logger.Debug("Traffic packet sent", zap.String("host", host), zap.Float64("latency_ms", latency))
	}
}

// -------------- Generate Reports --------------

func generateReport(destination string, hosts []string) error {
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		logger.Error("Failed to create report directory", zap.Error(err))
		return err
	}

	timestamp := time.Now().Format("20060102_150405")
	pdfFile := filepath.Join(reportDir, fmt.Sprintf("surveyor_report_%s.pdf", timestamp))
	csvFile := filepath.Join(reportDir, fmt.Sprintf("surveyor_report_%s.csv", timestamp))

	if err := generatePDFReport(pdfFile, destination, hosts); err != nil {
		return err
	}
	if err := generateCSVReport(csvFile, destination, hosts); err != nil {
		return err
	}
	logger.Info("Reports generated", zap.String("pdf", pdfFile), zap.String("csv", csvFile))
	return nil
}

func generatePDFReport(filePath, destination string, hosts []string) error {
	logger.Info("Generating PDF report", zap.String("file", filePath), zap.String("destination", destination))
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Let's use a real PDF library
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "Surveyor Report - Post Quantum")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	pdf.MultiCell(190, 6, fmt.Sprintf("Destination: %s\nHosts found: %d\n", destination, len(hosts)), "", "", false)
	pdf.Ln(5)

	for _, h := range hosts {
		// split into fields if we stored them as "IP|Hostname|OS|Ports..."
		fields := strings.Split(h, "|")
		line := fmt.Sprintf("Host: %s", fields[0])
		if len(fields) > 1 {
			line += fmt.Sprintf(", Hostname: %s", fields[1])
		}
		if len(fields) > 2 {
			line += fmt.Sprintf(", OS: %s", fields[2])
		}
		if len(fields) > 3 {
			line += fmt.Sprintf(", %s", fields[3])
		}
		pdf.MultiCell(190, 6, line, "", "", false)
	}

	return pdf.OutputFileAndClose(filePath)
}

func generateCSVReport(filePath, destination string, hosts []string) error {
	logger.Info("Generating CSV report", zap.String("file", filePath), zap.String("destination", destination))
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// minimal CSV
	f.WriteString("IP,Hostname,OS,OpenPorts\n")
	for _, h := range hosts {
		// parse the "IP|Hostname|OS|Ports" format
		fields := strings.Split(h, "|")
		ip := fields[0]
		hostname, osStr, ports := "", "", ""
		if len(fields) > 1 {
			hostname = fields[1]
		}
		if len(fields) > 2 {
			osStr = fields[2]
		}
		if len(fields) > 3 {
			ports = fields[3]
		}
		line := fmt.Sprintf("%s,%s,%s,%s\n", ip, hostname, osStr, ports)
		f.WriteString(line)
	}
	return nil
}

// -------------- Visual UI --------------

func visualizeInterface(destination string, hostCount int) {
	// We'll do a separate window for metrics
	const wWidth, wHeight = 900, 600
	rl.InitWindow(wWidth, wHeight, "Surveyor Metrics Visualization")
	defer rl.CloseWindow()
	rl.SetTargetFPS(30)

	// swirl of particles
	ps := makeParticles(30)

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)

		// update & draw
		for _, p := range ps {
			p.x += p.dx
			p.y += p.dy
			if p.x < 0 || p.x > float32(wWidth) {
				p.dx *= -1
			}
			if p.y < 0 || p.y > float32(wHeight) {
				p.dy *= -1
			}
			rl.DrawCircle(int32(p.x), int32(p.y), 3, p.color)
		}

		rl.DrawText(fmt.Sprintf("Destination: %s", destination), 20, 20, 20, rl.Black)
		rl.DrawText(fmt.Sprintf("Hosts enumerated: %d", hostCount), 20, 50, 20, rl.Black)
		// read metrics from trafficGenerated ...
		val := getCounterValue(trafficGenerated, destination)
		rl.DrawText(fmt.Sprintf("Traffic Generated: %.0f", val), 20, 80, 20, rl.Blue)

		rl.EndDrawing()
	}
}

type ParticleX struct {
	x, y   float32
	dx, dy float32
	color  rl.Color
}

func makeParticles(count int) []*ParticleX {
	rng := rand.New(rand.NewSource(uint64(time.Now().UnixNano())))
	ps := make([]*ParticleX, count)
	for i := 0; i < count; i++ {
		ps[i] = &ParticleX{
			x:  float32(rng.Intn(900)),
			y:  float32(rng.Intn(600)),
			dx: (rng.Float32()*2 - 1) * 2,
			dy: (rng.Float32()*2 - 1) * 2,
			color: rl.Color{
				R: uint8(rng.Intn(256)),
				G: uint8(rng.Intn(256)),
				B: uint8(rng.Intn(256)),
				A: 255,
			},
		}
	}
	return ps
}

// -------------- read from a CounterVec --------------

func getCounterValue(cv *prometheus.CounterVec, label string) float64 {
	var m []prometheus.Metric
	cv.WithLabelValues(label).Collect(&m)
	if len(m) == 0 {
		return 0
	}
	dto := &prometheus.Metric{}
	m[0].Write(dto)
	if dto.Counter == nil {
		return 0
	}
	return *dto.Counter.Value
}
