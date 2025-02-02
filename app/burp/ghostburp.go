package burp

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/jung-kurt/gofpdf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"golang.org/x/exp/rand"

	"ghostshell/options"
)

// -------------- Constants --------------

const (
	ScreenWidth   = 1280
	ScreenHeight  = 720
	FontPointSize = 24
	ParticleSize  = 5
	ReportDir     = "ghostshell/reporting"
	LogDir        = "ghostshell/logging"
)

// -------------- Logging --------------

var logger *zap.Logger

func setupLogger() error {
	if err := os.MkdirAll(LogDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	timestamp := time.Now().Format("20060102_150405") // e.g. 20261009_153012
	logFile := filepath.Join(LogDir, fmt.Sprintf("burpcollab_log_%s.log", timestamp))

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logFile, "stdout"}

	var err error
	logger, err = cfg.Build()
	if err != nil {
		return fmt.Errorf("failed to build logger: %v", err)
	}
	return nil
}

// -------------- Particles --------------

type Particle struct {
	x, y   float32
	dx, dy float32
	color  rl.Color
}

func generateParticles(count int) []*Particle {
	rand.Seed(uint64(time.Now().UnixNano()))
	ps := make([]*Particle, count)
	for i := 0; i < count; i++ {
		ps[i] = &Particle{
			x:  float32(rand.Intn(ScreenWidth)),
			y:  float32(rand.Intn(ScreenHeight)),
			dx: (rand.Float32()*2 - 1) * 2,
			dy: (rand.Float32()*2 - 1) * 2,
			color: rl.NewColor(
				uint8(rand.Intn(256)),
				uint8(rand.Intn(256)),
				uint8(rand.Intn(256)),
				255,
			),
		}
	}
	return ps
}

// -------------- Menu --------------

type MenuManager struct {
	items       []string
	currentItem int
	terminal    *Terminal
}

func (m *MenuManager) Update() {
	// Navigate menu with up/down
	if rl.IsKeyPressed(rl.KeyDown) {
		m.currentItem++
		if m.currentItem >= len(m.items) {
			m.currentItem = 0
		}
	}
	if rl.IsKeyPressed(rl.KeyUp) {
		m.currentItem--
		if m.currentItem < 0 {
			m.currentItem = len(m.items) - 1
		}
	}
	// Select with Enter
	if rl.IsKeyPressed(rl.KeyEnter) {
		m.executeItem()
	}
}

func (m *MenuManager) Draw() {
	baseY := float32(120)
	spacing := float32(30)
	for i, item := range m.items {
		color := rl.White
		if i == m.currentItem {
			color = rl.Yellow
		}
		rl.DrawText(item, 50, int32(baseY+float32(i)*spacing), 24, color)
	}
}

func (m *MenuManager) executeItem() {
	switch m.items[m.currentItem] {
	case "Start Collaborator Test":
		m.terminal.startCollaboratorTest()
	case "Generate Report":
		if err := m.terminal.generateReport(); err != nil {
			logger.Error("Report generation error", zap.Error(err))
		}
	case "Exit":
		m.terminal.shutdown()
	default:
		logger.Warn("Unknown menu item selected", zap.String("item", m.items[m.currentItem]))
	}
}

// -------------- Terminal --------------

type Terminal struct {
	font        rl.Font
	particles   []*Particle
	menuManager *MenuManager

	isScanning bool
	mu         sync.Mutex
	// results from the collaborator test
	results []string
}

func NewTerminal(parsedOptions *options.Options) (*Terminal, error) {
	if err := setupLogger(); err != nil {
		return nil, fmt.Errorf("failed to setup logger: %v", err)
	}

	fontPath := "resources/futuristic_font.ttf"
	font := rl.LoadFontEx(fontPath, FontPointSize, nil, 0)
	if font.BaseSize == 0 {
		logger.Warn("Failed to load custom font, using default", zap.String("font", fontPath))
		font = rl.GetFontDefault()
	}

	// e.g. parseOptions might contain a field for particleCount
	count := 50
	particles := generateParticles(count)

	menuItems := []string{"Start Collaborator Test", "Generate Report", "Exit"}
	menu := &MenuManager{
		items:       menuItems,
		currentItem: 0,
	}
	t := &Terminal{
		font:        font,
		particles:   particles,
		menuManager: menu,
		isScanning:  false,
		results:     []string{},
	}
	menu.terminal = t

	logger.Info("Terminal created successfully")
	return t, nil
}

func (t *Terminal) Update() {
	// update particles if not scanning
	if !t.isScanning {
		t.mu.Lock()
		for _, p := range t.particles {
			p.x += p.dx
			p.y += p.dy
			if p.x < 0 || p.x > ScreenWidth {
				p.dx *= -1
			}
			if p.y < 0 || p.y > ScreenHeight {
				p.dy *= -1
			}
		}
		t.mu.Unlock()
	}
	// update menu
	t.menuManager.Update()
}

