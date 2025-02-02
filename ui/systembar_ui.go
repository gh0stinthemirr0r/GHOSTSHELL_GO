package ui

import (
	"fmt"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Theme represents the color/style aspects we want from the theme manager.
type Theme struct {
	BackgroundColor rl.Color
	TextColor       rl.Color
	BorderColor     rl.Color
	HighlightColor  rl.Color
	// Add more fields as needed
}

// ThemeManager is an interface that provides the current Theme.
type ThemeManager interface {
	GetTheme() Theme
}

// SystemBar represents the system status bar at the bottom of the screen.
type SystemBar struct {
	show          bool // Toggle visibility
	width, height int32
	x, y          int32

	// Font for text rendering; you can load a custom one or default.
	font rl.Font

	// Stats
	cpuUsage float64
	ramUsage float64
	network  string
	aiModel  string
	aiCpu    float64
	aiRam    float64

	// Animation and effects
	blinkState   bool
	blinkCounter float32
	lastUpdate   time.Time

	// The ThemeManager to fetch color/style from
	themeManager ThemeManager
}

// NewSystemBar initializes a new SystemBar with default values.
func NewSystemBar(tm ThemeManager) *SystemBar {
	// Load a custom font or use default.
	font := rl.LoadFontEx("resources/futuristic_font.ttf", 20, nil, 0)
	if font.BaseSize == 0 {
		font = rl.GetFontDefault()
	}

	return &SystemBar{
		show:         true,
		width:        rl.GetScreenWidth(),
		height:       30,
		x:            0,
		y:            rl.GetScreenHeight() - 30,
		font:         font,
		themeManager: tm,
		lastUpdate:   time.Now(),
	}
}

// Update updates the system bar data.
func (sb *SystemBar) Update(cpu, ram float64, network, aiModel string, aiCpu, aiRam float64) {
	sb.cpuUsage = cpu
	sb.ramUsage = ram
	sb.network = network
	sb.aiModel = aiModel
	sb.aiCpu = aiCpu
	sb.aiRam = aiRam

	// Update animation states
	sb.blinkCounter += rl.GetFrameTime()
	if sb.blinkCounter >= 0.5 {
		sb.blinkCounter = 0
		sb.blinkState = !sb.blinkState
	}

	// Dynamically adjust position and dimensions if screen size changes
	sb.width = rl.GetScreenWidth()
	sb.height = 30
	sb.x = 0
	sb.y = rl.GetScreenHeight() - sb.height
}

// Draw renders the system bar on the screen with theme and animation.
func (sb *SystemBar) Draw() {
	if !sb.show {
		return
	}

	// Get theme colors or use fallback
	var bgColor, txtColor, borderColor rl.Color
	if sb.themeManager != nil {
		theme := sb.themeManager.GetTheme()
		bgColor = theme.BackgroundColor
		txtColor = theme.TextColor
		borderColor = theme.BorderColor
	} else {
		bgColor = rl.Color{R: 0, G: 0, B: 0, A: 200}
		txtColor = rl.Color{R: 0, G: 255, B: 0, A: 255}
		borderColor = rl.Gray
	}

	// Render background
	rl.DrawRectangle(sb.x, sb.y, sb.width, sb.height, bgColor)

	// Optionally add a border
	rl.DrawRectangleLinesEx(
		rl.NewRectangle(float32(sb.x), float32(sb.y), float32(sb.width), float32(sb.height)),
		2, borderColor,
	)

	// Draw stats text
	fontSize := float32(20)
	fontSpacing := float32(2)
	padding := int32(10)

	// CPU Usage
	rl.DrawTextEx(
		sb.font,
		fmt.Sprintf("CPU: %.1f%%", sb.cpuUsage),
		rl.Vector2{X: float32(sb.x + padding), Y: float32(sb.y + 8)},
		fontSize, fontSpacing, txtColor,
	)

	// RAM Usage
	rl.DrawTextEx(
		sb.font,
		fmt.Sprintf("RAM: %.1f%%", sb.ramUsage),
		rl.Vector2{X: float32(sb.x + padding + 120), Y: float32(sb.y + 8)},
		fontSize, fontSpacing, txtColor,
	)

	// Network
	if sb.network != "" {
		rl.DrawTextEx(
			sb.font,
			fmt.Sprintf("Network: %s", sb.network),
			rl.Vector2{X: float32(sb.x + padding + 250), Y: float32(sb.y + 8)},
			fontSize, fontSpacing, txtColor,
		)
	}

	// AI Model Info
	xOffset := float32(sb.width - 300)
	rl.DrawTextEx(
		sb.font,
		fmt.Sprintf("AI: %s", sb.aiModel),
		rl.Vector2{X: xOffset, Y: float32(sb.y + 8)},
		fontSize, fontSpacing, txtColor,
	)

	// Blinking animation for AI stats (example)
	if sb.blinkState {
		rl.DrawTextEx(
			sb.font,
			fmt.Sprintf("AI CPU: %.1f%% RAM: %.1f%%", sb.aiCpu, sb.aiRam),
			rl.Vector2{X: xOffset + 100, Y: float32(sb.y + 8)},
			fontSize, fontSpacing, rl.Red,
		)
	}
}

// ToggleVisibility toggles the visibility of the system bar.
func (sb *SystemBar) ToggleVisibility() {
	sb.show = !sb.show
}
