// File: command_authorization.go
package ghostcommand

import (
	"encoding/hex"
	"fmt"
	"sync"

	"go.uber.org/zap"

	// Importing the local post-quantum secure packages
	"ghostshell/ghostshell/oqs/sig"
)

// ErrorHandler defines the interface for handling errors.
// It should be implemented by the ErrorHandler struct.
type ErrorHandler interface {
	// HandleError logs or handles errors with a specific context and message.
	HandleError(context, message string)
}

// GhostAuth defines the interface for authentication operations.
// It should be implemented by the GhostAuth struct.
type GhostAuth interface {
	// GetUserPublicKey retrieves the public key for a given user.
	// Returns the public key as a hex-encoded string and a boolean indicating success.
	GetUserPublicKey(username string) (string, bool)
	// GetAuthorizedPrivateKey retrieves the authorized private key for a given user.
	// Returns the private key as a hex-encoded string and a boolean indicating success.
	GetAuthorizedPrivateKey(username string) (string, bool)
}

// CommandAuthorization manages user permissions and authorizations for executing specific commands.
type CommandAuthorization struct {
	userPermissions map[string]map[string]bool // Maps username to a map of commandName to authorization status
	errorHandler    ErrorHandler
	ghostAuth       GhostAuth
	oqsSig          *sig.Scheme
	logger          *zap.Logger
	mutex           sync.Mutex
}

// NewCommandAuthorization initializes and returns a new instance of CommandAuthorization.
// It requires implementations of GhostAuth and ErrorHandler interfaces.
func NewCommandAuthorization(ghostAuth GhostAuth, handler ErrorHandler) (*CommandAuthorization, error) {
	// Initialize zap logger
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize zap logger: %w", err)
	}

	logger.Info("Initializing CommandAuthorization with Post-Quantum Security.")

	// Initialize the post-quantum signature scheme (Dilithium-2)
	sigScheme, err := sig.NewScheme("Dilithium2")
	if err != nil {
		logger.Error("Failed to initialize Signature scheme", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize Signature scheme: %w", err)
	}

	// Initialize userPermissions map
	userPermissions := make(map[string]map[string]bool)

	return &CommandAuthorization{
		userPermissions: userPermissions,
		errorHandler:    handler,
		ghostAuth:       ghostAuth,
		oqsSig:          sigScheme,
		logger:          logger,
	}, nil
}

// Shutdown gracefully shuts down the CommandAuthorization and cleans up resources.
func (ca *CommandAuthorization) Shutdown() {
	ca.mutex.Lock()
	defer ca.mutex.Unlock()

	ca.logger.Info("Shutting down CommandAuthorization.")

	// Free the Signature scheme resources
	if err := ca.oqsSig.Free(); err != nil {
		ca.logger.Error("Failed to free Signature scheme", zap.Error(err))
	} else {
		ca.logger.Info("Signature scheme resources freed successfully.")
	}

	// Sync the logger to flush any pending logs
	_ = ca.logger.Sync()
}

// AddUserPermission adds user permission for executing a specific command.
// It returns true if the permission is added successfully, false otherwise.
func (ca *CommandAuthorization) AddUserPermission(username, commandName string) (bool, error) {
	ca.mutex.Lock()
	defer ca.mutex.Unlock()

	if _, exists := ca.userPermissions[username]; !exists {
		ca.userPermissions[username] = make(map[string]bool)
	}

	ca.userPermissions[username][commandName] = true
	ca.logger.Info("Added user permission.", zap.String("username", username), zap.String("command", commandName))
	return true, nil
}

// IsUserAuthorized verifies whether a user is authorized to execute a specific command.
// It returns true if authorized, false otherwise.
func (ca *CommandAuthorization) IsUserAuthorized(username, commandName string) (bool, error) {
	ca.mutex.Lock()
	defer ca.mutex.Unlock()

	if commands, exists := ca.userPermissions[username]; exists {
		if authorized, cmdExists := commands[commandName]; cmdExists {
			ca.logger.Info("User authorization checked.", zap.String("username", username), zap.String("command", commandName), zap.Bool("authorized", authorized))
			return authorized, nil
		}
	}

	ca.logger.Info("User authorization checked: not authorized.", zap.String("username", username), zap.String("command", commandName))
	return false, nil
}

// InitializePostQuantumSignature initializes the post-quantum signature scheme.
// This method is kept private as it's only used internally during initialization.
func (ca *CommandAuthorization) InitializePostQuantumSignature() error {
	// This method can include additional initialization steps if necessary
	// Currently, the signature scheme is initialized in the constructor
	return nil
}

// VerifyUserSignature verifies the user's signature to ensure authorization authenticity.
// It returns true if the signature is valid, false otherwise.
func (ca *CommandAuthorization) VerifyUserSignature(username, message, signature string) (bool, error) {
	ca.logger.Info("Verifying user signature.", zap.String("username", username), zap.String("message", message))

	// Retrieve the user's public key
	publicKeyHex, exists := ca.ghostAuth.GetUserPublicKey(username)
	if !exists {
		ca.errorHandler.HandleError("VerifyUserSignature", fmt.Sprintf("Public key not found for user: %s", username))
		return false, fmt.Errorf("public key not found for user: %s", username)
	}

	// Decode the hex-encoded public key
	publicKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		ca.errorHandler.HandleError("VerifyUserSignature", fmt.Sprintf("Failed to decode public key for user: %s", username))
		return false, fmt.Errorf("failed to decode public key for user: %s", username)
	}

	// Decode the hex-encoded signature
	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		ca.errorHandler.HandleError("VerifyUserSignature", fmt.Sprintf("Failed to decode signature for user: %s", username))
		return false, fmt.Errorf("failed to decode signature for user: %s", username)
	}

	// Verify the signature using the signature scheme
	err = ca.oqsSig.Verify([]byte(message), signatureBytes, publicKeyBytes)
	if err != nil {
		ca.errorHandler.HandleError("VerifyUserSignature", fmt.Sprintf("Signature verification failed for user: %s", username))
		return false, fmt.Errorf("signature verification failed for user: %s", username)
	}

	ca.logger.Info("User signature verified successfully.", zap.String("username", username))
	return true, nil
}