func (t *Terminal) Draw() {
	rl.BeginDrawing()
	rl.ClearBackground(rl.DarkBlue)

	// draw the particles
	t.mu.Lock()
	for _, p := range t.particles {
		rl.DrawCircle(int32(p.x), int32(p.y), ParticleSize, p.color)
	}
	t.mu.Unlock()

	// title
	rl.DrawTextEx(t.font, "BURP Collaborator Scanner", rl.NewVector2(40, 40), float32(FontPointSize), 2, rl.White)

	// if scanning
	if t.isScanning {
		rl.DrawTextEx(t.font, "Collaborator test in progress...", rl.NewVector2(40, 90), 24, 2, rl.Green)
	}

	// menu
	t.menuManager.Draw()

	rl.EndDrawing()
}

// shutdown cleans up
func (t *Terminal) shutdown() {
	logger.Sync()
	rl.CloseWindow()
	os.Exit(0)
}

// startCollaboratorTest concurrency-based test
func (t *Terminal) startCollaboratorTest() {
	if t.isScanning {
		logger.Warn("Collaborator test already in progress")
		return
	}
	t.isScanning = true
	t.results = []string{}

	go func() {
		start := time.Now()
		logger.Info("Starting collaborator concurrency test...")

		// For demonstration, let's do concurrency with random "targets"
		cTargets := []string{"collab1.burpcollab.net", "collab2.burpcollab.net", "test.burpcollab.net"}
		var wg sync.WaitGroup

		for _, c := range cTargets {
			wg.Add(1)
			go func(target string) {
				defer wg.Done()
				success, msg := collaboratorCheck(target)
				line := fmt.Sprintf("%s => %s", target, msg)
				t.mu.Lock()
				t.results = append(t.results, line)
				t.mu.Unlock()
				if success {
					logger.Info("Collaborator success", zap.String("target", target))
				} else {
					logger.Error("Collaborator fail", zap.String("target", target), zap.String("msg", msg))
				}
			}(c)
		}
		wg.Wait()

		dur := time.Since(start).Seconds()
		logger.Info("Collaborator concurrency test done", zap.Float64("duration_s", dur))

		// done scanning
		t.mu.Lock()
		t.isScanning = false
		t.mu.Unlock()
	}()
}

// collaboratorCheck simulates an out-of-band DNS or HTTP check to Burp collaborator
func collaboratorCheck(target string) (bool, string) {
	// stub logic: random success/failure
	// real logic might do e.g. an HTTP request to "http://<unique-id>.burpcollab.net" or DNS
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
	r := rand.Float32()
	if r < 0.4 {
		return false, "No out-of-band interaction recorded"
	}
	return true, "Interaction recorded"
}

// generateReport writes CSV & PDF
func (t *Terminal) generateReport() error {
	if len(t.results) == 0 {
		logger.Warn("No collaborator results to report on")
		return nil
	}

	if err := os.MkdirAll(ReportDir, 0755); err != nil {
		logger.Error("Failed to create report dir", zap.Error(err))
		return err
	}
	timestamp := time.Now().Format("20060102_150405")
	csvFile := filepath.Join(ReportDir, fmt.Sprintf("burp_collab_report_%s.csv", timestamp))
	pdfFile := filepath.Join(ReportDir, fmt.Sprintf("burp_collab_report_%s.pdf", timestamp))

	// CSV
	if err := t.writeCSV(csvFile); err != nil {
		return err
	}
	// PDF
	if err := t.writePDF(pdfFile); err != nil {
		return err
	}
	return nil
}

func (t *Terminal) writeCSV(path string) error {
	f, err := os.Create(path)
	if err != nil {
		logger.Error("Failed to create CSV file", zap.Error(err))
		return err
	}
	defer f.Close()

	// simple lines: "target => msg"
	for _, line := range t.results {
		f.WriteString(line + "\n")
	}

	logger.Info("CSV report generated", zap.String("file", path))
	return nil
}

func (t *Terminal) writePDF(path string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Burp Collaborator Report")
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 12)
	for _, line := range t.results {
		tokens := strings.SplitN(line, " => ", 2)
		if len(tokens) < 2 {
			tokens = append(tokens, "")
		}
		// e.g., tokens[0] is target, tokens[1] is result
		pdf.Cell(60, 8, tokens[0])
		x := pdf.GetX()
		y := pdf.GetY()
		pdf.MultiCell(120, 8, tokens[1], "", "", false)
		pdf.SetXY(x+60+120, y)
		pdf.Ln(8)
	}

	if err := pdf.OutputFileAndClose(path); err != nil {
		logger.Error("Failed to write PDF file", zap.Error(err))
		return err
	}
	logger.Info("PDF report generated", zap.String("file", path))
	return nil
}

// -------------- Entry Point --------------

func MainBurpScanner(parsedOptions *options.Options) {
	// Setup signal ctx
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Received shutdown signal, shutting down gracefully")
		cancel()
		rl.CloseWindow()
		os.Exit(0)
	}()

	// init
	term, err := NewTerminal(parsedOptions)
	if err != nil {
		fmt.Printf("Terminal init fail: %v\n", err)
		os.Exit(1)
	}

	rl.InitWindow(ScreenWidth, ScreenHeight, "Burp Collaborator Scanner")
	rl.SetTargetFPS(60)
	defer rl.CloseWindow()

	for !rl.WindowShouldClose() && ctx.Err() == nil {
		term.Update()
		term.Draw()
	}

	term.shutdown()
}
