// File: command_queue.go
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

// CommandQueue manages the encryption and queuing of commands.
type CommandQueue struct {
	encryptedQueue []string     // Slice to hold encrypted commands
	mutex          sync.Mutex   // Mutex to protect access to the queue
	condVar        *sync.Cond   // Condition variable for synchronization
	errorHandler   ErrorHandler // Error handler instance
	ghostVault     GhostVault   // GhostVault instance for key management
	kemScheme      *kem.Scheme  // KEM scheme for encryption/decryption
	logger         *zap.Logger  // Logger for structured logging
}

// NewCommandQueue initializes and returns a new instance of CommandQueue.
// It requires implementations of GhostVault and ErrorHandler interfaces.
func NewCommandQueue(ghostVault GhostVault, handler ErrorHandler, logger *zap.Logger) (*CommandQueue, error) {
	// Initialize the KEM scheme (Kyber-512)
	kemScheme, err := kem.NewScheme("Kyber512")
	if err != nil {
		logger.Error("Failed to initialize KEM scheme", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize KEM scheme: %w", err)
	}

	// Initialize the command queue slice
	encryptedQueue := make([]string, 0)

	// Initialize the mutex and condition variable
	mutex := sync.Mutex{}
	condVar := sync.NewCond(&mutex)

	return &CommandQueue{
		encryptedQueue: encryptedQueue,
		mutex:          mutex,
		condVar:        condVar,
		errorHandler:   handler,
		ghostVault:     ghostVault,
		kemScheme:      kemScheme,
		logger:         logger,
	}, nil
}

// Shutdown gracefully shuts down the CommandQueue and cleans up resources.
func (cq *CommandQueue) Shutdown() {
	cq.mutex.Lock()
	defer cq.mutex.Unlock()

	cq.logger.Info("Shutting down CommandQueue.")

	// Free the KEM scheme resources
	if err := cq.kemScheme.Free(); err != nil {
		cq.logger.Error("Failed to free KEM scheme", zap.Error(err))
	} else {
		cq.logger.Info("KEM scheme resources freed successfully.")
	}

	// Sync the logger to flush any pending logs
	_ = cq.logger.Sync()
}

// Enqueue adds a new command to the queue after encrypting it.
// It takes a command name and a slice of parameters.
// Returns true if the command is enqueued successfully, false otherwise.
func (cq *CommandQueue) Enqueue(commandName string, parameters []string) (bool, error) {
	cq.logger.Info("Enqueuing command.", zap.String("command", commandName), zap.Strings("parameters", parameters))

	// Encrypt the command
	encryptedCommand, err := cq.EncryptCommand(commandName, parameters)
	if err != nil {
		cq.errorHandler.HandleError("Enqueue", "Failed to encrypt command: "+err.Error())
		return false, fmt.Errorf("failed to encrypt command: %w", err)
	}

	// Lock the mutex before modifying the queue
	cq.mutex.Lock()
	defer cq.mutex.Unlock()

	// Append the encrypted command to the queue
	cq.encryptedQueue = append(cq.encryptedQueue, encryptedCommand)

	// Notify one waiting goroutine
	cq.condVar.Signal()

	cq.logger.Info("Command enqueued successfully.", zap.String("command", commandName))
	return true, nil
}

// Dequeue retrieves and decrypts the next command from the queue.
// It blocks until a command is available.
// Returns the command name and parameters, or an error if decryption fails.
func (cq *CommandQueue) Dequeue() (string, []string, error) {
	cq.mutex.Lock()
	defer cq.mutex.Unlock()

	// Wait until the queue is not empty
	cq.condVar.Wait()

	// Retrieve the first encrypted command
	if len(cq.encryptedQueue) == 0 {
		cq.errorHandler.HandleError("Dequeue", "Attempted to dequeue from an empty queue.")
		return "", nil, fmt.Errorf("command queue is empty")
	}
	encryptedCommand := cq.encryptedQueue[0]
	cq.encryptedQueue = cq.encryptedQueue[1:]

	cq.logger.Info("Dequeuing command.")

	// Decrypt the command
	commandName, parameters, err := cq.DecryptCommand(encryptedCommand)
	if err != nil {
		cq.errorHandler.HandleError("Dequeue", "Failed to decrypt command: "+err.Error())
		return "", nil, fmt.Errorf("failed to decrypt command: %w", err)
	}

	cq.logger.Info("Command dequeued and decrypted successfully.", zap.String("command", commandName))
	return commandName, parameters, nil
}

// EncryptCommand encrypts the command name and parameters using the KEM scheme.
// It returns the encrypted command as a hex-encoded string.
func (cq *CommandQueue) EncryptCommand(commandName string, parameters []string) (string, error) {
	// Generate a key pair for encryption
	publicKey, privateKey, err := cq.ghostVault.GenerateVaultKeyPair()
	if err != nil {
		return "", fmt.Errorf("failed to generate vault key pair: %w", err)
	}

	// Serialize the command name and parameters
	commandData := strings.Join(append([]string{commandName}, parameters...), " ")

	cq.logger.Debug("Serializing command data.", zap.String("commandData", commandData))

	// Encapsulate to generate ciphertext and shared secret
	ciphertext, sharedSecret, err := cq.kemScheme.Encapsulate([]byte(publicKey))
	if err != nil {
		return "", fmt.Errorf("failed to encapsulate shared secret: %w", err)
	}

	// For demonstration purposes, we'll concatenate ciphertext and sharedSecret
	// In a real-world scenario, use the sharedSecret to encrypt the commandData with a symmetric cipher
	encryptedCommandBytes := append(ciphertext, sharedSecret...)
	encryptedCommand := hex.EncodeToString(encryptedCommandBytes)

	cq.logger.Debug("Command encrypted successfully.", zap.String("encryptedCommand", encryptedCommand))
	return encryptedCommand, nil
}

// DecryptCommand decrypts the encrypted command string and retrieves the command name and parameters.
// It returns the command name, parameters slice, or an error if decryption fails.
func (cq *CommandQueue) DecryptCommand(encryptedCommand string) (string, []string, error) {
	cq.logger.Debug("Decrypting command.", zap.String("encryptedCommand", encryptedCommand))

	// Decode the hex-encoded encrypted command
	encryptedCommandBytes, err := hex.DecodeString(encryptedCommand)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decode encrypted command: %w", err)
	}

	// Split ciphertext and shared secret based on KEM scheme lengths
	expectedLength := cq.kemScheme.LengthCiphertext + cq.kemScheme.LengthSharedSecret
	if len(encryptedCommandBytes) < expectedLength {
		return "", nil, fmt.Errorf("encrypted command length is insufficient")
	}

	ciphertext := encryptedCommandBytes[:cq.kemScheme.LengthCiphertext]
	sharedSecret := encryptedCommandBytes[cq.kemScheme.LengthCiphertext:]

	cq.logger.Debug("Extracted ciphertext and shared secret.", zap.Int("ciphertextLength", len(ciphertext)), zap.Int("sharedSecretLength", len(sharedSecret)))

	// Retrieve the key pair for decryption
	publicKey, privateKey, err := cq.ghostVault.GenerateVaultKeyPair()
	if err != nil {
		return "", nil, fmt.Errorf("failed to retrieve vault key pair: %w", err)
	}

	// Decapsulate to retrieve the shared secret
	retrievedSharedSecret, err := cq.kemScheme.Decapsulate(ciphertext, []byte(privateKey))
	if err != nil {
		return "", nil, fmt.Errorf("failed to decapsulate shared secret: %w", err)
	}

	// Verify that the shared secrets match
	if !compareBytes(sharedSecret, retrievedSharedSecret) {
		return "", nil, fmt.Errorf("shared secret mismatch during decryption")
	}

	// For demonstration purposes, we'll assume that the shared secret is the serialized commandData
	// In a real-world scenario, use the sharedSecret with a symmetric cipher to decrypt the commandData
	commandData := string(sharedSecret)
	cq.logger.Debug("Deserialized command data.", zap.String("commandData", commandData))

	// Parse the commandData into command name and parameters
	parts := strings.Split(commandData, " ")
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("decrypted command data is empty")
	}

	commandName := parts[0]
	parameters := parts[1:]

	return commandName, parameters, nil
}

// compareBytes compares two byte slices for equality.
// Returns true if they are identical, false otherwise.
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
