// File: ghostshell/auth/auth_manager.go
package ghostauth

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"ghostshell/oqs/sig"
	"sync"

	"go.uber.org/zap"
)

// AuthManager manages user authentication, authorization, and MFA token operations.
type AuthManager struct {
	vault        *vault.Vault
	logger       *zap.Logger
	sigScheme    *sig.Scheme
	errorHandler ErrorHandler
	mutex        sync.Mutex
}

// NewAuthManager initializes and returns a new instance of AuthManager.
func NewAuthManager(vault *vault.Vault, logger *zap.Logger, errorHandler ErrorHandler) (*AuthManager, error) {
	// Initialize the quantum-safe signature scheme (Dilithium-2)
	scheme, err := sig.NewScheme("Dilithium2", logger)
	if err != nil {
		logger.Error("Failed to initialize signature scheme", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize signature scheme: %w", err)
	}

	return &AuthManager{
		vault:        vault,
		logger:       logger,
		sigScheme:    scheme,
		errorHandler: errorHandler,
	}, nil
}

// AddUser adds a new user with encrypted credentials to the vault.
func (am *AuthManager) AddUser(username, password, role string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	if username == "" || password == "" || role == "" {
		return errors.New("username, password, and role cannot be empty")
	}

	exists, err := am.vault.KeyExists(username)
	if err != nil {
		am.logger.Error("Failed to check key existence", zap.Error(err))
		return fmt.Errorf("failed to check key existence: %w", err)
	}
	if exists {
		am.logger.Warn("Username already exists", zap.String("username", username))
		return errors.New("username already exists")
	}

	encryptedPassword, err := am.vault.EncryptVault([]byte(password))
	if err != nil {
		am.logger.Error("Failed to encrypt password", zap.Error(err))
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	userData := map[string]string{
		"password": string(encryptedPassword),
		"role":     role,
	}
	jsonData, err := json.Marshal(userData)
	if err != nil {
		am.logger.Error("Failed to marshal user data", zap.Error(err))
		return fmt.Errorf("failed to marshal user data: %w", err)
	}

	if err := am.vault.Store(username, string(jsonData)); err != nil {
		am.logger.Error("Failed to store user data", zap.Error(err))
		return fmt.Errorf("failed to store user data: %w", err)
	}

	am.logger.Info("Added new user", zap.String("username", username))
	return nil
}

// Authenticate validates user credentials and generates a session ID.
func (am *AuthManager) Authenticate(username, password string) (string, error) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	if username == "" || password == "" {
		return "", errors.New("username and password cannot be empty")
	}

	userData, err := am.vault.Retrieve(username)
	if err != nil {
		am.logger.Warn("Invalid credentials", zap.String("username", username))
		return "", errors.New("invalid credentials")
	}

	var userMap map[string]string
	if err := json.Unmarshal([]byte(userData), &userMap); err != nil {
		am.logger.Error("Failed to parse user data", zap.Error(err))
		return "", fmt.Errorf("failed to parse user data: %w", err)
	}

	encryptedPassword := []byte(userMap["password"])
	decryptedPassword, err := am.vault.Decrypt(encryptedPassword)
	if err != nil || string(decryptedPassword) != password {
		am.logger.Warn("Invalid credentials", zap.String("username", username))
		return "", errors.New("invalid credentials")
	}

	sessionID, err := am.vault.GenerateRandomHex(32)
	if err != nil {
		am.logger.Error("Failed to generate session ID", zap.Error(err))
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	sessionData := map[string]string{
		"username": username,
		"role":     userMap["role"],
		"expires":  am.vault.GenerateExpiryTime(30 * 60), // 30 minutes in seconds
	}
	jsonSession, err := json.Marshal(sessionData)
	if err != nil {
		am.logger.Error("Failed to marshal session data", zap.Error(err))
		return "", fmt.Errorf("failed to marshal session data: %w", err)
	}

	if err := am.vault.Store(sessionID, string(jsonSession)); err != nil {
		am.logger.Error("Failed to store session data", zap.Error(err))
		return "", fmt.Errorf("failed to store session data: %w", err)
	}

	am.logger.Info("User authenticated successfully", zap.String("username", username), zap.String("sessionID", sessionID))
	return sessionID, nil
}

// Authorize checks the session ID validity and returns the user's role.
func (am *AuthManager) Authorize(sessionID string) (string, error) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	sessionData, err := am.vault.Retrieve(sessionID)
	if err != nil {
		am.logger.Warn("Invalid session", zap.String("sessionID", sessionID))
		return "", errors.New("invalid session")
	}

	var session map[string]string
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		am.logger.Error("Failed to parse session data", zap.Error(err))
		return "", fmt.Errorf("failed to parse session data: %w", err)
	}

	expiryTime, err := am.vault.ParseExpiryTime(session["expires"])
	if err != nil {
		am.logger.Error("Invalid session expiry", zap.Error(err))
		return "", fmt.Errorf("invalid session expiry: %w", err)
	}

	if am.vault.IsExpired(expiryTime) {
		am.vault.Delete(sessionID)
		am.logger.Warn("Session expired", zap.String("sessionID", sessionID))
		return "", errors.New("session expired")
	}

	am.logger.Info("Session authorized", zap.String("sessionID", sessionID), zap.String("role", session["role"]))
	return session["role"], nil
}

