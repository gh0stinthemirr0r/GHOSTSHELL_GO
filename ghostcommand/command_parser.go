// File: command_parser.go
package ghostcommand

import (
	"fmt"
	"sync"

	"go.uber.org/zap"

	// Importing the local post-quantum secure packages
	"ghostshell/oqs/kem"
)

// ErrorHandler defines the interface for handling errors.
// It should be implemented by the ErrorHandler struct.
type ErrorHandler interface {
	// HandleError logs or handles errors with a specific context and message.
	HandleError(context, message string)
}

// CommandParser manages the generation of quantum-safe key pairs and parsing of commands.
type CommandParser struct {
	errorHandler ErrorHandler
	kemScheme    *kem.Scheme
	logger       *zap.Logger
	mutex        sync.Mutex
}

// NewCommandParser initializes and returns a new instance of CommandParser.
// It requires implementations of ErrorHandler and a zap.Logger.
func NewCommandParser(handler ErrorHandler, logger *zap.Logger) (*CommandParser, error) {
	// Initialize the KEM scheme (Kyber-512)
	kemScheme, err := kem.NewScheme("Kyber512")
	if err != nil {
		logger.Error("Failed to initialize KEM scheme", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize KEM scheme: %w", err)
	}

	return &CommandParser{
		errorHandler: handler,
		kemScheme:    kemScheme,
		logger:       logger,
	}, nil
}

// Shutdown gracefully shuts down the CommandParser and cleans up resources.
func (cp *CommandParser) Shutdown() {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	cp.logger.Info("Shutting down CommandParser.")

	// Free the KEM scheme resources
	if err := cp.kemScheme.Free(); err != nil {
		cp.logger.Error("Failed to free KEM scheme", zap.Error(err))
	} else {
		cp.logger.Info("KEM scheme resources freed successfully.")
	}

	// Sync the logger to flush any pending logs
	_ = cp.logger.Sync()
}

// GenerateQuantumSafeKeyPair generates a quantum-safe key pair using Kyber-512.
// It returns the public and private keys as hex-encoded strings.
func (cp *CommandParser) GenerateQuantumSafeKeyPair() (string, string, error) {
	cp.logger.Info("Generating quantum-safe key pair using Kyber-512.")

	// Generate a key pair
	publicKeyBytes, privateKeyBytes, err := cp.kemScheme.Keypair()
	if err != nil {
		cp.logger.Error("Failed to generate key pair", zap.Error(err))
		cp.errorHandler.HandleError("GenerateQuantumSafeKeyPair", "Failed to generate quantum-safe key pair.")
		return "", "", fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Encode keys as hex strings
	publicKey := fmt.Sprintf("%x", publicKeyBytes)
	privateKey := fmt.Sprintf("%x", privateKeyBytes)

	cp.logger.Info("Quantum-safe key pair generated successfully.")
	return publicKey, privateKey, nil
}

// ParseCommand parses a given command string and returns the parsed output.
// It returns true if the command was parsed successfully, false otherwise.
func (cp *CommandParser) ParseCommand(command string) (string, bool) {
	cp.logger.Info("Parsing command.", zap.String("command", command))

	// Placeholder logic for parsing the command
	parsedOutput := "Parsed: " + command
	fmt.Println("[CommandParser] Command parsed successfully:", parsedOutput)

	return parsedOutput, true
}
