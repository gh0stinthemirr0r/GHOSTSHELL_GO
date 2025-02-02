// File: command_executor.go
package ghostcommand

import (
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"

	// Importing the local post-quantum secure packages
	"ghostshell/ghostshell/oqs/kem"
)

// ErrorHandler defines the interface for handling errors.
// It should be implemented by the ErrorHandler struct.
type ErrorHandler interface {
	// HandleError logs or handles errors with a specific context and message.
	HandleError(context, message string)
}

// GhostVault defines the interface for key pair generation and management.
type GhostVault interface {
	// GenerateVaultKeyPair generates a key pair for encryption/decryption.
	// Returns the public and private keys as hex-encoded strings.
	GenerateVaultKeyPair() (publicKey string, privateKey string, err error)
}

// CommandExecutor manages the registration and execution of commands with post-quantum security.
type CommandExecutor struct {
	commandRegistry map[string]func([]string)
	errorHandler    ErrorHandler
	ghostVault      GhostVault
	kemScheme       *kem.Scheme
	logger          *zap.Logger
	mutex           sync.Mutex
}

// NewCommandExecutor initializes and returns a new instance of CommandExecutor.
// It requires implementations of GhostVault and ErrorHandler interfaces.
func NewCommandExecutor(ghostVault GhostVault, handler ErrorHandler) (*CommandExecutor, error) {
	// Initialize zap logger
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize zap logger: %w", err)
	}

	logger.Info("Initializing CommandExecutor with Post-Quantum Security.")

	// Initialize the KEM scheme (Kyber-512)
	kemScheme, err := kem.NewScheme("Kyber512")
	if err != nil {
		logger.Error("Failed to initialize KEM scheme", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize KEM scheme: %w", err)
	}

	// Initialize commandRegistry map
	commandRegistry := make(map[string]func([]string))

	return &CommandExecutor{
		commandRegistry: commandRegistry,
		errorHandler:    handler,
		ghostVault:      ghostVault,
		kemScheme:       kemScheme,
		logger:          logger,
	}, nil
}

// Shutdown gracefully shuts down the CommandExecutor and cleans up resources.
func (ce *CommandExecutor) Shutdown() {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()

	ce.logger.Info("Shutting down CommandExecutor.")

	// Free the KEM scheme resources
	if err := ce.kemScheme.Free(); err != nil {
		ce.logger.Error("Failed to free KEM scheme", zap.Error(err))
	} else {
		ce.logger.Info("KEM scheme resources freed successfully.")
	}

	// Sync the logger to flush any pending logs
	_ = ce.logger.Sync()
}

// RegisterCommand adds a new command and its handler to the registry.
// It returns true if the command is registered successfully, false otherwise.
func (ce *CommandExecutor) RegisterCommand(commandName string, handler func([]string)) (bool, error) {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()

	if _, exists := ce.commandRegistry[commandName]; exists {
		ce.errorHandler.HandleError("RegisterCommand", "Command already registered: "+commandName)
		return false, fmt.Errorf("command already registered: %s", commandName)
	}

	ce.commandRegistry[commandName] = handler
	ce.logger.Info("Registered command.", zap.String("command", commandName))
	return true, nil
}

// ExecuteCommand encrypts the command and its parameters, decrypts them, and executes the associated handler.
// It returns true if the command is executed successfully, false otherwise.
func (ce *CommandExecutor) ExecuteCommand(commandName string, parameters []string) (bool, error) {
	// Encrypt the command and parameters before execution
	encryptedCommand, err := ce.EncryptCommand(commandName, parameters)
	if err != nil {
		ce.errorHandler.HandleError("ExecuteCommand", "Failed to encrypt command: "+err.Error())
		return false, fmt.Errorf("failed to encrypt command: %w", err)
	}

	// Decrypt the command and parameters before actually executing
	decryptedCommandName, decryptedParameters, err := ce.DecryptCommand(encryptedCommand)
	if err != nil {
		ce.errorHandler.HandleError("ExecuteCommand", "Failed to decrypt command: "+err.Error())
		return false, fmt.Errorf("failed to decrypt command: %w", err)
	}

	// Retrieve the command handler
	handler, exists := ce.commandRegistry[decryptedCommandName]
	if !exists {
		ce.errorHandler.HandleError("ExecuteCommand", "Command not found: "+decryptedCommandName)
		return false, fmt.Errorf("command not found: %s", decryptedCommandName)
	}

	// Execute the command handler
	defer func() {
		if r := recover(); r != nil {
			ce.errorHandler.HandleError("ExecuteCommand", fmt.Sprintf("Command execution panicked: %v", r))
		}
	}()

	handler(decryptedParameters)
	ce.logger.Info("Executed command successfully.", zap.String("command", decryptedCommandName))
	return true, nil
}

