package graphics

import (
	"image/color"
	"log"
	"runtime"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ThemeManager is assumed to be from your theme_manager.go
// and is used here for real-time theme updates.
type ThemeManager interface {
	GetTheme() Theme
	Subscribe() <-chan Theme
	Unsubscribe(ch <-chan Theme)
}

// Renderer handles drawing your Raylib-based UI
type Renderer struct {
	logger       *log.Logger
	isRunning    bool
	themeManager ThemeManager

	currentTheme Theme
	font         rl.Font // loaded font for text rendering
	themeCh      <-chan Theme
}

// init locks the main thread for Raylib since OpenGL contexts must run on the main thread.
func init() {
	runtime.LockOSThread()
}

// NewRenderer initializes the Raylib renderer.
// We accept a ThemeManager so we can subscribe to theme changes.
func NewRenderer(logger *log.Logger, themeManager ThemeManager) (*Renderer, error) {
	rl.InitWindow(800, 600, "GhostShell Terminal - Raylib")
	rl.SetTargetFPS(60)

	logger.Println("Raylib renderer initialized successfully")

	r := &Renderer{
		logger:       logger,
		isRunning:    false,
		themeManager: themeManager,
		currentTheme: themeManager.GetTheme(),
	}

	// Subscribe to theme changes so the renderer can update in real-time
	r.themeCh = themeManager.Subscribe()
	go r.handleThemeUpdates()

	// Attempt to load a custom font from the theme settings (if desired).
	// For example, if your theme.yaml says:
	//   font:
	//     family: "JetBrains Mono"
	//     size: 14
	// ... you'll need to map that to an actual TTF file. This is just an example path:
	fontFilePath := "./fonts/JetBrainsMono-Regular.ttf"
	fontSize := int32(14) // You might read from your full config if needed

	if rl.IsFileDropped() { /* optional file drop usage */
	}

	font := rl.LoadFontEx(fontFilePath, fontSize, nil, 0)
	if font.Texture.ID == 0 {
		logger.Printf("Warning: could not load font at path '%s'. Using default.\n", fontFilePath)
		r.font = rl.GetFontDefault()
	} else {
		r.font = font
		rl.SetTextureFilter(r.font.Texture, rl.FilterBilinear)
		logger.Printf("Loaded font '%s' (size %d)\n", fontFilePath, fontSize)
	}

	return r, nil
}

// handleThemeUpdates listens for new themes from the subscription channel
// and updates the renderer’s current theme accordingly.
func (r *Renderer) handleThemeUpdates() {
	for newTheme := range r.themeCh {
		r.logger.Println("Renderer received a new theme!")
		r.currentTheme = newTheme
		// If your theme also contains new font info, you could re-load fonts here.
	}
}

// Render starts the main rendering loop
func (r *Renderer) Render() {
	r.isRunning = true
	r.logger.Println("Starting render loop")

	for !rl.WindowShouldClose() && r.isRunning {
		// Start drawing
		rl.BeginDrawing()

		// Clear the background to the current theme’s background color
		rl.ClearBackground(colorToRaylib(r.currentTheme.BackgroundColor))

		// Example usage: draw text in the theme’s text color using the loaded font
		foregroundClr := colorToRaylib(r.currentTheme.TextColor)
		text := "Welcome to GhostShell Terminal!"
		textSize := float32(r.font.BaseSize) // or any scaling factor
		rl.DrawTextEx(r.font, text, rl.NewVector2(190, 200), textSize, 2, foregroundClr)

		// Example shape using theme’s border color
		borderClr := colorToRaylib(r.currentTheme.BorderColor)
		rl.DrawCircle(400, 300, 50, borderClr)

		// TODO: Implement any particle effects, glow, or animations from your YAML
		// For instance:
		// if animations.enable_particle_effects { ... }
		// if animations.enable_glow_effect { ... }

		// End drawing
		rl.EndDrawing()
	}

	r.logger.Println("Render loop ended")
}

// Shutdown stops the rendering, unsubscribes from theme updates, and closes the Raylib window.
func (r *Renderer) Shutdown() {
	r.isRunning = false

	// Unsubscribe from theme updates
	r.themeManager.Unsubscribe(r.themeCh)

	// If we loaded a custom font, unload it (if desired).
	// rl.UnloadFont(r.font)  // optional, can reduce memory usage

	rl.CloseWindow()
	r.logger.Println("Raylib renderer shut down")
}

// colorToRaylib is a small helper to convert Go’s color.RGBA to Raylib’s rl.Color
func colorToRaylib(c color.RGBA) rl.Color {
	return rl.Color{
		R: c.R,
		G: c.G,
		B: c.B,
		A: c.A,
	}
}
