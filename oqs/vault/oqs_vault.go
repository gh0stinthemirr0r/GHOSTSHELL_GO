package oqs_vault

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"ghostshell/oqs/aes"
	"ghostshell/oqs/rand"
	"ghostshell/oqs/sig"

	"go.uber.org/zap"
)

// Initialize logger with dynamic file naming
var logger *zap.SugaredLogger

func init() {
	currentTime := time.Now().UTC().Format("20060102_150405")
	logFileName1 := fmt.Sprintf("postquantumsecurity_log_%s.log", currentTime)
	logFileName2 := fmt.Sprintf("vaultlog_%s.log", currentTime)

	// Configure zap to write to both log files and stdout
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{
		"logging/" + logFileName1, // Ensure logs are placed in ghostshell/logging
		"logging/" + logFileName2,
		"stdout", // Also write logs to the console
	}

	// Build the logger
	log, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}

	// Set the global logger
	logger = log.Sugar()

	// Log initialization information
	logger.Infof("Logger initialized with files: %s and %s", logFileName1, logFileName2)
}

type OQS_STATUS int

const (
	OQS_SUCCESS OQS_STATUS = iota
	OQS_ERROR
)

var (
	ErrInvalidMasterKey      = errors.New("master key is invalid")
	ErrKeyManagementFailed   = errors.New("key management operation failed")
	ErrEncryptionFailed      = errors.New("encryption failed")
	ErrDecryptionFailed      = errors.New("decryption failed")
	ErrIntegrityCheckFailed  = errors.New("data integrity check failed")
	ErrKeyNotFound           = errors.New("key not found in vault")
	ErrNonceGenerationFailed = errors.New("nonce generation failed")
	ErrSignatureVerification = errors.New("signature verification failed")
	ErrInvalidCiphertext     = errors.New("invalid ciphertext")
)

// Vault manages secure storage and operations.
type Vault struct {
	data      map[string]string
	masterKey []byte
	signature *sig.Signature
	mutex     sync.RWMutex
}

// KeyManager interface defines secure key management operations.
type KeyManager interface {
	StoreKey(key []byte) error
	LoadKey() ([]byte, error)
	GeneratePQSPrivateKey() ([]byte, error) // Generates a post-quantum private key
	RotateMasterKey() ([]byte, error)       // Rotates the master key securely
	ValidateKey(key []byte) bool            // Validates a provided key
}

// EncryptVault encrypts arbitrary data using the vault's master key.
func (v *Vault) EncryptVault(data []byte) (string, error) {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	nonce := make([]byte, 12)
	if err := rand.RandomBytes(nonce); err != nil {
		return "", fmt.Errorf("%w: nonce generation failed", ErrNonceGenerationFailed)
	}

	ciphertext, err := aes.AES_GCMEncrypt(data, v.masterKey, nonce, nil)
	if err != nil {
		return "", fmt.Errorf("%w: encryption failed", ErrEncryptionFailed)
	}

	encrypted := base64.StdEncoding.EncodeToString(append(nonce, ciphertext...))
	return encrypted, nil
}

// DecryptVault decrypts previously encrypted data using the vault's master key.
func (v *Vault) DecryptVault(encrypted string) ([]byte, error) {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid ciphertext format", ErrInvalidCiphertext)
	}

	if len(data) < 12 {
		return nil, fmt.Errorf("%w: ciphertext too short", ErrInvalidCiphertext)
	}

	nonce := data[:12]
	ciphertext := data[12:]

	plaintext, err := aes.AES_GCMDecrypt(ciphertext, v.masterKey, nonce, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: decryption failed", ErrDecryptionFailed)
	}

	return plaintext, nil
}

// EncryptKey encrypts sensitive keys for storage or transmission.
func (v *Vault) EncryptKey(key []byte) (string, error) {
	return v.EncryptVault(key)
}

// DecryptKey decrypts sensitive keys for use in secure operations.
func (v *Vault) DecryptKey(encryptedKey string) ([]byte, error) {
	return v.DecryptVault(encryptedKey)
}

// EncryptConfigFile encrypts and writes configuration data to a file.
func (v *Vault) EncryptConfigFile(filePath string, configData []byte) error {
	encryptedData, err := v.EncryptVault(configData)
	if err != nil {
		return fmt.Errorf("failed to encrypt config data: %w", err)
	}

	if err := ioutil.WriteFile(filePath, []byte(encryptedData), 0600); err != nil {
		return fmt.Errorf("failed to write encrypted config file: %w", err)
	}

	logger.Infof("Configuration file encrypted and saved to '%s'", filePath)
	return nil
}

// DecryptConfigFile reads and decrypts configuration data from a file.
func (v *Vault) DecryptConfigFile(filePath string) ([]byte, error) {
	encryptedData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted config file: %w", err)
	}

	decryptedData, err := v.DecryptVault(string(encryptedData))
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt config file: %w", err)
	}

	logger.Infof("Configuration file decrypted successfully from '%s'", filePath)
	return decryptedData, nil
}

