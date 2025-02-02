// File: command_router.go
package ghostcommand

import (
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"

	// Importing the local post-quantum secure packages

	"ghostshell/ghostshell/oqs/kem"
	"ghostshell/ghostshell/oqs/sig"
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

// GhostAuth defines the interface for user authentication.
type GhostAuth interface {
	// GetUserPublicKey retrieves the public key of a user.
	GetUserPublicKey(username string) (publicKey string, err error)
	// SignMessage signs a message using the user's private key.
	SignMessage(username, message string) (signature string, err error)
}

// CryptoManager defines the interface for cryptographic operations.
type CryptoManager interface {
	// GenerateKeyPair generates a quantum-safe key pair.
	GenerateKeyPair() (publicKey, privateKey string, err error)
	// Encrypt encrypts the given data using the provided public key.
	Encrypt(data string, publicKey string) (encryptedData string, err error)
	// Decrypt decrypts the given encrypted data using the provided private key.
	Decrypt(encryptedData string, privateKey string) (decryptedData string, err error)
}

// CommandHandler defines the type for command handler functions.
type CommandHandlerFunc func(username string, output *string) bool

// CommandRouter manages the registration and execution of commands with quantum-safe encryption and authentication.
type CommandRouter struct {
	commandRegistry map[string]CommandHandlerFunc
	commandMutex    sync.Mutex
	errorHandler    ErrorHandler
	ghostAuth       GhostAuth
	cryptoManager   CryptoManager
	logger          *zap.Logger
	kemScheme       *kem.Scheme
	sigScheme       *sig.Scheme
}

// NewCommandRouter initializes and returns a new instance of CommandRouter.
// It requires implementations of GhostAuth, CryptoManager, GhostVault, and ErrorHandler interfaces.
func NewCommandRouter(
	ghostAuth GhostAuth,
	cryptoManager CryptoManager,
	ghostVault GhostVault,
	handler ErrorHandler,
	logger *zap.Logger,
) (*CommandRouter, error) {
	// Initialize the KEM scheme (Kyber-512)
	kemScheme, err := kem.NewScheme("Kyber512")
	if err != nil {
		logger.Error("Failed to initialize KEM scheme", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize KEM scheme: %w", err)
	}

	// Initialize the Signature scheme (Dilithium 2)
	sigScheme, err := sig.NewScheme("Dilithium2")
	if err != nil {
		logger.Error("Failed to initialize Signature scheme", zap.Error(err))
		kemScheme.Free()
		return nil, fmt.Errorf("failed to initialize Signature scheme: %w", err)
	}

	return &CommandRouter{
		commandRegistry: make(map[string]CommandHandlerFunc),
		errorHandler:    handler,
		ghostAuth:       ghostAuth,
		cryptoManager:   cryptoManager,
		logger:          logger,
		kemScheme:       kemScheme,
		sigScheme:       sigScheme,
	}, nil
}

// Shutdown gracefully shuts down the CommandRouter and cleans up resources.
func (cr *CommandRouter) Shutdown() {
	cr.commandMutex.Lock()
	defer cr.commandMutex.Unlock()

	cr.logger.Info("Shutting down CommandRouter.")

	// Free the KEM and Signature scheme resources
	if err := cr.kemScheme.Free(); err != nil {
		cr.logger.Error("Failed to free KEM scheme", zap.Error(err))
	} else {
		cr.logger.Info("KEM scheme resources freed successfully.")
	}

	if err := cr.sigScheme.Free(); err != nil {
		cr.logger.Error("Failed to free Signature scheme", zap.Error(err))
	} else {
		cr.logger.Info("Signature scheme resources freed successfully.")
	}

	// Sync the logger to flush any pending logs
	_ = cr.logger.Sync()
}

// RegisterCommand registers a command with its handler function.
// Returns true if the command was registered successfully, false otherwise.
func (cr *CommandRouter) RegisterCommand(commandName string, handler CommandHandlerFunc) (bool, error) {
	cr.commandMutex.Lock()
	defer cr.commandMutex.Unlock()

	if _, exists := cr.commandRegistry[commandName]; exists {
		cr.errorHandler.HandleError("RegisterCommand", "Command already registered: "+commandName)
		return false, fmt.Errorf("command already registered: %s", commandName)
	}

	cr.commandRegistry[commandName] = handler
	cr.logger.Info("Registered command.", zap.String("command", commandName))
	return true, nil
}

// ExecuteCommand executes a registered command with quantum-safe encryption and authentication.
// It takes the username, command name, and parameters.
// Returns true if the command was executed successfully, false otherwise.
func (cr *CommandRouter) ExecuteCommand(username, commandName string, parameters []string) (bool, error) {
	cr.logger.Info("Executing command.", zap.String("username", username), zap.String("command", commandName), zap.Strings("parameters", parameters))

	// Authenticate the user
	if !cr.AuthenticateUser(username) {
		cr.errorHandler.HandleError("ExecuteCommand", "Authentication failed for user: "+username)
		return false, fmt.Errorf("authentication failed for user: %s", username)
	}

	// Encrypt the command
	encryptedCommand, err := cr.EncryptCommand(commandName, parameters)
	if err != nil {
		cr.errorHandler.HandleError("ExecuteCommand", "Failed to encrypt command.")
		return false, fmt.Errorf("failed to encrypt command: %w", err)
	}

	// Decrypt the command
	decryptedCommandName, decryptedParameters, err := cr.DecryptCommand(encryptedCommand)
	if err != nil {
		cr.errorHandler.HandleError("ExecuteCommand", "Failed to decrypt command.")
		return false, fmt.Errorf("failed to decrypt command: %w", err)
	}

	// Find the command handler
	cr.commandMutex.Lock()
	handler, exists := cr.commandRegistry[decryptedCommandName]
	cr.commandMutex.Unlock()

	if !exists {
		cr.errorHandler.HandleError("ExecuteCommand", "Command not found: "+decryptedCommandName)
		return false, fmt.Errorf("command not found: %s", decryptedCommandName)
	}

	// Execute the command handler
	var output string
	success := handler(username, &output)
	if success {
		cr.logger.Info("Command executed successfully.", zap.String("command", decryptedCommandName))
	} else {
		cr.errorHandler.HandleError("ExecuteCommand", "Command execution failed: "+decryptedCommandName)
		cr.logger.Error("Command execution failed.", zap.String("command", decryptedCommandName))
	}

	return success, nil
}

// AuthenticateUser authenticates a user using post-quantum signature verification.
// Returns true if authentication is successful, false otherwise.
func (cr *CommandRouter) AuthenticateUser(username string) bool {
	cr.logger.Info("Authenticating user.", zap.String("username", username))

	publicKey, err := cr.ghostAuth.GetUserPublicKey(username)
	if err != nil {
		cr.errorHandler.HandleError("AuthenticateUser", "Failed to retrieve public key for user: "+username)
		cr.logger.Error("Failed to retrieve public key.", zap.String("username", username), zap.Error(err))
		return false
	}

	message := "AuthenticateUser"
	signature, err := cr.ghostAuth.SignMessage(username, message)
	if err != nil {
		cr.errorHandler.HandleError("AuthenticateUser", "Failed to sign message for user: "+username)
		cr.logger.Error("Failed to sign message.", zap.String("username", username), zap.Error(err))
		return false
	}

	// Quantum-safe signature verification using Dilithium-2 from liboqs.
	isVerified, err := cr.sigScheme.Verify([]byte(message), []byte(signature), []byte(publicKey))
	if err != nil {
		cr.errorHandler.HandleError("AuthenticateUser", "Signature verification error for user: "+username)
		cr.logger.Error("Signature verification error.", zap.String("username", username), zap.Error(err))
		return false
	}

	if !isVerified {
		cr.errorHandler.HandleError("AuthenticateUser", "Signature verification failed for user: "+username)
		cr.logger.Error("Signature verification failed.", zap.String("username", username))
		return false
	}

	cr.logger.Info("User authenticated successfully.", zap.String("username", username))
	return true
}

// EncryptCommand encrypts the command name and parameters using the KEM scheme.
// It returns the encrypted command as a hex-encoded string.
func (cr *CommandRouter) EncryptCommand(commandName string, parameters []string) (string, error) {
	// Generate a quantum-safe key pair
	publicKey, privateKey, err := cr.cryptoManager.GenerateKeyPair()
	if err != nil {
		return "", fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Serialize the command name and parameters
	commandData := strings.Join(append([]string{commandName}, parameters...), " ")
	cr.logger.Debug("Serializing command data.", zap.String("commandData", commandData))

	// Encapsulate to generate ciphertext and shared secret
	ciphertext, sharedSecret, err := cr.kemScheme.Encapsulate([]byte(publicKey))
	if err != nil {
		return "", fmt.Errorf("failed to encapsulate shared secret: %w", err)
	}

	// In a real-world scenario, use the shared secret to encrypt the commandData using a symmetric cipher (e.g., AES-GCM)
	// For demonstration, we'll concatenate ciphertext and sharedSecret and encode them as a hex string
	encryptedCommandBytes := append(ciphertext, sharedSecret...)
	encryptedCommand := fmt.Sprintf("%x", encryptedCommandBytes)

	cr.logger.Debug("Command encrypted successfully.", zap.String("encryptedCommand", encryptedCommand))
	return encryptedCommand, nil
}

// DecryptCommand decrypts the encrypted command string and retrieves the command name and parameters.
// It returns the command name, parameters slice, or an error if decryption fails.
func (cr *CommandRouter) DecryptCommand(encryptedCommand string) (string, []string, error) {
	cr.logger.Debug("Decrypting command.", zap.String("encryptedCommand", encryptedCommand))

	// Decode the hex-encoded encrypted command
	encryptedCommandBytes, err := fmt.Sscanf(encryptedCommand, "%x", &encryptedCommand)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decode encrypted command: %w", err)
	}

	// Split ciphertext and shared secret based on KEM scheme lengths
	expectedLength := cr.kemScheme.LengthCiphertext + cr.kemScheme.LengthSharedSecret
	if len(encryptedCommandBytes) < expectedLength {
		return "", nil, fmt.Errorf("encrypted command length is insufficient")
	}

	ciphertext := encryptedCommandBytes[:cr.kemScheme.LengthCiphertext]
	sharedSecret := encryptedCommandBytes[cr.kemScheme.LengthCiphertext:]

	cr.logger.Debug("Extracted ciphertext and shared secret.", zap.Int("ciphertextLength", len(ciphertext)), zap.Int("sharedSecretLength", len(sharedSecret)))

	// Retrieve the key pair for decryption
	_, privateKey, err := cr.cryptoManager.GenerateKeyPair()
	if err != nil {
		return "", nil, fmt.Errorf("failed to retrieve vault key pair: %w", err)
	}

	// Decapsulate to retrieve the shared secret
	retrievedSharedSecret, err := cr.kemScheme.Decapsulate(ciphertext, []byte(privateKey))
	if err != nil {
		return "", nil, fmt.Errorf("failed to decapsulate shared secret: %w", err)
	}

	// Verify that the shared secrets match
	if !compareBytes(sharedSecret, retrievedSharedSecret) {
		return "", nil, fmt.Errorf("shared secret mismatch during decryption")
	}

	// In a real-world scenario, use the shared secret to decrypt the commandData
	// For demonstration, we'll assume that the shared secret is the serialized commandData
	commandData := string(sharedSecret)
	cr.logger.Debug("Deserialized command data.", zap.String("commandData", commandData))

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
