package ui

import (
	"math/rand"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Theme is a simple struct representing color styling from your theme manager.
type Theme struct {
	BackgroundColor rl.Color
	TextColor       rl.Color
	BorderColor     rl.Color
	CursorColor     rl.Color
	SelectionColor  rl.Color
	NeonGlowColor   rl.Color
}

// ThemeManager is an interface for obtaining the current theme.
// If you already have one, adjust accordingly.
type ThemeManager interface {
	GetTheme() Theme
}

// Particle represents a single particle’s position, velocity, color, and properties.
type Particle struct {
	x, y      float32 // Position
	dx, dy    float32 // Velocity
	color     rl.Color
	size      float32
	lifetime  float32 // Lifetime of the particle
	fadeSpeed float32 // Speed of fading
}

// ParticleSystem holds a list of particles, references a ThemeManager (for color),
// and has parameters for controlling behavior (e.g., speed, count, effects).
type ParticleSystem struct {
	particles      []*Particle
	themeManager   ThemeManager
	speed          float32 // Base speed multiplier
	enabled        bool    // Are particle effects enabled?
	maxParticles   int     // Maximum number of particles
	enableFading   bool    // Whether particles fade out over time
	gravityEnabled bool    // Apply gravity to particles
	gravity        float32 // Gravity strength
}

// NewParticleSystem initializes a ParticleSystem with the desired number of particles,
// a speed multiplier, and theming.
func NewParticleSystem(count int, speed float32, tm ThemeManager, enabled bool) *ParticleSystem {
	ps := &ParticleSystem{
		themeManager:   tm,
		speed:          speed,
		enabled:        enabled,
		maxParticles:   count,
		enableFading:   true,
		gravityEnabled: true,
		gravity:        0.1,
	}
	ps.generateParticles(count)
	return ps
}

// generateParticles creates an initial batch of particles at random positions/speeds.
func (ps *ParticleSystem) generateParticles(count int) {
	rand.Seed(time.Now().UnixNano())
	ps.particles = make([]*Particle, count)

	var colorFromTheme rl.Color
	if ps.themeManager != nil {
		colorFromTheme = ps.themeManager.GetTheme().NeonGlowColor
	} else {
		colorFromTheme = rl.Color{R: 255, G: 255, B: 255, A: 255} // Fallback color
	}

	for i := 0; i < count; i++ {
		x := rand.Float32() * float32(rl.GetScreenWidth())
		y := rand.Float32() * float32(rl.GetScreenHeight())
		dx := (rand.Float32()*2 - 1) * ps.speed
		dy := (rand.Float32()*2 - 1) * ps.speed
		size := rand.Float32()*2 + 2          // Random size between 2-4 px
		lifetime := rand.Float32()*5 + 1      // Lifetime between 1-5 seconds
		fadeSpeed := rand.Float32()*0.5 + 0.1 // Fade speed

		ps.particles[i] = &Particle{
			x:         x,
			y:         y,
			dx:        dx,
			dy:        dy,
			color:     colorFromTheme,
			size:      size,
			lifetime:  lifetime,
			fadeSpeed: fadeSpeed,
		}
	}
}

// Update moves each particle according to its velocity and applies effects like fading and gravity.
func (ps *ParticleSystem) Update(deltaTime float32) {
	if !ps.enabled {
		return
	}

	screenW := float32(rl.GetScreenWidth())
	screenH := float32(rl.GetScreenHeight())

	for _, p := range ps.particles {
		p.x += p.dx * deltaTime
		p.y += p.dy * deltaTime

		// Apply gravity if enabled
		if ps.gravityEnabled {
			p.dy += ps.gravity * deltaTime
		}

		// Wrap particles around screen edges
		if p.x < 0 {
			p.x = screenW
		} else if p.x > screenW {
			p.x = 0
		}
		if p.y < 0 {
			p.y = screenH
		} else if p.y > screenH {
			p.y = 0
		}

		// Reduce lifetime and fade particles if enabled
		if ps.enableFading {
			p.lifetime -= deltaTime
			if p.lifetime > 0 {
				p.color.A = uint8(float32(p.color.A) * (1 - p.fadeSpeed*deltaTime))
			} else {
				p.color.A = 0 // Fully transparent
			}
		}
	}
}

// Draw renders each particle on the screen.
func (ps *ParticleSystem) Draw() {
	if !ps.enabled {
		return
	}

	for _, p := range ps.particles {
		if p.color.A > 0 { // Only draw particles that are not fully transparent
			rl.DrawCircleV(rl.NewVector2(p.x, p.y), p.size, p.color)
		}
	}
}

// Enable toggles particle effect usage.
func (ps *ParticleSystem) Enable(enable bool) {
	ps.enabled = enable
}

// ResetParticles regenerates the particle system.
func (ps *ParticleSystem) ResetParticles(count int) {
	ps.generateParticles(count)
}

// SetGravity enables/disables gravity and adjusts its strength.
func (ps *ParticleSystem) SetGravity(enabled bool, strength float32) {
	ps.gravityEnabled = enabled
	ps.gravity = strength
}

// SetFading toggles fading effects for particles.
func (ps *ParticleSystem) SetFading(enable bool) {
	ps.enableFading = enable
}

// clamp8 is a small helper to clamp int32 to 0-255 range for color channel.
func clamp8(val int32) uint8 {
	if val < 0 {
		return 0
	}
	if val > 255 {
		return 255
	}
	return uint8(val)
}
