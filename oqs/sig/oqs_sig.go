// OQS-GO
// Aaron Stovall

package sig

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Initialize logger with dynamic file naming
var logger *zap.SugaredLogger

func init() {
	currentTime := time.Now().UTC().Format("20060102_150405")
	logFileName1 := fmt.Sprintf("ghostshell/logging/postquantumsecurity_log_%s.log", currentTime)

	// Configure zap to write to both log files and stdout
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{
		logFileName1, // Ensure logs are placed in ghostshell/logging
		"stdout",     // Also write logs to the console
	}

	// Build the logger
	log, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}

	// Set the global logger
	logger = log.Sugar()

	// Log initialization information
	logger.Infof("Logger initialized with file: %s", logFileName1)
}

// OQS_STATUS defines the status codes for operations.
type OQS_STATUS int

const (
	OQS_SUCCESS OQS_STATUS = iota
	OQS_ERROR
)

// Error definitions
var (
	ErrInvalidSignatureStructure = errors.New("invalid signature structure: missing key lengths")
	ErrKeyGenerationFailed       = errors.New("failed to generate key pair")
	ErrSecretKeyNotInitialized   = errors.New("secret key is not initialized")
	ErrPublicKeyNotInitialized   = errors.New("public key is not initialized")
	ErrSignatureGenerationFailed = errors.New("failed to generate signature")
	ErrSignatureVerification     = errors.New("signature verification failed")
	ErrInvalidSignatureFormat    = errors.New("invalid signature format")
	ErrKeyManagerNotInitialized  = errors.New("key manager not initialized")
	ErrNilPrivateKey             = errors.New("private key is nil")
	ErrNilPublicKey              = errors.New("public key is nil")
)

// Signature represents a digital signature scheme.
type Signature struct {
	Name      string
	PublicKey *ecdsa.PublicKey
	SecretKey *ecdsa.PrivateKey
	Signature string
	KeyMutex  sync.RWMutex // Ensures thread-safe access to keys
}

// KeyManager defines the interface for key management operations.
type KeyManager interface {
	StoreKey(key *ecdsa.PrivateKey) error
	LoadKey() (*ecdsa.PrivateKey, error)
}

// DummyKeyManager is a placeholder implementation for KeyManager.
// Replace with actual implementation interfacing with HSM or cloud KMS.
type DummyKeyManager struct{}

// StoreKey securely stores the private key.
func (dkm *DummyKeyManager) StoreKey(key *ecdsa.PrivateKey) error {
	logger.Infof("Storing private key securely (DummyKeyManager)")
	// Implement secure storage logic here
	return nil
}

// LoadKey securely retrieves the private key.
func (dkm *DummyKeyManager) LoadKey() (*ecdsa.PrivateKey, error) {
	logger.Infof("Loading private key securely (DummyKeyManager)")
	// Implement secure retrieval logic here
	return nil, errors.New("key not found")
}

// Global variables for key management
var (
	keyManager KeyManager   = &DummyKeyManager{}
	kmMutex    sync.RWMutex // Ensures thread-safe access to keyManager
)

// SetKeyManager sets the key manager for secure key operations.
func SetKeyManager(km KeyManager) error {
	if km == nil {
		return errors.New("key manager cannot be nil")
	}
	kmMutex.Lock()
	defer kmMutex.Unlock()
	keyManager = km
	logger.Infof("Key manager has been set")
	return nil
}

// NewSignature initializes a Signature structure.
func NewSignature(name string) *Signature {
	return &Signature{
		Name:     name,
		KeyMutex: sync.RWMutex{},
	}
}

// GenerateKeypair generates a public and secret key for the signature scheme.
func (s *Signature) GenerateKeypair() error {
	s.KeyMutex.Lock()
	defer s.KeyMutex.Unlock()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		logger.Errorf("Key generation failed: %v", err)
		return fmt.Errorf("key generation failed: %w", err)
	}

	s.SecretKey = privateKey
	s.PublicKey = &privateKey.PublicKey

	// Optionally store the key using KeyManager
	if keyManager != nil {
		if err := keyManager.StoreKey(privateKey); err != nil {
			logger.Errorf("Failed to store private key: %v", err)
			return fmt.Errorf("failed to store private key: %w", err)
		}
		logger.Infof("Private key stored successfully using KeyManager")
	} else {
		logger.Warnf("KeyManager not initialized, private key not stored securely")
	}

	logger.Infof("Keypair generated successfully for Signature scheme: %s", s.Name)
	return nil
}

