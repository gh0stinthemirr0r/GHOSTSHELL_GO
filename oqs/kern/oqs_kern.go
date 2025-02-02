// OQS-GO
// Aaron Stovall

package kern

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Initialize logger with dynamic file naming
var logger *zap.SugaredLogger

func init() {
	currentTime := time.Now().UTC().Format("20060102_150405")
	logFileName1 := fmt.Sprintf("postquantumsecurity_log_%s.log", currentTime)

	// Configure zap to write to both log files and stdout
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{
		logFileName1,
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
	logger.Infof("Logger initialized with files: %s and %s", logFileName1)
}

func init() {
	currentTime := time.Now().UTC().Format("20060102_150405")
	logFileName := fmt.Sprintf("postquantumsecurity_log_%s.log", currentTime)
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{
		logFileName,
		"stdout",
	}
	log, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	logger = log.Sugar()
	logger.Infof("Logger initialized with file: %s", logFileName)
}

func init() {
	configureLogger()
	keyManager = &DummyKeyManager{}
}

// Configure zap logger
func configureLogger() {
	timeSuffix := time.Now().UTC().Format("2006-01-02_15-04-05")
	logFilename := fmt.Sprintf("postquantumsecurity_log_%s.log", timeSuffix)
	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{
		logFilename,
		"stdout",
	}
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	var err error
	logger, err = cfg.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize zap logger: %v", err))
	}
	logger.Info("Logger initialized", zap.String("filename", logFilename))
}

// Error definitions
var (
	ErrNilCallbacks             = errors.New("invalid AES callbacks: nil")
	ErrAESCallbacksNotSet       = errors.New("AES callbacks not set")
	ErrInvalidAESKeyLength      = errors.New("invalid AES key length")
	ErrInvalidAESNonceLength    = errors.New("invalid nonce length for AES-GCM")
	ErrSignatureVerification    = errors.New("signature verification failed")
	ErrKeyGenerationFailed      = errors.New("failed to generate new key")
	ErrInvalidSignatureFormat   = errors.New("invalid signature format")
	ErrPrivateKeyNil            = errors.New("private key is nil")
	ErrPublicKeyNil             = errors.New("public key is nil")
	ErrKeyManagerNotInitialized = errors.New("key manager not initialized")
	ErrInvalidKEMInitialization = errors.New("KEM structure is not properly initialized")
	ErrMissingPublicKey         = errors.New("public key is missing")
	ErrMissingCiphertext        = errors.New("missing ciphertext")
	ErrMissingSecretKey         = errors.New("missing secret key")
	ErrInvalidKeySize           = errors.New("invalid key size")
)

// Constants for key lengths and sizes
const (
	LengthPublicKey348864    = 261120
	LengthSecretKey348864    = 6492
	LengthCiphertext348864   = 96
	LengthSharedSecret348864 = 32
)

// AESContext represents the AES encryption/decryption context.
type AESContext struct {
	Schedule []byte
}

// OQSAESCallbacks defines the callback API for AES operations.
type OQSAESCallbacks struct {
	AESGCMEncrypt func(plaintext []byte, key []byte, nonce []byte, additionalData []byte) ([]byte, error)
	AESGCMDecrypt func(ciphertext []byte, key []byte, nonce []byte, additionalData []byte) ([]byte, error)
}

// Global variables for AES callbacks
var (
	currentAESCallbacks *OQSAESCallbacks
	callbacksMutex      sync.RWMutex
)

// SetAESCallbacks sets the callback functions for AES operations.
func SetAESCallbacks(newCallbacks *OQSAESCallbacks) error {
	if newCallbacks == nil {
		return ErrNilCallbacks
	}
	callbacksMutex.Lock()
	defer callbacksMutex.Unlock()
	currentAESCallbacks = newCallbacks
	logger.Info("AES callbacks have been set")
	return nil
}

// GetAESCallbacks retrieves the current AES callbacks.
func GetAESCallbacks() (*OQSAESCallbacks, error) {
	callbacksMutex.RLock()
	defer callbacksMutex.RUnlock()
	if currentAESCallbacks == nil {
		return nil, ErrAESCallbacksNotSet
	}
	return currentAESCallbacks, nil
}
