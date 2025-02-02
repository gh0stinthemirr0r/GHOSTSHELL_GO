package ui

import (
	"fmt"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type SettingsUI struct {
	x, y, width, height int32

	themeManager ThemeManager
	font         rl.Font

	settings       map[string]interface{} // Key-value pairs for settings
	selectedIndex  int                    // Index of the currently selected setting
	isEditing      bool                   // If true, editing mode is active
	editValueInput string                 // Temporary value input for editing
}

// NewSettingsUI creates a new settings UI instance.
func NewSettingsUI(tm ThemeManager, font rl.Font, x, y, width, height int32) *SettingsUI {
	return &SettingsUI{
		x:            x,
		y:            y,
		width:        width,
		height:       height,
		themeManager: tm,
		font:         font,
		settings: map[string]interface{}{
			"Enable Particles": true,
			"Max Particles":    100,
			"Theme":            "NeonNight",
			"Glow Effect":      true,
		},
		selectedIndex:  0,
		isEditing:      false,
		editValueInput: "",
	}
}

// Update handles user input and updates the UI state.
func (ui *SettingsUI) Update() {
	if ui.isEditing {
		// Handle editing input
		if rl.IsKeyPressed(rl.KeyEnter) {
			ui.applyEdit()
		} else if rl.IsKeyPressed(rl.KeyEscape) {
			ui.isEditing = false
			ui.editValueInput = ""
		} else {
			for {
				typed := rl.GetCharPressed()
				if typed == 0 {
					break
				}

				// Allow backspace to remove characters
				if typed == '' && len(ui.editValueInput) > 0 {
					ui.editValueInput = ui.editValueInput[:len(ui.editValueInput)-1]
				} else if typed >= 32 && typed <= 126 {
					ui.editValueInput += string(typed)
				}
			}
		}
		return
	}

	// Navigation and selection
	if rl.IsKeyPressed(rl.KeyDown) {
		ui.selectedIndex++
		if ui.selectedIndex >= len(ui.settings) {
			ui.selectedIndex = 0
		}
	} else if rl.IsKeyPressed(rl.KeyUp) {
		ui.selectedIndex--
		if ui.selectedIndex < 0 {
			ui.selectedIndex = len(ui.settings) - 1
		}
	} else if rl.IsKeyPressed(rl.KeyEnter) {
		// Enter editing mode
		ui.isEditing = true
		ui.editValueInput = fmt.Sprintf("%v", ui.getSelectedValue())
	}
}

// Draw renders the settings UI.
func (ui *SettingsUI) Draw() {
	// Get theme
	bgColor := rl.Color{R: 30, G: 30, B: 46, A: 255}
	textColor := rl.White
	borderColor := rl.Gray

	if ui.themeManager != nil {
		theme := ui.themeManager.GetTheme()
		bgColor = theme.BackgroundColor
		textColor = theme.TextColor
		borderColor = theme.BorderColor
	}

	// Draw background
	rl.DrawRectangle(ui.x, ui.y, ui.width, ui.height, bgColor)
	// Draw border
	rl.DrawRectangleLinesEx(
		rl.Rectangle{X: float32(ui.x), Y: float32(ui.y), Width: float32(ui.width), Height: float32(ui.height)},
		2, borderColor,
	)

	// Draw settings title
	rl.DrawTextEx(ui.font, "Settings", rl.Vector2{X: float32(ui.x + 10), Y: float32(ui.y + 10)}, 20, 2, textColor)

	// Draw settings entries
	offsetY := float32(40)
	lineHeight := float32(30)
	i := 0
	for key, value := range ui.settings {
		isSelected := (i == ui.selectedIndex)
		entryText := fmt.Sprintf("%s: %v", key, value)
		entryColor := textColor
		if isSelected {
			entryColor = rl.Green
		}

		// Draw entry
		entryPos := rl.Vector2{X: float32(ui.x + 10), Y: float32(ui.y) + offsetY + (lineHeight * float32(i))}
		rl.DrawTextEx(ui.font, entryText, entryPos, 18, 2, entryColor)

		if isSelected && ui.isEditing {
			// Highlight the editable value
			editPos := rl.Vector2{X: entryPos.X + 250, Y: entryPos.Y}
			editValueText := fmt.Sprintf("[Editing: %s]", ui.editValueInput)
			rl.DrawTextEx(ui.font, editValueText, editPos, 18, 2, rl.Red)
		}
		i++
	}
}

// applyEdit applies the edited value to the currently selected setting.
func (ui *SettingsUI) applyEdit() {
	key := ui.getSelectedKey()
	if key == "" {
		ui.isEditing = false
		return
	}

	// Convert to appropriate type based on current value
	switch ui.settings[key].(type) {
	case bool:
		ui.settings[key] = (strings.ToLower(ui.editValueInput) == "true")
	case int:
		var intValue int
		fmt.Sscanf(ui.editValueInput, "%d", &intValue)
		ui.settings[key] = intValue
	case string:
		ui.settings[key] = ui.editValueInput
	}

	ui.isEditing = false
	ui.editValueInput = ""
}

// getSelectedKey returns the key of the currently selected setting.
func (ui *SettingsUI) getSelectedKey() string {
	i := 0
	for key := range ui.settings {
		if i == ui.selectedIndex {
			return key
		}
		i++
	}
	return ""
}

// getSelectedValue returns the value of the currently selected setting.
func (ui *SettingsUI) getSelectedValue() interface{} {
	key := ui.getSelectedKey()
	if key == "" {
		return nil
	}
	return ui.settings[key]
}
