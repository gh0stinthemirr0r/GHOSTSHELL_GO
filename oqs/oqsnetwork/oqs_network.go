// OQS_Network.go
// Enhanced for modular and robust quantum-safe networking

package oqs_network

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// Initialize logger with dynamic file naming
var logger *zap.SugaredLogger

func init() {
	currentTime := time.Now().UTC().Format("20060102_150405")
	logFileName1 := fmt.Sprintf("postquantumsecurity_log_%s.log", currentTime)
	logFileName2 := fmt.Sprintf("netlog_%s.log", currentTime)

	// Configure zap to write to both log files and stdout
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{
		logFileName1,
		logFileName2,
		"stdout", // Also write logs to the console
	}

	// Build the logger
	log, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}

	// Set the global logger
	logger = log.Sugar()

	// Log initialization information
	logger.Infof("Logger initialized with files: %s and %s", logFileName1, logFileName2)
}

// OQS_STATUS defines the status codes for operations.
type OQS_STATUS int

const (
	OQS_SUCCESS OQS_STATUS = iota
	OQS_ERROR
)

// Error definitions
var (
	ErrAlreadyConnected    = errors.New("already connected to the specified address")
	ErrConnectionNotFound  = errors.New("connection not found")
	ErrFailedToConnect     = errors.New("failed to connect")
	ErrFailedToSendData    = errors.New("failed to send data")
	ErrFailedToReceiveData = errors.New("failed to receive data")
	ErrFailedToDisconnect  = errors.New("failed to disconnect")
	ErrTLSConfiguration    = errors.New("invalid TLS configuration")
	ErrInvalidProtocol     = errors.New("invalid protocol")
)

// CertManager interface manages certificates.
type CertManager interface {
	LoadClientCert() (tls.Certificate, error)
	LoadRootCAs() (*x509.CertPool, error)
}

// OQSNetwork encapsulates secure network communication.
type OQSNetwork struct {
	TLSConfig   *tls.Config
	Connections map[string]interface{}
	mutex       sync.RWMutex
	certManager CertManager
}

// NewOQSNetwork initializes a new OQSNetwork instance.
func NewOQSNetwork(certMgr CertManager) (*OQSNetwork, error) {
	rootCAs, err := certMgr.LoadRootCAs()
	if err != nil {
		logger.Errorf("Failed to load root CAs: %v", err)
		return nil, fmt.Errorf("%w: %v", ErrTLSConfiguration, err)
	}

	clientCert, err := certMgr.LoadClientCert()
	if err != nil {
		logger.Warnf("Failed to load client certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS13,
		RootCAs:      rootCAs,
		Certificates: []tls.Certificate{clientCert},
	}

	return &OQSNetwork{
		TLSConfig:   tlsConfig,
		Connections: make(map[string]interface{}),
		certManager: certMgr,
	}, nil
}

// Connect establishes a connection for various protocols.
func (n *OQSNetwork) Connect(address string, protocol string) (interface{}, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if _, exists := n.Connections[address]; exists {
		return nil, fmt.Errorf("%w: %s", ErrAlreadyConnected, address)
	}

	var conn interface{}
	var err error

	switch protocol {
	case "tcp":
		conn, err = tls.Dial("tcp", address, n.TLSConfig)
	case "ssh":
		sshConfig := &ssh.ClientConfig{
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
		conn, err = ssh.Dial("tcp", address, sshConfig)
	default:
		return nil, ErrInvalidProtocol
	}

	if err != nil {
		logger.Errorf("Failed to connect to %s: %v", address, err)
		return nil, fmt.Errorf("%w to %s: %v", ErrFailedToConnect, address, err)
	}

	n.Connections[address] = conn
	logger.Infof("Successfully connected to %s via %s", address, protocol)
	return conn, nil
}

// Disconnect securely closes the connection to the given address.
func (n *OQSNetwork) Disconnect(address string) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	conn, exists := n.Connections[address]
	if !exists {
		return fmt.Errorf("%w: %s", ErrConnectionNotFound, address)
	}

	netConn, ok := conn.(net.Conn)
	if ok {
		err := netConn.Close()
		if err != nil {
			logger.Errorf("Failed to disconnect from %s: %v", address, err)
			return fmt.Errorf("%w from %s: %v", ErrFailedToDisconnect, address, err)
		}
	}

	delete(n.Connections, address)
	logger.Infof("Disconnected from %s", address)
	return nil
}
