// File: ghostauth.go
package ghostauth

import (
	"encoding/hex"
	"fmt"
	"sync"

	"go.uber.org/zap"

	// Importing the local post-quantum secure packages
	"ghostshell/oqs/kem"
	"ghostshell/oqs/sig"
)

// UserKey holds the public and private keys for a user.
type UserKey struct {
	PublicKey  string
	PrivateKey string
}

// UserStorageManager defines the interface for loading and saving user keys.
type UserStorageManager interface {
	LoadUsers(userKeys map[string]UserKey) error
	SaveUsers(userKeys map[string]UserKey) error
}

// GhostAuth manages user authentication using post-quantum cryptography.
type GhostAuth struct {
	logger             *zap.Logger
	mutex              sync.Mutex
	userKeys           map[string]UserKey
	userStorageManager UserStorageManager
	kemScheme          *kem.Scheme
	sigScheme          *sig.Scheme
	errorHandler       ErrorHandler
}

// ErrorHandler defines the interface for handling errors.
type ErrorHandler interface {
	// HandleError logs or handles errors with a specific context and message.
	HandleError(context, message string)
}

// NewGhostAuth initializes and returns a new instance of GhostAuth.
// It requires implementations of UserStorageManager and ErrorHandler interfaces.
func NewGhostAuth(storageManager UserStorageManager, handler ErrorHandler) (*GhostAuth, error) {
	// Initialize zap logger
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize zap logger: %w", err)
	}

	logger.Info("Initializing GhostAuth with Post-Quantum Security.")

	// Initialize the KEM scheme (Kyber-512)
	kemScheme, err := kem.NewScheme("Kyber512")
	if err != nil {
		logger.Error("Failed to initialize KEM scheme", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize KEM scheme: %w", err)
	}

	// Initialize the Signature scheme (Dilithium-2)
	sigScheme, err := sig.NewScheme("Dilithium2")
	if err != nil {
		logger.Error("Failed to initialize Signature scheme", zap.Error(err))
		kemScheme.Free()
		return nil, fmt.Errorf("failed to initialize Signature scheme: %w", err)
	}

	// Initialize userKeys map
	userKeys := make(map[string]UserKey)

	// Load existing users from persistent storage
	if err := storageManager.LoadUsers(userKeys); err != nil {
		logger.Error("Failed to load users from storage", zap.Error(err))
		kemScheme.Free()
		sigScheme.Free()
		return nil, fmt.Errorf("failed to load users from storage: %w", err)
	}

	return &GhostAuth{
		logger:             logger,
		userKeys:           userKeys,
		userStorageManager: storageManager,
		kemScheme:          kemScheme,
		sigScheme:          sigScheme,
		errorHandler:       handler,
	}, nil
}

// Shutdown gracefully shuts down the GhostAuth and cleans up resources.
func (ga *GhostAuth) Shutdown() {
	ga.mutex.Lock()
	defer ga.mutex.Unlock()

	ga.logger.Info("Cleaning up GhostAuth resources.")

	// Save users to persistent storage before shutting down
	if err := ga.userStorageManager.SaveUsers(ga.userKeys); err != nil {
		ga.logger.Error("Failed to save users to storage", zap.Error(err))
	} else {
		ga.logger.Info("Users saved to storage successfully.")
	}

	// Free the KEM scheme resources
	if err := ga.kemScheme.Free(); err != nil {
		ga.logger.Error("Failed to free KEM scheme", zap.Error(err))
	} else {
		ga.logger.Info("KEM scheme resources freed successfully.")
	}

	// Free the Signature scheme resources
	if err := ga.sigScheme.Free(); err != nil {
		ga.logger.Error("Failed to free Signature scheme", zap.Error(err))
	} else {
		ga.logger.Info("Signature scheme resources freed successfully.")
	}

	// Sync the logger to flush any pending logs
	_ = ga.logger.Sync()
}

// GeneratePostQuantumKeyPair generates a post-quantum key pair using Kyber-512.
// It returns the public and private keys as hex-encoded strings.
func (ga *GhostAuth) GeneratePostQuantumKeyPair() (string, string, error) {
	ga.logger.Info("Generating post-quantum key pair using Kyber-512.")

	// Use the KEM scheme to generate a key pair
	publicKeyBytes, privateKeyBytes, err := ga.kemScheme.Keypair()
	if err != nil {
		ga.logger.Error("Failed to generate Kyber key pair", zap.Error(err))
		return "", "", fmt.Errorf("failed to generate Kyber key pair: %w", err)
	}

	// Encode the keys as hex strings
	publicKey := hex.EncodeToString(publicKeyBytes)
	privateKey := hex.EncodeToString(privateKeyBytes)

	ga.logger.Info("Post-quantum key pair generated successfully.")
	return publicKey, privateKey, nil
}

