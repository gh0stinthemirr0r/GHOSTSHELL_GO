// Package grpc provides a secure gRPC client integrated with Open Quantum Safe (OQS) features for terminal communications.
package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"sync"

	usermanagementpb "ghostshell/gRPC/Proto/usermanagementpb"
	"ghostshell/ghostshell/oqs"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GRPCClient manages the gRPC connection and provides methods to interact with the server.
type GRPCClient struct {
	connection *grpc.ClientConn
	mutex      sync.RWMutex
	logger     *zap.Logger
	client     usermanagementpb.UserServiceClient // Client for the UserService
}

// NewGRPCClient creates a new GRPCClient instance with a secure OQS TLS connection.
func NewGRPCClient(target, certFile, keyFile, caFile string, logger *zap.Logger) (*GRPCClient, error) {
	// Generate OQS-supported key pairs.
	oqsCert, oqsKey, err := oqs.GenerateCertificate(certFile, keyFile)
	if err != nil {
		logger.Error("Failed to generate OQS certificate and key", zap.Error(err))
		return nil, fmt.Errorf("failed to generate OQS certificate and key: %w", err)
	}

	// Load the client's certificate and private key.
	clientCert, err := tls.X509KeyPair(oqsCert, oqsKey)
	if err != nil {
		logger.Error("Failed to load client certificate and key", zap.Error(err))
		return nil, fmt.Errorf("failed to load client certificate and key: %w", err)
	}

	// Load the CA certificate.
	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		logger.Error("Failed to read CA certificate", zap.Error(err))
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		logger.Error("Failed to append CA certificate to pool")
		return nil, errors.New("failed to append CA certificate to pool")
	}

	// Configure TLS with OQS-supported cipher suites.
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{clientCert},
		RootCAs:            caCertPool,
		MinVersion:         tls.VersionTLS13, // Enforce TLS 1.3 for enhanced security
		CipherSuites:       oqs.GetSupportedCipherSuites(),
		InsecureSkipVerify: false,
	}

	// Establish the gRPC connection with the secure TLS credentials.
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		logger.Error("Failed to connect to gRPC server", zap.String("target", target), zap.Error(err))
		return nil, fmt.Errorf("failed to connect to gRPC server at %s: %w", target, err)
	}

	client := usermanagementpb.NewUserServiceClient(conn) // Initialize the UserService client

	logger.Info("gRPC client connected", zap.String("target", target))
	return &GRPCClient{
		connection: conn,
		logger:     logger,
		client:     client,
	}, nil
}

// Close terminates the gRPC connection safely.
func (c *GRPCClient) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.connection == nil {
		return errors.New("gRPC connection is already closed or was never established")
	}

	err := c.connection.Close()
	if err != nil {
		c.logger.Error("Failed to close gRPC connection", zap.Error(err))
		return fmt.Errorf("failed to close gRPC connection: %w", err)
	}

	c.connection = nil
	c.logger.Info("gRPC connection closed successfully.")
	return nil
}

// CreateUser sends a CreateUserRequest to the server and returns the response.
func (c *GRPCClient) CreateUser(ctx context.Context, username, email, password string) (*usermanagementpb.CreateUserResponse, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.connection == nil {
		return nil, errors.New("gRPC connection is not established")
	}

	req := &usermanagementpb.CreateUserRequest{
		Username: username,
		Email:    email,
		Password: password,
	}

	resp, err := c.client.CreateUser(ctx, req)
	if err != nil {
		c.logger.Error("Failed to create user", zap.Error(err))
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return resp, nil
}

// GetUserProfile retrieves the user profile from the server.
func (c *GRPCClient) GetUserProfile(ctx context.Context, username string) (*usermanagementpb.GetUserProfileResponse, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.connection == nil {
		return nil, errors.New("gRPC connection is not established")
	}

	req := &usermanagementpb.GetUserProfileRequest{Username: username}
	resp, err := c.client.GetUserProfile(ctx, req)
	if err != nil {
		c.logger.Error("Failed to retrieve user profile", zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve user profile: %w", err)
	}

	return resp, nil
}

// IsConnected checks if the gRPC client is currently connected.
func (c *GRPCClient) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connection != nil
}