// Securely erases sensitive data from memory.
func (v *Vault) SecureErase() {
	v.mutex.Lock()
	for k := range v.data {
		delete(v.data, k)
	}
	zeroKey(v.masterKey)
	v.mutex.Unlock()
	logger.Infof("Vault securely erased")
}

// Helper function to zero out sensitive keys.
func zeroKey(key []byte) {
	for i := range key {
		key[i] = 0
	}
}

// NewVault initializes a quantum-safe vault.
func NewVault(masterKey []byte) (*Vault, error) {
	if len(masterKey) != 32 {
		return nil, ErrInvalidMasterKey
	}

	signature := sig.NewSignature("Dilithium2")
	if err := signature.GenerateKeypair(); err != nil {
		return nil, fmt.Errorf("failed to generate signing keypair: %w", err)
	}

	return &Vault{
		data:      make(map[string]string),
		masterKey: masterKey,
		signature: signature,
	}, nil
}

// NewConfiguration initializes a configuration file manager.
func NewConfiguration(configFilePath string, masterKey []byte) (*Vault, error) {
	vault, err := NewVault(masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize configuration vault: %w", err)
	}

	// Load existing configuration from the file
	if _, err := ioutil.ReadFile(configFilePath); err == nil {
		logger.Infof("Existing configuration loaded from %s", configFilePath)
	}

	return vault, nil
}

// NewStorage initializes secure data storage.
func NewStorage(storageType string, masterKey []byte) (*Vault, error) {
	vault, err := NewVault(masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage vault: %w", err)
	}

	logger.Infof("Storage type '%s' initialized securely", storageType)
	return vault, nil
}

// StoreInMemory securely stores key-value pairs in memory.
func (v *Vault) StoreInMemory(key string, value string) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	encryptedValue, err := v.EncryptVault([]byte(value))
	if err != nil {
		return fmt.Errorf("failed to encrypt value: %w", err)
	}

	v.data[key] = encryptedValue
	logger.Infof("Stored in-memory key '%s' securely", key)
	return nil
}

// RetrieveFromMemory securely retrieves key-value pairs from memory.
func (v *Vault) RetrieveFromMemory(key string) (string, error) {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	encryptedValue, exists := v.data[key]
	if !exists {
		return "", ErrKeyNotFound
	}

	decryptedValue, err := v.DecryptVault(encryptedValue)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt value: %w", err)
	}

	return string(decryptedValue), nil
}

// SecureMemoryOperations securely wipes in-memory data.
func (v *Vault) SecureMemoryOperations(key string) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	encryptedValue, exists := v.data[key]
	if !exists {
		return ErrKeyNotFound
	}

	zeroKey([]byte(encryptedValue))
	delete(v.data, key)
	logger.Infof("In-memory data for key '%s' securely wiped", key)
	return nil
}

// Store securely stores key-value pairs using the appropriate method.
func (v *Vault) Store(key string, value string) error {
	return v.StoreInMemory(key, value)
}

// Retrieve securely retrieves key-value pairs using the appropriate method.
func (v *Vault) Retrieve(key string) (string, error) {
	return v.RetrieveFromMemory(key)
}

// StorageCommands defines high-level storage operations.
func (v *Vault) StorageCommands(command string, args ...interface{}) (interface{}, error) {
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
		return nil, v.Store(key, value)
	case "retrieve":
		if len(args) != 1 {
			return nil, errors.New("retrieve command requires a key")
		}
		key, ok := args[0].(string)
		if !ok {
			return nil, errors.New("retrieve command requires a string argument")
		}
		return v.Retrieve(key)
	case "delete":
		if len(args) != 1 {
			return nil, errors.New("delete command requires a key")
		}
		key, ok := args[0].(string)
		if !ok {
			return nil, errors.New("delete command requires a string argument")
		}
		return nil, v.SecureMemoryOperations(key)
	default:
		return nil, errors.New("unsupported command")
	}
}

// Example for integrating vault-based secure storage
func ExampleVaultIntegration() {
	masterKey := make([]byte, 32)
	if err := rand.RandomBytes(masterKey); err != nil {
		logger.Fatalf("Failed to generate master key: %v", err)
	}

	vault, err := NewVault(masterKey)
	if err != nil {
		logger.Fatalf("Vault initialization failed: %v", err)
	}

	// Example usage
	if err := vault.Store("api_key", "my_secure_api_key"); err != nil {
		logger.Errorf("Failed to store value: %v", err)
	}

	value, err := vault.Retrieve("api_key")
	if err != nil {
		logger.Errorf("Failed to retrieve value: %v", err)
	} else {
		logger.Infof("Retrieved value: %s", value)
	}
}
