package network

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/shirou/gopsutil/net"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type NetworkAdapterInfo struct {
	Name         string
	HardwareAddr string
	IPAddresses  []string
	MTU          int32
	IsUp         bool
	Speed        uint64
}

type Route struct {
	Destination string
	Gateway     string
	Genmask     string
	Flags       string
	Metric      int
	Ref         int
	Use         int
	Iface       string
}

type DNSInfo struct {
	Servers []string
}

type NetworkCollector interface {
	GetAdapterDetails() ([]NetworkAdapterInfo, error)
	GetRouteTable() ([]Route, error)
	GetDNSInfo() (DNSInfo, error)
	EncryptAndStoreSensitiveData(data string) (string, error)
	DecryptSensitiveData(encryptedData string) (string, error)
}

type systemNetworkCollector struct {
	logger *zap.Logger
	mu     sync.RWMutex
}

// initializeLogger initializes a Zap logger with dynamic filenames and UTC timestamps.
func initializeLogger() (*zap.Logger, error) {
	currentTime := time.Now().UTC().Format("20060102_1504")
	logFilename := fmt.Sprintf("ghostshell/logging/netlog_%s.log", currentTime)

	if err := os.MkdirAll("ghostshell/logging", 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:      false,
		Encoding:         "json",
		OutputPaths:      []string{logFilename, "stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:      "timestamp",
			LevelKey:     "level",
			MessageKey:   "message",
			CallerKey:    "caller",
			EncodeTime:   zapcore.ISO8601TimeEncoder,
			EncodeLevel:  zapcore.CapitalLevelEncoder,
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}

	return config.Build()
}

// NewSystemNetworkCollector initializes a new NetworkCollector with a Zap logger.
func NewSystemNetworkCollector() (NetworkCollector, error) {
	logger, err := initializeLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	return &systemNetworkCollector{
		logger: logger,
	}, nil
}

func (c *systemNetworkCollector) GetDNSInfo() (DNSInfo, error) {
	// Implement DNS info retrieval logic here
	return DNSInfo{}, nil
}

func (c *systemNetworkCollector) DecryptSensitiveData(encryptedData string) (string, error) {
	// Implement DecryptSensitiveData() method
	return "", nil
}

func (c *systemNetworkCollector) GetAdapterDetails() ([]NetworkAdapterInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	interfaces, err := net.Interfaces()
	if err != nil {
		c.logger.Error("Error retrieving network interfaces", zap.Error(err))
		return nil, fmt.Errorf("error retrieving network interfaces: %w", err)
	}

	var adapters []NetworkAdapterInfo
	for _, iface := range interfaces {
		var ipAddrs []string
		for _, addr := range iface.Addrs {
			ipAddrs = append(ipAddrs, addr.Addr)
		}
		adapter := NetworkAdapterInfo{
			Name:         iface.Name,
			HardwareAddr: iface.HardwareAddr,
			IPAddresses:  ipAddrs,
			MTU:          int32(iface.MTU),
			IsUp:         iface.Flags&net.FlagUp != 0,
			Speed:        iface.Speed,
		}
		adapters = append(adapters, adapter)
		c.logger.Info("Retrieved adapter details", zap.String("adapter", iface.Name), zap.Bool("is_up", adapter.IsUp))
	}

	return adapters, nil
}

func ExampleUsage() {
	collector, err := NewSystemNetworkCollector()
	if err != nil {
		log.Fatalf("Failed to initialize network collector: %v", err)
	}

	adapters, err := collector.GetAdapterDetails()
	if err != nil {
		log.Fatalf("Failed to retrieve adapter details: %v", err)
	}

	collector.(*systemNetworkCollector).logger.Info("Adapters retrieved", zap.Any("adapters", adapters))
}
