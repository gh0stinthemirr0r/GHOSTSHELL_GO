package sources

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/xid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Constants & Paths
const (
	LogDir = "ghostshell/logging"
)

// DNSRecord represents a DNS record with host, type, and value.
type DNSRecord struct {
	Host  string
	Type  string
	Value string
}

// WildcardDetector detects and manages wildcard subdomains using native DNS queries and Zap logger.
type WildcardDetector struct {
	logger       *zap.Logger
	cache        map[string][]string
	cacheMutex   sync.RWMutex
	wildcardMap  map[string]struct{}
	wildcardLock sync.RWMutex
}

// NewWildcardDetector creates a new instance of WildcardDetector with native DNS queries and Zap logger.
func NewWildcardDetector() (*WildcardDetector, error) {
	// Initialize Zap logger
	logger, err := setupLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to setup logger: %v", err)
	}

	return &WildcardDetector{
		logger:      logger,
		cache:       make(map[string][]string),
		wildcardMap: make(map[string]struct{}),
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

// IsWildcard checks if a host is a wildcard by comparing its DNS response with responses from random subdomains.
func (wd *WildcardDetector) IsWildcard(host string, baseDomain string) bool {
	origIPs, err := wd.queryDNS(host, "A")
	if err != nil {
		wd.logger.Error("Failed to query DNS for original host", zap.String("host", host), zap.Error(err))
		return false
	}

	// Generate random subdomains and compare results
	for i := 0; i < 3; i++ {
		randSubdomain := xid.New().String() + "." + baseDomain
		randIPs, err := wd.queryDNS(randSubdomain, "A")
		if err != nil {
			wd.logger.Warn("Failed to query DNS for random subdomain", zap.String("subdomain", randSubdomain), zap.Error(err))
			continue
		}

		for _, ip := range randIPs {
			wd.wildcardLock.Lock()
			wd.wildcardMap[ip] = struct{}{}
			wd.wildcardLock.Unlock()
		}
	}

	// Compare original IPs to wildcard IPs
	for _, ip := range origIPs {
		wd.wildcardLock.RLock()
		if _, exists := wd.wildcardMap[ip]; exists {
			wd.wildcardLock.RUnlock()
			return true
		}
		wd.wildcardLock.RUnlock()
	}

	return false
}

// queryDNS performs a DNS query for the given host and record type using Go's native net package.
func (wd *WildcardDetector) queryDNS(host string, recordType string) ([]string, error) {
	wd.cacheMutex.RLock()
	if cachedIPs, found := wd.cache[host]; found {
		wd.cacheMutex.RUnlock()
		return cachedIPs, nil
	}
	wd.cacheMutex.RUnlock()

	var ips []string
	var err error

	switch recordType {
	case "A":
		ips, err = net.LookupHost(host)
	case "AAAA":
		ips, err = net.LookupIP(host)
	case "CNAME":
		cname, err := net.LookupCNAME(host)
		if err != nil {
			return nil, err
		}
		ips = []string{cname}
	case "MX":
		mxs, err := net.LookupMX(host)
		if err != nil {
			return nil, err
		}
		for _, mx := range mxs {
			ips = append(ips, fmt.Sprintf("%d %s", mx.Pref, mx.Host))
		}
	case "NS":
		nss, err := net.LookupNS(host)
		if err != nil {
			return nil, err
		}
		for _, ns := range nss {
			ips = append(ips, ns.Host)
		}
	case "TXT":
		txts, err := net.LookupTXT(host)
		if err != nil {
			return nil, err
		}
		for _, txt := range txts {
			ips = append(ips, txt)
		}
	case "SRV":
		srvs, err := net.LookupSRV("", "", host)
		if err != nil {
			return nil, err
		}
		for _, srv := range srvs {
			ips = append(ips, fmt.Sprintf("%d %d %d %s", srv.Priority, srv.Weight, srv.Port, srv.Target))
		}
	default:
		return nil, fmt.Errorf("unsupported DNS record type: %s", recordType)
	}

	if err != nil {
		return nil, err
	}

	// Cache the results
	wd.cacheMutex.Lock()
	wd.cache[host] = ips
	wd.cacheMutex.Unlock()

	return ips, nil
}

// Shutdown gracefully shuts down the WildcardDetector, ensuring all logs are flushed.
func (wd *WildcardDetector) Shutdown() {
	if wd.logger != nil {
		wd.logger.Info("Shutting down WildcardDetector...")
		_ = wd.logger.Sync()
	}
}

// Example usage of WildcardDetector
func main() {
	// Initialize WildcardDetector
	wd, err := NewWildcardDetector()
	if err != nil {
		fmt.Printf("Failed to initialize WildcardDetector: %v\n", err)
		os.Exit(1)
	}
	defer wd.Shutdown()

	// Example domain
	domain := "example.com"

	// Check if the domain has wildcard DNS records
	isWildcard := wd.IsWildcard(domain, "example.com")
	if isWildcard {
		wd.logger.Info("Wildcard DNS detected", zap.String("domain", domain))
	} else {
		wd.logger.Info("No wildcard DNS detected", zap.String("domain", domain))
	}
}
