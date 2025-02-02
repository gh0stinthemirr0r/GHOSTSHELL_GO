// File: extended_command_handler.go
package ghostcommand

import (
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"

	// Importing the local post-quantum secure packages
	"ghostshell/ghostshell/oqs/kem"
	"ghostshell/ghostshell/oqs/sig"
)

// ErrorHandler defines the interface for handling errors.
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

// ExtendedCommandHandler manages the registration and execution of specialized commands with quantum-safe encryption and authentication.
type ExtendedCommandHandler struct {
	specializedCommandRegistry map[string]func(parameters []string)
	handlerMutex               sync.Mutex
	errorHandler               ErrorHandler
	ghostAuth                  GhostAuth
	ghostVault                 GhostVault
	kemScheme                  *kem.Scheme
	sigScheme                  *sig.Scheme
	logger                     *zap.Logger
}

// NewExtendedCommandHandler initializes and returns a new instance of ExtendedCommandHandler.
func NewExtendedCommandHandler(
	ghostAuth GhostAuth,
	ghostVault GhostVault,
	errorHandler ErrorHandler,
	kemScheme *kem.Scheme,
	sigScheme *sig.Scheme,
	logger *zap.Logger,
) (*ExtendedCommandHandler, error) {
	return &ExtendedCommandHandler{
		specializedCommandRegistry: make(map[string]func(parameters []string)),
		errorHandler:               errorHandler,
		ghostAuth:                  ghostAuth,
		ghostVault:                 ghostVault,
		kemScheme:                  kemScheme,
		sigScheme:                  sigScheme,
		logger:                     logger,
	}, nil
}

// RegisterSpecializedCommand registers a specialized command with its handler function.
// Returns true if the command was registered successfully, false otherwise.
func (ech *ExtendedCommandHandler) RegisterSpecializedCommand(commandName string, handler func(parameters []string)) (bool, error) {
	ech.handlerMutex.Lock()
	defer ech.handlerMutex.Unlock()

	if _, exists := ech.specializedCommandRegistry[commandName]; exists {
		ech.errorHandler.HandleError("RegisterSpecializedCommand", "Command already registered: "+commandName)
		return false, fmt.Errorf("command already registered: %s", commandName)
	}

	ech.specializedCommandRegistry[commandName] = handler
	ech.logger.Info("Registered specialized command.", zap.String("command", commandName))
	return true, nil
}

// ExecuteSpecializedCommand executes a specialized command with post-quantum encryption and authentication.
// It takes the username, command name, and parameters.
// Returns true if the command was executed successfully, false otherwise.
func (ech *ExtendedCommandHandler) ExecuteSpecializedCommand(username, commandName string, parameters []string) (bool, error) {
	ech.logger.Info("Executing specialized command.", zap.String("username", username), zap.String("command", commandName), zap.Strings("parameters", parameters))

	// Authenticate the user
	if !ech.AuthenticateUser(username) {
		ech.errorHandler.HandleError("ExecuteSpecializedCommand", "Authentication failed for user: "+username)
		return false, fmt.Errorf("authentication failed for user: %s", username)
	}

	// Encrypt the command and parameters
	encryptedCommand, err := ech.EncryptCommand(commandName, parameters)
	if err != nil {
		ech.errorHandler.HandleError("ExecuteSpecializedCommand", "Failed to encrypt command: "+err.Error())
		return false, fmt.Errorf("failed to encrypt command: %w", err)
	}

	// Decrypt the command and parameters
	decryptedCommandName, decryptedParameters, err := ech.DecryptCommand(encryptedCommand)
	if err != nil {
		ech.errorHandler.HandleError("ExecuteSpecializedCommand", "Failed to decrypt command: "+err.Error())
		return false, fmt.Errorf("failed to decrypt command: %w", err)
	}

	// Find the command handler
	ech.handlerMutex.Lock()
	handler, exists := ech.specializedCommandRegistry[decryptedCommandName]
	ech.handlerMutex.Unlock()

	if !exists {
		ech.errorHandler.HandleError("ExecuteSpecializedCommand", "Command not found: "+decryptedCommandName)
		return false, fmt.Errorf("command not found: %s", decryptedCommandName)
	}

	// Execute the command handler
	defer func() {
		if r := recover(); r != nil {
			ech.errorHandler.HandleError("ExecuteSpecializedCommand", fmt.Sprintf("Command execution panicked: %v", r))
		}
	}()

	handler(decryptedParameters)
	ech.logger.Info("Specialized command executed successfully.", zap.String("command", decryptedCommandName))
	return true, nil
}

