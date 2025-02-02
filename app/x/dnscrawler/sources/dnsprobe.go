package sources

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jung-kurt/gofpdf"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Constants & Paths
const (
	LogDir        = "ghostshell/logging"
	ReportDir     = "ghostshell/reporting"
	SecureDataDir = "ghostshell/secure_data"
)

// DNSProbe handles DNS lookups using Go's native net package with retry logic.
type DNSProbe struct {
	logger       *zap.Logger
	resolvers    []string
	maxRetries   int
	questionType uint16
	mu           sync.Mutex // Guards access to scanResults
	scanResults  []DNSRecord
}

// DNSRecord represents a DNS record with host, type, and value.
type DNSRecord struct {
	Host  string
	Type  string
	Value string
}

// Options defines configuration options for DNSProbe.
type Options struct {
	Resolvers    []string
	MaxRetries   int
	QuestionType uint16
}

// DefaultOptions provides default DNSProbe configuration.
var DefaultOptions = Options{
	Resolvers: []string{
		"1.1.1.1:53", // Cloudflare
		"8.8.8.8:53", // Google
		"9.9.9.9:53", // Quad9
	},
	MaxRetries:   3,
	QuestionType: net.TypeA, // Using net package constants
}

// NewDNSProbe creates a new instance of DNSProbe with native DNS enumeration and Zap logger.
func NewDNSProbe(options Options) (*DNSProbe, error) {
	// Initialize Zap logger
	logger, err := setupLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to setup logger: %v", err)
	}

	// Validate question type
	if !isValidQuestionType(options.QuestionType) {
		logger.Error("Invalid question type provided", zap.Uint16("questionType", options.QuestionType))
		return nil, fmt.Errorf("invalid question type: %d", options.QuestionType)
	}

	return &DNSProbe{
		logger:       logger,
		resolvers:    options.Resolvers,
		maxRetries:   options.MaxRetries,
		questionType: options.QuestionType,
		scanResults:  []DNSRecord{},
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

// isValidQuestionType checks if the provided question type is supported.
func isValidQuestionType(qType uint16) bool {
	switch qType {
	case net.TypeA, net.TypeAAAA, net.TypeCNAME, net.TypeMX, net.TypeNS, net.TypeTXT, net.TypeSRV:
		return true
	default:
		return false
	}
}

// Enumerate performs DNS enumeration for a domain and saves encrypted results.
func (dp *DNSProbe) Enumerate(domain string, outputFile string) error {
	dp.logger.Info("Starting DNS enumeration", zap.String("domain", domain))

	// Perform enumeration
	results, err := dp.nativeDNSEnumerate(domain)
	if err != nil {
		dp.logger.Error("DNS enumeration failed", zap.String("domain", domain), zap.Error(err))
		return err
	}

	// Encrypt and store the results
	var encryptedResults []string
	for _, res := range results {
		line := fmt.Sprintf("Host: %s - Type: %s - Value: %s", res.Host, res.Type, res.Value)
		dp.logger.Info("DNS Record", zap.String("record", line))

		// Encrypt the line
		encryptedLine, err := dp.encryptData(line)
		if err != nil {
			dp.logger.Error("Failed to encrypt DNS record", zap.String("record", line), zap.Error(err))
			continue // Skip this entry but continue with others
		}
		encryptedResults = append(encryptedResults, encryptedLine)

		// Append to scanResults safely
		dp.mu.Lock()
		dp.scanResults = append(dp.scanResults, res)
		dp.mu.Unlock()
	}

	// Write encrypted results to output file
	if err := dp.writeEncryptedResults(outputFile, encryptedResults); err != nil {
		dp.logger.Error("Failed to write encrypted results", zap.String("outputFile", outputFile), zap.Error(err))
		return err
	}

	dp.logger.Info("DNS enumeration completed", zap.String("outputFile", outputFile))
	return nil
}

// nativeDNSEnumerate performs DNS enumeration using Go's standard library.
func (dp *DNSProbe) nativeDNSEnumerate(domain string) ([]DNSRecord, error) {
	var records []DNSRecord

	// Define a custom resolver
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Second * 5,
			}
			return d.DialContext(ctx, network, address)
		},
	}

	switch dp.questionType {
	case net.TypeA:
		ips, err := resolver.LookupIP(context.Background(), "ip4", domain)
		if err != nil {
			dp.logger.Warn("LookupIP failed", zap.String("domain", domain), zap.Error(err))
		} else {
			for _, ip := range ips {
				records = append(records, DNSRecord{Host: domain, Type: "A", Value: ip.String()})
			}
		}
	case net.TypeAAAA:
		ips, err := resolver.LookupIP(context.Background(), "ip6", domain)
		if err != nil {
			dp.logger.Warn("LookupIP failed", zap.String("domain", domain), zap.Error(err))
		} else {
			for _, ip := range ips {
				records = append(records, DNSRecord{Host: domain, Type: "AAAA", Value: ip.String()})
			}
		}
	case net.TypeCNAME:
		cname, err := resolver.LookupCNAME(context.Background(), domain)
		if err != nil {
			dp.logger.Warn("LookupCNAME failed", zap.String("domain", domain), zap.Error(err))
		} else {
			records = append(records, DNSRecord{Host: domain, Type: "CNAME", Value: cname})
		}
	case net.TypeMX:
		mxs, err := resolver.LookupMX(context.Background(), domain)
		if err != nil {
			dp.logger.Warn("LookupMX failed", zap.String("domain", domain), zap.Error(err))
		} else {
			for _, mx := range mxs {
				value := fmt.Sprintf("%d %s", mx.Pref, mx.Host)
				records = append(records, DNSRecord{Host: domain, Type: "MX", Value: value})
			}
		}
	case net.TypeNS:
		nss, err := resolver.LookupNS(context.Background(), domain)
		if err != nil {
			dp.logger.Warn("LookupNS failed", zap.String("domain", domain), zap.Error(err))
		} else {
			for _, ns := range nss {
				records = append(records, DNSRecord{Host: domain, Type: "NS", Value: ns.Host})
			}
		}
	case net.TypeTXT:
		txts, err := resolver.LookupTXT(context.Background(), domain)
		if err != nil {
			dp.logger.Warn("LookupTXT failed", zap.String("domain", domain), zap.Error(err))
		} else {
			for _, txt := range txts {
				records = append(records, DNSRecord{Host: domain, Type: "TXT", Value: txt})
			}
		}
	case net.TypeSRV:
		// Example SRV record lookup; adjust service and protocol as needed
		srvs, err := resolver.LookupSRV(context.Background(), "sip", "tcp", domain)
		if err != nil {
			dp.logger.Warn("LookupSRV failed", zap.String("domain", domain), zap.Error(err))
		} else {
			for _, srv := range srvs {
				value := fmt.Sprintf("%d %d %d %s", srv.Priority, srv.Weight, srv.Port, srv.Target)
				records = append(records, DNSRecord{Host: srv.Target, Type: "SRV", Value: value})
			}
		}
	default:
		return nil, errors.New("unsupported DNS question type")
	}

	return records, nil
}

