package aes

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Initialize logger with dynamic filename
var (
	logger *zap.Logger
	keyManager AES_KeyManager
)

func init() {
	timestamp := time.Now().UTC().Format("20060102_150405")
	logFilename := fmt.Sprintf("postquantumsecurity_log_%s.log", timestamp)

	config := zap.NewProductionConfig()
	config.OutputPaths = []string{
		logFilename,
		"stdout",
	}

	var err error
	logger, err = config.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	keyManager = &AES_DummyKeyManager{}
}

// AESContext represents the AES encryption/decryption context.
type AESContext struct {}

// AES_KeyManager defines the interface for key management operations.
type AES_KeyManager interface {
	StoreKey(key []byte) error
	LoadKey() ([]byte, error)
}

// AES_DummyKeyManager is a placeholder implementation for AES_KeyManager.
type AES_DummyKeyManager struct{}

func (dkm *AES_DummyKeyManager) StoreKey(key []byte) error {
	logger.Info("Storing key securely")
	return nil
}

func (dkm *AES_DummyKeyManager) LoadKey() ([]byte, error) {
	logger.Info("Loading key securely")
	return nil, errors.New("key not found")
}

// AES_GCMEncrypt encrypts plaintext using AES-GCM mode.
func AES_GCMEncrypt(plaintext, key, nonce, additionalData []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		logger.Error("Invalid AES key length", zap.Int("length", len(key)))
		return nil, fmt.Errorf("invalid AES key length")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		logger.Error("Failed to create AES cipher", zap.Error(err))
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		logger.Error("Failed to create GCM cipher mode", zap.Error(err))
		return nil, fmt.Errorf("failed to create GCM cipher mode: %w", err)
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, additionalData)
	logger.Info("AES_GCMEncrypt successful", zap.Int("ciphertext_length", len(ciphertext)))
	return ciphertext, nil
}

// AES_GCMDecrypt decrypts ciphertext using AES-GCM mode.
func AES_GCMDecrypt(ciphertext, key, nonce, additionalData []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		logger.Error("Invalid AES key length", zap.Int("length", len(key)))
		return nil, fmt.Errorf("invalid AES key length")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		logger.Error("Failed to create AES cipher", zap.Error(err))
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		logger.Error("Failed to create GCM cipher mode", zap.Error(err))
		return nil, fmt.Errorf("failed to create GCM cipher mode: %w", err)
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, additionalData)
	if err != nil {
		logger.Error("Failed to decrypt ciphertext", zap.Error(err))
		return nil, fmt.Errorf("failed to decrypt ciphertext: %w", err)
	}

	logger.Info("AES_GCMDecrypt successful", zap.Int("plaintext_length", len(plaintext)))
	return plaintext, nil
}

// AES_SecureKeyRotation rotates the encryption key securely.
func AES_SecureKeyRotation(oldKey []byte) ([]byte, error) {
	newKey := make([]byte, len(oldKey))
	if _, err := rand.Read(newKey); err != nil {
		logger.Error("Failed to generate new key", zap.Error(err))
		return nil, fmt.Errorf("failed to generate new key: %w", err)
	}
	logger.Info("AES_SecureKeyRotation successful")
	return newKey, nil
}

// AES_ExampleUsage demonstrates AES encryption and decryption.
func AES_ExampleUsage() {
	key := []byte("examplekey123456")
	nonce := []byte("exampleNonce12")
	plaintext := []byte("Hello, Post-Quantum World!")

	ciphertext, err := AES_GCMEncrypt(plaintext, key, nonce, nil)
	if err != nil {
		logger.Error("Encryption failed", zap.Error(err))
		return
	}
	logger.Info("Encryption successful", zap.String("ciphertext", fmt.Sprintf("%x", ciphertext)))

	decrypted, err := AES_GCMDecrypt(ciphertext, key, nonce, nil)
	if err != nil {
		logger.Error("Decryption failed", zap.Error(err))
		return
	}
	logger.Info("Decryption successful", zap.String("plaintext", string(decrypted)))
}

func main() {
	AES_ExampleUsage()
}

package aes

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Initialize logger with dynamic filename
var (
	logger *zap.Logger
	keyManager AES_KeyManager
)

func init() {
	timestamp := time.Now().UTC().Format("20060102_150405")
	logFilename := fmt.Sprintf("postquantumsecurity_log_%s.log", timestamp)

	config := zap.NewProductionConfig()
	config.OutputPaths = []string{
		logFilename,
		"stdout",
	}

	var err error
	logger, err = config.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	keyManager = &AES_DummyKeyManager{}
}

// AESContext represents the AES encryption/decryption context.
type AESContext struct {}