// EncryptCommand encrypts the command name and parameters using the KEM scheme.
// It returns the encrypted command as a string.
func (ce *CommandExecutor) EncryptCommand(commandName string, parameters []string) (string, error) {
	// Generate a key pair for encryption
	publicKey, privateKey, err := ce.ghostVault.GenerateVaultKeyPair()
	if err != nil {
		return "", fmt.Errorf("failed to generate vault key pair: %w", err)
	}

	// Serialize the command name and parameters
	commandData := strings.Join(append([]string{commandName}, parameters...), " ")

	// Encapsulate to generate ciphertext and shared secret
	ciphertext, sharedSecret, err := ce.kemScheme.Encapsulate([]byte(publicKey))
	if err != nil {
		return "", fmt.Errorf("failed to encapsulate shared secret: %w", err)
	}

	// For demonstration purposes, we'll concatenate ciphertext and sharedSecret
	// In a real-world scenario, you'd use the sharedSecret to encrypt the commandData with a symmetric cipher
	encryptedCommandBytes := append(ciphertext, sharedSecret...)
	encryptedCommand := hex.EncodeToString(encryptedCommandBytes)

	ce.logger.Info("Command encrypted successfully.", zap.String("command", commandName))
	return encryptedCommand, nil
}

// DecryptCommand decrypts the encrypted command using the KEM scheme.
// It returns the decrypted command name and parameters.
func (ce *CommandExecutor) DecryptCommand(encryptedCommand string) (string, []string, error) {
	// Decode the hex-encoded encrypted command
	encryptedCommandBytes, err := hex.DecodeString(encryptedCommand)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decode encrypted command: %w", err)
	}

	// Split ciphertext and shared secret based on KEM scheme lengths
	if len(encryptedCommandBytes) < ce.kemScheme.LengthCiphertext+ce.kemScheme.LengthSharedSecret {
		return "", nil, fmt.Errorf("encrypted command length is insufficient")
	}

	ciphertext := encryptedCommandBytes[:ce.kemScheme.LengthCiphertext]
	sharedSecret := encryptedCommandBytes[ce.kemScheme.LengthCiphertext:]

	// Retrieve the key pair for decryption
	publicKey, privateKey, err := ce.ghostVault.GenerateVaultKeyPair()
	if err != nil {
		return "", nil, fmt.Errorf("failed to retrieve vault key pair: %w", err)
	}

	// Decapsulate to retrieve the shared secret
	retrievedSharedSecret, err := ce.kemScheme.Decapsulate(ciphertext, []byte(privateKey))
	if err != nil {
		return "", nil, fmt.Errorf("failed to decapsulate shared secret: %w", err)
	}

	// Verify that the shared secrets match
	if !compareBytes(sharedSecret, retrievedSharedSecret) {
		return "", nil, fmt.Errorf("shared secret mismatch during decryption")
	}

	// For demonstration purposes, we'll assume that the shared secret is the serialized commandData
	// In a real-world scenario, you'd decrypt the commandData using the sharedSecret with a symmetric cipher
	commandData := string(sharedSecret)
	parts := strings.Split(commandData, " ")
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("decrypted command data is empty")
	}

	commandName := parts[0]
	parameters := parts[1:]

	ce.logger.Info("Command decrypted successfully.", zap.String("command", commandName))
	return commandName, parameters, nil
}

// compareBytes compares two byte slices for equality.
func compareBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, byteA := range a {
		if byteA != b[i] {
			return false
		}
	}
	return true
}
