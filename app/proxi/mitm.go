package proxi

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"

	"ghostshell/ghostshell/oqs"

	"github.com/elazarl/goproxy"
	"go.uber.org/zap"
)

// MITMProxy handles Man-in-the-Middle proxying with TLS interception
type MITMProxy struct {
	proxy         *goproxy.ProxyHttpServer
	logger        *zap.Logger
	encryptionKey []byte
}

// NewMITMProxy creates a new MITMProxy instance with enhanced logging and security
func NewMITMProxy(logger *zap.Logger) (*MITMProxy, error) {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true

	// Generate encryption key using OQS for enhanced security
	encryptionKey, err := oqs.GenerateRandomBytes(32)
	if err != nil {
		logger.Error("Failed to generate encryption key", zap.Error(err))
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		logger.Info("Intercepted request", zap.String("method", req.Method), zap.String("url", req.URL.String()))
		return req, nil
	})

	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		logger.Info("Intercepted response", zap.String("status", resp.Status))
		return resp
	})

	logger.Info("MITM proxy initialized")
	return &MITMProxy{proxy: proxy, logger: logger, encryptionKey: encryptionKey}, nil
}

// Start starts the MITM proxy on the given address and port
func (m *MITMProxy) Start(listenAddress string, port int) error {
	address := fmt.Sprintf("%s:%d", listenAddress, port)
	m.logger.Info("Starting MITM proxy", zap.String("address", address))
	return http.ListenAndServe(address, m.proxy)
}

// GenerateTLSConfig generates a TLS configuration for MITM with OQS support
func GenerateTLSConfig(logger *zap.Logger) (*tls.Config, error) {
	caCert, caKey, err := GenerateCA(logger)
	if err != nil {
		logger.Error("Failed to generate CA", zap.Error(err))
		return nil, fmt.Errorf("failed to generate CA: %w", err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*caCert},
		RootCAs:      certPool,
		ClientCAs:    certPool,
		MinVersion:   tls.VersionTLS13, // Enforce TLS 1.3
	}

	logger.Info("TLS configuration generated with TLS 1.3 enforced")
	return tlsConfig, nil
}

// GenerateCA generates a self-signed Certificate Authority for MITM
func GenerateCA(logger *zap.Logger) (*tls.Certificate, *tls.Certificate, error) {
	// Placeholder implementation
	logger.Warn("GenerateCA not implemented")
	return nil, nil, errors.New("GenerateCA not implemented")
}

// Close releases resources used by the proxy
func (m *MITMProxy) Close() {
	if len(m.encryptionKey) > 0 {
		// Securely clear the encryption key
		oqs.ZeroMemory(m.encryptionKey)
		m.logger.Info("Encryption key securely cleared")
	}
	m.logger.Sync()
}
