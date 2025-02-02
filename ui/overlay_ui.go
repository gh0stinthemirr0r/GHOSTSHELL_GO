package ui

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// MetricsOverlayUI represents the visual metrics overlay using Raylib
type MetricsOverlayUI struct {
	logger           *zap.Logger
	cpuUsageGauge    prometheus.Gauge
	memoryUsageGauge prometheus.Gauge
	pqSecurityGauge  prometheus.Gauge
	dynamicData      []float64 // Simulated dynamic data for graphs
}

// NewMetricsOverlayUI initializes the metrics overlay with default values
func NewMetricsOverlayUI(logger *zap.Logger) *MetricsOverlayUI {
	if logger == nil {
		logger = zap.NewExample()
	}

	cpuUsage := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ghostshell_cpu_usage_percent",
		Help: "Overall CPU usage percentage.",
	})

	memoryUsage := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ghostshell_memory_usage_percent",
		Help: "Overall memory usage percentage.",
	})

	pqSecurity := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ghostshell_post_quantum_security",
		Help: "Indicates if post-quantum security is engaged (1) or not (0).",
	})

	return &MetricsOverlayUI{
		logger:           logger,
		cpuUsageGauge:    cpuUsage,
		memoryUsageGauge: memoryUsage,
		pqSecurityGauge:  pqSecurity,
		dynamicData:      make([]float64, 0),
	}
}

// StartDashboard starts the Raylib-based dashboard for displaying metrics
func (overlay *MetricsOverlayUI) StartDashboard() {
	rl.InitWindow(800, 600, "Metrics Overlay Dashboard")
	rl.SetTargetFPS(60)
	defer rl.CloseWindow()

	for !rl.WindowShouldClose() {
		overlay.updateMetrics() // Simulate metric updates

		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)

		// Draw title
		rl.DrawText("Metrics Overlay Dashboard", 10, 10, 20, rl.DarkGray)

		// Draw CPU Usage
		cpuUsage := overlay.cpuUsageGauge.Desc()
		cpuColor := rl.Green
		if cpuUsage.Value() > 80 {
			cpuColor = rl.Red
		}

	}

}
