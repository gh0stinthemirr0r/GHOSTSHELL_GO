// File: ghostauth_mfa.go
package ghostauth

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	// Importing the local post-quantum secure packages
	"ghostshell/oqs/kem"
	"ghostshell/oqs/sig"
)

// UserMFA holds MFA-related information for a user.
type UserMFA struct {
	MFAEnabled  bool
	MFAToken    string
	TokenExpiry time.Time
}

// GhostAuth defines the interface for authentication operations.
// It should be implemented by the GhostAuth struct.
type GhostAuth interface {
	// GetAuthorizedPrivateKey retrieves the authorized private key for a given user.
	// Returns the private key as a string and a boolean indicating success.
	GetAuthorizedPrivateKey(username string) (string, bool)
	// GetUserPublicKey retrieves the public key for a given user.
	// Returns the public key as a string and a boolean indicating success.
	GetUserPublicKey(username string) (string, bool)
}

// ErrorHandler defines the interface for handling errors.
// It should be implemented by the ErrorHandler struct.
type ErrorHandler interface {
	// HandleError logs or handles errors with a specific context and message.
	HandleError(context, message string)
}

// GhostAuthMFA manages MFA operations using post-quantum cryptography.
type GhostAuthMFA struct {
	logger          *zap.Logger
	mutex           sync.Mutex
	userMFA         map[string]UserMFA
	errorHandler    ErrorHandler
	ghostAuth       GhostAuth
	kemScheme       *kem.Scheme
	signatureScheme *sig.Scheme
}

// NewGhostAuthMFA initializes and returns a new instance of GhostAuthMFA.
// It requires implementations of GhostAuth and ErrorHandler interfaces.
func NewGhostAuthMFA(auth GhostAuth, handler ErrorHandler) (*GhostAuthMFA, error) {
	// Initialize zap logger
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize zap logger: %w", err)
	}

	logger.Info("Initializing GhostAuthMFA with Post-Quantum Security.")

	// Initialize the KEM scheme (Kyber512)
	kemScheme, err := kem.NewScheme("Kyber512")
	if err != nil {
		logger.Error("Failed to initialize KEM scheme", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize KEM scheme: %w", err)
	}

	// Initialize the Signature scheme (Dilithium2) for additional security (optional)
	signatureScheme, err := sig.NewScheme("Dilithium2")
	if err != nil {
		logger.Error("Failed to initialize Signature scheme", zap.Error(err))
		kemScheme.Free()
		return nil, fmt.Errorf("failed to initialize Signature scheme: %w", err)
	}

	// Initialize userMFA map
	userMFA := make(map[string]UserMFA)

	return &GhostAuthMFA{
		logger:          logger,
		userMFA:         userMFA,
		errorHandler:    handler,
		ghostAuth:       auth,
		kemScheme:       kemScheme,
		signatureScheme: signatureScheme,
	}, nil
}

// Shutdown gracefully shuts down the GhostAuthMFA and cleans up resources.
func (mfa *GhostAuthMFA) Shutdown() {
	mfa.mutex.Lock()
	defer mfa.mutex.Unlock()

	mfa.logger.Info("Cleaning up GhostAuthMFA resources.")

	// Free the KEM scheme resources
	if err := mfa.kemScheme.Free(); err != nil {
		mfa.logger.Error("Failed to free KEM scheme", zap.Error(err))
	} else {
		mfa.logger.Info("KEM scheme resources freed successfully.")
	}

	// Free the Signature scheme resources
	if err := mfa.signatureScheme.Free(); err != nil {
		mfa.logger.Error("Failed to free Signature scheme", zap.Error(err))
	} else {
		mfa.logger.Info("Signature scheme resources freed successfully.")
	}

	// Additional cleanup if necessary

	// Sync the logger to flush any pending logs
	_ = mfa.logger.Sync()
}

// EnableMFA enables MFA for a specific user.
func (mfa *GhostAuthMFA) EnableMFA(username string) error {
	mfa.mutex.Lock()
	defer mfa.mutex.Unlock()

	user, exists := mfa.userMFA[username]
	if !exists {
		// Initialize user MFA data if not present
		user = UserMFA{}
	}

	user.MFAEnabled = true
	mfa.userMFA[username] = user

	mfa.logger.Info("MFA enabled for user.", zap.String("username", username))
	return nil
}

