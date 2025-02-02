// File: command.go
package ghostcommand

import (
	"encoding/hex"
	"fmt"
	"sync"

	"go.uber.org/zap"

	// Importing the local post-quantum secure packages
	"ghostshell/oqs/kem"
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

// Command represents a command with a name, description, and an execution function.
type Command struct {
	name         string
	description  string
	execute      func(parameters string) // Execution function
	errorHandler ErrorHandler
	ghostVault   GhostVault
	kemScheme    *kem.Scheme
	mutex        sync.Mutex
	logger       *zap.Logger
}

// NewCommand initializes and returns a new Command instance.
// It requires the command name, description, execution function, GhostVault, ErrorHandler, KEM scheme, and logger.
func NewCommand(
	name string,
	description string,
	execute func(parameters string),
	ghostVault GhostVault,
	errorHandler ErrorHandler,
	kemScheme *kem.Scheme,
	logger *zap.Logger,
) *Command {
	return &Command{
		name:         name,
		description:  description,
		execute:      execute,
		ghostVault:   ghostVault,
		errorHandler: errorHandler,
		kemScheme:    kemScheme,
		logger:       logger,
	}
}

// Execute runs the command with the given parameters.
// It encrypts the parameters, decrypts them, and then executes the command.
// Returns true if execution was successful, false otherwise.
func (c *Command) Execute(parameters string) bool {
	c.logger.Info("Executing command.", zap.String("command", c.name), zap.String("parameters", parameters))

	// Encrypt the parameters
	encryptedParameters, err := c.EncryptParameters(parameters)
	if err != nil {
		c.errorHandler.HandleError("Execute", "Failed to encrypt command parameters: "+err.Error())
		return false
	}

	// Decrypt the parameters
	decryptedParameters, err := c.DecryptParameters(encryptedParameters)
	if err != nil {
		c.errorHandler.HandleError("Execute", "Failed to decrypt command parameters: "+err.Error())
		return false
	}

	// Execute the command with decrypted parameters
	defer func() {
		if r := recover(); r != nil {
			c.errorHandler.HandleError("Execute", fmt.Sprintf("Command execution panicked: %v", r))
		}
	}()

	c.execute(decryptedParameters)
	c.logger.Info("Command executed successfully.", zap.String("command", c.name))
	return true
}

// GetName returns the name of the command.
func (c *Command) GetName() string {
	return c.name
}

// GetDescription returns the description of the command.
func (c *Command) GetDescription() string {
	return c.description
}

// EncryptParameters encrypts the command parameters using the KEM scheme.
// It returns the encrypted parameters as a hex-encoded string.
func (c *Command) EncryptParameters(parameters string) (string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Generate a quantum-safe key pair
	publicKey, _, err := c.ghostVault.GenerateVaultKeyPair()
	if err != nil {
		return "", fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Convert public key from hex to bytes
	publicKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode public key: %w", err)
	}

	// Encapsulate to generate ciphertext and shared secret
	ciphertext, sharedSecret, err := c.kemScheme.Encapsulate(publicKeyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to encapsulate shared secret: %w", err)
	}

	// For demonstration, concatenate ciphertext and sharedSecret and encode as hex string
	encryptedBytes := append(ciphertext, sharedSecret...)
	encryptedParameters := hex.EncodeToString(encryptedBytes)

	c.logger.Debug("Parameters encrypted successfully.", zap.String("encryptedParameters", encryptedParameters))
	return encryptedParameters, nil
}

// DecryptParameters decrypts the encrypted command parameters.
// It returns the decrypted parameters as a string.
func (c *Command) DecryptParameters(encryptedParameters string) (string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Decode the hex-encoded encrypted parameters
	encryptedBytes, err := hex.DecodeString(encryptedParameters)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted parameters: %w", err)
	}

	// Split ciphertext and shared secret based on KEM scheme lengths
	expectedLength := c.kemScheme.LengthCiphertext + c.kemScheme.LengthSharedSecret
	if len(encryptedBytes) != expectedLength {
		return "", fmt.Errorf("invalid encrypted parameters length")
	}

	ciphertext := encryptedBytes[:c.kemScheme.LengthCiphertext]
	sharedSecret := encryptedBytes[c.kemScheme.LengthCiphertext:]

	// Generate the key pair (assuming the same key pair is used for encapsulation and decapsulation)
	// In practice, you should store and reuse the key pair instead of generating it every time.
	_, privateKey, err := c.ghostVault.GenerateVaultKeyPair()
	if err != nil {
		return "", fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Convert private key from hex to bytes
	privateKeyBytes, err := hex.DecodeString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %w", err)
	}

	// Decapsulate to retrieve the shared secret
	retrievedSharedSecret, err := c.kemScheme.Decapsulate(ciphertext, privateKeyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to decapsulate shared secret: %w", err)
	}

	// Verify that the shared secrets match
	if !compareBytes(sharedSecret, retrievedSharedSecret) {
		return "", fmt.Errorf("shared secret mismatch during decryption")
	}

	// For demonstration, assume that the shared secret is the decrypted parameters
	decryptedParameters := string(retrievedSharedSecret)

	c.logger.Debug("Parameters decrypted successfully.", zap.String("decryptedParameters", decryptedParameters))
	return decryptedParameters, nil
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
