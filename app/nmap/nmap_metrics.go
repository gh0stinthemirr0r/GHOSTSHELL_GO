// nmap.go

// Package network provides functionalities for performing Nmap scans,
// parsing results, and integrating with monitoring systems like Prometheus.
package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Ullaakut/nmap"
	"github.com/prometheus/client_golang/prometheus"
)

// NmapScanResult represents the structured outcome of an Nmap scan.
type NmapScanResult struct {
	Hosts []NmapHost `json:"hosts"`
}

// NmapHost represents a single host's scan results.
type NmapHost struct {
	Addresses []NmapAddress `json:"addresses"`
	Ports     []NmapPort    `json:"ports"`
}

// NmapAddress represents an address associated with a host.
type NmapAddress struct {
	Addr string `json:"addr"`
	Type string `json:"type"`
}

// NmapPort represents a single port's scan results.
type NmapPort struct {
	ID       int    `json:"id"`
	Protocol string `json:"protocol"`
	State    string `json:"state"`
	Service  string `json:"service"`
}

// NmapScannerConfig holds configuration parameters for NmapScanner.
type NmapScannerConfig struct {
	// Targets specifies the list of target hosts or networks.
	Targets []string
	// Options allows specifying custom Nmap command-line options.
	Options []string
	// Timeout defines the maximum duration for the scan.
	Timeout time.Duration
	// Retries specifies the number of retry attempts for failed scans.
	Retries int
	// RetryInterval defines the wait time between retries.
	RetryInterval time.Duration
	// Logger is used for logging scan activities and errors.
	Logger *log.Logger
	// PrometheusMetrics enables Prometheus metrics collection.
	EnablePrometheus bool
}

// NmapScanner encapsulates the configuration, execution, and parsing of Nmap scans.
type NmapScanner struct {
	config       NmapScannerConfig
	scanner      *nmap.Scanner
	mutex        sync.RWMutex
	totalScans   prometheus.Counter
	successScans prometheus.Counter
	failedScans  prometheus.Counter
	scanDuration prometheus.Histogram
	scanResults  prometheus.GaugeVec
	metrics      bool
}

// NewNmapScanner initializes and returns a new NmapScanner instance.
func NewNmapScanner(cfg NmapScannerConfig) (*NmapScanner, error) {
	if len(cfg.Targets) == 0 {
		return nil, errors.New("no targets specified for Nmap scan")
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second // Default timeout
	}

	if cfg.Retries < 0 {
		cfg.Retries = 0 // Default to no retries
	}

	if cfg.RetryInterval == 0 {
		cfg.RetryInterval = 5 * time.Second // Default retry interval
	}

	if cfg.Logger == nil {
		cfg.Logger = log.New(os.Stdout, "[NmapScanner] ", log.LstdFlags)
	}

	scanner, err := nmap.NewScanner(
		nmap.WithTargets(cfg.Targets...),
		nmap.WithCustomArguments(cfg.Options...),
		nmap.WithTimeout(cfg.Timeout),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Nmap scanner: %w", err)
	}

	ns := &NmapScanner{
		config:  cfg,
		scanner: scanner,
	}

	// Initialize Prometheus metrics if enabled
	if cfg.EnablePrometheus {
		ns.initPrometheusMetrics()
		ns.metrics = true
	}

	return ns, nil
}

// initPrometheusMetrics initializes Prometheus metrics for NmapScanner.
func (ns *NmapScanner) initPrometheusMetrics() {
	ns.totalScans = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "nmap_total_scans",
		Help: "Total number of Nmap scans performed.",
	})
	ns.successScans = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "nmap_success_scans",
		Help: "Total number of successful Nmap scans.",
	})
	ns.failedScans = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "nmap_failed_scans",
		Help: "Total number of failed Nmap scans.",
	})
	ns.scanDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "nmap_scan_duration_seconds",
		Help:    "Duration of Nmap scans in seconds.",
		Buckets: prometheus.DefBuckets,
	})
	ns.scanResults = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "nmap_scan_results",
		Help: "Status of the latest Nmap scan per target.",
	}, []string{"target", "status"})

	// Register metrics
	prometheus.MustRegister(ns.totalScans, ns.successScans, ns.failedScans, ns.scanDuration, ns.scanResults)
}

// Run executes the Nmap scan with the provided context.
// It handles retries based on the configuration and returns the scan results.
func (ns *NmapScanner) Run(ctx context.Context) (*NmapScanResult, error) {
	var (
		result   *nmap.Run
		err      error
		warnings []string
	)

	for attempt := 0; attempt <= ns.config.Retries; attempt++ {
		if attempt > 0 {
			ns.config.Logger.Printf("Retrying Nmap scan (Attempt %d/%d)...", attempt, ns.config.Retries)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(ns.config.RetryInterval):
			}
		}

		ns.totalScans.Inc()

		startTime := time.Now()
		result, warnings, err = ns.scanner.RunWithContext(ctx)
		duration := time.Since(startTime).Seconds()
		ns.scanDuration.Observe(duration)

		if err != nil {
			ns.config.Logger.Printf("Nmap scan attempt %d failed: %v", attempt+1, err)
			ns.failedScans.Inc()
			ns.scanResults.WithLabelValues("all", "failed").Set(1)
			continue
		}

		if len(warnings) > 0 {
			ns.config.Logger.Printf("Nmap scan warnings: %v", warnings)
		}

		ns.successScans.Inc()
		ns.scanResults.WithLabelValues("all", "success").Set(1)

		scanResult, parseErr := ns.parseResults(result)
		if parseErr != nil {
			ns.config.Logger.Printf("Error parsing Nmap results: %v", parseErr)
			ns.failedScans.Inc()
			ns.scanResults.WithLabelValues("all", "failed").Set(1)
			err = parseErr
			continue
		}

		return scanResult, nil
	}

	return nil, fmt.Errorf("all Nmap scan attempts failed: %w", err)
}

// parseResults parses the Nmap scan results into a structured format.
func (ns *NmapScanner) parseResults(scan *nmap.Run) (*NmapScanResult, error) {
	if scan == nil || len(scan.Hosts) == 0 {
		return nil, errors.New("no hosts found in Nmap scan results")
	}

	var scanResult NmapScanResult
	for _, host := range scan.Hosts {
		var nmapHost NmapHost
		for _, addr := range host.Addresses {
			nmapHost.Addresses = append(nmapHost.Addresses, NmapAddress{
				Addr: addr.Addr,
				Type: addr.AddrType,
			})
		}

		for _, port := range host.Ports {
			nmapHost.Ports = append(nmapHost.Ports, NmapPort{
				ID:       port.ID,
				Protocol: port.Protocol,
				State:    port.State.State,
				Service:  port.Service.Name,
			})
		}

		scanResult.Hosts = append(scanResult.Hosts, nmapHost)
	}

	return &scanResult, nil
}

// SaveResultsToFile saves the scan results to a specified JSON file.
func (ns *NmapScanner) SaveResultsToFile(scanResult *NmapScanResult, filepath string) error {
	if scanResult == nil {
		return errors.New("scanResult cannot be nil")
	}

	data, err := json.MarshalIndent(scanResult, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal scan results to JSON: %w", err)
	}

	err = os.WriteFile(filepath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write scan results to file: %w", err)
	}

	ns.config.Logger.Printf("Scan results saved to %s", filepath)
	return nil
}

// Close releases resources held by the NmapScanner instance.
func (ns *NmapScanner) Close() {
	// Currently, the nmap.Scanner does not require explicit resource release.
	// This method is provided for future enhancements and to adhere to best practices.
}
