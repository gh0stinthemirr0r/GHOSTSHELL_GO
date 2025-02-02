package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/jung-kurt/gofpdf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/crypto/ssh"
	"golang.org/x/exp/rand"

	// Hypothetical modules for quantum-safe crypto & key management

	"yourproject/metrics" // for auth failure increments or similar
	"yourproject/oqs_vault"
)

const (
	logDir       = "ghostshell/logging"
	reportDir    = "ghostshell/reporting"
	fontSize     = 24
	windowWidth  = 1280
	windowHeight = 720
	maxParticles = 50
)

// -------------- Logging --------------

var logger *zap.Logger

func setupLogger() error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(logDir, fmt.Sprintf("ssh_log_%s.log", timestamp))

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

// -------------- Raylib Particles --------------

type Particle struct {
	x, y   float32
	dx, dy float32
	color  rl.Color
}

func generateParticles(count int) []*Particle {
	ps := make([]*Particle, count)
	rng := rand.New(rand.NewSource(uint64(time.Now().UnixNano())))
	for i := 0; i < count; i++ {
		ps[i] = &Particle{
			x:  float32(rng.Intn(windowWidth)),
			y:  float32(rng.Intn(windowHeight)),
			dx: (rng.Float32()*2 - 1) * 2,
			dy: (rng.Float32()*2 - 1) * 2,
			color: rl.NewColor(
				uint8(rng.Intn(256)),
				uint8(rng.Intn(256)),
				uint8(rng.Intn(256)),
				255,
			),
		}
	}
	return ps
}

func updateParticles(ps []*Particle) {
	for _, p := range ps {
		p.x += p.dx
		p.y += p.dy

		if p.x < 0 || p.x > float32(windowWidth) {
			p.dx *= -1
		}
		if p.y < 0 || p.y > float32(windowHeight) {
			p.dy *= -1
		}
		rl.DrawCircle(int32(p.x), int32(p.y), 5, p.color)
	}
}

// -------------- Terminal UI --------------

type Terminal struct {
	font rl.Font
}

func newTerminal() (*Terminal, error) {
	fontPath := "resources/futuristic_font.ttf"
	font := rl.LoadFontEx(fontPath, fontSize, nil, 0)
	if font.BaseSize == 0 {
		logger.Warn("Failed to load custom font, fallback to default", zap.String("font", fontPath))
		font = rl.GetFontDefault()
	}
	return &Terminal{font: font}, nil
}

// -------------- SSH Manager (Quantum-Safe) --------------

type SSHManager struct {
	serverConfig *ssh.ServerConfig
	address      string
	connections  []string // track remote addresses
	commands     []string // track executed commands
	mu           sync.Mutex
	// concurrency
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	// optional metrics
	authFailuresCounter metrics.Counter
}

