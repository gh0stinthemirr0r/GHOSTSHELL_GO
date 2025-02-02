package ui

import (
	"fmt"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// VPNManagerUI represents the user interface for managing VPN connections.
type VPNManagerUI struct {
	x, y, width, height int32
	themeManager        ThemeManager
	font                rl.Font

	// Data for VPN metrics and logs
	activeVPN         bool
	connectionMetrics map[string]int // e.g., {"Total": X, "Success": Y, "Failure": Z}
	latestLogs        []string       // Holds recent log entries for display

	// Buttons
	startButton  Button
	stopButton   Button
	reportButton Button
	logScroll    ScrollableLog
}

// NewVPNManagerUI initializes a new instance of VPNManagerUI.
func NewVPNManagerUI(tm ThemeManager, font rl.Font, x, y, width, height int32) *VPNManagerUI {
	return &VPNManagerUI{
		x:                 x,
		y:                 y,
		width:             width,
		height:            height,
		themeManager:      tm,
		font:              font,
		activeVPN:         false,
		connectionMetrics: map[string]int{"Total": 0, "Success": 0, "Failure": 0},
		latestLogs:        []string{},
		startButton:       NewButton(x+20, y+height-60, 120, 40, "Start VPN"),
		stopButton:        NewButton(x+160, y+height-60, 120, 40, "Stop VPN"),
		reportButton:      NewButton(x+300, y+height-60, 120, 40, "Generate Report"),
		logScroll:         NewScrollableLog(x+20, y+60, width-40, height-140, font),
	}
}

// Update handles input and UI updates.
func (ui *VPNManagerUI) Update() {
	// Update buttons
	if ui.startButton.Update() {
		ui.handleStartVPN()
	}

	if ui.stopButton.Update() {
		ui.handleStopVPN()
	}

	if ui.reportButton.Update() {
		ui.handleGenerateReport()
	}

	// Update log scrolling
	ui.logScroll.Update()
}

// Draw renders the VPN Manager UI.
func (ui *VPNManagerUI) Draw() {
	// Get theme colors
	bgColor := rl.DarkGray
	textColor := rl.White
	borderColor := rl.LightGray
	if ui.themeManager != nil {
		theme := ui.themeManager.GetTheme()
		bgColor = theme.BackgroundColor
		textColor = theme.TextColor
		borderColor = theme.BorderColor
	}

	// Draw background and border
	rl.DrawRectangle(ui.x, ui.y, ui.width, ui.height, bgColor)
	rl.DrawRectangleLinesEx(
		rl.Rectangle{X: float32(ui.x), Y: float32(ui.y), Width: float32(ui.width), Height: float32(ui.height)},
		2,
		borderColor,
	)

	// Draw title
	rl.DrawTextEx(ui.font, "VPN Manager", rl.Vector2{X: float32(ui.x + 20), Y: float32(ui.y + 10)}, 20, 2, textColor)

	// Draw metrics
	metricsY := ui.y + 40
	for key, value := range ui.connectionMetrics {
		metricsText := fmt.Sprintf("%s: %d", key, value)
		rl.DrawTextEx(ui.font, metricsText, rl.Vector2{X: float32(ui.x + 20), Y: float32(metricsY)}, 16, 2, textColor)
		metricsY += 20
	}

	// Draw buttons
	ui.startButton.Draw()
	ui.stopButton.Draw()
	ui.reportButton.Draw()

	// Draw logs
	ui.logScroll.Draw()
}

// handleStartVPN handles starting the VPN connection.
func (ui *VPNManagerUI) handleStartVPN() {
	ui.activeVPN = true
	ui.connectionMetrics["Total"]++
	ui.connectionMetrics["Success"]++ // Mock success; replace with actual logic
	ui.latestLogs = append(ui.latestLogs, fmt.Sprintf("[%s] VPN started successfully.", time.Now().Format("15:04:05")))
}

// handleStopVPN handles stopping the VPN connection.
func (ui *VPNManagerUI) handleStopVPN() {
	if !ui.activeVPN {
		return
	}
	ui.activeVPN = false
	ui.latestLogs = append(ui.latestLogs, fmt.Sprintf("[%s] VPN stopped.", time.Now().Format("15:04:05")))
}

// handleGenerateReport handles report generation.
func (ui *VPNManagerUI) handleGenerateReport() {
	ui.latestLogs = append(ui.latestLogs, fmt.Sprintf("[%s] Report generated.", time.Now().Format("15:04:05")))
	// Integrate with report generation logic
}

// ScrollableLog represents a scrollable log viewer.
type ScrollableLog struct {
	x, y, width, height int32
	font                rl.Font
	lines               []string
	scrollOffset        int
}

// NewScrollableLog initializes a new ScrollableLog.
func NewScrollableLog(x, y, width, height int32, font rl.Font) ScrollableLog {
	return ScrollableLog{
		x:            x,
		y:            y,
		width:        width,
		height:       height,
		font:         font,
		lines:        []string{},
		scrollOffset: 0,
	}
}

// Update handles scrolling input for the log.
func (sl *ScrollableLog) Update() {
	if rl.IsKeyPressed(rl.KeyUp) {
		if sl.scrollOffset > 0 {
			sl.scrollOffset--
		}
	} else if rl.IsKeyPressed(rl.KeyDown) {
		if sl.scrollOffset < len(sl.lines)-1 {
			sl.scrollOffset++
		}
	}
}

// Draw renders the scrollable log.
func (sl *ScrollableLog) Draw() {
	rl.DrawRectangle(sl.x, sl.y, sl.width, sl.height, rl.Black)
	startIdx := sl.scrollOffset
	endIdx := startIdx + int(sl.height)/20 // Assuming 20px line height
	if endIdx > len(sl.lines) {
		endIdx = len(sl.lines)
	}
	for i, line := range sl.lines[startIdx:endIdx] {
		lineY := sl.y + int32(i*20)
		rl.DrawTextEx(sl.font, line, rl.Vector2{X: float32(sl.x + 5), Y: float32(lineY)}, 16, 1, rl.White)
	}
}

// Button represents a clickable button in the UI.
type Button struct {
	x, y, width, height int32
	label               string
	isHovered           bool
}

// NewButton initializes a new Button.
func NewButton(x, y, width, height int32, label string) Button {
	return Button{
		x:      x,
		y:      y,
		width:  width,
		height: height,
		label:  label,
	}
}

// Update handles input for the button.
func (b *Button) Update() bool {
	mouseX := rl.GetMouseX()
	mouseY := rl.GetMouseY()
	b.isHovered = mouseX > b.x && mouseX < b.x+b.width && mouseY > b.y && mouseY < b.y+b.height

	if b.isHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		return true
	}
	return false
}

// Draw renders the button.
func (b *Button) Draw() {
	bgColor := rl.DarkGray
	if b.isHovered {
		bgColor = rl.Gray
	}
	rl.DrawRectangle(b.x, b.y, b.width, b.height, bgColor)
	textWidth := rl.MeasureText(b.label, 20)
	rl.DrawText(b.label, b.x+(b.width-int32(textWidth))/2, b.y+(b.height-20)/2, 20, rl.White)
}
