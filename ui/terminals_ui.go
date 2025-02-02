package ui

import (
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// TerminalWindow represents a single terminal window
// This version includes scrollbars, text wrapping, and additional interactive features.
type TerminalWindow struct {
	id       int
	title    string
	active   bool
	buffer   []string // Output history
	cmdInput string   // Command input
	llmInput string   // LLM interface input

	themeManager  ThemeManager
	font          rl.Font
	x, y          int32
	width, height int32

	scrollOffset int      // Vertical scrolling offset for the buffer
	hasScrollbar bool     // Whether the window has a scrollbar
	maxBuffer    int      // Maximum lines of buffer history to display
	contextMenu  []string // Context menu options
}

// NewTerminalWindow creates a new terminal window with enhanced features.
func NewTerminalWindow(
	id int,
	title string,
	tm ThemeManager,
	font rl.Font,
	x, y, width, height int32,
) *TerminalWindow {
	return &TerminalWindow{
		id:           id,
		title:        title,
		active:       true,
		themeManager: tm,
		font:         font,
		x:            x,
		y:            y,
		width:        width,
		height:       height,
		scrollOffset: 0,
		hasScrollbar: true,
		maxBuffer:    100, // Limit buffer display to 100 lines by default
		contextMenu:  []string{"Clear", "Copy", "Paste"},
	}
}

// Draw renders the terminal window, including scrollbars and optional context menus.
func (tw *TerminalWindow) Draw() {
	if tw.themeManager == nil {
		// Fallback theme if no ThemeManager is provided
		rl.DrawRectangle(tw.x, tw.y, tw.width, tw.height, rl.Color{R: 0, G: 0, B: 0, A: 200})
		return
	}

	// Fetch current theme
	theme := tw.themeManager.GetTheme()
	bgColor := colorToRaylib(theme.BackgroundColor)
	txtColor := colorToRaylib(theme.TextColor)
	borderCol := colorToRaylib(theme.BorderColor)

	// Draw background rectangle for the terminal
	rl.DrawRectangle(tw.x, tw.y, tw.width, tw.height, bgColor)

	// Draw border if the terminal is active
	if tw.active {
		rl.DrawRectangleLinesEx(
			rl.NewRectangle(float32(tw.x), float32(tw.y), float32(tw.width), float32(tw.height)),
			2, // Border thickness
			borderCol,
		)
	}

	// Draw the window title
	rl.DrawTextEx(
		tw.font,
		tw.title,
		rl.NewVector2(float32(tw.x+8), float32(tw.y+4)),
		float32(tw.font.BaseSize), 1, txtColor,
	)

	// Draw the terminal buffer
	lineHeight := float32(tw.font.BaseSize + 4)
	startY := float32(tw.y + 30)

	visibleLines := int((float32(tw.height-60) / lineHeight))
	bufferStart := len(tw.buffer) - visibleLines - tw.scrollOffset
	if bufferStart < 0 {
		bufferStart = 0
	}

	for i, line := range tw.buffer[bufferStart:] {
		if i >= visibleLines {
			break
		}
		linePos := rl.NewVector2(float32(tw.x+8), startY+(float32(i)*lineHeight))
		rl.DrawTextEx(tw.font, line, linePos, float32(tw.font.BaseSize), 1, txtColor)
	}

	// Draw scrollbar if enabled
	if tw.hasScrollbar {
		totalBufferHeight := lineHeight * float32(len(tw.buffer))
		if totalBufferHeight > float32(tw.height-60) {
			scrollBarHeight := float32(tw.height-60) * (float32(tw.height-60) / totalBufferHeight)
			scrollBarY := float32(tw.y+30) + float32(tw.scrollOffset)*lineHeight*(float32(tw.height-60)/totalBufferHeight)
			rl.DrawRectangle(tw.x+tw.width-10, int32(scrollBarY), 8, int32(scrollBarHeight), txtColor)
		}
	}

	// Draw the command input prompt
	cmdPromptY := float32(tw.y + tw.height - 30)
	rl.DrawTextEx(
		tw.font,
		"> "+tw.cmdInput,
		rl.NewVector2(float32(tw.x+8), cmdPromptY),
		float32(tw.font.BaseSize), 1, txtColor,
	)

	// Optionally draw LLM input area
	llmY := cmdPromptY - lineHeight - 5
	rl.DrawTextEx(
		tw.font,
		"LLM: "+tw.llmInput,
		rl.NewVector2(float32(tw.x+8), llmY),
		float32(tw.font.BaseSize), 1, txtColor,
	)
}

// HandleInput processes user interaction with the terminal.
func (tw *TerminalWindow) HandleInput() {
	// Scroll buffer
	if rl.IsKeyDown(rl.KeyPageUp) {
		tw.scrollOffset += 1
		if tw.scrollOffset > len(tw.buffer) {
			tw.scrollOffset = len(tw.buffer)
		}
	} else if rl.IsKeyDown(rl.KeyPageDown) {
		tw.scrollOffset -= 1
		if tw.scrollOffset < 0 {
			tw.scrollOffset = 0
		}
	}

	// Handle command input
	for {
		typed := rl.GetCharPressed()
		if typed == 0 {
			break
		}
		if typed == '\b' && len(tw.cmdInput) > 0 {
			// Handle backspace
			tw.cmdInput = tw.cmdInput[:len(tw.cmdInput)-1]
		} else if typed >= 32 && typed <= 126 {
			// Append printable ASCII characters
			tw.cmdInput += string(rune(typed))
		}
	}

	// Enter to execute command
	if rl.IsKeyPressed(rl.KeyEnter) {
		tw.ExecuteCommand()
	}
}

// ExecuteCommand processes the current command input.
func (tw *TerminalWindow) ExecuteCommand() {
	if tw.cmdInput == "" {
		return
	}
	tw.buffer = append(tw.buffer, "> "+tw.cmdInput)
	tw.buffer = append(tw.buffer, "Executed: "+tw.cmdInput)
	tw.cmdInput = ""
	if len(tw.buffer) > tw.maxBuffer {
		tw.buffer = tw.buffer[1:]
	}
}

// colorToRaylib converts image/color.RGBA to rl.Color for Raylib drawing.
func colorToRaylib(c color.RGBA) rl.Color {
	return rl.Color{R: c.R, G: c.G, B: c.B, A: c.A}
}
