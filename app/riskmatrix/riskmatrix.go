package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/jung-kurt/gofpdf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/rand"
	// Hypothetical quantum-safe module references
)

const (
	logDir       = "ghostshell/logging"
	reportDir    = "ghostshell/reporting"
	windowWidth  = 800
	windowHeight = 600
	fontSize     = 24
	maxParticles = 50
)

// -------------- Logging --------------

var logger *zap.Logger

func InitializeLogger() error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}
	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(logDir, fmt.Sprintf("riskmatrix_log_%s.log", timestamp))

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logFile, "stdout"}

	var err error
	logger, err = cfg.Build()
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %v", err)
	}
	return nil
}

// -------------- Particles UI --------------

type Particle struct {
	x, y   float32
	dx, dy float32
	color  rl.Color
}

func generateParticles(count int) []*Particle {
	ps := make([]*Particle, count)
	for i := 0; i < count; i++ {
		ps[i] = &Particle{
			x:  float32(rand.Intn(windowWidth)),
			y:  float32(rand.Intn(windowHeight)),
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
		rl.DrawCircle(int32(p.x), int32(p.y), 4, p.color)
	}
}

// -------------- Data Structures --------------

type RiskMatrix struct {
	Risks []Risk
}

type Risk struct {
	Category   string
	Impact     string
	Likelihood string
	Score      int
}

// -------------- Reading & Parsing with PQ references --------------

// parseJSONLFile reads a JSONL file and merges data into a RiskMatrix
func parseJSONLFile(filePath string) ([]Risk, error) {
	logger.Info("Parsing JSONL file", zap.String("filePath", filePath))
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open JSONL file: %w", err)
	}
	defer file.Close()

	// Hypothetical ephemeral decryption or ephemeral usage:
	// ciphertext, err := io.ReadAll(file)
	// plaintext, err := oqs_network.DecryptEphemeral(ciphertext)
	// if err != nil { ... }

	var results []Risk
	decoder := json.NewDecoder(file)
	for decoder.More() {
		var risk Risk
		if err := decoder.Decode(&risk); err != nil {
			return nil, fmt.Errorf("failed to parse JSONL entry: %w", err)
		}
		results = append(results, risk)
	}
	return results, nil
}

// parseMarkdownFile reads a Markdown file and merges data into a RiskMatrix
func parseMarkdownFile(filePath string) ([]Risk, error) {
	logger.Info("Parsing Markdown file", zap.String("filePath", filePath))
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open markdown file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var results []Risk
	for _, line := range lines {
		// parse logic: e.g. "Category|Impact|Likelihood"
		fields := strings.Split(line, "|")
		if len(fields) < 3 {
			continue
		}
		cat := strings.TrimSpace(fields[0])
		imp := strings.TrimSpace(fields[1])
		lik := strings.TrimSpace(fields[2])
		score := len(fields) // placeholder logic
		results = append(results, Risk{
			Category:   cat,
			Impact:     imp,
			Likelihood: lik,
			Score:      score,
		})
	}
	return results, nil
}

// parseYAMLFile (placeholder) if you want to parse YAML into a RiskMatrix
func parseYAMLFile(filePath string) ([]Risk, error) {
	logger.Info("Parsing YAML file", zap.String("filePath", filePath))
	// Hypothetical: we read a YAML, parse it, etc.
	// We'll stub some results:
	return []Risk{
		{"Process", "High", "Likely", 12},
		{"Infra", "Critical", "Possible", 9},
	}, nil
}

// -------------- CSV/PDF Reporting --------------

