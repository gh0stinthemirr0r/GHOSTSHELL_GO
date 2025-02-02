package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prometheus metrics registry
var (
	moduleStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "oqs_module_status",
			Help: "Status of OQS Modules (1 = online, 0 = offline)",
		},
		[]string{"module"},
	)
	encryptionLatency = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "oqs_encryption_latency_ms",
			Help:    "Latency of encryption operations in milliseconds",
			Buckets: prometheus.LinearBuckets(5, 5, 10),
		},
	)
)

func init() {
	// Register Prometheus metrics
	prometheus.MustRegister(moduleStatus)
	prometheus.MustRegister(encryptionLatency)
}

func main() {
	// Serve Prometheus metrics
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Fatal(http.ListenAndServe(":2112", nil))
	}()

	// Initialize Raylib window
	rl.InitWindow(1000, 700, "Quantum Security Dashboard")
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	// Module statuses
	modules := []string{"Vault", "Signature", "AES", "Network", "Random"}
	moduleHealth := make(map[string]bool)

	// Initialize all modules as online
	for _, module := range modules {
		moduleHealth[module] = true
		moduleStatus.WithLabelValues(module).Set(1)
	}

	// Simulate metrics updates
	go simulateMetrics(modules, moduleHealth)

	// Dashboard render loop
	for !rl.WindowShouldClose() {
		renderDashboard(modules, moduleHealth)
	}
}

// simulateMetrics simulates random status changes and latency metrics for demonstration purposes
func simulateMetrics(modules []string, moduleHealth map[string]bool) {
	for {
		time.Sleep(2 * time.Second)

		// Randomly toggle module statuses
		for _, module := range modules {
			status := rand.Intn(2) // 0 or 1
			moduleHealth[module] = status == 1
			moduleStatus.WithLabelValues(module).Set(float64(status))
		}

		// Record random encryption latency
		encryptionLatency.Observe(float64(rand.Intn(50) + 5))
	}
}

// renderDashboard draws the OQS dashboard using Raylib
func renderDashboard(modules []string, moduleHealth map[string]bool) {
	rl.BeginDrawing()
	rl.ClearBackground(rl.RayWhite)

	// Dashboard title
	rl.DrawText("Quantum Security Dashboard", 350, 20, 24, rl.DarkGray)

	// Draw module statuses
	drawModuleStatuses(modules, moduleHealth, 50, 80)

	// Draw encryption latency graph
	drawLatencyGraph(450, 300, 500, 200)

	rl.EndDrawing()
}

// drawModuleStatuses displays the status of each module on the screen
func drawModuleStatuses(modules []string, moduleHealth map[string]bool, x, y int32) {
	for _, module := range modules {
		status := moduleHealth[module]
		color := rl.Green
		if !status {
			color = rl.Red
		}
		rl.DrawText(fmt.Sprintf("%s: %s", module, statusString(status)), x, y, 20, color)
		y += 40
	}
}

// drawLatencyGraph renders a simulated latency graph
func drawLatencyGraph(x, y, width, height int32) {
	// Background for the graph
	rl.DrawRectangle(x, y, width, height, rl.LightGray)
	rl.DrawRectangleLinesEx(rl.Rectangle{
		X: float32(x), Y: float32(y), Width: float32(width), Height: float32(height),
	}, 2, rl.DarkGray)

	// Simulate latency data (replace with real Prometheus query if available)
	latencyData := []float64{}
	for i := 0; i < 20; i++ {
		latencyData = append(latencyData, float64(rand.Intn(50)+5))
	}

	// Draw graph
	maxLatency := 60.0 // Adjust as needed
	barWidth := float32(width) / float32(len(latencyData))

	for i, latency := range latencyData {
		h := (float32(latency) / float32(maxLatency)) * float32(height)
		barX := float32(x) + float32(i)*barWidth
		barY := float32(y) + float32(height) - h
		barColor := rl.Fade(rl.DarkBlue, 0.8)

		if latency > 40 {
			barColor = rl.Red
		} else if latency > 20 {
			barColor = rl.Orange
		}

		rl.DrawRectangle(int32(barX), int32(barY), int32(barWidth)-2, int32(h), barColor)
	}

	// Add labels
	rl.DrawText("Latency (ms)", x+10, y-20, 16, rl.DarkGray)
	rl.DrawText("Time", x+width-60, y+height+10, 16, rl.DarkGray)
}

// statusString converts a boolean status to a string
func statusString(online bool) string {
	if online {
		return "Online"
	}
	return "Offline"
}
