package ui

import (
	"fmt"
	"math/rand"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Dashboard represents the main system dashboard with graphs, metrics, and widgets.
type Dashboard struct {
	x, y          int32 // Position of the dashboard
	width, height int32 // Size of the dashboard
	font          rl.Font
	themeManager  ThemeManager

	// Data for metrics
	cpuUsageHistory []float64
	ramUsageHistory []float64
	networkHistory  []float64

	// Widgets
	clockVisible    bool
	networkVisible  bool
	cpuGraphVisible bool
	ramGraphVisible bool
	updateInterval  time.Duration
	lastUpdate      time.Time
}

// NewDashboard creates and initializes a new Dashboard instance.
func NewDashboard(x, y, width, height int32, tm ThemeManager, font rl.Font) *Dashboard {
	if font.BaseSize == 0 {
		font = rl.GetFontDefault()
	}

	rand.Seed(time.Now().UnixNano())

	return &Dashboard{
		x:               x,
		y:               y,
		width:           width,
		height:          height,
		font:            font,
		themeManager:    tm,
		cpuUsageHistory: make([]float64, 60),
		ramUsageHistory: make([]float64, 60),
		networkHistory:  make([]float64, 60),
		clockVisible:    true,
		networkVisible:  true,
		cpuGraphVisible: true,
		ramGraphVisible: true,
		updateInterval:  2 * time.Second,
		lastUpdate:      time.Now(),
	}
}

// Update updates the metrics and widget data periodically.
func (db *Dashboard) Update() {
	// Check update interval
	if time.Since(db.lastUpdate) < db.updateInterval {
		return
	}
	db.lastUpdate = time.Now()

	// Update CPU, RAM, and Network histories with random data for now (replace with real metrics)
	db.cpuUsageHistory = append(db.cpuUsageHistory[1:], rand.Float64()*100)
	db.ramUsageHistory = append(db.ramUsageHistory[1:], rand.Float64()*100)
	db.networkHistory = append(db.networkHistory[1:], rand.Float64()*100)
}

// Draw renders the dashboard, including widgets and graphs.
func (db *Dashboard) Draw() {
	// Get theme colors
	theme := db.themeManager.GetTheme()
	bgColor := colorToRaylib(theme.BackgroundColor)
	textColor := colorToRaylib(theme.TextColor)
	borderColor := colorToRaylib(theme.BorderColor)

	// Draw dashboard background
	rl.DrawRectangle(db.x, db.y, db.width, db.height, bgColor)

	// Optionally draw a border
	rl.DrawRectangleLinesEx(
		rl.Rectangle{X: float32(db.x), Y: float32(db.y), Width: float32(db.width), Height: float32(db.height)},
		2,
		borderColor,
	)

	// Draw widgets and graphs
	currentY := db.y + 10
	lineHeight := int32(30)

	// Draw Clock Widget
	if db.clockVisible {
		db.drawClock(int32(db.x+10), currentY, textColor)
		currentY += lineHeight + 10
	}

	// Draw Network Status
	if db.networkVisible {
		db.drawNetworkWidget(int32(db.x+10), currentY, textColor)
		currentY += lineHeight + 10
	}

	// Draw CPU Usage Graph
	if db.cpuGraphVisible {
		db.drawGraph("CPU Usage", db.cpuUsageHistory, int32(db.x+10), currentY, 300, 100, textColor, borderColor)
		currentY += 120
	}

	// Draw RAM Usage Graph
	if db.ramGraphVisible {
		db.drawGraph("RAM Usage", db.ramUsageHistory, int32(db.x+10), currentY, 300, 100, textColor, borderColor)
	}
}

// drawClock renders the current time on the dashboard.
func (db *Dashboard) drawClock(x, y int32, textColor rl.Color) {
	timeString := time.Now().Format("15:04:05")
	rl.DrawTextEx(
		db.font,
		fmt.Sprintf("Clock: %s", timeString),
		rl.Vector2{X: float32(x), Y: float32(y)},
		float32(db.font.BaseSize),
		1,
		textColor,
	)
}

// drawNetworkWidget displays a placeholder for network stats.
func (db *Dashboard) drawNetworkWidget(x, y int32, textColor rl.Color) {
	networkStatus := "Online" // Replace with real network data
	rl.DrawTextEx(
		db.font,
		fmt.Sprintf("Network: %s", networkStatus),
		rl.Vector2{X: float32(x), Y: float32(y)},
		float32(db.font.BaseSize),
		1,
		textColor,
	)
}

// drawGraph renders a line graph for a given dataset.
func (db *Dashboard) drawGraph(title string, data []float64, x, y, width, height int32, textColor, lineColor rl.Color) {
	// Draw graph title
	rl.DrawTextEx(
		db.font,
		title,
		rl.Vector2{X: float32(x), Y: float32(y)},
		float32(db.font.BaseSize),
		1,
		textColor,
	)

	// Calculate graph points
	graphX := x
	graphY := y + 20
	graphWidth := width
	graphHeight := height
	maxValue := float64(100)

	numPoints := len(data)
	if numPoints < 2 {
		return
	}

	stepX := float32(graphWidth) / float32(numPoints-1)

	// Draw graph lines
	for i := 0; i < numPoints-1; i++ {
		x1 := float32(graphX) + float32(i)*stepX
		y1 := float32(graphY+graphHeight) - float32(data[i])/float32(maxValue)*float32(graphHeight)
		x2 := float32(graphX) + float32(i+1)*stepX
		y2 := float32(graphY+graphHeight) - float32(data[i+1])/float32(maxValue)*float32(graphHeight)

		rl.DrawLineV(rl.Vector2{X: x1, Y: y1}, rl.Vector2{X: x2, Y: y2}, lineColor)
	}

	// Draw graph border
	rl.DrawRectangleLines(
		graphX, graphY, graphWidth, graphHeight, lineColor,
	)
}

// ToggleVisibility toggles the visibility of widgets or graphs.
func (db *Dashboard) ToggleVisibility(widget string) {
	switch widget {
	case "clock":
		db.clockVisible = !db.clockVisible
	case "network":
		db.networkVisible = !db.networkVisible
	case "cpu":
		db.cpuGraphVisible = !db.cpuGraphVisible
	case "ram":
		db.ramGraphVisible = !db.ramGraphVisible
	}
}

// colorToRaylib converts `image/color` to Raylib colors.
func colorToRaylib(c rl.Color) rl.Color {
	return rl.Color{R: c.R, G: c.G, B: c.B, A: c.A}
}