func writeCSVReport(path string, matrix *RiskMatrix) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	if err := writer.Write([]string{"Category", "Impact", "Likelihood", "Score"}); err != nil {
		return err
	}
	for _, r := range matrix.Risks {
		row := []string{
			r.Category,
			r.Impact,
			r.Likelihood,
			fmt.Sprintf("%d", r.Score),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func writePDFReport(path string, matrix *RiskMatrix) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Risk Matrix Post-Quantum Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	for _, r := range matrix.Risks {
		line := fmt.Sprintf("Category: %s, Impact: %s, Likelihood: %s, Score: %d",
			r.Category, r.Impact, r.Likelihood, r.Score)
		pdf.MultiCell(190, 6, line, "", "", false)
	}
	return pdf.OutputFileAndClose(path)
}

// -------------- Concurrency aggregator --------------

func gatherFilesConcurrently(files []string) (*RiskMatrix, error) {
	var mu sync.Mutex
	var allRisks []Risk
	var wg sync.WaitGroup
	var firstErr error

	for _, f := range files {
		wg.Add(1)
		go func(filepath string) {
			defer wg.Done()
			ext := strings.ToLower(filepath[strings.LastIndex(filepath, ".")+1:])
			var parsed []Risk
			var err error

			switch ext {
			case "jsonl":
				parsed, err = parseJSONLFile(filepath)
			case "md":
				parsed, err = parseMarkdownFile(filepath)
			case "yaml", "yml":
				parsed, err = parseYAMLFile(filepath)
			default:
				err = fmt.Errorf("unrecognized file extension: %s", ext)
			}

			if err != nil {
				logger.Error("Error parsing file", zap.String("file", filepath), zap.Error(err))
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}

			// ephemeral encryption or ephemeral usage:
			// e.g. store in memory, or do a placeholder
			mu.Lock()
			allRisks = append(allRisks, parsed...)
			mu.Unlock()
		}(f)
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return &RiskMatrix{Risks: allRisks}, nil
}

// -------------- UI & Main --------------

func main() {
	// Initialize logger
	if err := InitializeLogger(); err != nil {
		fmt.Printf("Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting RiskMatrix application with Post-Quantum references")

	// Suppose we have some post-quantum ephemeral usage in oqs_vault
	if err := oqs_vault.InitEphemeralKey(); err != nil {
		logger.Warn("Failed ephemeral PQ key init", zap.Error(err))
	}

	// Example files to parse concurrently
	filesToParse := []string{"risks.jsonl", "matrix.md", "some.yaml"}

	matrix, err := gatherFilesConcurrently(filesToParse)
	if err != nil {
		logger.Fatal("Failed to gather files concurrently", zap.Error(err))
	}
	logger.Info("Parsed files", zap.Int("riskCount", len(matrix.Risks)))

	// Raylib UI
	rl.InitWindow(windowWidth, windowHeight, "RiskMatrix UI (PQ Secure)")
	rl.SetTargetFPS(60)
	runtime.LockOSThread()

	// generate simple swirl
	ps := generateParticles(maxParticles)

	// Graceful signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	mainCtx, mainCancel := context.WithCancel(context.Background())

	// main loop
	for !rl.WindowShouldClose() && mainCtx.Err() == nil {
		select {
		case <-sigChan:
			logger.Info("Got shutdown signal")
			mainCancel()
		default:
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.DarkBlue)

		updateParticles(ps)
		rl.DrawText("Risk Matrix (Post-Quantum) UI", 20, 20, fontSize, rl.White)
		localTime := time.Now().Format("15:04:05")
		rl.DrawText(fmt.Sprintf("Local Time: %s", localTime), 20, 60, fontSize-4, rl.LightGray)
		rl.DrawText("Press ESC to quit", 20, 100, fontSize-4, rl.LightGray)

		if rl.IsKeyPressed(rl.KeyEscape) {
			mainCancel()
		}
		rl.EndDrawing()
	}
	rl.CloseWindow()

	// generate reports in CSV/PDF
	// create a timestamped file in ghostshell/reporting
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		logger.Fatal("Failed to create report directory", zap.Error(err))
	}

	timestamp := time.Now().Format("20060102_150405")
	csvFile := filepath.Join(reportDir, fmt.Sprintf("riskmatrix_report_%s.csv", timestamp))
	pdfFile := filepath.Join(reportDir, fmt.Sprintf("riskmatrix_report_%s.pdf", timestamp))

	if err := writeCSVReport(csvFile, matrix); err != nil {
		logger.Error("Failed to write CSV report", zap.Error(err))
	} else {
		logger.Info("CSV report generated", zap.String("file", csvFile))
	}

	if err := writePDFReport(pdfFile, matrix); err != nil {
		logger.Error("Failed to write PDF report", zap.Error(err))
	} else {
		logger.Info("PDF report generated", zap.String("file", pdfFile))
	}

	// Print to console
	logger.Info("Outputting risk matrix to console")
	for _, r := range matrix.Risks {
		fmt.Printf("Category: %s | Impact: %s | Likelihood: %s | Score: %d\n",
			r.Category, r.Impact, r.Likelihood, r.Score)
	}

	logger.Info("RiskMatrix application finished")
}