// AuthenticateUser authenticates a user using post-quantum signature verification.
// Returns true if authentication is successful, false otherwise.
func (ech *ExtendedCommandHandler) AuthenticateUser(username string) bool {
	ech.logger.Info("Authenticating user for specialized command.", zap.String("username", username))

	publicKey, err := ech.ghostAuth.GetUserPublicKey(username)
	if err != nil {
		ech.errorHandler.HandleError("AuthenticateUser", "Failed to retrieve public key for user: "+username)
		ech.logger.Error("Failed to retrieve public key.", zap.String("username", username), zap.Error(err))
		return false
	}

	message := "Authenticate user for specialized command: " + username
	signature, err := ech.ghostAuth.SignMessage(username, message)
	if err != nil {
		ech.errorHandler.HandleError("AuthenticateUser", "Failed to sign message for user: "+username)
		ech.logger.Error("Failed to sign message.", zap.String("username", username), zap.Error(err))
		return false
	}

	// Convert message, signature, and publicKey from strings to byte slices
	messageBytes := []byte(message)
	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		ech.errorHandler.HandleError("AuthenticateUser", "Failed to decode signature.")
		ech.logger.Error("Failed to decode signature.", zap.Error(err))
		return false
	}
	publicKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		ech.errorHandler.HandleError("AuthenticateUser", "Failed to decode public key.")
		ech.logger.Error("Failed to decode public key.", zap.Error(err))
		return false
	}

	// Quantum-safe signature verification using Dilithium-2 from liboqs.
	isVerified, err := ech.sigScheme.Verify(messageBytes, signatureBytes, publicKeyBytes)
	if err != nil {
		ech.errorHandler.HandleError("AuthenticateUser", "Signature verification error.")
		ech.logger.Error("Signature verification error.", zap.Error(err))
		return false
	}

	if !isVerified {
		ech.errorHandler.HandleError("AuthenticateUser", "Signature verification failed for user: "+username)
		ech.logger.Error("Signature verification failed.", zap.String("username", username))
		return false
	}

	ech.logger.Info("User authenticated successfully for specialized command.", zap.String("username", username))
	return true
}

// EncryptCommand encrypts the command name and parameters using the KEM scheme.
// It returns the encrypted command as a hex-encoded string.
func (ech *ExtendedCommandHandler) EncryptCommand(commandName string, parameters []string) (string, error) {
	ech.logger.Debug("Encrypting command and parameters.", zap.String("commandName", commandName), zap.Strings("parameters", parameters))

	// Serialize the command name and parameters
	commandData := strings.Join(append([]string{commandName}, parameters...), " ")
	ech.logger.Debug("Serialized command data.", zap.String("commandData", commandData))

	// Generate a quantum-safe key pair
	publicKey, _, err := ech.ghostVault.GenerateVaultKeyPair()
	if err != nil {
		return "", fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Convert public key from hex to bytes
	publicKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode public key: %w", err)
	}

	// Encapsulate to generate ciphertext and shared secret
	ciphertext, sharedSecret, err := ech.kemScheme.Encapsulate(publicKeyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to encapsulate shared secret: %w", err)
	}

	// For demonstration, concatenate ciphertext and sharedSecret and encode as hex string
	encryptedBytes := append(ciphertext, sharedSecret...)
	encryptedCommand := hex.EncodeToString(encryptedBytes)

	ech.logger.Debug("Command encrypted successfully.", zap.String("encryptedCommand", encryptedCommand))
	return encryptedCommand, nil
}

// DecryptCommand decrypts the encrypted command string and retrieves the command name and parameters.
// It returns the command name, parameters slice, or an error if decryption fails.
func (ech *ExtendedCommandHandler) DecryptCommand(encryptedCommand string) (string, []string, error) {
	ech.logger.Debug("Decrypting command.", zap.String("encryptedCommand", encryptedCommand))

	// Decode the hex-encoded encrypted command
	encryptedBytes, err := hex.DecodeString(encryptedCommand)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decode encrypted command: %w", err)
	}

	// Split ciphertext and shared secret based on KEM scheme lengths
	expectedLength := ech.kemScheme.LengthCiphertext + ech.kemScheme.LengthSharedSecret
	if len(encryptedBytes) != expectedLength {
		return "", nil, fmt.Errorf("invalid encrypted command length")
	}

	ciphertext := encryptedBytes[:ech.kemScheme.LengthCiphertext]
	sharedSecret := encryptedBytes[ech.kemScheme.LengthCiphertext:]

	ech.logger.Debug("Extracted ciphertext and shared secret.", zap.Int("ciphertextLength", len(ciphertext)), zap.Int("sharedSecretLength", len(sharedSecret)))

	// Generate the key pair (assuming the same key pair is used for encapsulation and decapsulation)
	// In practice, you should store and reuse the key pair instead of generating it every time.
	_, privateKey, err := ech.ghostVault.GenerateVaultKeyPair()
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Convert private key from hex to bytes
	privateKeyBytes, err := hex.DecodeString(privateKey)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	// Decapsulate to retrieve the shared secret
	retrievedSharedSecret, err := ech.kemScheme.Decapsulate(ciphertext, privateKeyBytes)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decapsulate shared secret: %w", err)
	}

	// Verify that the shared secrets match
	if !compareBytes(sharedSecret, retrievedSharedSecret) {
		return "", nil, fmt.Errorf("shared secret mismatch during decryption")
	}

	// Deserialize command data back into command name and parameters
	commandData := string(retrievedSharedSecret)
	ech.logger.Debug("Deserialized command data.", zap.String("commandData", commandData))

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
