package sources

import (
	"context"
	"encoding/csv"
	"fmt"
	"net"
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

	"ghostshell/oqs"        // Hypothetical post-quantum security package
	"ghostshell/securedata" // Hypothetical secure data handling package
)

// Constants & Paths
const (
	LogDir        = "ghostshell/logging"
	ReportDir     = "ghostshell/reporting"
	SecureDataDir = "ghostshell/secure_data"
)

// DNSQuantum enumerates DNS records for a domain with post-quantum security.
type DNSQuantum struct {
	logger      *zap.Logger
	vault       *securedata.Vault
	mu          sync.Mutex // Guards access to scanResults
	scanResults []string
}

// NewDNSQuantum creates a new instance of DNSQuantum with post-quantum security initialized.
func NewDNSQuantum() (*DNSQuantum, error) {
	// Initialize Zap logger
	logger, err := setupLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to setup logger: %v", err)
	}

	// Initialize secure vault
	vault, err := initializeVault(logger)
	if err != nil {
		logger.Error("Failed to initialize secure vault", zap.Error(err))
		return nil, err
	}

	return &DNSQuantum{
		logger:      logger,
		vault:       vault,
		scanResults: []string{},
	}, nil
}

// setupLogger initializes a Zap logger with a timestamped log file in ISO8601 format.
func setupLogger() (*zap.Logger, error) {
	if err := os.MkdirAll(LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	// ISO8601 timestamp e.g., 2023-10-25T15-30-45Z
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	logFileName := fmt.Sprintf("dnscrawler_log_%s.log", strings.ReplaceAll(timestamp, ":", "-"))
	logFilePath := filepath.Join(LogDir, logFileName)

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{logFilePath, "stdout"}

	logger, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %v", err)
	}
	return logger, nil
}