// DisableMFA disables MFA for a specific user.
func (mfa *GhostAuthMFA) DisableMFA(username string) error {
	mfa.mutex.Lock()
	defer mfa.mutex.Unlock()

	user, exists := mfa.userMFA[username]
	if !exists {
		return fmt.Errorf("user '%s' not found", username)
	}

	user.MFAEnabled = false
	user.MFAToken = ""
	user.TokenExpiry = time.Time{}
	mfa.userMFA[username] = user

	mfa.logger.Info("MFA disabled for user.", zap.String("username", username))
	return nil
}

// GenerateMFAToken generates a multi-factor authentication (MFA) token for the specified user.
// It returns the token as a string and an error if any occurred during the process.
func (mfa *GhostAuthMFA) GenerateMFAToken(username string) (string, error) {
	mfa.mutex.Lock()
	defer mfa.mutex.Unlock()

	// Check if MFA is enabled for the user
	user, exists := mfa.userMFA[username]
	if !exists || !user.MFAEnabled {
		mfa.errorHandler.HandleError("GenerateMFAToken", "MFA not enabled for user.")
		return "", fmt.Errorf("MFA not enabled for user: %s", username)
	}

	// Generate a quantum-safe token
	token, err := mfa.QuantumSafeGenerateToken()
	if err != nil {
		mfa.errorHandler.HandleError("GenerateMFAToken", "Failed to generate quantum-safe MFA token.")
		return "", fmt.Errorf("failed to generate MFA token: %w", err)
	}

	// Set token and expiry time (e.g., 5 minutes from now)
	user.MFAToken = token
	user.TokenExpiry = time.Now().Add(5 * time.Minute).UTC()
	mfa.userMFA[username] = user

	mfa.logger.Info("Generated MFA token for user.", zap.String("username", username))
	return token, nil
}

// VerifyMFAToken verifies the MFA token for the specified user.
// It returns true if the token is valid, false otherwise, along with an error if any occurred.
func (mfa *GhostAuthMFA) VerifyMFAToken(username, token string) (bool, error) {
	mfa.mutex.Lock()
	defer mfa.mutex.Unlock()

	mfa.logger.Info("Verifying MFA token.", zap.String("username", username))

	user, exists := mfa.userMFA[username]
	if !exists {
		mfa.errorHandler.HandleError("VerifyMFAToken", "Username not found.")
		return false, fmt.Errorf("user '%s' not found", username)
	}

	// Check if MFA is enabled
	if !user.MFAEnabled {
		mfa.errorHandler.HandleError("VerifyMFAToken", "MFA not enabled for user.")
		return false, fmt.Errorf("MFA not enabled for user: %s", username)
	}

	// Check if token matches and is not expired
	if !mfa.IsMFATokenValid(user, token) {
		mfa.errorHandler.HandleError("VerifyMFAToken", "Invalid MFA token.")
		return false, fmt.Errorf("invalid MFA token for user: %s", username)
	}

	mfa.logger.Info("MFA token verified successfully.", zap.String("username", username))
	return true, nil
}

// IsMFATokenValid checks if the MFA token is valid and not expired.
func (mfa *GhostAuthMFA) IsMFATokenValid(user UserMFA, token string) bool {
	mfa.logger.Info("Checking if MFA token is valid.", zap.String("token", token))

	// Check if the token has expired
	if time.Now().UTC().After(user.TokenExpiry) {
		mfa.errorHandler.HandleError("IsMFATokenValid", "MFA token has expired.")
		return false
	}

	// Check if the token matches
	if user.MFAToken != token {
		mfa.errorHandler.HandleError("IsMFATokenValid", "MFA token does not match.")
		return false
	}

	return true
}

// QuantumSafeGenerateToken generates a quantum-safe MFA token using KEM.
func (mfa *GhostAuthMFA) QuantumSafeGenerateToken() (string, error) {
	// Use the KEM scheme to encapsulate a shared secret
	ciphertext, sharedSecret, err := mfa.kemScheme.Encapsulate(nil) // nil can be used if peer's public key is not needed for token generation
	if err != nil {
		mfa.logger.Error("Failed to encapsulate shared secret.", zap.Error(err))
		return "", fmt.Errorf("failed to encapsulate shared secret: %w", err)
	}

	// Convert sharedSecret to a string (you may choose to encode it in hex or base64)
	token := string(sharedSecret)

	// Optionally, you can include additional information or encoding
	return token, nil
}
