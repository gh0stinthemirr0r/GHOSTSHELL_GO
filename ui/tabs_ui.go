package ui

import (
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// TabType identifies each tab
type TabType int

const (
	WorkspaceTab TabType = iota + 1
	FeaturesTab
)

// ThemeManager is an interface that provides the current theme.
// Typically, you'd define this in your theme_manager.go file or a shared package.
type ThemeManager interface {
	GetTheme() Theme
}

// Theme is a struct or interface that provides color fields (BackgroundColor, TextColor, etc.).
// Here we assume a simple struct with RGBA fields. Adjust to match your actual theme definition.
type Theme struct {
	BackgroundColor color.RGBA
	TextColor       color.RGBA
	BorderColor     color.RGBA
	HighlightColor  color.RGBA
	// ... additional fields as needed
}

// TabManager holds the state of tabs and their contents
type TabManager struct {
	activeTab    TabType
	workspace    *WorkspaceTab
	features     *FeaturesTab
	themeManager ThemeManager
	font         rl.Font // if you want a custom font for tab labels
}

// NewTabManager initializes a new TabManager with default values
func NewTabManager(tm ThemeManager, font rl.Font) *TabManager {
	return &TabManager{
		activeTab:    WorkspaceTab,
		workspace:    NewWorkspaceTab(tm, font),
		features:     NewFeaturesTab(tm, font),
		themeManager: tm,
		font:         font,
	}
}

// SwitchTab switches between tabs based on the provided type
func (tm *TabManager) SwitchTab(tabType TabType) {
	tm.activeTab = tabType

	// If you want each tab to know whether it's active:
	tm.workspace.SetActive(false)
	tm.features.SetActive(false)

	switch tabType {
	case WorkspaceTab:
		tm.workspace.SetActive(true)
	case FeaturesTab:
		tm.features.SetActive(true)
	}
}

// Draw draws the tab bar and then the currently active tab
func (tm *TabManager) Draw() {
	// 1. Draw the tab bar (with theming)
	tm.drawTabBar()

	// 2. Draw the active tab content
	switch tm.activeTab {
	case WorkspaceTab:
		tm.workspace.Draw()
	case FeaturesTab:
		tm.features.Draw()
	}
}

// drawTabBar renders a dynamic tab bar at the top with clickable tab labels.
func (tm *TabManager) drawTabBar() {
	if tm.themeManager == nil {
		// If no theme manager, fallback
		rl.DrawRectangle(0, 0, rl.GetScreenWidth(), 50, rl.Gray)
		rl.DrawText("Workspace", 20, 15, 20, rl.White)
		rl.DrawText("Features", 140, 15, 20, rl.White)
		return
	}

	// Fetch theme
	theme := tm.themeManager.GetTheme()
	bgColor := colorToRaylib(theme.BackgroundColor)
	txtColor := colorToRaylib(theme.TextColor)
	borderColor := colorToRaylib(theme.BorderColor)
	highlightColor := colorToRaylib(theme.HighlightColor)

	// Draw background for the tab bar
	rl.DrawRectangle(0, 0, rl.GetScreenWidth(), 50, bgColor)

	// Optionally draw a bottom border line for the tab bar
	rl.DrawLine(0, 50, rl.GetScreenWidth(), 50, borderColor)

	// Tab label positions
	tabs := []struct {
		Label string
		Type  TabType
		X     int32
		Y     int32
	}{
		{"Workspace", WorkspaceTab, 20, 15},
		{"Features", FeaturesTab, 140, 15},
	}

	for _, tab := range tabs {
		color := txtColor
		if tm.activeTab == tab.Type {
			color = highlightColor
		}
		rl.DrawTextEx(
			tm.font,
			tab.Label,
			rl.NewVector2(float32(tab.X), float32(tab.Y)),
			float32(tm.font.BaseSize),
			1,
			color,
		)

		// Check if the mouse clicks on a tab
		mouseX := rl.GetMouseX()
		mouseY := rl.GetMouseY()
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) &&
			mouseX > tab.X && mouseX < tab.X+100 && mouseY > tab.Y && mouseY < tab.Y+30 {
			tm.SwitchTab(tab.Type)
		}
	}
}

// colorToRaylib is a small helper to convert Go’s color.RGBA to Raylib’s rl.Color.
func colorToRaylib(c color.RGBA) rl.Color {
	return rl.Color{
		R: c.R,
		G: c.G,
		B: c.B,
		A: c.A,
	}
}

// WorkspaceTab is a placeholder example
type WorkspaceTab struct {
	active       bool
	themeManager ThemeManager
	font         rl.Font
}

func NewWorkspaceTab(tm ThemeManager, font rl.Font) *WorkspaceTab {
	return &WorkspaceTab{
		active:       true,
		themeManager: tm,
		font:         font,
	}
}

func (w *WorkspaceTab) IsActive() bool {
	return w.active
}

func (w *WorkspaceTab) SetActive(active bool) {
	w.active = active
}

func (w *WorkspaceTab) Draw() {
	// Example: fill the rest of the screen below the tab bar
	yOffset := int32(50)
	screenWidth := rl.GetScreenWidth()
	screenHeight := rl.GetScreenHeight() - yOffset

	if w.themeManager == nil {
		// fallback
		rl.DrawRectangle(0, yOffset, screenWidth, screenHeight, rl.DarkGray)
		rl.DrawText("Workspace Tab Content Here", 50, 100, 20, rl.White)
		return
	}

	theme := w.themeManager.GetTheme()
	bgColor := colorToRaylib(theme.BackgroundColor)
	txtColor := colorToRaylib(theme.TextColor)

	// Fill background with theme color
	rl.DrawRectangle(0, yOffset, screenWidth, screenHeight, bgColor)

	// Draw some text
	rl.DrawTextEx(w.font, "Workspace Tab Content Here", rl.NewVector2(50, 100), float32(w.font.BaseSize), 1, txtColor)
}

// FeaturesTab is a placeholder example
type FeaturesTab struct {
	active       bool
	themeManager ThemeManager
	font         rl.Font
}

func NewFeaturesTab(tm ThemeManager, font rl.Font) *FeaturesTab {
	return &FeaturesTab{
		active:       false,
		themeManager: tm,
		font:         font,
	}
}

func (f *FeaturesTab) IsActive() bool {
	return f.active
}

func (f *FeaturesTab) SetActive(active bool) {
	f.active = active
}

func (f *FeaturesTab) Draw() {
	// Similar to WorkspaceTab, fill the screen below the tab bar with theme color
	yOffset := int32(50)
	screenWidth := rl.GetScreenWidth()
	screenHeight := rl.GetScreenHeight() - yOffset

	if f.themeManager == nil {
		rl.DrawRectangle(0, yOffset, screenWidth, screenHeight, rl.DarkGray)
		rl.DrawText("Features Tab Content Here", 50, 100, 20, rl.White)
		return
	}

	theme := f.themeManager.GetTheme()
	bgColor := colorToRaylib(theme.BackgroundColor)
	txtColor := colorToRaylib(theme.TextColor)

	rl.DrawRectangle(0, yOffset, screenWidth, screenHeight, bgColor)
	rl.DrawTextEx(f.font, "Features Tab Content Here", rl.NewVector2(50, 100), float32(f.font.BaseSize), 1, txtColor)
}
