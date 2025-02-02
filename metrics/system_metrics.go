package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"go.uber.org/zap"
)

// SystemMetricsConfig holds the main configuration for the metrics collector.
type SystemMetricsConfig struct {
	CollectionInterval time.Duration
	DiskPaths          []string
	EnableNetwork      bool
	Logger             *zap.Logger
	EnableEncryption   bool
	EncryptionKey      []byte // 32 bytes if encryption is used
	PrometheusPort     string

	// Additional AI usage config
	EnableAIModelMetrics bool
	AiModelManager       AiModelManager // a placeholder interface for your manager
}

// AiModelManager is a placeholder interface that your code implements to retrieve usage stats
type AiModelManager interface {
	// GetAiModelUsage returns a slice of usage stats for each loaded AI model.
	// e.g. model name, memory usage (MB), CPU usage (percent).
	GetAiModelUsage() ([]ModelUsage, error)
}

// ModelUsage holds usage data for a single AI model
type ModelUsage struct {
	ModelName  string
	MemoryMB   float64
	CPUPercent float64
}

// SystemMetrics is the main collector object
type SystemMetrics struct {
	config             SystemMetricsConfig
	cpuUsageGaugeVec   *prometheus.GaugeVec
	memUsageGauge      prometheus.Gauge
	diskUsageGaugeVec  *prometheus.GaugeVec
	netIOGaugeVec      *prometheus.GaugeVec
	aiMemUsageGaugeVec *prometheus.GaugeVec
	aiCPUUsageGaugeVec *prometheus.GaugeVec

	stopChan chan struct{}
	wg       sync.WaitGroup
	server   *http.Server
}

// NewSystemMetrics returns a pointer to a new SystemMetrics object with the config.
func NewSystemMetrics(cfg SystemMetricsConfig) (*SystemMetrics, error) {
	if cfg.CollectionInterval <= 0 {
		return nil, errors.New("collection interval must be greater than zero")
	}
	if len(cfg.DiskPaths) == 0 {
		return nil, errors.New("at least one disk path must be specified")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewExample() // fallback logger
	}
	if cfg.EnableEncryption && len(cfg.EncryptionKey) != 32 {
		return nil, errors.New("encryption key must be 32 bytes if encryption is enabled")
	}

	sm := &SystemMetrics{
		config:   cfg,
		stopChan: make(chan struct{}),
	}

	// CPU usage
	sm.cpuUsageGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_cpu_usage_percent",
			Help: "CPU usage percentage per core.",
		},
		[]string{"core"},
	)

	// Memory usage
	sm.memUsageGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_memory_usage_percent",
		Help: "Memory usage percentage.",
	})

	// Disk usage
	sm.diskUsageGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_disk_usage_percent",
			Help: "Disk usage percentage per mount point.",
		},
		[]string{"mountpoint"},
	)

	// Network I/O
	sm.netIOGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_network_io_bytes",
			Help: "Network I/O in bytes per interface/direction.",
		},
		[]string{"interface", "direction"},
	)

	// AI usage (optional)
	if cfg.EnableAIModelMetrics {
		sm.aiMemUsageGaugeVec = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ai_model_memory_usage_mb",
				Help: "Memory usage (MB) per AI model.",
			},
			[]string{"model_name"},
		)
		sm.aiCPUUsageGaugeVec = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ai_model_cpu_usage_percent",
				Help: "CPU usage (percent) per AI model.",
			},
			[]string{"model_name"},
		)
	}

	// Register with Prometheus
	prometheus.MustRegister(
		sm.cpuUsageGaugeVec,
		sm.memUsageGauge,
		sm.diskUsageGaugeVec,
		sm.netIOGaugeVec,
	)
	if cfg.EnableAIModelMetrics {
		prometheus.MustRegister(sm.aiMemUsageGaugeVec, sm.aiCPUUsageGaugeVec)
	}

	return sm, nil
}

// Start sets up the Prometheus server and concurrency for collecting metrics
func (sm *SystemMetrics) Start() error {
	sm.config.Logger.Info("Starting system metrics collection...")

	// Launch Prometheus server
	sm.server = &http.Server{
		Addr:    ":" + sm.config.PrometheusPort,
		Handler: promhttp.Handler(),
	}
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		sm.config.Logger.Info("Starting Prometheus server", zap.String("port", sm.config.PrometheusPort))
		if err := sm.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			sm.config.Logger.Error("Prometheus server error", zap.Error(err))
		}
	}()

	// Launch concurrency for collecting metrics
	sm.wg.Add(1)
	go sm.collectMetrics()

	return nil
}

// Stop signals the collector to stop and shuts down the Prometheus server
func (sm *SystemMetrics) Stop() {
	sm.config.Logger.Info("Stopping system metrics collection")
	close(sm.stopChan)

	// Shut down the server gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if sm.server != nil {
		sm.server.Shutdown(ctx)
	}

	sm.wg.Wait()
	sm.config.Logger.Info("System metrics collection stopped")
}

func (sm *SystemMetrics) collectMetrics() {
	defer sm.wg.Done()
	ticker := time.NewTicker(sm.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sm.stopChan:
			return
		case <-ticker.C:
			sm.collectCPUUsage()
			sm.collectMemoryUsage()
			sm.collectDiskUsage()
			if sm.config.EnableNetwork {
				sm.collectNetworkIO()
			}
			if sm.config.EnableAIModelMetrics && sm.config.AiModelManager != nil {
				sm.collectAiModelUsage()
			}
		}
	}
}