// initializeVault sets up the secure vault using post-quantum cryptography.
func initializeVault(logger *zap.Logger) (*securedata.Vault, error) {
	logger.Info("Initializing post-quantum secure vault")

	// Ensure the secure data directory exists
	if err := os.MkdirAll(SecureDataDir, 0700); err != nil {
		logger.Error("Failed to create secure data directory", zap.Error(err))
		return nil, fmt.Errorf("failed to create secure data directory: %w", err)
	}

	// Generate or load a post-quantum encryption key
	encryptionKeyPath := filepath.Join(SecureDataDir, "encryption_key.key")
	var encryptionKey []byte
	if _, err := os.Stat(encryptionKeyPath); os.IsNotExist(err) {
		// Generate a new encryption key
		key, err := oqs.GenerateRandomBytes(32) // Assuming 256-bit key
		if err != nil {
			logger.Error("Failed to generate encryption key", zap.Error(err))
			return nil, fmt.Errorf("failed to generate encryption key: %w", err)
		}
		encryptionKey = key

		// Save the encryption key securely
		if err := oqs.SaveKey(encryptionKeyPath, encryptionKey); err != nil {
			logger.Error("Failed to save encryption key", zap.Error(err))
			return nil, fmt.Errorf("failed to save encryption key: %w", err)
		}
		logger.Info("Generated and saved new encryption key", zap.String("path", encryptionKeyPath))
	} else {
		// Load existing encryption key
		key, err := oqs.LoadKey(encryptionKeyPath)
		if err != nil {
			logger.Error("Failed to load existing encryption key", zap.Error(err))
			return nil, fmt.Errorf("failed to load existing encryption key: %w", err)
		}
		encryptionKey = key
		logger.Info("Loaded existing encryption key", zap.String("path", encryptionKeyPath))
	}

	// Initialize the secure vault
	vault, err := securedata.NewVault(encryptionKey)
	if err != nil {
		logger.Error("Failed to initialize secure vault", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize secure vault: %w", err)
	}
	logger.Info("Secure vault initialized successfully")

	return vault, nil
}

// Enumerate performs DNS enumeration for a domain and saves encrypted results.
func (dq *DNSQuantum) Enumerate(domain string, outputFile string) error {
	dq.logger.Info("Starting DNS enumeration", zap.String("domain", domain))

	// Perform native DNS enumeration
	results, err := nativeDNSEnumerate(domain)
	if err != nil {
		dq.logger.Error("DNS enumeration failed", zap.String("domain", domain), zap.Error(err))
		return err
	}

	// Encrypt and store the results
	var encryptedResults []string
	for _, res := range results {
		line := fmt.Sprintf("Host: %s - Type: %s - Value: %s", res.Host, res.Type, res.Value)
		dq.logger.Info("DNS Record", zap.String("record", line))

		// Encrypt the line
		encryptedLine, err := dq.vault.EncryptData(line)
		if err != nil {
			dq.logger.Error("Failed to encrypt DNS record", zap.String("record", line), zap.Error(err))
			continue // Skip this entry but continue with others
		}
		encryptedResults = append(encryptedResults, encryptedLine)

		// Append to scanResults safely
		dq.mu.Lock()
		dq.scanResults = append(dq.scanResults, line)
		dq.mu.Unlock()
	}

	// Write encrypted results to output file
	if err := dq.writeEncryptedResults(outputFile, encryptedResults); err != nil {
		dq.logger.Error("Failed to write encrypted results", zap.String("outputFile", outputFile), zap.Error(err))
		return err
	}

	dq.logger.Info("DNS enumeration completed", zap.String("outputFile", outputFile))
	return nil
}

// nativeDNSEnumerate performs DNS enumeration using Go's standard library.
func nativeDNSEnumerate(domain string) ([]DNSRecord, error) {
	var records []DNSRecord
	recordTypes := []string{"A", "AAAA", "CNAME", "MX", "NS", "TXT", "SRV"}

	for _, rType := range recordTypes {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		switch rType {
		case "A":
			ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", domain)
			if err != nil {
				continue
			}
			for _, ip := range ips {
				records = append(records, DNSRecord{Host: domain, Type: "A", Value: ip.String()})
			}
		case "AAAA":
			ips, err := net.DefaultResolver.LookupIP(ctx, "ip6", domain)
			if err != nil {
				continue
			}
			for _, ip := range ips {
				records = append(records, DNSRecord{Host: domain, Type: "AAAA", Value: ip.String()})
			}
		case "CNAME":
			cname, err := net.DefaultResolver.LookupCNAME(ctx, domain)
			if err != nil {
				continue
			}
			records = append(records, DNSRecord{Host: domain, Type: "CNAME", Value: cname})
		case "MX":
			mxs, err := net.DefaultResolver.LookupMX(ctx, domain)
			if err != nil {
				continue
			}
			for _, mx := range mxs {
				records = append(records, DNSRecord{Host: domain, Type: "MX", Value: fmt.Sprintf("%d %s", mx.Pref, mx.Host)})
			}
		case "NS":
			nss, err := net.DefaultResolver.LookupNS(ctx, domain)
			if err != nil {
				continue
			}
			for _, ns := range nss {
				records = append(records, DNSRecord{Host: domain, Type: "NS", Value: ns.Host})
			}
		case "TXT":
			txts, err := net.DefaultResolver.LookupTXT(ctx, domain)
			if err != nil {
				continue
			}
			for _, txt := range txts {
				records = append(records, DNSRecord{Host: domain, Type: "TXT", Value: txt})
			}
		case "SRV":
			// Example SRV record lookup, adjust service and protocol as needed
			srvs, err := net.DefaultResolver.LookupSRV(ctx, "sip", "tcp", domain)
			if err != nil {
				continue
			}
			for _, srv := range srvs {
				records = append(records, DNSRecord{Host: srv.Target, Type: "SRV", Value: fmt.Sprintf("%d %d %d %s", srv.Priority, srv.Weight, srv.Port, srv.Target)})
			}
		}
	}

	return records, nil
}

// DNSRecord represents a DNS record with host, type, and value.
type DNSRecord struct {
	Host  string
	Type  string
	Value string
}

// writeEncryptedResults writes encrypted scan results to the specified file.
func (dq *DNSQuantum) writeEncryptedResults(outputPath string, encryptedData []string) error {
	// Ensure the reporting directory exists
	if err := os.MkdirAll(ReportDir, 0755); err != nil {
		dq.logger.Error("Failed to create report directory", zap.Error(err))
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	// Create the output file
	file, err := os.Create(outputPath)
	if err != nil {
		dq.logger.Error("Failed to create output file", zap.String("path", outputPath), zap.Error(err))
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Write encrypted data
	for _, line := range encryptedData {
		_, err := file.WriteString(line + "\n")
		if err != nil {
			dq.logger.Error("Failed to write to output file", zap.String("path", outputPath), zap.Error(err))
			return fmt.Errorf("failed to write to output file: %w", err)
		}
	}

	dq.logger.Info("Encrypted DNS results written to file", zap.String("path", outputPath))
	return nil
}

// GenerateReports decrypts scan results and writes them to CSV and PDF.
func (dq *DNSQuantum) GenerateReports(encryptedFile string) error {
	dq.logger.Info("Generating reports", zap.String("encryptedFile", encryptedFile))

	// Read encrypted results
	encryptedResults, err := dq.readEncryptedResults(encryptedFile)
	if err != nil {
		dq.logger.Error("Failed to read encrypted results", zap.Error(err))
		return err
	}

	// Decrypt results
	var decryptedResults []string
	for _, encLine := range encryptedResults {
		decryptedLine, err := dq.vault.DecryptData(encLine)
		if err != nil {
			dq.logger.Error("Failed to decrypt a scan result", zap.String("encryptedLine", encLine), zap.Error(err))
			continue // Skip this entry but continue with others
		}
		decryptedResults = append(decryptedResults, decryptedLine)
	}

	if len(decryptedResults) == 0 {
		dq.logger.Warn("No decrypted results available for reporting")
		return nil
	}

	// Write CSV report
	csvPath := filepath.Join(ReportDir, fmt.Sprintf("dnscrawler_report_%s.csv", time.Now().Format("20060102T150405Z")))
	if err := writeCSVReport(csvPath, decryptedResults); err != nil {
		dq.logger.Error("Failed to write CSV report", zap.Error(err))
		return err
	}

	// Write PDF report
	pdfPath := filepath.Join(ReportDir, fmt.Sprintf("dnscrawler_report_%s.pdf", time.Now().Format("20060102T150405Z")))
	if err := writePDFReport(pdfPath, decryptedResults); err != nil {
		dq.logger.Error("Failed to write PDF report", zap.Error(err))
		return err
	}

	dq.logger.Info("Reports generated successfully", zap.String("csv", csvPath), zap.String("pdf", pdfPath))
	return nil
}

// readEncryptedResults reads encrypted scan results from the specified file.
func (dq *DNSQuantum) readEncryptedResults(inputPath string) ([]string, error) {
	file, err := os.Open(inputPath)
	if err != nil {
		dq.logger.Error("Failed to open encrypted results file", zap.String("path", inputPath), zap.Error(err))
		return nil, fmt.Errorf("failed to open encrypted results file: %w", err)
	}
	defer file.Close()

	var encryptedResults []string
	scanner := NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		encryptedResults = append(encryptedResults, line)
	}

	if err := scanner.Err(); err != nil {
		dq.logger.Error("Error reading encrypted results file", zap.Error(err))
		return nil, fmt.Errorf("error reading encrypted results file: %w", err)
	}

	return encryptedResults, nil
}

// writeCSVReport writes decrypted scan results to a CSV file.
func writeCSVReport(outputPath string, data []string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create CSV report file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"Host", "Type", "Value"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data
	for _, line := range data {
		tokens := strings.SplitN(line, " - ", 3)
		if len(tokens) != 3 {
			continue // Skip malformed lines
		}
		host := strings.TrimPrefix(tokens[0], "Host: ")
		rType := strings.TrimPrefix(tokens[1], "Type: ")
		value := strings.TrimPrefix(tokens[2], "Value: ")
		if err := writer.Write([]string{host, rType, value}); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// writePDFReport writes decrypted scan results to a PDF file.
func writePDFReport(outputPath string, data []string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "DNS Crawler Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(60, 8, "Host")
	pdf.Cell(40, 8, "Type")
	pdf.Cell(80, 8, "Value")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 12)
	for _, line := range data {
		tokens := strings.SplitN(line, " - ", 3)
		if len(tokens) != 3 {
			continue // Skip malformed lines
		}
		host := strings.TrimPrefix(tokens[0], "Host: ")
		rType := strings.TrimPrefix(tokens[1], "Type: ")
		value := strings.TrimPrefix(tokens[2], "Value: ")

		pdf.Cell(60, 8, host)
		pdf.Cell(40, 8, rType)
		pdf.MultiCell(80, 8, value, "", "", false)
	}

	if err := pdf.OutputFileAndClose(outputPath); err != nil {
		return fmt.Errorf("failed to write PDF report: %w", err)
	}

	return nil
}

// Shutdown handles the graceful shutdown of DNSQuantum.
func (dq *DNSQuantum) Shutdown() {
	dq.logger.Info("Shutting down DNSQuantum...")

	// Close the secure vault
	if dq.vault != nil {
		if err := dq.vault.Close(); err != nil {
			dq.logger.Error("Failed to close secure vault", zap.Error(err))
		} else {
			dq.logger.Info("Secure vault closed successfully")
		}
	}

	// Flush and sync logger
	if dq.logger != nil {
		_ = dq.logger.Sync()
	}

	os.Exit(0)
}

// DNSScannerResult represents a single DNS scan result.
type DNSScannerResult struct {
	Host  string
	Type  string
	Value string
}

// NewScanner creates a new Scanner for reading lines from a file.
func NewScanner(file *os.File) *Scanner {
	return &Scanner{
		file: file,
		buf:  make([]byte, 0),
	}
}

// Scanner is a simple line scanner.
type Scanner struct {
	file *os.File
	buf  []byte
}

// Scan reads the next line.
func (s *Scanner) Scan() bool {
	s.buf = s.buf[:0]
	for {
		b := make([]byte, 1)
		_, err := s.file.Read(b)
		if err != nil {
			return len(s.buf) > 0
		}
		if b[0] == '\n' {
			break
		}
		s.buf = append(s.buf, b[0])
	}
	return true
}

// Text returns the current line as a string.
func (s *Scanner) Text() string {
	return string(s.buf)
}

// EnumerateDNS scans DNS records for a given domain and saves encrypted results.
func EnumerateDNS(domain string, encryptedOutput string) {
	dq, err := NewDNSQuantum()
	if err != nil {
		fmt.Printf("Failed to initialize DNSQuantum: %v\n", err)
		os.Exit(1)
	}

	// Handle graceful shutdown
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-shutdownChan
		dq.Shutdown()
	}()

	// Perform enumeration
	if err := dq.Enumerate(domain, encryptedOutput); err != nil {
		dq.logger.Error("DNS enumeration failed", zap.Error(err))
		dq.Shutdown()
	}

	// Optionally generate reports
	if err := dq.GenerateReports(encryptedOutput); err != nil {
		dq.logger.Error("Report generation failed", zap.Error(err))
	}

	dq.Shutdown()
}
