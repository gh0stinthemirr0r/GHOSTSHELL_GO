package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"ghostshell/ui"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	logDir    = "ghostshell/logging"
	reportDir = "ghostshell/reporting"
)

var logger *zap.Logger

func setupLogger() error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(logDir, fmt.Sprintf("vpn_log_%s.log", timestamp))

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

type VPNManager struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex

	currentVPNProcess *exec.Cmd
	currentVPNConfig  string
	connections       []string

	totalAttempts prometheus.Counter
	successConn   prometheus.Counter
	failedConn    prometheus.Counter
	vpnDuration   prometheus.Histogram
	vpnActive     prometheus.Gauge

	wg sync.WaitGroup
}

func newVPNManager() *VPNManager {
	ctx, cancel := context.WithCancel(context.Background())
	vm := &VPNManager{
		ctx:         ctx,
		cancel:      cancel,
		connections: []string{},
	}
	vm.initMetrics()
	return vm
}

func (vm *VPNManager) initMetrics() {
	vm.totalAttempts = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "vpn_total_attempts",
		Help: "Total VPN connection attempts",
	})
	vm.successConn = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "vpn_successful_connections",
		Help: "Successful VPN connections",
	})
	vm.failedConn = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "vpn_failed_connections",
		Help: "Failed VPN connections",
	})
	vm.vpnDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "vpn_connection_duration_seconds",
		Help:    "Duration of VPN connections in seconds",
		Buckets: prometheus.DefBuckets,
	})
	vm.vpnActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vpn_active",
		Help: "Indicates if a VPN connection is active (1) or not (0)",
	})

	prometheus.MustRegister(vm.totalAttempts, vm.successConn, vm.failedConn, vm.vpnDuration, vm.vpnActive)
}

func (vm *VPNManager) StartVPN(configPath string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if vm.currentVPNProcess != nil {
		return fmt.Errorf("VPN already running with config: %s", vm.currentVPNConfig)
	}

	vm.totalAttempts.Inc()

	cmd := exec.CommandContext(vm.ctx, "openvpn", "--config", configPath)
	err := cmd.Start()
	if err != nil {
		vm.failedConn.Inc()
		return fmt.Errorf("failed to start VPN process: %w", err)
	}

	vm.currentVPNProcess = cmd
	vm.currentVPNConfig = configPath
	vm.connections = append(vm.connections, configPath)
	vm.vpnActive.Set(1)

	vm.wg.Add(1)
	go vm.monitorVPN(cmd, time.Now())
	return nil
}

func (vm *VPNManager) monitorVPN(cmd *exec.Cmd, startTime time.Time) {
	defer vm.wg.Done()
	err := cmd.Wait()
	dur := time.Since(startTime).Seconds()

	vm.mu.Lock()
	defer vm.mu.Unlock()

	vm.vpnDuration.Observe(dur)
	if err == nil {
		vm.successConn.Inc()
		logger.Info("VPN process exited successfully", zap.String("config", vm.currentVPNConfig))
	} else {
		vm.failedConn.Inc()
		logger.Warn("VPN process error", zap.String("config", vm.currentVPNConfig), zap.Error(err))
	}
	vm.currentVPNProcess = nil
	vm.currentVPNConfig = ""
	vm.vpnActive.Set(0)
}

func (vm *VPNManager) Stop() {
	vm.cancel()
	if vm.currentVPNProcess != nil {
		_ = vm.currentVPNProcess.Process.Kill()
	}
	vm.wg.Wait()
}

func main() {
	if err := setupLogger(); err != nil {
		fmt.Printf("Logger setup error: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	vm := newVPNManager()
	go startPrometheus()

	rl.InitWindow(1280, 720, "VPN Manager")
	rl.SetTargetFPS(60)

	uim := ui.NewVPNManagerUI(nil, rl.GetFontDefault(), 20, 20, 1240, 680)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	for !rl.WindowShouldClose() {
		select {
		case <-sigChan:
			logger.Info("Received signal, shutting down")
			vm.Stop()
			rl.CloseWindow()
			return
		default:
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.DarkGray)
		uim.Update()
		uim.Draw()
		rl.EndDrawing()
	}

	vm.Stop()
	logger.Info("Application shutdown gracefully")
}

func startPrometheus() {
	http.Handle("/metrics", promhttp.Handler())
	logger.Info("Prometheus metrics available on :8080/metrics")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Warn("Prometheus server ended", zap.Error(err))
	}
}
