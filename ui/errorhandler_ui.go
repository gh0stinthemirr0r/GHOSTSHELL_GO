package ui

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// ErrorHandlerUI handles displaying and managing error notifications.
type ErrorHandlerUI struct {
	errors              []string  // List of error messages
	font                rl.Font   // Font for displaying errors
	x, y, width, height int32     // Position and size of the error UI panel
	dismissAfter        float32   // Time in seconds to auto-dismiss an error
	timers              []float32 // Timers tracking how long each error has been displayed
	visible             bool      // Toggle visibility of the error panel
}

// NewErrorHandlerUI initializes a new ErrorHandlerUI.
func NewErrorHandlerUI(x, y, width, height int32, font rl.Font, dismissAfter float32) *ErrorHandlerUI {
	return &ErrorHandlerUI{
		errors:       make([]string, 0),
		font:         font,
		x:            x,
		y:            y,
		width:        width,
		height:       height,
		dismissAfter: dismissAfter,
		timers:       make([]float32, 0),
		visible:      true,
	}
}

// AddError adds a new error message to the error handler.
func (e *ErrorHandlerUI) AddError(message string) {
	e.errors = append(e.errors, message)
	e.timers = append(e.timers, 0.0) // Initialize the timer for the new error
}

// Update updates the error handler state, managing auto-dismiss timers.
func (e *ErrorHandlerUI) Update(deltaTime float32) {
	if len(e.errors) == 0 {
		return
	}

	for i := 0; i < len(e.timers); i++ {
		e.timers[i] += deltaTime
		if e.timers[i] >= e.dismissAfter {
			// Remove the error and its timer
			e.errors = append(e.errors[:i], e.errors[i+1:]...)
			e.timers = append(e.timers[:i], e.timers[i+1:]...)
			i-- // Adjust index after removal
		}
	}
}

// Draw renders the error messages on the screen.
func (e *ErrorHandlerUI) Draw() {
	if !e.visible || len(e.errors) == 0 {
		return
	}

	// Background panel
	panelColor := rl.Color{R: 50, G: 50, B: 50, A: 200}
	rl.DrawRectangle(e.x, e.y, e.width, e.height, panelColor)

	// Border
	borderColor := rl.Color{R: 255, G: 50, B: 50, A: 255}
	rl.DrawRectangleLinesEx(
		rl.Rectangle{X: float32(e.x), Y: float32(e.y), Width: float32(e.width), Height: float32(e.height)},
		2, borderColor,
	)

	// Draw error messages
	txtColor := rl.Color{R: 255, G: 255, B: 255, A: 255}
	padding := int32(10)
	lineHeight := int32(rl.MeasureTextEx(e.font, "A", float32(e.font.BaseSize), 1).Y) + 4

	for i, message := range e.errors {
		if int32(i)*lineHeight+padding > e.height-padding {
			break // Stop rendering if out of bounds
		}
		pos := rl.Vector2{
			X: float32(e.x + padding),
			Y: float32(e.y + padding + int32(i)*lineHeight),
		}
		rl.DrawTextEx(e.font, message, pos, float32(e.font.BaseSize), 1, txtColor)
	}
}

// Clear removes all error messages from the handler.
func (e *ErrorHandlerUI) Clear() {
	e.errors = make([]string, 0)
	e.timers = make([]float32, 0)
}

// ToggleVisibility toggles the visibility of the error UI panel.
func (e *ErrorHandlerUI) ToggleVisibility() {
	e.visible = !e.visible
}

// Example usage of the ErrorHandlerUI in a Raylib loop
func ExampleUsage() {
	screenWidth := int32(800)
	screenHeight := int32(600)
	rl.InitWindow(screenWidth, screenHeight, "Error Handler Example")
	defer rl.CloseWindow()

	font := rl.LoadFontEx("resources/roboto.ttf", 20, nil, 0)
	if font.BaseSize == 0 {
		font = rl.GetFontDefault()
	}
	errorUI := NewErrorHandlerUI(10, 10, screenWidth-20, 100, font, 5.0)

	deltaTime := float32(0.0)

	for !rl.WindowShouldClose() {
		deltaTime = rl.GetFrameTime()

		if rl.IsKeyPressed(rl.KeyA) {
			errorUI.AddError("An example error occurred!")
		}
		if rl.IsKeyPressed(rl.KeyC) {
			errorUI.Clear()
		}
		if rl.IsKeyPressed(rl.KeyV) {
			errorUI.ToggleVisibility()
		}

		errorUI.Update(deltaTime)

		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)

		errorUI.Draw()

		rl.EndDrawing()
	}
}
