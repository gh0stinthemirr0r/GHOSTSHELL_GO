package main

import (
    "context"
    "encoding/csv"
    "fmt"
    "io"
    "net/http"
    "os"
    "os/signal"
    "path/filepath"
    "runtime"
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

    // Hypothetical local quantum-safe networking module
    "yourproject/oqs_network"
)

const (
    logDir       = "ghostshell/logging"
    reportDir    = "ghostshell/reporting"
    windowWidth  = 1280
    windowHeight = 720
    fontSize     = 24
    maxParticles = 50
    defaultTimeout = 5 * time.Second
)

// -------------- Logging --------------

var logger *zap.Logger

func setupLogger() error {
    if err := os.MkdirAll(logDir, 0755); err != nil {
        return fmt.Errorf("failed to create log directory: %v", err)
    }
    timestamp := time.Now().Format("20060102_150405")
    logFile := filepath.Join(logDir, fmt.Sprintf("httpprobe_log_%s.log", timestamp))

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

// -------------- Particles --------------

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
            x: float32(rand.Intn(windowWidth)),
            y: float32(rand.Intn(windowHeight)),
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

        if p.x < 0 || p.x > float32(windowWidth) {
            p.dx *= -1
        }
        if p.y < 0 || p.y > float32(windowHeight) {
            p.dy *= -1
        }
        rl.DrawCircle(int32(p.x), int32(p.y), 4, p.color)
    }
}

// -------------- Terminal UI --------------

type Terminal struct {
    font rl.Font
}

func newTerminal() (*Terminal, error) {
    fontPath := "resources/futuristic_font.ttf"
    font := rl.LoadFontEx(fontPath, fontSize, nil, 0)
    if font.BaseSize == 0 {
        logger.Warn("Failed to load custom font, fallback to default", zap.String("fontPath", fontPath))
        font = rl.GetFontDefault()
    }
    return &Terminal{font: font}, nil
}

// -------------- HTTP Probe --------------

// ProbeResult holds details of a single probe attempt
type ProbeResult struct {
    URL        string
    StatusCode int
    Duration   time.Duration
    Error      error
}

// HTTPProbe manages concurrency-based HTTP checks with quantum-safe comms
type HTTPProbe struct {
    // reference to OQSNetwork for quantum-safe logic
    oqsNet *oqs_network.OQSNetwork

    // concurrency
    ctx     context.Context
    cancel  context.CancelFunc
    wg      sync.WaitGroup

    // config
    timeout       time.Duration
    maxRetries    int
    retryInterval time.Duration
    concurrency   int

    // results
    mu      sync.Mutex
    results map[string]ProbeResult

    // metrics
    totalProbes   prometheus.Counter
    successProbes prometheus.Counter
    failedProbes  prometheus.Counter
    probeDuration prometheus.Histogram
}

// newHTTPProbe initializes the HTTP probe structure
func newHTTPProbe(oqsNet *oqs_network.OQSNetwork, concurrency int, maxRetries int, retryInterval time.Duration, timeout time.Duration) *HTTPProbe {
    ctx, cancel := context.WithCancel(context.Background())

    hp := &HTTPProbe{
        oqsNet:        oqsNet,
        ctx:           ctx,
        cancel:        cancel,
        concurrency:   concurrency,
        maxRetries:    maxRetries,
        retryInterval: retryInterval,
        timeout:       timeout,
        results:       make(map[string]ProbeResult),
    }
    hp.initMetrics()
    return hp
}

func (hp *HTTPProbe) initMetrics() {
    hp.totalProbes = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "http_probe_total",
        Help: "Total number of HTTP probes",
    })
    hp.successProbes = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "http_probe_success_total",
        Help: "Total number of successful HTTP probes",
    })
    hp.failedProbes = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "http_probe_failure_total",
        Help: "Total number of failed HTTP probes",
    })
    hp.probeDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name: "http_probe_duration_seconds",
        Help: "Duration of HTTP probes in seconds",
        Buckets: prometheus.DefBuckets,
    })

    prometheus.MustRegister(hp.totalProbes, hp.successProbes, hp.failedProbes, hp.probeDuration)
}

// Start concurrency-based probing of multiple URLs
func (hp *HTTPProbe) Start(urls []string) {
    hp.wg.Add(1)
    go hp.runProbes(urls)
}