// GenerateMFAToken generates a multi-factor authentication (MFA) token for the specified user.
func (am *AuthManager) GenerateMFAToken(username string) (string, error) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Define the MFA message that will be signed
	message := "MFA request for user: " + username

	// Retrieve the private key securely, ensuring proper authorization
	privateKey, ok := am.ghostAuth.GetAuthorizedPrivateKey(username)
	if !ok {
		am.errorHandler.HandleError("GenerateMFAToken", fmt.Sprintf("Private key not found or unauthorized access for user: %s", username))
		return "", fmt.Errorf("private key not found or unauthorized for user: %s", username)
	}

	// Sign the message with the retrieved private key using quantum-safe signature
	signature, err := am.sigScheme.Sign([]byte(message), []byte(privateKey))
	if err != nil {
		am.errorHandler.HandleError("GenerateMFAToken", "Failed to sign MFA token.")
		return "", fmt.Errorf("failed to sign MFA token: %w", err)
	}

	// Convert signature to hexadecimal string for safe transmission/storage
	token := fmt.Sprintf("%x", signature)

	am.logger.Info("Generated MFA Token", zap.String("username", username), zap.String("token", token))
	return token, nil
}

// VerifyMFAToken verifies the MFA token for the specified user.
func (am *AuthManager) VerifyMFAToken(username, token string) (bool, error) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Define the message that was signed to generate the MFA token
	message := "MFA request for user: " + username

	// Retrieve the public key for the user
	publicKey, ok := am.ghostAuth.GetUserPublicKey(username)
	if !ok {
		am.errorHandler.HandleError("VerifyMFAToken", fmt.Sprintf("Failed to retrieve public key for user: %s", username))
		return false, fmt.Errorf("failed to retrieve public key for user: %s", username)
	}

	// Decode the hexadecimal token back to bytes
	signature, err := hexToBytes(token)
	if err != nil {
		am.errorHandler.HandleError("VerifyMFAToken", "Invalid token format.")
		return false, fmt.Errorf("invalid token format: %w", err)
	}

	// Verify the MFA token (signature) with the user's public key
	valid, err := am.sigScheme.Verify([]byte(message), signature, []byte(publicKey))
	if err != nil {
		am.errorHandler.HandleError("VerifyMFAToken", "Failed to verify MFA token.")
		return false, fmt.Errorf("failed to verify MFA token: %w", err)
	}

	if !valid {
		am.errorHandler.HandleError("VerifyMFAToken", fmt.Sprintf("Signature verification failed for MFA token of user: %s", username))
	}

	return valid, nil
}

// hexToBytes converts a hexadecimal string to a byte slice.
func hexToBytes(hexStr string) ([]byte, error) {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, err
	}
	return data, nil
}