// AES_KeyManager defines the interface for key management operations.
type AES_KeyManager interface {
	StoreKey(key []byte) error
	LoadKey() ([]byte, error)
}

// AES_DummyKeyManager is a placeholder implementation for AES_KeyManager.
type AES_DummyKeyManager struct{}

func (dkm *AES_DummyKeyManager) StoreKey(key []byte) error {
	logger.Info("Storing key securely")
	return nil
}

func (dkm *AES_DummyKeyManager) LoadKey() ([]byte, error) {
	logger.Info("Loading key securely")
	return nil, errors.New("key not found")
}

// AES_GCMEncrypt encrypts plaintext using AES-GCM mode.
func AES_GCMEncrypt(plaintext, key, nonce, additionalData []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		logger.Error("Invalid AES key length", zap.Int("length", len(key)))
		return nil, fmt.Errorf("invalid AES key length")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		logger.Error("Failed to create AES cipher", zap.Error(err))
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		logger.Error("Failed to create GCM cipher mode", zap.Error(err))
		return nil, fmt.Errorf("failed to create GCM cipher mode: %w", err)
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, additionalData)
	logger.Info("AES_GCMEncrypt successful", zap.Int("ciphertext_length", len(ciphertext)))
	return ciphertext, nil
}

// AES_GCMDecrypt decrypts ciphertext using AES-GCM mode.
func AES_GCMDecrypt(ciphertext, key, nonce, additionalData []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		logger.Error("Invalid AES key length", zap.Int("length", len(key)))
		return nil, fmt.Errorf("invalid AES key length")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		logger.Error("Failed to create AES cipher", zap.Error(err))
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		logger.Error("Failed to create GCM cipher mode", zap.Error(err))
		return nil, fmt.Errorf("failed to create GCM cipher mode: %w", err)
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, additionalData)
	if err != nil {
		logger.Error("Failed to decrypt ciphertext", zap.Error(err))
		return nil, fmt.Errorf("failed to decrypt ciphertext: %w", err)
	}

	logger.Info("AES_GCMDecrypt successful", zap.Int("plaintext_length", len(plaintext)))
	return plaintext, nil
}

// AES_SecureKeyRotation rotates the encryption key securely.
func AES_SecureKeyRotation(oldKey []byte) ([]byte, error) {
	newKey := make([]byte, len(oldKey))
	if _, err := rand.Read(newKey); err != nil {
		logger.Error("Failed to generate new key", zap.Error(err))
		return nil, fmt.Errorf("failed to generate new key: %w", err)
	}
	logger.Info("AES_SecureKeyRotation successful")
	return newKey, nil
}

// AES_ExampleUsage demonstrates AES encryption and decryption.
func AES_ExampleUsage() {
	key := []byte("examplekey123456")
	nonce := []byte("exampleNonce12")
	plaintext := []byte("Hello, Post-Quantum World!")

	ciphertext, err := AES_GCMEncrypt(plaintext, key, nonce, nil)
	if err != nil {
		logger.Error("Encryption failed", zap.Error(err))
		return
	}
	logger.Info("Encryption successful", zap.String("ciphertext", fmt.Sprintf("%x", ciphertext)))

	decrypted, err := AES_GCMDecrypt(ciphertext, key, nonce, nil)
	if err != nil {
		logger.Error("Decryption failed", zap.Error(err))
		return
	}
	logger.Info("Decryption successful", zap.String("plaintext", string(decrypted)))
}

func main() {
	AES_ExampleUsage()
}
// AES_SecureKeyRotation rotates an old key into a new key securely.
func AES_SecureKeyRotation(oldKey []byte) ([]byte, error) {
	if err := AES_ValidateKey(oldKey); err != nil {
		logger.Printf("Invalid old key: %v", err)
		return nil, err
	}

	newKey := make([]byte, len(oldKey))
	if _, err := rand.Read(newKey); err != nil {
		logger.Printf("Failed to generate new key: %v", err)
		return nil, fmt.Errorf("failed to generate new key: %w", err)
	}

	// Zero out the old key for security
	AES_ZeroKey(oldKey)
	logger.Printf("Old key securely erased.")

	// Store the new key
	if err := AES_StoreEncryptionKey(newKey); err != nil {
		logger.Printf("Failed to store new key: %v", err)
		return nil, err
	}
	logger.Printf("New key successfully rotated and stored.")

	return newKey, nil
}

// AES_ZeroKey securely erases a key from memory.
func AES_ZeroKey(key []byte) {
	for i := range key {
		key[i] = 0
	}
	logger.Printf("Key securely erased from memory.")
}