// Sign generates a signature for the given message using the secret key.
func (s *Signature) Sign(message []byte) (string, error) {
	s.KeyMutex.RLock()
	defer s.KeyMutex.RUnlock()

	if s.SecretKey == nil {
		return "", ErrSecretKeyNotInitialized
	}

	hash := sha256.Sum256(message)
	r, sSig, err := ecdsa.Sign(rand.Reader, s.SecretKey, hash[:])
	if err != nil {
		logger.Errorf("Signature generation failed: %v", err)
		return "", fmt.Errorf("signature generation failed: %w", err)
	}

	// ASN.1 encode the signature
	signatureBytes, err := asn1.Marshal(struct {
		R, S *big.Int
	}{r, sSig})
	if err != nil {
		logger.Errorf("Failed to marshal signature: %v", err)
		return "", fmt.Errorf("failed to marshal signature: %w", err)
	}

	s.Signature = hex.EncodeToString(signatureBytes)
	logger.Infof("Signature generated successfully for message: %x", message)
	return s.Signature, nil
}

// Verify verifies the signature of the given message.
func (s *Signature) Verify(message []byte, signatureHex string) (bool, error) {
	s.KeyMutex.RLock()
	defer s.KeyMutex.RUnlock()

	if s.PublicKey == nil {
		return false, ErrPublicKeyNotInitialized
	}

	// Decode the hex-encoded signature
	signatureBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		logger.Errorf("Failed to decode signature: %v", err)
		return false, fmt.Errorf("%w: %v", ErrInvalidSignatureFormat, err)
	}

	// Unmarshal the ASN.1 encoded signature
	var sigStruct struct {
		R, S *big.Int
	}
	if _, err := asn1.Unmarshal(signatureBytes, &sigStruct); err != nil {
		logger.Errorf("Failed to unmarshal signature: %v", err)
		return false, fmt.Errorf("%w: %v", ErrInvalidSignatureFormat, err)
	}

	// Hash the message
	hash := sha256.Sum256(message)

	// Verify the signature
	valid := ecdsa.Verify(s.PublicKey, hash[:], sigStruct.R, sigStruct.S)
	if valid {
		logger.Infof("Signature verification successful for message: %x", message)
	} else {
		logger.Warnf("Signature verification failed for message: %x", message)
	}

	return valid, nil
}

// SecureKeyRotation rotates the secret key securely.
func (s *Signature) SecureKeyRotation() error {
	s.KeyMutex.Lock()
	defer s.KeyMutex.Unlock()

	if s.SecretKey == nil {
		return ErrSecretKeyNotInitialized
	}

	// Generate a new keypair
	newPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		logger.Errorf("New key generation failed: %v", err)
		return fmt.Errorf("new key generation failed: %w", err)
	}

	// Optionally store the new key using KeyManager
	if keyManager != nil {
		if err := keyManager.StoreKey(newPrivateKey); err != nil {
			logger.Errorf("Failed to store new private key: %v", err)
			return fmt.Errorf("failed to store new private key: %w", err)
		}
		logger.Infof("New private key stored successfully using KeyManager")
	} else {
		logger.Warnf("KeyManager not initialized, new private key not stored securely")
	}

	// Zero out the old key
	zeroPrivateKey(s.SecretKey)

	// Replace with the new key
	s.SecretKey = newPrivateKey
	s.PublicKey = &newPrivateKey.PublicKey

	logger.Infof("Secure key rotation successful for Signature scheme: %s", s.Name)
	return nil
}

// zeroPrivateKey securely erases a private key from memory.
func zeroPrivateKey(key *ecdsa.PrivateKey) {
	if key == nil {
		return
	}
	key.D.SetInt64(0)
	key.X.SetInt64(0)
	key.Y.SetInt64(0)
}

// StoreSignatureKey securely stores the signature's secret key using the key manager.
func (s *Signature) StoreSignatureKey() error {
	s.KeyMutex.RLock()
	defer s.KeyMutex.RUnlock()

	if s.SecretKey == nil {
		return ErrSecretKeyNotInitialized
	}

	kmMutex.RLock()
	defer kmMutex.RUnlock()

	if keyManager == nil {
		return ErrKeyManagerNotInitialized
	}

	if err := keyManager.StoreKey(s.SecretKey); err != nil {
		logger.Errorf("Failed to store signature key: %v", err)
		return fmt.Errorf("failed to store signature key: %w", err)
	}

	logger.Infof("Signature key stored successfully using KeyManager")
	return nil
}

// LoadSignatureKey securely retrieves the signature's secret key using the key manager.
func (s *Signature) LoadSignatureKey() error {
	kmMutex.RLock()
	defer kmMutex.RUnlock()

	if keyManager == nil {
		return ErrKeyManagerNotInitialized
	}

	privateKey, err := keyManager.LoadKey()
	if err != nil {
		logger.Errorf("Failed to load signature key: %v", err)
		return fmt.Errorf("failed to load signature key: %w", err)
	}

	s.KeyMutex.Lock()
	defer s.KeyMutex.Unlock()

	s.SecretKey = privateKey
	s.PublicKey = &privateKey.PublicKey
	logger.Infof("Signature key loaded successfully using KeyManager")
	return nil
}

// Store securely stores key-value pairs using the appropriate method.
func (s *Signature) Store(key string, value string) error {
	return s.StoreInMemory(key, value)
}

// Retrieve securely retrieves key-value pairs using the appropriate method.
func (s *Signature) Retrieve(key string) (string, error) {
	return s.RetrieveFromMemory(key)
}

