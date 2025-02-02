package main

import (
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jung-kurt/gofpdf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	DefaultBaseURL  = "https://cve.mitre.org/api/v3/cves"
	DefaultTimeout  = 10 * time.Second
	LogDir          = "ghostshell/logging"
	ReportDir       = "ghostshell/reporting"
	DefaultPDFTitle = "CVE Mapper Report"
)

// CVEData represents the structured data about a single CVE or set of CVEs
type CVEData struct {
	ID           string
	Description  string
	Score        float64
	Published    string
	LastModified string
	// You can add more fields like references, exploit code maturity, etc.
}

// -------------- Logging Setup --------------

var logger *zap.Logger

func setupLogger() (*zap.Logger, error) {
	if err := os.MkdirAll(LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(LogDir, fmt.Sprintf("cvemap_log_%s.log", timestamp))

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logFile, "stdout"}

	logger, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}
	return logger, nil
}

// -------------- CLI Flags --------------

type Options struct {
	Debug   bool
	CVEIDs  string
	BaseURL string
	Timeout time.Duration
}

func parseOptions() (*Options, error) {
	var debug bool
	var cveIDs string
	var baseURL string
	var timeout int

	flag.BoolVar(&debug, "debug", false, "Enable debug logs")
	flag.StringVar(&cveIDs, "cve", "", "Comma-separated list of CVE IDs to retrieve")
	flag.StringVar(&baseURL, "baseurl", DefaultBaseURL, "Base URL for the MITRE CVE API")
	flag.IntVar(&timeout, "timeout", 10, "Request timeout in seconds")

	flag.Parse()

	opts := &Options{
		Debug:   debug,
		CVEIDs:  cveIDs,
		BaseURL: baseURL,
		Timeout: time.Duration(timeout) * time.Second,
	}
	return opts, nil
}

// -------------- Concurrency / CVE fetching --------------

func doRequest(url string, method string, logger *zap.Logger, timeout time.Duration) (*http.Response, error) {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		logger.Error("Failed to create request", zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Request failed", zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("request failed for %s: %v", url, err)
	}

	// We won't close body here, as we might parse it. The caller should close it.
	if resp.StatusCode == http.StatusUnauthorized {
		logger.Warn("Authorization failed", zap.String("url", url), zap.Int("status", resp.StatusCode))
		return nil, fmt.Errorf("unauthorized: %s", url)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Warn("Non-2xx response", zap.String("url", url), zap.Int("status", resp.StatusCode))
		return resp, fmt.Errorf("non-OK status %d for url %s", resp.StatusCode, url)
	}
	return resp, nil
}

func fetchCVEData(cveID, baseURL string, timeout time.Duration, logger *zap.Logger) (CVEData, error) {
	// For demonstration, we will do a stub. Real logic might do something like:
	// url := fmt.Sprintf("%s/%s", baseURL, cveID)
	// resp, err := doRequest(url, "GET", logger, timeout)
	// parse JSON etc.

	// Stub logic: 50% success/fail
	time.Sleep(time.Duration(500) * time.Millisecond)
	if strings.TrimSpace(cveID) == "" {
		return CVEData{}, errors.New("empty CVE ID")
	}
	r := time.Now().UnixNano() % 2
	if r == 0 {
		logger.Warn("Failed to fetch CVE data", zap.String("cveID", cveID))
		return CVEData{}, fmt.Errorf("failed to fetch %s", cveID)
	}

	// Return a random CVSS score, published date, etc.
	return CVEData{
		ID:           cveID,
		Description:  fmt.Sprintf("Description for %s", cveID),
		Score:        float64((time.Now().UnixNano() % 1000)) / 100.0,
		Published:    time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
		LastModified: time.Now().Format("2006-01-02"),
	}, nil
}

// -------------- CSV/PDF Reporting --------------