// encryptData encrypts the given data string using the secure vault.
func (dp *DNSProbe) encryptData(data string) (string, error) {
	encrypted, err := dp.vault.EncryptData(data)
	if err != nil {
		return "", fmt.Errorf("encryption failed: %w", err)
	}
	return encrypted, nil
}

// writeEncryptedResults writes encrypted scan results to the specified file.
func (dp *DNSProbe) writeEncryptedResults(outputPath string, encryptedData []string) error {
	// Ensure the reporting directory exists
	if err := os.MkdirAll(ReportDir, 0755); err != nil {
		dp.logger.Error("Failed to create report directory", zap.Error(err))
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	// Create the output file
	file, err := os.Create(outputPath)
	if err != nil {
		dp.logger.Error("Failed to create output file", zap.String("path", outputPath), zap.Error(err))
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Write encrypted data
	for _, line := range encryptedData {
		_, err := file.WriteString(line + "\n")
		if err != nil {
			dp.logger.Error("Failed to write to output file", zap.String("path", outputPath), zap.Error(err))
			return fmt.Errorf("failed to write to output file: %w", err)
		}
	}

	dp.logger.Info("Encrypted DNS results written to file", zap.String("path", outputPath))
	return nil
}

// GenerateReports decrypts scan results and writes them to CSV and PDF.
func (dp *DNSProbe) GenerateReports(encryptedFile string) error {
	dp.logger.Info("Generating reports", zap.String("encryptedFile", encryptedFile))

	// Read encrypted results
	encryptedResults, err := dp.readEncryptedResults(encryptedFile)
	if err != nil {
		dp.logger.Error("Failed to read encrypted results", zap.Error(err))
		return err
	}

	// Decrypt results
	var decryptedResults []DNSRecord
	for _, encLine := range encryptedResults {
		decryptedLine, err := dp.vault.DecryptData(encLine)
		if err != nil {
			dp.logger.Error("Failed to decrypt a scan result", zap.String("encryptedLine", encLine), zap.Error(err))
			continue // Skip this entry but continue with others
		}
		record, err := parseDNSRecord(decryptedLine)
		if err != nil {
			dp.logger.Error("Failed to parse decrypted scan result", zap.String("decryptedLine", decryptedLine), zap.Error(err))
			continue
		}
		decryptedResults = append(decryptedResults, record)
	}

	if len(decryptedResults) == 0 {
		dp.logger.Warn("No decrypted results available for reporting")
		return nil
	}

	// Write CSV report
	csvPath := filepath.Join(ReportDir, fmt.Sprintf("dnscrawler_report_%s.csv", time.Now().Format("20060102T150405Z")))
	if err := writeCSVReport(csvPath, decryptedResults); err != nil {
		dp.logger.Error("Failed to write CSV report", zap.Error(err))
		return err
	}

	// Write PDF report
	pdfPath := filepath.Join(ReportDir, fmt.Sprintf("dnscrawler_report_%s.pdf", time.Now().Format("20060102T150405Z")))
	if err := writePDFReport(pdfPath, decryptedResults); err != nil {
		dp.logger.Error("Failed to write PDF report", zap.Error(err))
		return err
	}

	dp.logger.Info("Reports generated successfully", zap.String("csv", csvPath), zap.String("pdf", pdfPath))
	return nil
}

// readEncryptedResults reads encrypted scan results from the specified file.
func (dp *DNSProbe) readEncryptedResults(inputPath string) ([]string, error) {
	file, err := os.Open(inputPath)
	if err != nil {
		dp.logger.Error("Failed to open encrypted results file", zap.String("path", inputPath), zap.Error(err))
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
		dp.logger.Error("Error reading encrypted results file", zap.Error(err))
		return nil, fmt.Errorf("error reading encrypted results file: %w", err)
	}

	return encryptedResults, nil
}

// parseDNSRecord parses a decrypted DNS record line into a DNSRecord struct.
func parseDNSRecord(line string) (DNSRecord, error) {
	parts := strings.Split(line, " - ")
	if len(parts) != 3 {
		return DNSRecord{}, fmt.Errorf("invalid DNS record format: %s", line)
	}

	host := strings.TrimPrefix(parts[0], "Host: ")
	rType := strings.TrimPrefix(parts[1], "Type: ")
	value := strings.TrimPrefix(parts[2], "Value: ")

	return DNSRecord{
		Host:  host,
		Type:  rType,
		Value: value,
	}, nil
}

// writeCSVReport writes decrypted scan results to a CSV file.
func writeCSVReport(outputPath string, data []DNSRecord) error {
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
	for _, record := range data {
		if err := writer.Write([]string{record.Host, record.Type, record.Value}); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// writePDFReport writes decrypted scan results to a PDF file.
func writePDFReport(outputPath string, data []DNSRecord) error {
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
	for _, record := range data {
		pdf.Cell(60, 8, record.Host)
		pdf.Cell(40, 8, record.Type)
		pdf.MultiCell(80, 8, record.Value, "", "", false)
	}

	if err := pdf.OutputFileAndClose(outputPath); err != nil {
		return fmt.Errorf("failed to write PDF report: %w", err)
	}

	return nil
}

// Shutdown handles the graceful shutdown of DNSProbe.
func (dp *DNSProbe) Shutdown() {
	dp.logger.Info("Shutting down DNSProbe...")

	// Close the secure vault
	if dp.vault != nil {
		if err := dp.vault.Close(); err != nil {
			dp.logger.Error("Failed to close secure vault", zap.Error(err))
		} else {
			dp.logger.Info("Secure vault closed successfully")
		}
	}

	// Flush and sync logger
	if dp.logger != nil {
		_ = dp.logger.Sync()
	}

	os.Exit(0)
}

// DNSScannerResult represents a single DNS scan result.
type DNSScannerResult struct {
	Host  string
	Type  string
	Value string
}

// Scanner is a simple line scanner.
type Scanner struct {
	file *os.File
	buf  []byte
}

// NewScanner creates a new Scanner for reading lines from a file.
func NewScanner(file *os.File) *Scanner {
	return &Scanner{
		file: file,
		buf:  make([]byte, 0),
	}
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
