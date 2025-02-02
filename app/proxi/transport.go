package proxi

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"ghostshell/ghostshell/oqs"
	"ghostshell/ghostshell/vault"

	"go.uber.org/zap"
)

// Transport encapsulates a secure HTTP transport configuration
// with post-quantum security features.
type Transport struct {
	TLSConfig     *tls.Config
	HTTPClient    *http.Client
	logger        *zap.Logger
	encryptionKey []byte
	keyMutex      sync.RWMutex
}

// NewTransport creates and returns a new secure Transport instance
func NewTransport(certFile, keyFile, caFile string, logger *zap.Logger) (*Transport, error) {
	// Load client certificate and key
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		logger.Error("Failed to load client certificate and key", zap.Error(err))
		return nil, fmt.Errorf("failed to load client certificate and key: %w", err)
	}

	// Load CA certificate
	caCert, err := vault.ReadFileSecurely(caFile)
	if err != nil {
		logger.Error("Failed to read CA certificate", zap.Error(err))
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		logger.Error("Failed to append CA certificate to pool")
		return nil, errors.New("failed to append CA certificate to pool")
	}

	// Generate post-quantum encryption key
	encryptionKey, err := oqs.GenerateRandomBytes(32)
	if err != nil {
		logger.Error("Failed to generate encryption key", zap.Error(err))
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Create TLS configuration
	tlsConfig := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		RootCAs:                  caCertPool,
		MinVersion:               tls.VersionTLS13,
		CipherSuites:             oqs.SupportedCipherSuites(),
		PreferServerCipherSuites: true,
	}

	// Create an HTTP client with secure transport
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
		Timeout: 30 * time.Second,
	}

	logger.Info("Transport initialized successfully")

	return &Transport{
		TLSConfig:     tlsConfig,
		HTTPClient:    httpClient,
		logger:        logger,
		encryptionKey: encryptionKey,
	}, nil
}

// SecureDo performs a secure HTTP request with additional encryption support
func (t *Transport) SecureDo(req *http.Request) (*http.Response, error) {
	// Encrypt request body if present
	if req.Body != nil {
		encryptedBody, err := oqs.EncryptRequestBody(req.Body, t.encryptionKey)
		if err != nil {
			t.logger.Error("Failed to encrypt request body", zap.Error(err))
			return nil, fmt.Errorf("failed to encrypt request body: %w", err)
		}
		req.Body = encryptedBody
	}

	t.logger.Info("Sending secure HTTP request", zap.String("url", req.URL.String()))
	resp, err := t.HTTPClient.Do(req)
	if err != nil {
		t.logger.Error("HTTP request failed", zap.Error(err))
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	// Decrypt response body if present
	if resp.Body != nil {
		decryptedBody, err := oqs.DecryptResponseBody(resp.Body, t.encryptionKey)
		if err != nil {
			t.logger.Error("Failed to decrypt response body", zap.Error(err))
			return nil, fmt.Errorf("failed to decrypt response body: %w", err)
		}
		resp.Body = decryptedBody
	}

	t.logger.Info("Secure HTTP request completed", zap.String("url", req.URL.String()), zap.Int("status", resp.StatusCode))
	return resp, nil
}

// RotateEncryptionKey rotates the encryption key securely
func (t *Transport) RotateEncryptionKey() error {
	t.keyMutex.Lock()
	defer t.keyMutex.Unlock()

	newKey, err := oqs.GenerateRandomBytes(32)
	if err != nil {
		t.logger.Error("Failed to generate new encryption key", zap.Error(err))
		return fmt.Errorf("failed to generate new encryption key: %w", err)
	}

	t.encryptionKey = newKey
	t.logger.Info("Encryption key rotated successfully")
	return nil
}
