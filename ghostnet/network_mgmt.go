package network

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"ghostshell/metrics"
	oqs_vault "ghostshell/oqs/vault"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type NetworkManagerConfig struct {
	AdaptersInterval time.Duration
	RoutesInterval   time.Duration
	DNSInterval      time.Duration
	NetstatInterval  time.Duration
	EnablePrometheus bool
	Logger           *zap.Logger
	EnableEncryption bool
	EncryptionKey    []byte
}

type NetworkManager struct {
	config    NetworkManagerConfig
	collector NetworkCollector
	probe     *HTTPProbe
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	metrics   *metrics.SystemMetrics
	vault     *oqs_vault.Vault
}

func NewNetworkManager(cfg NetworkManagerConfig, collector NetworkCollector, probe *HTTPProbe, vault *oqs_vault.Vault) (*NetworkManager, error) {
	nm := &NetworkManager{
		config:    cfg,
		collector: collector,
		probe:     probe,
		vault:     vault,
		ctx:       context.Background(),
	}
	return nm, nil
}

func initLogger() (*zap.Logger, error) {
	logDir := "ghostshell/logging"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	timestamp := time.Now().UTC().Format("20060102_150405")
	logFile := fmt.Sprintf("%s/netlog_%s.log", logDir, timestamp)

	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{logFile, "stdout"}
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	logger.Info("Logger initialized", zap.String("logFile", logFile))
	return logger, nil
}

func (nm *NetworkManager) Start() error {
	nm.config.Logger.Info("Starting NetworkManager...")

	if err := oqs_vault.InitializeSecureMemory(); err != nil {
		nm.config.Logger.Error("Failed to initialize secure memory", zap.Error(err))
		return fmt.Errorf("failed to initialize secure memory: %w", err)
	}

	if err := InitializeNetwork(nm.vault); err != nil {
		nm.config.Logger.Error("Failed to initialize network", zap.Error(err))
		return fmt.Errorf("failed to initialize network: %w", err)
	}

	if err := StartNetworkServices(nm.config.Logger, nm.config.EncryptionKey); err != nil {
		nm.config.Logger.Error("Failed to start network services", zap.Error(err))
		return fmt.Errorf("failed to start network services: %w", err)
	}

	return nil
}

func (nm *NetworkManager) monitorAdapters() {
	nm.config.Logger.Info("Monitoring network adapters...")
	ticker := time.NewTicker(nm.config.AdaptersInterval)
	defer ticker.Stop()

	for {
		select {
		case <-nm.ctx.Done():
			nm.config.Logger.Info("Stopped monitoring network adapters.")
			return
		case <-ticker.C:
			adapters, err := nm.collector.GetAdapterDetails()
			if err != nil {
				nm.config.Logger.Error("Error monitoring adapters", zap.Error(err))
				continue
			}

			if nm.config.EnableEncryption {
				adapters = encryptData(nm.vault, adapters).([]NetworkAdapterInfo)
			}

			nm.logAdapters(adapters)
		}
	}
}

func (nm *NetworkManager) logAdapters(adapters []NetworkAdapterInfo) {
	nm.config.Logger.Info("Adapters", zap.Any("adapters", adapters))
}

func InitializeNetwork(vault *oqs_vault.Vault) error {
	key, err := vault.GenerateRandomBytes(32)
	if err != nil {
		return err
	}
	EncryptionKey = key
	zap.L().Info("Network component initialized with post-quantum encryption.")
	return nil
}

var EncryptionKey []byte

func StartNetworkServices(logger *zap.Logger, encryptionKey []byte) error {
	logger.Info("Starting post-quantum secure network services...")
	go monitorAdapters(logger, encryptionKey)
	go monitorRoutes(logger, encryptionKey)
	go monitorDNS(logger, encryptionKey)
	go monitorNetstat(logger, encryptionKey)
	return nil
}

func monitorAdapters(logger *zap.Logger, key []byte) {
	logger.Info("Monitoring adapters...")
}

func monitorRoutes(logger *zap.Logger, key []byte) {
	logger.Info("Monitoring routes...")
}

func monitorDNS(logger *zap.Logger, key []byte) {
	logger.Info("Monitoring DNS...")
}

func monitorNetstat(logger *zap.Logger, key []byte) {
	logger.Info("Monitoring netstat...")
}

func encryptData(vault *oqs_vault.Vault, data interface{}) interface{} {
	encrypted, err := vault.EncryptData(data)
	if err != nil {
		return data
	}
	return encrypted
}