// collectCPUUsage updates the CPU usage gauge vector
func (sm *SystemMetrics) collectCPUUsage() {
	usage, err := cpu.Percent(0, true)
	if err != nil {
		sm.config.Logger.Error("Error collecting CPU usage", zap.Error(err))
		return
	}
	sm.cpuUsageGaugeVec.Reset()
	for i, val := range usage {
		gaugeValue := val
		if sm.config.EnableEncryption {
			gaugeValue = applyEphemeralEncryption(val, sm.config.EncryptionKey, sm.config.Logger)
		}
		label := fmt.Sprintf("core_%d", i)
		sm.cpuUsageGaugeVec.WithLabelValues(label).Set(gaugeValue)
		sm.config.Logger.Debug("CPU usage updated", zap.String("core", label), zap.Float64("usage", val))
	}
}

// collectMemoryUsage updates the memory usage gauge
func (sm *SystemMetrics) collectMemoryUsage() {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		sm.config.Logger.Error("Error collecting memory usage", zap.Error(err))
		return
	}
	usage := memInfo.UsedPercent
	if sm.config.EnableEncryption {
		usage = applyEphemeralEncryption(usage, sm.config.EncryptionKey, sm.config.Logger)
	}
	sm.memUsageGauge.Set(usage)
	sm.config.Logger.Debug("Memory usage updated", zap.Float64("usage", usage))
}

// collectDiskUsage updates the disk usage gauge vector for each path
func (sm *SystemMetrics) collectDiskUsage() {
	for _, path := range sm.config.DiskPaths {
		diskInfo, err := disk.Usage(path)
		if err != nil {
			sm.config.Logger.Error("Error collecting disk usage", zap.String("path", path), zap.Error(err))
			continue
		}
		usage := diskInfo.UsedPercent
		if sm.config.EnableEncryption {
			usage = applyEphemeralEncryption(usage, sm.config.EncryptionKey, sm.config.Logger)
		}
		sm.diskUsageGaugeVec.WithLabelValues(path).Set(usage)
		sm.config.Logger.Debug("Disk usage updated", zap.String("path", path), zap.Float64("usage", diskInfo.UsedPercent))
	}
}

// collectNetworkIO updates the net I/O gauge vector per interface
func (sm *SystemMetrics) collectNetworkIO() {
	ioCounters, err := net.IOCounters(true)
	if err != nil {
		sm.config.Logger.Error("Error collecting network I/O", zap.Error(err))
		return
	}
	// reset old values
	sm.netIOGaugeVec.Reset()

	for _, counter := range ioCounters {
		rx := float64(counter.BytesRecv)
		tx := float64(counter.BytesSent)
		if sm.config.EnableEncryption {
			rx = applyEphemeralEncryption(rx, sm.config.EncryptionKey, sm.config.Logger)
			tx = applyEphemeralEncryption(tx, sm.config.EncryptionKey, sm.config.Logger)
		}
		sm.netIOGaugeVec.WithLabelValues(counter.Name, "rx").Set(rx)
		sm.netIOGaugeVec.WithLabelValues(counter.Name, "tx").Set(tx)
		sm.config.Logger.Debug("Network I/O updated", zap.String("interface", counter.Name), zap.Uint64("rx", counter.BytesRecv), zap.Uint64("tx", counter.BytesSent))
	}
}

// collectAiModelUsage queries the AiModelManager for usage data and updates gauge vectors
func (sm *SystemMetrics) collectAiModelUsage() {
	usageData, err := sm.config.AiModelManager.GetAiModelUsage()
	if err != nil {
		sm.config.Logger.Error("Error collecting AI model usage", zap.Error(err))
		return
	}
	sm.aiMemUsageGaugeVec.Reset()
	sm.aiCPUUsageGaugeVec.Reset()

	for _, mu := range usageData {
		memVal := mu.MemoryMB
		cpuVal := mu.CPUPercent

		if sm.config.EnableEncryption {
			memVal = applyEphemeralEncryption(memVal, sm.config.EncryptionKey, sm.config.Logger)
			cpuVal = applyEphemeralEncryption(cpuVal, sm.config.EncryptionKey, sm.config.Logger)
		}
		sm.aiMemUsageGaugeVec.WithLabelValues(mu.ModelName).Set(memVal)
		sm.aiCPUUsageGaugeVec.WithLabelValues(mu.ModelName).Set(cpuVal)
		sm.config.Logger.Debug("AI Model usage updated",
			zap.String("model", mu.ModelName),
			zap.Float64("mem_MB", mu.MemoryMB),
			zap.Float64("cpu_percent", mu.CPUPercent),
		)
	}
}

// applyEphemeralEncryption is a placeholder function that "encrypts" a float by turning it into a length of ciphertext
func applyEphemeralEncryption(value float64, key []byte, logger *zap.Logger) float64 {
	// In a real usage, you'd ephemeral-encrypt the numeric value. This function simulates by converting the float to string and "encrypting."
	plaintext := fmt.Sprintf("%.3f", value)
	encrypted, err := encryptMetric(plaintext, key)
	if err != nil {
		logger.Warn("Failed ephemeral encryption, returning raw value", zap.Error(err))
		return value
	}
	return float64(len(encrypted))
}

// encryptMetric is a placeholder that does a simple transformation
func encryptMetric(plaintext string, key []byte) ([]byte, error) {
	// real ephemeral encryption might do KEM, ephemeral key usage, etc.
	// We'll do a trivial approach: reverse the string + key length
	reversed := reverseString(plaintext)
	cipher := fmt.Sprintf("%s|%d", reversed, len(key))
	return []byte(cipher), nil
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