// AES_Sign creates a digital signature of the data using ECDSA.
func AES_Sign(privateKey *ecdsa.PrivateKey, data []byte) (string, error) {
	if privateKey == nil {
		logger.Printf("Private key is nil.")
		return "", ErrPrivateKeyNil
	}

	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		logger.Printf("Signing failed: %v", err)
		return "", fmt.Errorf("signing failed: %w", err)
	}

	signatureBytes, err := asn1.Marshal(struct {
		R, S *big.Int
	}{r, s})
	if err != nil {
		logger.Printf("Failed to marshal signature: %v", err)
		return "", fmt.Errorf("failed to marshal signature: %w", err)
	}

	signature := hex.EncodeToString(signatureBytes)
	logger.Printf("Signature successfully created: %s", signature)
	return signature, nil
}

// AES_Verify verifies a digital signature using ECDSA.
func AES_Verify(publicKey *ecdsa.PublicKey, data []byte, signatureHex string) (bool, error) {
	if publicKey == nil {
		logger.Printf("Public key is nil.")
		return false, ErrPublicKeyNil
	}

	signatureBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		logger.Printf("Failed to decode signature: %v", err)
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}

	var rs struct {
		R, S *big.Int
	}
	if _, err := asn1.Unmarshal(signatureBytes, &rs); err != nil {
		logger.Printf("Failed to unmarshal signature: %v", err)
		return false, fmt.Errorf("failed to unmarshal signature: %w", err)
	}

	hash := sha256.Sum256(data)
	valid := ecdsa.Verify(publicKey, hash[:], rs.R, rs.S)
	if !valid {
		logger.Printf("Signature verification failed.")
		return false, ErrSignatureVerification
	}

	logger.Printf("Signature successfully verified.")
	return true, nil
}
// AES_Sign creates a digital signature of the data using ECDSA.
func AES_Sign(privateKey *ecdsa.PrivateKey, data []byte) (string, error) {
	if privateKey == nil {
		logger.Printf("Private key is nil.")
		return "", ErrPrivateKeyNil
	}

	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		logger.Printf("Signing failed: %v", err)
		return "", fmt.Errorf("signing failed: %w", err)
	}

	signatureBytes, err := asn1.Marshal(struct {
		R, S *big.Int
	}{r, s})
	if err != nil {
		logger.Printf("Failed to marshal signature: %v", err)
		return "", fmt.Errorf("failed to marshal signature: %w", err)
	}

	signature := hex.EncodeToString(signatureBytes)
	logger.Printf("Signature successfully created: %s", signature)
	return signature, nil
}

// AES_Verify verifies a digital signature using ECDSA.
func AES_Verify(publicKey *ecdsa.PublicKey, data []byte, signatureHex string) (bool, error) {
	if publicKey == nil {
		logger.Printf("Public key is nil.")
		return false, ErrPublicKeyNil
	}

	signatureBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		logger.Printf("Failed to decode signature: %v", err)
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}

	var rs struct {
		R, S *big.Int
	}
	if _, err := asn1.Unmarshal(signatureBytes, &rs); err != nil {
		logger.Printf("Failed to unmarshal signature: %v", err)
		return false, fmt.Errorf("failed to unmarshal signature: %w", err)
	}

	hash := sha256.Sum256(data)
	valid := ecdsa.Verify(publicKey, hash[:], rs.R, rs.S)
	if !valid {
		logger.Printf("Signature verification failed.")
		return false, ErrSignatureVerification
	}

	logger.Printf("Signature successfully verified.")
	return true, nil
}

func init() {
	// Initialize the logger with dynamic filename
	var err error
	logger, err = NewLogger("postquantumsecurity_log")
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	logger.Info("Logger initialized successfully with UTC timestamps.")
}

// AES_GenerateRandomKey generates a secure random key for AES encryption.
func AES_GenerateRandomKey(length int) ([]byte, error) {
	if length != 16 && length != 24 && length != 32 {
		return nil, ErrInvalidAESKeyLength
	}

	key := make([]byte, length)
	if _, err := rand.Read(key); err != nil {
		logger.Printf("Failed to generate random AES key: %v", err)
		return nil, fmt.Errorf("failed to generate random AES key: %w", err)
	}

	logger.Printf("Generated random AES key of length: %d", length)
	return key, nil
}

// AES_GenerateNonce generates a secure random nonce for AES-GCM encryption.
func AES_GenerateNonce() ([]byte, error) {
	nonce := make([]byte, 12) // 96 bits for AES-GCM
	if _, err := rand.Read(nonce); err != nil {
		logger.Printf("Failed to generate nonce: %v", err)
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	logger.Printf("Generated random AES-GCM nonce.")
	return nonce, nil
}