// AddUser adds a new user and generates a key pair for them.
// It returns true if the user is added successfully, false otherwise.
func (ga *GhostAuth) AddUser(username string) (bool, error) {
	ga.mutex.Lock()
	defer ga.mutex.Unlock()

	ga.logger.Info("Adding new user.", zap.String("username", username))

	// Check if the user already exists
	if _, exists := ga.userKeys[username]; exists {
		ga.errorHandler.HandleError("AddUser", fmt.Sprintf("User already exists: %s", username))
		return false, fmt.Errorf("user already exists: %s", username)
	}

	// Generate a post-quantum key pair for the user
	publicKey, privateKey, err := ga.GeneratePostQuantumKeyPair()
	if err != nil {
		ga.errorHandler.HandleError("AddUser", fmt.Sprintf("Failed to generate key pair for user: %s", username))
		return false, fmt.Errorf("failed to generate key pair for user: %s", username)
	}

	// Store the key pair in the userKeys map
	ga.userKeys[username] = UserKey{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}

	// Save the new user to persistent storage
	if err := ga.userStorageManager.SaveUsers(ga.userKeys); err != nil {
		ga.errorHandler.HandleError("AddUser", fmt.Sprintf("Failed to save user: %s", username))
		return false, fmt.Errorf("failed to save user: %s", username)
	}

	ga.logger.Info("User added successfully.", zap.String("username", username))
	return true, nil
}

// AuthenticateUser authenticates a user using their signature.
// It returns true if authentication is successful, false otherwise.
func (ga *GhostAuth) AuthenticateUser(username, signature string) (bool, error) {
	ga.logger.Info("Authenticating user.", zap.String("username", username))

	ga.mutex.Lock()
	defer ga.mutex.Unlock()

	// Retrieve the user's public key
	publicKeyHex, exists := ga.userKeys[username]
	if !exists {
		ga.errorHandler.HandleError("AuthenticateUser", fmt.Sprintf("Public key not found for user: %s", username))
		return false, fmt.Errorf("public key not found for user: %s", username)
	}

	// Decode the hex-encoded public key
	publicKeyBytes, err := hex.DecodeString(publicKeyHex.PublicKey)
	if err != nil {
		ga.errorHandler.HandleError("AuthenticateUser", fmt.Sprintf("Failed to decode public key for user: %s", username))
		return false, fmt.Errorf("failed to decode public key for user: %s", username)
	}

	// Initialize the signature mechanism (Dilithium-2)
	sigScheme := ga.sigScheme

	// Define the message that was signed
	message := fmt.Sprintf("Authenticate user: %s", username)
	messageBytes := []byte(message)

	// Decode the hex-encoded signature
	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		ga.errorHandler.HandleError("AuthenticateUser", fmt.Sprintf("Failed to decode signature for user: %s", username))
		return false, fmt.Errorf("failed to decode signature for user: %s", username)
	}

	// Verify the signature
	err = sigScheme.Verify(messageBytes, signatureBytes, publicKeyBytes)
	if err != nil {
		ga.errorHandler.HandleError("AuthenticateUser", fmt.Sprintf("Signature verification failed for user: %s", username))
		return false, fmt.Errorf("signature verification failed for user: %s", username)
	}

	ga.logger.Info("User authenticated successfully.", zap.String("username", username))
	return true, nil
}

// GetUserPublicKey retrieves the public key of a user.
// It returns the public key as a hex-encoded string and true if found, false otherwise.
func (ga *GhostAuth) GetUserPublicKey(username string) (string, bool) {
	ga.mutex.Lock()
	defer ga.mutex.Unlock()

	userKey, exists := ga.userKeys[username]
	if !exists {
		ga.logger.Warn("Public key not found for user.", zap.String("username", username))
		return "", false
	}

	return userKey.PublicKey, true
}

// GetAuthorizedPrivateKey securely retrieves the private key of a user.
// It returns the private key as a hex-encoded string and true if found, false otherwise.
func (ga *GhostAuth) GetAuthorizedPrivateKey(username string) (string, bool) {
	ga.mutex.Lock()
	defer ga.mutex.Unlock()

	userKey, exists := ga.userKeys[username]
	if !exists {
		ga.logger.Warn("Private key not found for user.", zap.String("username", username))
		return "", false
	}

	return userKey.PrivateKey, true
}

// SignMessage signs a message with the user's private key using Dilithium-2.
// It returns the signature as a hex-encoded string.
func (ga *GhostAuth) SignMessage(username, message string) (string, error) {
	ga.logger.Info("Signing message.", zap.String("username", username), zap.String("message", message))

	ga.mutex.Lock()
	defer ga.mutex.Unlock()

	// Retrieve the user's private key
	privateKeyHex, exists := ga.userKeys[username]
	if !exists {
		ga.errorHandler.HandleError("SignMessage", fmt.Sprintf("Private key not found for user: %s", username))
		return "", fmt.Errorf("private key not found for user: %s", username)
	}

	// Decode the hex-encoded private key
	privateKeyBytes, err := hex.DecodeString(privateKeyHex.PrivateKey)
	if err != nil {
		ga.errorHandler.HandleError("SignMessage", fmt.Sprintf("Failed to decode private key for user: %s", username))
		return "", fmt.Errorf("failed to decode private key for user: %s", username)
	}

	// Initialize the signature mechanism (Dilithium-2)
	sigScheme := ga.sigScheme

	// Sign the message
	signatureBytes, err := sigScheme.Sign([]byte(message), privateKeyBytes)
	if err != nil {
		ga.errorHandler.HandleError("SignMessage", fmt.Sprintf("Failed to sign message for user: %s", username))
		return "", fmt.Errorf("failed to sign message for user: %s", username)
	}

	// Encode the signature as a hex string
	signatureHex := hex.EncodeToString(signatureBytes)

	ga.logger.Info("Message signed successfully.", zap.String("username", username))
	return signatureHex, nil
}
