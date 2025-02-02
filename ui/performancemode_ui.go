package ui

import (
	"fmt"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// PerformanceModeUI represents the performance monitoring interface
// that displays CPU, GPU, memory, and network usage in a compact, high-performance mode.
type PerformanceModeUI struct {
	font           rl.Font       // Custom font for rendering
	width, height  int32         // Dimensions of the UI panel
	x, y           int32         // Position of the panel on the screen
	cpuUsage       float64       // CPU usage percentage
	gpuUsage       float64       // GPU usage percentage
	ramUsage       float64       // RAM usage percentage
	networkStats   string        // Summary of network usage
	lastUpdate     time.Time     // Timestamp of the last update
	updateInterval time.Duration // Update interval for metrics
	themeManager   ThemeManager  // To fetch theming details
}

// NewPerformanceModeUI initializes a new instance of PerformanceModeUI
func NewPerformanceModeUI(themeManager ThemeManager, font rl.Font) *PerformanceModeUI {
	return &PerformanceModeUI{
		font:           font,
		width:          400,
		height:         200,
		x:              20,
		y:              20,
		cpuUsage:       0.0,
		gpuUsage:       0.0,
		ramUsage:       0.0,
		networkStats:   "Initializing...",
		lastUpdate:     time.Now(),
		updateInterval: 2 * time.Second, // Default update interval
		themeManager:   themeManager,
	}
}

// Update refreshes the system metrics displayed on the UI.
func (pm *PerformanceModeUI) Update() {
	// Update only if the interval has passed
	if time.Since(pm.lastUpdate) < pm.updateInterval {
		return
	}

	pm.lastUpdate = time.Now()

	// Here you would collect real performance stats.
	// For demonstration, we'll simulate random metrics.
	pm.cpuUsage = getSimulatedMetric()
	pm.gpuUsage = getSimulatedMetric()
	pm.ramUsage = getSimulatedMetric()
	pm.networkStats = fmt.Sprintf("Up: %.2f Mbps | Down: %.2f Mbps", getSimulatedMetric(), getSimulatedMetric())
}

// Draw renders the performance monitoring panel to the screen.
func (pm *PerformanceModeUI) Draw() {
	// Fetch theme colors
	bgColor := rl.DarkGray
	txtColor := rl.White
	if pm.themeManager != nil {
		theme := pm.themeManager.GetTheme()
		bgColor = theme.BackgroundColor
		txtColor = theme.TextColor
	}

	// Draw the panel background
	rl.DrawRectangle(pm.x, pm.y, pm.width, pm.height, bgColor)

	// Draw the panel border
	borderColor := rl.LightGray
	if pm.themeManager != nil {
		borderColor = pm.themeManager.GetTheme().BorderColor
	}
	rl.DrawRectangleLinesEx(
		rl.Rectangle{X: float32(pm.x), Y: float32(pm.y), Width: float32(pm.width), Height: float32(pm.height)},
		2, borderColor,
	)

	// Draw the performance metrics
	padding := int32(10)
	fontSize := float32(20)
	lineHeight := int32(fontSize + 8)

	metrics := []string{
		fmt.Sprintf("CPU Usage: %.2f%%", pm.cpuUsage),
		fmt.Sprintf("GPU Usage: %.2f%%", pm.gpuUsage),
		fmt.Sprintf("RAM Usage: %.2f%%", pm.ramUsage),
		fmt.Sprintf("Network: %s", pm.networkStats),
	}

	for i, metric := range metrics {
		posX := float32(pm.x + padding)
		posY := float32(pm.y + padding + lineHeight*int32(i))
		rl.DrawTextEx(pm.font, metric, rl.NewVector2(posX, posY), fontSize, 1, txtColor)
	}
}

// SetPosition sets the position of the performance panel on the screen.
func (pm *PerformanceModeUI) SetPosition(x, y int32) {
	pm.x = x
	pm.y = y
}

// SetSize adjusts the size of the performance panel.
func (pm *PerformanceModeUI) SetSize(width, height int32) {
	pm.width = width
	pm.height = height
}

// SetUpdateInterval modifies the update interval for system metrics.
func (pm *PerformanceModeUI) SetUpdateInterval(interval time.Duration) {
	pm.updateInterval = interval
}

// Simulated metric function for demonstration purposes.
func getSimulatedMetric() float64 {
	return float64(rl.GetRandomValue(10, 90)) // Random value between 10 and 90
}