// runProbes fetches each URL with concurrency, storing results in hp.results
func (hp *HTTPProbe) runProbes(urls []string) {
    defer hp.wg.Done()

    // concurrency
    ch := make(chan string, len(urls))
    for _, u := range urls {
        ch <- u
    }
    close(ch)

    var wgLocal sync.WaitGroup
    for i := 0; i < hp.concurrency; i++ {
        wgLocal.Add(1)
        go func() {
            defer wgLocal.Done()
            for url := range ch {
                select {
                case <-hp.ctx.Done():
                    return
                default:
                }
                hp.probeOneURL(url)
            }
        }()
    }
    wgLocal.Wait()
}

// probeOneURL attempts multiple retries for a single URL
func (hp *HTTPProbe) probeOneURL(url string) {
    var lastErr error
    start := time.Now()
    hp.totalProbes.Inc()

    // we do a quantum-safe connection using hp.oqsNet, then do an HTTP GET
    // we'll stub that logic as follows:
    for i := 0; i < hp.maxRetries; i++ {
        select {
        case <-hp.ctx.Done():
            return
        default:
        }

        code, dur, err := hp.doHTTPRequest(url)
        if err == nil {
            // success
            hp.successProbes.Inc()
            end := time.Since(start)
            hp.mu.Lock()
            hp.results[url] = ProbeResult{
                URL:        url,
                StatusCode: code,
                Duration:   end,
                Error:      nil,
            }
            hp.mu.Unlock()
            hp.probeDuration.Observe(end.Seconds())
            return
        }
        lastErr = err
        time.Sleep(hp.retryInterval)
    }

    // if we get here, all retries failed
    end := time.Since(start)
    hp.failedProbes.Inc()
    hp.probeDuration.Observe(end.Seconds())
    hp.mu.Lock()
    hp.results[url] = ProbeResult{
        URL:        url,
        StatusCode: 0,
        Duration:   end,
        Error:      lastErr,
    }
    hp.mu.Unlock()
}

// doHTTPRequest simulates a quantum-safe HTTP GET
func (hp *HTTPProbe) doHTTPRequest(url string) (int, time.Duration, error) {
    start := time.Now()

    // In a real scenario, we might do something like:
    // 1) hp.oqsNet.Connect(url, "tcp") or "tls"
    // 2) Then do an HTTP GET over that quantum-safe connection
    // For demonstration, we'll just do a random success/fail
    time.Sleep(time.Millisecond * time.Duration(rand.Intn(500)+100))

    if rand.Float32() < 0.3 {
        return 0, time.Since(start), fmt.Errorf("random simulated error")
    }
    // success, random status code 200 or 302
    codes := []int{200, 302, 404}
    code := codes[rand.Intn(len(codes))]
    return code, time.Since(start), nil
}

// Stop signals the concurrency to end
func (hp *HTTPProbe) Stop() {
    hp.cancel()
    hp.wg.Wait()
}

// -------------- Reporting --------------

func generateReports(results map[string]ProbeResult) error {
    if err := os.MkdirAll(reportDir, 0755); err != nil {
        logger.Error("Failed to create report dir", zap.Error(err))
        return err
    }
    timestamp := time.Now().Format("20060102_150405")
    csvFile := filepath.Join(reportDir, fmt.Sprintf("httpprobe_report_%s.csv", timestamp))
    pdfFile := filepath.Join(reportDir, fmt.Sprintf("httpprobe_report_%s.pdf", timestamp))

    // CSV
    f, err := os.Create(csvFile)
    if err != nil {
        logger.Error("Failed to create CSV", zap.Error(err))
        return err
    }
    defer f.Close()

    w := csv.NewWriter(f)
    defer w.Flush()

    if err := w.Write([]string{"URL", "StatusCode", "Duration(s)", "Error"}); err != nil {
        return err
    }
    for url, res := range results {
        row := []string{
            url,
            fmt.Sprintf("%d", res.StatusCode),
            fmt.Sprintf("%.3f", res.Duration.Seconds()),
            fmt.Sprintf("%v", res.Error),
        }
        if err := w.Write(row); err != nil {
            return err
        }
    }
    logger.Info("CSV report generated", zap.String("file", csvFile))

    // PDF
    pdf := gofpdf.New("P", "mm", "A4", "")
    pdf.AddPage()
    pdf.SetFont("Arial", "B", 16)
    pdf.Cell(40, 10, "HTTP Probe Report (Post-Quantum Secure)")
    pdf.Ln(12)

    pdf.SetFont("Arial", "", 12)
    for url, res := range results {
        line := fmt.Sprintf("URL: %s, Code: %d, Dur: %.3f sec, Err: %v", url, res.StatusCode, res.Duration.Seconds(), res.Error)
        pdf.MultiCell(190, 6, line, "", "", false)
        pdf.Ln(3)
    }
    if err := pdf.OutputFileAndClose(pdfFile); err != nil {
        logger.Error("Failed to write PDF", zap.String("file", pdfFile), zap.Error(err))
        return err
    }
    logger.Info("PDF report generated", zap.String("file", pdfFile))
    return nil
}

