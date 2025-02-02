package ui

import (
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Theme is a simple struct for the background, text, and cursor colors.
type Theme struct {
	BackgroundColor  rl.Color
	TextColor        rl.Color
	BorderColor      rl.Color
	CursorColor      rl.Color
	ActiveTabColor   rl.Color
	InactiveTabColor rl.Color
}

// ThemeManager is an interface that returns the current Theme.
type ThemeManager interface {
	GetTheme() Theme
}

// EditableBreadcrumb represents an editable breadcrumb bar.
type EditableBreadcrumb struct {
	currentPath  []rune  // Stores path in runes for easier cursor manipulation.
	editMode     bool    // Whether the user is editing the path.
	font         rl.Font // Font for text rendering.
	width        int32   // Width of the breadcrumb bar.
	height       int32   // Height of the bar.
	x, y         int32   // Position (top-left corner).
	cursorPos    int     // Cursor position in the rune slice.
	blinkCounter float32 // For cursor blinking timing.
	blinkState   bool    // Toggles cursor visibility.

	themeManager ThemeManager
}

// NewEditableBreadcrumb initializes a new EditableBreadcrumb with default values.
func NewEditableBreadcrumb(tm ThemeManager, x, y, width, height int32, font rl.Font) *EditableBreadcrumb {
	// If no font is provided, load one from disk (example).
	if font.BaseSize == 0 {
		loadedFont := rl.LoadFontEx("resources/futuristic_font.ttf", 20, nil, 0)
		if loadedFont.BaseSize == 0 {
			// Fallback to default font.
			loadedFont = rl.GetFontDefault()
		}
		font = loadedFont
	}

	defaultPath := []rune("/home/user")
	return &EditableBreadcrumb{
		currentPath:  defaultPath,
		editMode:     false,
		font:         font,
		width:        width,
		height:       height,
		x:            x,
		y:            y,
		cursorPos:    len(defaultPath),
		blinkCounter: 0,
		blinkState:   true,
		themeManager: tm,
	}
}

// Update handles user input and updates the breadcrumb state.
func (eb *EditableBreadcrumb) Update(delta float32) {
	// Update blink timer.
	eb.blinkCounter += delta
	if eb.blinkCounter >= 0.5 {
		eb.blinkCounter = 0
		eb.blinkState = !eb.blinkState
	}

	if eb.editMode {
		// Press ENTER to apply changes.
		if rl.IsKeyPressed(rl.KeyEnter) {
			eb.setCurrentPath(string(eb.currentPath))
			eb.editMode = false
		} else if rl.IsKeyPressed(rl.KeyEscape) {
			// Cancel editing.
			eb.editMode = false
		}

		// Left/Right arrow to move cursor.
		if rl.IsKeyPressed(rl.KeyLeft) && eb.cursorPos > 0 {
			eb.cursorPos--
		} else if rl.IsKeyPressed(rl.KeyRight) && eb.cursorPos < len(eb.currentPath) {
			eb.cursorPos++
		}

		// Backspace to delete character before cursor.
		if rl.IsKeyPressed(rl.KeyBackspace) && eb.cursorPos > 0 {
			eb.currentPath = append(eb.currentPath[:eb.cursorPos-1], eb.currentPath[eb.cursorPos:]...)
			eb.cursorPos--
		}

		// Regular character input.
		for {
			typed := rl.GetCharPressed()
			if typed == 0 {
				break
			}
			// Only accept typical printable range for demonstration.
			if typed >= 32 && typed <= 126 {
				left := eb.currentPath[:eb.cursorPos]
				right := eb.currentPath[eb.cursorPos:]
				eb.currentPath = append(left, append([]rune{typed}, right...)...)
				eb.cursorPos++
			}
		}
	}
}

// Draw renders the breadcrumb bar on the screen.
func (eb *EditableBreadcrumb) Draw() {
	// Fetch theme colors or fallback to default.
	bgColor := rl.Color{R: 0, G: 0, B: 0, A: 200}
	txtColor := rl.Color{R: 100, G: 255, B: 255, A: 255}
	cursorClr := rl.Color{R: 100, G: 255, B: 255, A: 255}

	if eb.themeManager != nil {
		theme := eb.themeManager.GetTheme()
		bgColor = theme.BackgroundColor
		txtColor = theme.TextColor
		cursorClr = theme.CursorColor
	}

	// Render background.
	rl.DrawRectangle(eb.x, eb.y, eb.width, eb.height, bgColor)

	// Render border if applicable.
	rl.DrawRectangleLines(eb.x, eb.y, eb.width, eb.height, rl.Color{R: 255, G: 255, B: 255, A: 100})

	// Draw breadcrumb text and cursor.
	if eb.editMode {
		// Draw the path in real-time.
		pathString := string(eb.currentPath)
		position := rl.Vector2{X: float32(eb.x + 10), Y: float32(eb.y) + 6}
		rl.DrawTextEx(eb.font, pathString, position, 20, 2, txtColor)

		// Calculate cursor x offset and draw it.
		cursorXOffset := eb.x + 10 + measureTextWidth([]rune(pathString[:eb.cursorPos]), eb.font)
		if eb.blinkState {
			rl.DrawRectangle(cursorXOffset, eb.y+5, 2, 20, cursorClr)
		}
	} else {
		// Not in edit mode, just draw the finalized path string.
		rl.DrawTextEx(eb.font, eb.GetCurrentPath(), rl.Vector2{X: float32(eb.x + 10), Y: float32(eb.y) + 6}, 20, 2, txtColor)
	}
}

// ToggleEditMode toggles the editing mode of the breadcrumb.
func (eb *EditableBreadcrumb) ToggleEditMode() {
	eb.editMode = !eb.editMode
	if eb.editMode {
		// Move cursor to end of current path.
		eb.cursorPos = len(eb.currentPath)
	}
}

// GetCurrentPath returns the current path as a string.
func (eb *EditableBreadcrumb) GetCurrentPath() string {
	return string(eb.currentPath)
}

// setCurrentPath updates the current path internally (trimming spaces).
func (eb *EditableBreadcrumb) setCurrentPath(path string) {
	trimmed := strings.TrimSpace(path)
	eb.currentPath = []rune(trimmed)
	eb.cursorPos = len(eb.currentPath)
}

// measureTextWidth helps measure the pixel width of a given rune slice,
// so we can position the cursor accurately.
func measureTextWidth(runes []rune, font rl.Font) int32 {
	text := string(runes)
	size := rl.MeasureTextEx(font, text, 20, 2)
	return int32(size.X)
}