// StoreInMemory securely stores key-value pairs in memory.
func (s *Signature) StoreInMemory(key string, value string) error {
	s.KeyMutex.Lock()
	defer s.KeyMutex.Unlock()

	// For demonstration, we'll use base64 encoding as a placeholder.
	// Replace this with actual encryption logic as needed.
	encodedValue := base64.StdEncoding.EncodeToString([]byte(value))
	s.Signature = encodedValue // Example storage
	logger.Infof("Stored in-memory key '%s' securely", key)
	return nil
}

// RetrieveFromMemory securely retrieves key-value pairs from memory.
func (s *Signature) RetrieveFromMemory(key string) (string, error) {
	s.KeyMutex.RLock()
	defer s.KeyMutex.RUnlock()

	if s.Signature == "" {
		return "", ErrKeyNotFound
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(s.Signature)
	if err != nil {
		logger.Errorf("Failed to decode value for key '%s': %v", key, err)
		return "", fmt.Errorf("failed to decode value: %w", err)
	}

	value := string(decodedBytes)
	logger.Infof("Retrieved in-memory key '%s' successfully", key)
	return value, nil
}

// StorageCommands defines high-level storage operations.
func (s *Signature) StorageCommands(command string, args ...interface{}) (interface{}, error) {
	switch command {
	case "store":
		if len(args) != 2 {
			return nil, errors.New("store command requires a key and value")
		}
		key, ok1 := args[0].(string)
		value, ok2 := args[1].(string)
		if !ok1 || !ok2 {
			return nil, errors.New("store command requires string arguments")
		}
		return nil, s.Store(key, value)
	case "retrieve":
		if len(args) != 1 {
			return nil, errors.New("retrieve command requires a key")
		}
		key, ok := args[0].(string)
		if !ok {
			return nil, errors.New("retrieve command requires a string argument")
		}
		return s.Retrieve(key)
	case "delete":
		if len(args) != 1 {
			return nil, errors.New("delete command requires a key")
		}
		key, ok := args[0].(string)
		if !ok {
			return nil, errors.New("delete command requires a string argument")
		}
		return nil, s.SecureMemoryOperations(key)
	default:
		return nil, errors.New("unsupported command")
	}
}

// Verify verifies the signature of the given message.
func (s *Signature) Verify(message []byte, signatureHex string) (bool, error) {
	s.KeyMutex.RLock()
	defer s.KeyMutex.RUnlock()

	if s.PublicKey == nil {
		return false, ErrPublicKeyNotInitialized
	}

	// Decode the hex-encoded signature
	signatureBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		logger.Errorf("Failed to decode signature: %v", err)
		return false, fmt.Errorf("%w: %v", ErrInvalidSignatureFormat, err)
	}

	// Unmarshal the ASN.1 encoded signature
	var sigStruct struct {
		R, S *big.Int
	}
	if _, err := asn1.Unmarshal(signatureBytes, &sigStruct); err != nil {
		logger.Errorf("Failed to unmarshal signature: %v", err)
		return false, fmt.Errorf("%w: %v", ErrInvalidSignatureFormat, err)
	}

	// Hash the message
	hash := sha256.Sum256(message)

	// Verify the signature
	valid := ecdsa.Verify(s.PublicKey, hash[:], sigStruct.R, sigStruct.S)
	if valid {
		logger.Infof("Signature verification successful for message: %x", message)
	} else {
		logger.Warnf("Signature verification failed for message: %x", message)
	}

	return valid, nil
}

// ExampleUsage demonstrates signature generation and verification.
func SIGExampleUsage() {
	// Initialize a signature scheme (e.g., ECDSA P-256)
	sigScheme := NewSignature("ECDSA-P256")

	// Generate keypair
	if err := sigScheme.GenerateKeypair(); err != nil {
		logger.Fatalf("Keypair generation failed: %v", err)
	}

	message := []byte("Hello, World!")
	signature, err := sigScheme.Sign(message)
	if err != nil {
		logger.Fatalf("Signing failed: %v", err)
	}

	isValid, err := sigScheme.Verify(message, signature)
	if err != nil {
		logger.Fatalf("Verification error: %v", err)
	}

	if isValid {
		logger.Infof("Signature is valid.")
	} else {
		logger.Warnf("Signature is invalid.")
	}

	// Rotate the secret key
	if err := sigScheme.SecureKeyRotation(); err != nil {
		logger.Fatalf("Key rotation failed: %v", err)
	}

	// Sign and verify again with the new key
	newSignature, err := sigScheme.Sign(message)
	if err != nil {
		logger.Fatalf("Signing after key rotation failed: %v", err)
	}

	isValid, err = sigScheme.Verify(message, newSignature)
	if err != nil {
		logger.Fatalf("Verification after key rotation failed: %v", err)
	}

	if isValid {
		logger.Infof("Signature after key rotation is valid.")
	} else {
		logger.Warnf("Signature after key rotation is invalid.")
	}
}
