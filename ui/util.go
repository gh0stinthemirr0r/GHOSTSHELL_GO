package ui

import (
	"image/color"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// DrawNeonGlow draws a glowing outline around the specified rectangle with multiple layers for a softer glow effect.
func DrawNeonGlow(x, y, width, height int32, intensity int, color rl.Color) {
	for i := 0; i < intensity; i++ {
		alpha := uint8(float64(color.A) * (1.0 - float64(i)/float64(intensity)))
		glowColor := rl.Color{R: color.R, G: color.G, B: color.B, A: alpha}
		rl.DrawRectangleLinesEx(rl.Rectangle{
			X:      float32(x - int32(i)),
			Y:      float32(y - int32(i)),
			Width:  float32(width + 2*int32(i)),
			Height: float32(height + 2*int32(i)),
		}, 1, glowColor)
	}
}

// DrawRoundedRectangle draws a rectangle with rounded corners.
func DrawRoundedRectangle(x, y, width, height, radius int32, color rl.Color) {
	if radius*2 > width || radius*2 > height {
		radius = int32(math.Min(float64(width/2), float64(height/2)))
	}

	// Draw the rounded corners
	rl.DrawCircle(x+radius, y+radius, float32(radius), color)
	rl.DrawCircle(x+width-radius-1, y+radius, float32(radius), color)
	rl.DrawCircle(x+radius, y+height-radius-1, float32(radius), color)
	rl.DrawCircle(x+width-radius-1, y+height-radius-1, float32(radius), color)

	// Draw the edges
	rl.DrawRectangle(x+radius, y, width-2*radius, height, color)
	rl.DrawRectangle(x, y+radius, radius, height-2*radius, color)
	rl.DrawRectangle(x+width-radius, y+radius, radius, height-2*radius, color)
}

// DrawGradientRectangle draws a rectangle with a vertical gradient.
func DrawGradientRectangle(x, y, width, height int32, topColor, bottomColor rl.Color) {
	for i := int32(0); i < height; i++ {
		factor := float32(i) / float32(height)
		interpolatedColor := rl.Color{
			R: uint8(float32(topColor.R)*(1.0-factor) + float32(bottomColor.R)*factor),
			G: uint8(float32(topColor.G)*(1.0-factor) + float32(bottomColor.G)*factor),
			B: uint8(float32(topColor.B)*(1.0-factor) + float32(bottomColor.B)*factor),
			A: uint8(float32(topColor.A)*(1.0-factor) + float32(bottomColor.A)*factor),
		}
		rl.DrawRectangle(x, y+i, width, 1, interpolatedColor)
	}
}

// ClampColorValue clamps a color component value to the range [0, 255].
func ClampColorValue(value int32) uint8 {
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return uint8(value)
}

// AdjustColorBrightness adjusts the brightness of a color by a given factor.
func AdjustColorBrightness(color rl.Color, factor float32) rl.Color {
	return rl.Color{
		R: ClampColorValue(int32(float32(color.R) * factor)),
		G: ClampColorValue(int32(float32(color.G) * factor)),
		B: ClampColorValue(int32(float32(color.B) * factor)),
		A: color.A,
	}
}

// DrawShadowedText draws text with a shadow underneath for better readability.
func DrawShadowedText(font rl.Font, text string, pos rl.Vector2, fontSize float32, spacing float32, textColor, shadowColor rl.Color) {
	shadowOffset := rl.Vector2{X: pos.X + 2, Y: pos.Y + 2}
	rl.DrawTextEx(font, text, shadowOffset, fontSize, spacing, shadowColor)
	rl.DrawTextEx(font, text, pos, fontSize, spacing, textColor)
}

// DrawCircularProgressBar draws a circular progress bar.
func DrawCircularProgressBar(center rl.Vector2, radius float32, progress float32, barColor rl.Color, backgroundColor rl.Color) {
	// Clamp progress between 0 and 1
	if progress < 0 {
		progress = 0
	} else if progress > 1 {
		progress = 1
	}

	// Draw the background circle
	rl.DrawCircleV(center, radius, backgroundColor)

	// Draw the progress arc
	rl.DrawCircleSector(center, radius, 270, 270+360*progress, 64, barColor)
}

// colorToRaylib converts an image/color.RGBA to an rl.Color.
func colorToRaylib(c color.RGBA) rl.Color {
	return rl.Color{
		R: c.R,
		G: c.G,
		B: c.B,
		A: c.A,
	}
}
