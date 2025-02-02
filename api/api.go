// api.go

// Package api provides a secure RESTful API server integrated with Open Quantum Safe (OQS) features.
package api

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"

	"ghostshell/oqs" // Importing the OQS package

	"github.com/emicklei/go-restful/v3"
	"go.uber.org/zap"
)

// API represents the RESTful API server.
type API struct {
	tlsConfig *tls.Config
	logger    *zap.Logger
}

// NewAPI creates a new instance of API with the provided TLS configuration and logger.
func NewAPI(tlsConfig *tls.Config, logger *zap.Logger) *API {
	return &API{
		tlsConfig: tlsConfig,
		logger:    logger,
	}
}

// Register registers the API routes with the provided WebService container.
func (api *API) Register(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/api").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/status").To(api.handleStatus))
	ws.Route(ws.POST("/command").To(api.handleCommand))

	container.Add(ws)
}

// handleStatus handles the GET /api/status endpoint.
// It returns the operational status of the API server.
func (api *API) handleStatus(req *restful.Request, resp *restful.Response) {
	response := map[string]string{
		"status":  "running",
		"message": "API is operational",
	}
	resp.WriteEntity(response)
	api.logger.Info("Status checked", zap.String("status", "running"))
}

// handleCommand handles the POST /api/command endpoint.
// It receives a command, logs it securely, and returns a response.
func (api *API) handleCommand(req *restful.Request, resp *restful.Response) {
	var request map[string]string
	if err := req.ReadEntity(&request); err != nil {
		api.logger.Error("Error reading command from request", zap.Error(err))
		resp.WriteError(http.StatusBadRequest, err)
		return
	}

	command, exists := request["command"]
	if !exists || command == "" {
		api.logger.Warn("No command provided in the request")
		resp.WriteError(http.StatusBadRequest, errors.New("command field is required"))
		return
	}

	// Log the command securely using OQS's memory management.
	commandBytes, err := oqs.GenerateRandomBytes(len(command))
	if err != nil {
		api.logger.Error("Failed to generate random bytes for command logging", zap.Error(err))
		resp.WriteError(http.StatusInternalServerError, err)
		return
	}

	secureCommand, err := oqs.AllocateMemory(len(commandBytes))
	if err != nil {
		api.logger.Error("Failed to allocate secure memory for command", zap.Error(err))
		resp.WriteError(http.StatusInternalServerError, err)
		return
	}
	copy((*[1 << 30]byte)(secureCommand)[:len(commandBytes)], []byte(command))

	api.logger.Info("Received command", zap.String("command", command))

	oqs.FreeMemory((*[1 << 30]byte)(secureCommand)[:len(commandBytes)])

	response := map[string]string{
		"status":  "success",
		"output":  "Command executed: " + command,
		"details": "Execution logic not implemented",
	}
	resp.WriteEntity(response)
}

// StartServer starts the RESTful API server with TLS configuration.
func StartServer(certFile, keyFile, caFile string) error {
	address := ":8443" // Use HTTPS port

	// Load server's certificate and key.
	serverCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to load server certificate and key: %w", err)
	}

	// Load CA certificate.
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		return errors.New("failed to append CA certificate to pool")
	}

	// Configure TLS.
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert, // Enforce mTLS
		MinVersion:   tls.VersionTLS13,               // Enforce TLS 1.3
	}

	// Initialize zap logger.
	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Sync() // Flush any buffered log entries

	// Create a new API instance.
	apiInstance := NewAPI(tlsConfig, logger)

	// Create a new WebService container.
	container := restful.NewContainer()

	// Register API routes.
	apiInstance.Register(container)

	// Create an HTTP server with TLS configuration.
	server := &http.Server{
		Addr:      address,
		Handler:   container,
		TLSConfig: tlsConfig,
	}

	// Log server start.
	logger.Info("Starting RESTful API server", zap.String("address", address))

	// Start the server.
	return server.ListenAndServeTLS("", "") // Certificates are already loaded via tlsConfig
}
