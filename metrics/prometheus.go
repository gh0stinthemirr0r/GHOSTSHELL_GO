package metrics

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	// Hypothetical references to ephemeral encryption
)

// Config holds the configuration for the Prometheus metrics manager.
type Config struct {
	Port        string // e.g. "9090"
	Path        string // e.g. "/metrics"
	EnableHTTPS bool
	CertFile    string
	KeyFile     string

	Username string
	Password string // for BasicAuth

	// Optional ephemeral encryption
	EnableEncryption bool
	EncryptionKey    []byte

	// Potential placeholders for system or AI usage integration
	AiUsageEnabled       bool
	SystemMetricsEnabled bool
}

// MetricsManager manages a Prometheus registry & server with optional BasicAuth, ephemeral encryption, etc.
type MetricsManager struct {
	registry *prometheus.Registry
	server   *http.Server
	logger   *zap.Logger

	authCredentials  string
	encryptionKey    []byte
	ephemeralEnabled bool

	wg           sync.WaitGroup
	shutdownChan chan struct{}
}

// NewMetricsManager constructs a new manager with default process & Go runtime collectors, ephemeral init, etc.
func NewMetricsManager(logger *zap.Logger, cfg Config) (*MetricsManager, error) {
	if cfg.Port == "" {
		cfg.Port = "9090"
	}
	if cfg.Path == "" {
		cfg.Path = "/metrics"
	}
	if cfg.EnableHTTPS && (cfg.CertFile == "" || cfg.KeyFile == "") {
		return nil, errors.New("HTTPS enabled but CertFile/KeyFile not provided")
	}

	if logger == nil {
		logger = zap.NewExample()
	}

	// Post-quantum ephemeral init if requested
	if cfg.EnableEncryption {
		if len(cfg.EncryptionKey) != 32 {
			return nil, errors.New("encryption key must be 32 bytes if EnableEncryption is true")
		}
		// Potential ephemeral memory init
		if err := oqs_vault.InitializeSecureMemory(); err != nil {
			return nil, fmt.Errorf("failed to initialize secure memory: %w", err)
		}
		logger.Info("Post-quantum ephemeral encryption is enabled")
	}

	reg := prometheus.NewRegistry()
	mm := &MetricsManager{
		registry:         reg,
		logger:           logger,
		ephemeralEnabled: cfg.EnableEncryption,
		encryptionKey:    cfg.EncryptionKey,
		shutdownChan:     make(chan struct{}),
	}

	// Setup BasicAuth if username/pass provided
	if cfg.Username != "" && cfg.Password != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(cfg.Username + ":" + cfg.Password))
		mm.authCredentials = auth
	}

	// Register default collectors
	if err := mm.registerDefaultMetrics(); err != nil {
		return nil, err
	}

	// Create HTTP server with a custom mux for the route
	mux := http.NewServeMux()
	mux.Handle(cfg.Path, mm.authMiddleware(promhttp.HandlerFor(mm.registry, promhttp.HandlerOpts{})))

	mm.server = &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}
	if cfg.EnableHTTPS {
		mm.server.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return mm, nil
}

// registerDefaultMetrics adds the default process and Go runtime collectors
func (m *MetricsManager) registerDefaultMetrics() error {
	if err := m.registry.Register(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{})); err != nil {
		m.logger.Error("Failed to register process collector", zap.Error(err))
		return err
	}
	if err := m.registry.Register(prometheus.NewGoCollector()); err != nil {
		m.logger.Error("Failed to register go collector", zap.Error(err))
		return err
	}
	return nil
}

// RegisterMetric allows adding a custom metric collector
func (m *MetricsManager) RegisterMetric(col prometheus.Collector) error {
	if err := m.registry.Register(col); err != nil {
		m.logger.Error("Failed to register custom metric", zap.Error(err))
		return err
	}
	m.logger.Info("Registered custom metric", zap.String("metric", fmt.Sprintf("%T", col)))
	return nil
}

// authMiddleware handles BasicAuth if configured
func (m *MetricsManager) authMiddleware(handler http.Handler) http.Handler {
	if m.authCredentials == "" {
		return handler
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		expected := "Basic " + m.authCredentials
		if authHeader != expected {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			m.logger.Warn("Unauthorized metrics access attempt", zap.String("remote_addr", r.RemoteAddr))
			return
		}
		handler.ServeHTTP(w, r)
	})
}

// ServeMetrics starts the server (blocking call). Typically you'd run in a goroutine
func (m *MetricsManager) ServeMetrics(cfg Config) {
	m.wg.Add(1)
	defer m.wg.Done()

	m.logger.Info("Starting Prometheus metrics server", zap.String("port", cfg.Port), zap.String("path", cfg.Path))

	var err error
	if cfg.EnableHTTPS {
		err = m.server.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile)
	} else {
		err = m.server.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		m.logger.Fatal("Metrics server failed", zap.Error(err))
	}
	m.logger.Info("Metrics server stopped")
}

// StopMetricsServer gracefully stops the server
func (m *MetricsManager) StopMetricsServer(ctx context.Context) error {
	if m.server == nil {
		return errors.New("metrics server not running")
	}
	m.logger.Info("Shutting down metrics server...")
	err := m.server.Shutdown(ctx)
	if err != nil {
		m.logger.Error("Error shutting down metrics server", zap.Error(err))
	}
	close(m.shutdownChan)
	m.wg.Wait()
	m.logger.Info("Metrics server shut down gracefully")
	return err
}

// RunWithGracefulShutdown runs the metrics server in a goroutine and waits for signals
func (m *MetricsManager) RunWithGracefulShutdown(cfg Config) {
	go m.ServeMetrics(cfg)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	s := <-signalChan
	m.logger.Info("Received shutdown signal", zap.String("signal", s.String()))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := m.StopMetricsServer(ctx); err != nil {
		m.logger.Error("Error during server shutdown", zap.Error(err))
	} else {
		m.logger.Info("Metrics server shut down successfully")
	}
}

// ephemeral encryption placeholder for numeric values
func (m *MetricsManager) ApplyEphemeralEncryption(value float64) float64 {
	if !m.ephemeralEnabled || len(m.encryptionKey) == 0 {
		// no encryption, return raw
		return value
	}
	encVal := ephemeralEncryptFloat(value, m.encryptionKey)
	return encVal
}

// ephemeralEncryptFloat is a toy example that might do partial ephemeral encryption
func ephemeralEncryptFloat(raw float64, key []byte) float64 {
	// Real ephemeral encryption might do KEM/AES etc.
	// For demonstration, we just convert to string and fake “cipher length”
	s := fmt.Sprintf("%.3f", raw)
	// e.g. reverse + key-based offset
	rev := reverseString(s)
	// final length
	clen := float64(len(rev) + len(key))
	return clen
}

func reverseString(in string) string {
	runes := []rune(in)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// GetRegistry returns the underlying registry for advanced usage
func (m *MetricsManager) GetRegistry() *prometheus.Registry {
	return m.registry
}