func generateReport(cveData []CVEData, logger *zap.Logger) error {
	if err := os.MkdirAll(ReportDir, 0755); err != nil {
		logger.Error("Failed to create report directory", zap.Error(err))
		return err
	}
	timestamp := time.Now().Format("20060102_150405")
	csvFile := filepath.Join(ReportDir, fmt.Sprintf("cvemap_report_%s.csv", timestamp))
	pdfFile := filepath.Join(ReportDir, fmt.Sprintf("cvemap_report_%s.pdf", timestamp))

	if err := writeCSVReport(csvFile, cveData, logger); err != nil {
		return err
	}
	if err := writePDFReport(pdfFile, cveData, logger); err != nil {
		return err
	}

	return nil
}

func writeCSVReport(path string, cveData []CVEData, logger *zap.Logger) error {
	f, err := os.Create(path)
	if err != nil {
		logger.Error("Failed to create CSV", zap.String("file", path), zap.Error(err))
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// header
	if err := w.Write([]string{"CVE_ID", "Description", "Score", "Published", "LastModified"}); err != nil {
		return err
	}
	for _, c := range cveData {
		row := []string{
			c.ID,
			c.Description,
			fmt.Sprintf("%.2f", c.Score),
			c.Published,
			c.LastModified,
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	logger.Info("CSV report generated", zap.String("file", path))
	return nil
}

func writePDFReport(path string, cveData []CVEData, logger *zap.Logger) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(40, 10, "CVE Mapper Report")
	pdf.Ln(12)

	// headers
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(40, 8, "CVE ID")
	pdf.Cell(40, 8, "Score")
	pdf.Cell(50, 8, "Published")
	pdf.Cell(50, 8, "LastModified")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 12)
	for _, c := range cveData {
		// row
		pdf.Cell(40, 8, c.ID)
		pdf.Cell(40, 8, fmt.Sprintf("%.2f", c.Score))
		pdf.Cell(50, 8, c.Published)
		pdf.Cell(50, 8, c.LastModified)
		pdf.Ln(8)

		// Description multiline
		pdf.MultiCell(190, 6, fmt.Sprintf("Desc: %s", c.Description), "", "", false)
		pdf.Ln(4)
	}

	if err := pdf.OutputFileAndClose(path); err != nil {
		logger.Error("Failed to write PDF", zap.String("file", path), zap.Error(err))
		return err
	}
	logger.Info("PDF report generated", zap.String("file", path))
	return nil
}

// -------------- Main Logic --------------

func main() {
	opts, err := parseOptions()
	if err != nil {
		fmt.Printf("Error parsing options: %v\n", err)
		os.Exit(1)
	}

	logger, logErr := setupLogger()
	if logErr != nil {
		fmt.Printf("Failed to init logger: %v\n", logErr)
		os.Exit(1)
	}
	defer logger.Sync()

	if opts.Debug {
		logger.Info("Debug mode enabled")
	}

	// Setup signal-based graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Received shutdown signal, exiting gracefully.")
		cancel()
		// If you need additional cleanup, do it here
		os.Exit(0)
	}()

	// concurrency-based retrieval of CVEs
	cves := strings.Split(opts.CVEIDs, ",")
	if len(cves) == 1 && cves[0] == "" {
		logger.Warn("No CVEs requested, skipping retrieval.")
		cves = []string{}
	}
	logger.Info("Starting concurrency to fetch CVEs", zap.Strings("cveIDs", cves))

	var wg sync.WaitGroup
	cveResults := make([]CVEData, 0, len(cves))
	mu := sync.Mutex{}

	for _, cveID := range cves {
		cveID = strings.TrimSpace(cveID)
		if cveID == "" {
			continue
		}
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			data, err := fetchCVEData(id, opts.BaseURL, opts.Timeout, logger)
			if err != nil {
				logger.Warn("Failed to fetch a CVE", zap.String("CVE", id), zap.Error(err))
				return
			}
			mu.Lock()
			cveResults = append(cveResults, data)
			mu.Unlock()
		}(cveID)
	}

	wg.Wait()
	logger.Info("Finished fetching CVEs", zap.Int("retrieved_count", len(cveResults)))

	if len(cveResults) > 0 {
		if err := generateReport(cveResults, logger); err != nil {
			logger.Error("Failed to generate report", zap.Error(err))
		}
	} else {
		logger.Warn("No CVE data retrieved, skipping report.")
	}

	logger.Info("CVE mapping complete. Exiting now.")
}