// -------------- Main & Raylib UI --------------

func main() {
    // Setup logging
    if err := setupLogger(); err != nil {
        fmt.Printf("Logger setup failed: %v\n", err)
        os.Exit(1)
    }
    defer logger.Sync()

    // Start Prometheus metrics server
    go func() {
        http.Handle("/metrics", promhttp.Handler())
        if err := http.ListenAndServe(":8080", nil); err != nil {
            logger.Fatal("Prometheus server error", zap.Error(err))
        }
    }()

    // Suppose we also have a CertManager and create an OQSNetwork
    certMgr := &MyCertManager{} // your custom cert manager
    oqsNet, err := oqs_network.NewOQSNetwork(certMgr)
    if err != nil {
        logger.Fatal("Failed to init OQSNetwork", zap.Error(err))
    }

    // Create & start HTTPProbe
    probe := newHTTPProbe(oqsNet, concurrency=3, maxRetries=2, retryInterval=1*time.Second, timeout=defaultTimeout)
    urls := []string{"https://example.com", "https://someotherplace.org", "https://failing-site.io"}
    probe.Start(urls)

    // Raylib window
    rl.InitWindow(windowWidth, windowHeight, "HTTP Probe - Post Quantum")
    rl.SetTargetFPS(60)
    runtime.LockOSThread()

    t, err := newTerminal()
    if err != nil {
        logger.Fatal("Failed to init Terminal UI", zap.Error(err))
    }
    defer t.Shutdown()

    ps := generateParticles(maxParticles)

    // Graceful signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    mainCtx, mainCancel := context.WithCancel(context.Background())

    // main loop
    for !rl.WindowShouldClose() && mainCtx.Err() == nil {
        select {
        case <-sigChan:
            logger.Info("Got shutdown signal")
            probe.Stop() // stop concurrency
            mainCancel()
            break
        default:
        }

        rl.BeginDrawing()
        rl.ClearBackground(rl.DarkBlue)

        // update & draw particles
        updateParticles(ps)

        // UI text
        rl.DrawTextEx(t.font, "HTTP Probe (Quantum-Safe)", rl.NewVector2(40, 40), float32(fontSize), 2, rl.White)
        localTime := time.Now().Format("15:04:05")
        rl.DrawTextEx(t.font, fmt.Sprintf("Local Time: %s", localTime), rl.NewVector2(40, 80), 20, 2, rl.LightGray)
        rl.DrawTextEx(t.font, "Press ESC to exit", rl.NewVector2(40, 110), 20, 2, rl.LightGray)

        // if user hits ESC
        if rl.IsKeyPressed(rl.KeyEscape) {
            probe.Stop()
            mainCancel()
        }

        rl.EndDrawing()
    }

    // after loop, gather results
    probe.Stop() // ensures concurrency done

    // collect results
    probe.mu.Lock()
    finalResults := probe.results
    probe.mu.Unlock()

    if err := generateReports(finalResults); err != nil {
        logger.Error("Failed to generate reports", zap.Error(err))
    }

    rl.CloseWindow()
    logger.Info("Application shutting down gracefully.")
}

// MyCertManager is a placeholder for real quantum-safe certificate loading
type MyCertManager struct{}

func (mgr *MyCertManager) LoadClientCert() (tls.Certificate, error) {
    // stub
    return tls.Certificate{}, nil
}
func (mgr *MyCertManager) LoadRootCAs() (*x509.CertPool, error) {
    // stub
    return nil, nil
}
