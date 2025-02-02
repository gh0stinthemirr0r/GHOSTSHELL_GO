package openport

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Constants & Paths
const (
	LogDir         = "ghostshell/logging"
	MaxConcurrency = 100 // Number of concurrent port scanners
)

// PortStatus represents the status of a port.
type PortStatus struct {
	Port       int
	Protocol   string
	IsOpen     bool
	IsInsecure bool
}

// InsecurePorts is a predefined list of ports considered insecure.
var InsecurePorts = map[int]bool{
	21:   true, // FTP
	22:   true, // SSH
	23:   true, // Telnet
	25:   true, // SMTP
	53:   true, // DNS
	80:   true, // HTTP
	110:  true, // POP3
	143:  true, // IMAP
	443:  true, // HTTPS
	445:  true, // SMB
	3389: true, // RDP
	5900: true, // VNC
}

// FindOpenPorts identifies open ports within the specified range and protocol.
// It returns a sorted list of PortStatus, prioritizing insecure ports first.
func FindOpenPorts(startPort, endPort int, protocol string) ([]PortStatus, error) {
	// Initialize Zap logger
	logger, err := setupLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to setup logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting port scan",
		zap.Int("startPort", startPort),
		zap.Int("endPort", endPort),
		zap.String("protocol", protocol),
	)

	var (
		openPorts   []PortStatus
		mu          sync.Mutex
		wg          sync.WaitGroup
		portChan    = make(chan int, MaxConcurrency)
		resultsChan = make(chan PortStatus, MaxConcurrency)
	)

	// Worker function to scan ports
	worker := func() {
		defer wg.Done()
		for port := range portChan {
			address := fmt.Sprintf("127.0.0.1:%d", port)
			conn, err := net.DialTimeout(protocol, address, 500*time.Millisecond)
			if err == nil {
				conn.Close()
				isInsecure := InsecurePorts[port]
				portStatus := PortStatus{
					Port:       port,
					Protocol:   protocol,
					IsOpen:     true,
					IsInsecure: isInsecure,
				}
				resultsChan <- portStatus
				logger.Debug("Port is open",
					zap.Int("port", port),
					zap.String("protocol", protocol),
					zap.Bool("isInsecure", isInsecure),
				)
			} else {
				logger.Debug("Port is closed",
					zap.Int("port", port),
					zap.String("protocol", protocol),
					zap.Error(err),
				)
			}
		}
	}

	// Start worker pool
	for i := 0; i < MaxConcurrency; i++ {
		wg.Add(1)
		go worker()
	}

	// Send ports to be scanned
	go func() {
		for port := startPort; port <= endPort; port++ {
			portChan <- port
		}
		close(portChan)
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for portStatus := range resultsChan {
		mu.Lock()
		openPorts = append(openPorts, portStatus)
		mu.Unlock()
	}

	// Sort open ports: insecure ports first, then ascending order
	sort.Slice(openPorts, func(i, j int) bool {
		if openPorts[i].IsInsecure && !openPorts[j].IsInsecure {
			return true
		}
		if openPorts[j].IsInsecure && !openPorts[i].IsInsecure {
			return false
		}
		return openPorts[i].Port < openPorts[j].Port
	})

	logger.Info("Port scan completed", zap.Int("openPortsFound", len(openPorts)))

	return openPorts, nil
}

// setupLogger initializes a Zap logger with a timestamped log file in ISO8601 format.
func setupLogger() (*zap.Logger, error) {
	if err := os.MkdirAll(LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	// ISO8601 timestamp e.g., 2023-10-25T15-30-45Z
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	logFileName := fmt.Sprintf("openport_log_%s.log", strings.ReplaceAll(timestamp, ":", "-"))
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

// DisplayOpenPorts prints the open ports, listing insecure ports first.
func DisplayOpenPorts(openPorts []PortStatus) {
	fmt.Println("Open Ports:")
	fmt.Println("------------")
	fmt.Printf("%-10s %-10s %-10s\n", "Port", "Protocol", "Insecure")
	fmt.Printf("%-10s %-10s %-10s\n", "----", "--------", "---------")
	for _, port := range openPorts {
		insecure := "No"
		if port.IsInsecure {
			insecure = "Yes"
		}
		fmt.Printf("%-10d %-10s %-10s\n", port.Port, port.Protocol, insecure)
	}
}

// Example usage of FindOpenPorts and DisplayOpenPorts
func main() {
	startPort := 1
	endPort := 1024
	protocol := "tcp"

	openPorts, err := FindOpenPorts(startPort, endPort, protocol)
	if err != nil {
		fmt.Printf("Error scanning ports: %v\n", err)
		return
	}

	DisplayOpenPorts(openPorts)
}