// NewSSHManager sets up a post-quantum SSH server config
func NewSSHManager(address string) (*SSHManager, error) {
	// generate logger
	if err := setupLogger(); err != nil {
		return nil, fmt.Errorf("logger setup failed: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sm := &SSHManager{
		address:     address,
		connections: []string{},
		commands:    []string{},
		ctx:         ctx,
		cancel:      cancel,
	}

	if err := sm.setupServerConfig(); err != nil {
		return nil, err
	}
	sm.authFailuresCounter = metrics.NewCounter("ssh_auth_failures", "Count of SSH authentication failures")

	return sm, nil
}

// setupServerConfig loads a post-quantum private key from oqs_vault & configures SSH
func (sm *SSHManager) setupServerConfig() error {
	// load a post-quantum private key from the vault
	pqPrivateKey, err := oqs_vault.GetKeyManager().GeneratePQSPrivateKey()
	if err != nil {
		logger.Error("Failed to generate PQ private key", zap.Error(err))
		return fmt.Errorf("failed to generate post-quantum key: %w", err)
	}

	signer, err := ssh.NewSignerFromKey(pqPrivateKey)
	if err != nil {
		logger.Error("Failed to create SSH signer from PQ key", zap.Error(err))
		return fmt.Errorf("failed to create signer: %w", err)
	}

	// build server config
	sm.serverConfig = &ssh.ServerConfig{
		NoClientAuth: false, // or true if no auth
	}
	sm.serverConfig.AddHostKey(signer)

	// Example post-quantum KEX + ciphers (some are hypothetical or require custom patch sets)
	sm.serverConfig.Config.KeyExchanges = []string{
		"oqs-kyber-512-sha3-256", // example, not a standard upstream
		"curve25519-sha256",      // fallback
	}
	sm.serverConfig.Config.Ciphers = []string{
		"aes256-gcm@openssh.com",
		"chacha20-poly1305@openssh.com",
	}

	return nil
}

// Start runs a goroutine that listens for inbound SSH connections
func (sm *SSHManager) Start() error {
	ln, err := net.Listen("tcp", sm.address)
	if err != nil {
		logger.Error("Failed to listen on address", zap.String("address", sm.address), zap.Error(err))
		return err
	}
	logger.Info("SSH server listening", zap.String("address", sm.address))

	sm.wg.Add(1)
	go sm.acceptLoop(ln)
	return nil
}

// acceptLoop handles incoming connections until context is canceled
func (sm *SSHManager) acceptLoop(ln net.Listener) {
	defer sm.wg.Done()
	defer ln.Close()

	for {
		select {
		case <-sm.ctx.Done():
			logger.Info("Stop accepting new SSH connections (context canceled)")
			return
		default:
		}

		ln.SetDeadline(time.Now().Add(2 * time.Second))
		conn, err := ln.Accept()
		if err != nil {
			// check for net timeout
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			logger.Warn("Accept error", zap.Error(err))
			continue
		}
		sm.wg.Add(1)
		go sm.handleConnection(conn)
	}
}

// handleConnection performs the SSH handshake & channel management
func (sm *SSHManager) handleConnection(conn net.Conn) {
	defer sm.wg.Done()
	defer conn.Close()

	sshConn, channels, requests, err := ssh.NewServerConn(conn, sm.serverConfig)
	if err != nil {
		logger.Warn("SSH handshake failed", zap.Error(err))
		sm.authFailuresCounter.Increment(1.0)
		return
	}

	remote := sshConn.RemoteAddr().String()
	logger.Info("SSH connection established", zap.String("remote", remote))

	// track connection
	sm.mu.Lock()
	sm.connections = append(sm.connections, remote)
	sm.mu.Unlock()

	go ssh.DiscardRequests(requests)

	for newCh := range channels {
		switch newCh.ChannelType() {
		case "session":
			ch, reqs, err := newCh.Accept()
			if err != nil {
				logger.Warn("Channel accept error", zap.Error(err))
				continue
			}
			go sm.handleSession(ch, reqs, remote)
		default:
			newCh.Reject(ssh.UnknownChannelType, "unsupported channel type")
		}
	}
}

// handleSession processes "exec" requests
func (sm *SSHManager) handleSession(ch ssh.Channel, reqs <-chan *ssh.Request, remote string) {
	defer ch.Close()

	for req := range reqs {
		switch req.Type {
		case "exec":
			cmd := string(req.Payload[4:]) // typical offset to skip exec length
			logger.Info("Executing command", zap.String("command", cmd), zap.String("remote", remote))
			sm.mu.Lock()
			sm.commands = append(sm.commands, fmt.Sprintf("Remote: %s, CMD: %s", remote, cmd))
			sm.mu.Unlock()

			// write some response
			ch.Write([]byte(fmt.Sprintf("Executed: %s\n", cmd)))
			req.Reply(true, nil)
		default:
			req.Reply(false, nil)
		}
	}
}

// Stop signals the SSH manager to shut down
func (sm *SSHManager) Stop() {
	sm.cancel()
	sm.wg.Wait()
}

// -------------- Raylib Particles UI --------------

func main() {
	// 1) Setup logging
	if err := setupLogger(); err != nil {
		fmt.Printf("Logger setup error: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// 2) Initialize raylib
	rl.InitWindow(windowWidth, windowHeight, "SSH Manager (Post-Quantum)")
	rl.SetTargetFPS(60)
	runtime.LockOSThread()

	t, err := newTerminal()
	if err != nil {
		logger.Fatal("Failed to create Terminal UI", zap.Error(err))
	}
	defer t.Shutdown()

	ps := generateParticles(maxParticles)

	// 3) Initialize metrics & manager
	metricsOverlay := metrics.NewMetricsOverlay() // hypothetical
	manager, err := NewSSHManager(":2222")
	if err != nil {
		logger.Fatal("Failed to init SSH manager", zap.Error(err))
	}

	// 4) Start the SSH server
	if err := manager.Start(); err != nil {
		logger.Fatal("Failed to start SSH server", zap.Error(err))
	}

	// 5) Main loop with graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	mainCtx, mainCancel := context.WithCancel(context.Background())

	for !rl.WindowShouldClose() && mainCtx.Err() == nil {
		select {
		case <-sigChan:
			logger.Info("Got shutdown signal")
			mainCancel()
		default:
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		updateParticles(ps)
		// Draw text
		rl.DrawTextEx(t.font, "SSH Manager (Post-Quantum)", rl.NewVector2(40, 40), float32(fontSize), 2, rl.White)
		localTime := time.Now().Format("15:04:05")
		rl.DrawTextEx(t.font, fmt.Sprintf("Local Time: %s", localTime), rl.NewVector2(40, 80), 20, 2, rl.White)
		rl.DrawTextEx(t.font, "Press ESC to Exit", rl.NewVector2(40, 110), 20, 2, rl.White)

		if rl.IsKeyPressed(rl.KeyEscape) {
			mainCancel()
		}

		rl.EndDrawing()
	}

	// 6) Stop the SSH manager
	manager.Stop()

	// 7) Generate a report with connection & command logs
	manager.mu.Lock()
	connections := append([]string{}, manager.connections...)
	commands := append([]string{}, manager.commands...)
	manager.mu.Unlock()

	lines := []string{"=== SSH Connections ==="}
	lines = append(lines, connections...)
	lines = append(lines, "=== Commands Executed ===")
	lines = append(lines, commands...)

	if err := generateReports(lines); err != nil {
		logger.Error("Failed to generate reports", zap.Error(err))
	}

	logger.Info("Application shutting down gracefully.")
	rl.CloseWindow()
}

// -------------- generateReports writes CSV & PDF --------------

func generateReports(data []string) error {
	if len(data) == 0 {
		logger.Warn("No data to report on, skipping generateReports")
		return nil
	}
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		logger.Error("Failed to create report dir", zap.Error(err))
		return err
	}
	timestamp := time.Now().Format("20060102_150405")
	csvPath := filepath.Join(reportDir, fmt.Sprintf("ssh_report_%s.csv", timestamp))
	pdfPath := filepath.Join(reportDir, fmt.Sprintf("ssh_report_%s.pdf", timestamp))

	// CSV
	f, err := os.Create(csvPath)
	if err != nil {
		logger.Error("Failed to create CSV file", zap.Error(err))
		return err
	}
	defer f.Close()

	for _, line := range data {
		fmt.Fprintln(f, line)
	}
	logger.Info("CSV report generated", zap.String("file", csvPath))

	// PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "SSH Manager Post-Quantum Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	for _, line := range data {
		pdf.MultiCell(190, 6, line, "", "", false)
	}
	if err := pdf.OutputFileAndClose(pdfPath); err != nil {
		logger.Error("Failed to write PDF file", zap.Error(err))
		return err
	}
	logger.Info("PDF report generated", zap.String("file", pdfPath))
	return nil
}

// MyCertManager or Key loading logic from vault, if needed
// ...
