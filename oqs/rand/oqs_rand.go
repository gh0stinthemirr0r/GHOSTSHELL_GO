// OQS-GO
// Aaron Stovall

package rand

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"sync"

	"ghostshell/logger" // Import the centralized logger
)

// OQS_STATUS defines the status codes for operations.
type OQS_STATUS int

const (
	OQS_SUCCESS OQS_STATUS = iota
	OQS_ERROR
)

// Error definitions
var (
	ErrNilAlgorithmFunc         = errors.New("random algorithm function cannot be nil")
	ErrUnsupportedAlgorithm     = errors.New("unsupported random algorithm")
	ErrAlgorithmNotSelected     = errors.New("random algorithm not selected")
	ErrInvalidBuffer            = errors.New("buffer is empty")
	ErrInvalidCustomAlgorithm   = errors.New("invalid custom RNG function")
	ErrAlgorithmSwitchFailed    = errors.New("failed to switch random algorithm")
	ErrRandomBytesGeneration    = errors.New("failed to generate random bytes")
	ErrKeyManagerNotInitialized = errors.New("key manager not initialized")
	ErrEntropyValidationFailed  = errors.New("entropy validation failed")
	ErrKeyNotFound              = errors.New("key not found")
)

// RandomAlgorithm is a function type for custom random number generators.
type RandomAlgorithm func([]byte) (int, error)

// KeyManager defines the interface for key management operations.
type KeyManager interface {
	StoreKey(key []byte) error
	LoadKey() ([]byte, error)
}

// DummyKeyManager is a placeholder implementation for KeyManager.
// Replace with actual implementation interfacing with HSM or cloud KMS.a// OQS-GO
// Aaron Stovall

package rand

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sync"

	"ghostshell/ghostshell/logger" // Import the centralized logger
)

// OQS_STATUS defines the status codes for operations.
type OQS_STATUS int

const (
	OQS_SUCCESS OQS_STATUS = iota
	OQS_ERROR
)

// Error definitions
var (
	ErrNilAlgorithmFunc         = errors.New("random algorithm function cannot be nil")
	ErrUnsupportedAlgorithm     = errors.New("unsupported random algorithm")
	ErrAlgorithmNotSelected     = errors.New("random algorithm not selected")
	ErrInvalidBuffer            = errors.New("buffer is empty")
	ErrInvalidCustomAlgorithm   = errors.New("invalid custom RNG function")
	ErrAlgorithmSwitchFailed    = errors.New("failed to switch random algorithm")
	ErrRandomBytesGeneration    = errors.New("failed to generate random bytes")
	ErrKeyManagerNotInitialized = errors.New("key manager not initialized")
	ErrEntropyValidationFailed  = errors.New("entropy validation failed")
	ErrKeyNotFound              = errors.New("key not found")
)

// RandomAlgorithm is a function type for custom random number generators.
type RandomAlgorithm func([]byte) (int, error)

// KeyManager defines the interface for key management operations.
type KeyManager interface {
	StoreKey(key []byte) error
	LoadKey() ([]byte, error)
}

// DummyKeyManager is a placeholder implementation for KeyManager.
// Replace with actual implementation interfacing with HSM or cloud KMS.
type DummyKeyManager struct{}

func (dkm *DummyKeyManager) StoreKey(key []byte) error {
	logger.Logger.Infof("Storing key securely (DummyKeyManager)")
	return nil
}

func (dkm *DummyKeyManager) LoadKey() ([]byte, error) {
	logger.Logger.Infof("Loading key securely (DummyKeyManager)")
	return nil, errors.New("key not found")
}

// Global variables for key management and current random algorithm
var (
	keyManager       KeyManager      = &DummyKeyManager{}
	currentAlgorithm RandomAlgorithm = defaultRandomAlgorithm
	algorithmMutex   sync.RWMutex    // Ensures thread-safe access to currentAlgorithm
	entropyPool      []byte          // Entropy pool for randomness
	poolMutex        sync.Mutex      // Mutex for entropy pool
)

// defaultRandomAlgorithm uses the crypto/rand package to generate random bytes.
func defaultRandomAlgorithm(b []byte) (int, error) {
	n, err := rand.Read(b)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// SwitchAlgorithm switches the random number generator to the specified algorithm.
func SwitchAlgorithm(algorithm string, customFunc ...RandomAlgorithm) OQS_STATUS {
	algorithmMutex.Lock()
	defer algorithmMutex.Unlock()

	switch algorithm {
	case OQS_RAND_alg_system:
		currentAlgorithm = defaultRandomAlgorithm
		logger.Logger.Infof("Switched to system RNG")
		return OQS_SUCCESS
	case OQS_RAND_alg_custom:
		if len(customFunc) == 0 || customFunc[0] == nil {
			logger.Logger.Errorf("Custom RNG function not provided or is nil")
			return OQS_ERROR
		}
		currentAlgorithm = customFunc[0]
		logger.Logger.Infof("Switched to custom RNG")
		return OQS_SUCCESS
	default:
		logger.Logger.Errorf("Unsupported RNG algorithm: %s", algorithm)
		return OQS_ERROR
	}
}

// RandomBytes fills the given slice with random bytes.
func RandomBytes(randomArray []byte) error {
	if len(randomArray) == 0 {
		return ErrInvalidBuffer
	}

	algorithmMutex.RLock()
	alg := currentAlgorithm
	algorithmMutex.RUnlock()

	if alg == nil {
		return ErrAlgorithmNotSelected
	}

	_, err := alg(randomArray)
	if err != nil {
		logger.Logger.Errorf("RandomBytes failed: %v", err)
		return ErrRandomBytesGeneration
	}

	logger.Logger.Infof("RandomBytes generated successfully, bytes: %d", len(randomArray))
	return nil
}

// AddEntropy adds data to the entropy pool.
func AddEntropy(data []byte) {
	poolMutex.Lock()
	entropyPool = append(entropyPool, data...)
	poolMutex.Unlock()
	logger.Logger.Infof("Added %d bytes to the entropy pool", len(data))
}

// GetEntropy retrieves random bytes from the entropy pool.
func GetEntropy(size int) ([]byte, error) {
	poolMutex.Lock()
	defer poolMutex.Unlock()

	if len(entropyPool) < size {
		logger.Logger.Errorf("Insufficient entropy in pool. Requested: %d, Available: %d", size, len(entropyPool))
		return nil, ErrKeyNotFound
	}

	entropy := entropyPool[:size]
	entropyPool = entropyPool[size:]
	logger.Logger.Infof("Retrieved %d bytes from the entropy pool", size)
	return entropy, nil
}

// ValidateEntropy computes the Shannon entropy of the data.
func ValidateEntropy(data []byte) (float64, error) {
	if len(data) == 0 {
		return 0, ErrInvalidBuffer
	}

	byteCounts := make(map[byte]int)
	for _, b := range data {
		byteCounts[b]++
	}

	entropy := 0.0
	total := float64(len(data))
	for _, count := range byteCounts {
		p := float64(count) / total
		entropy -= p * math.Log2(p)
	}

	logger.Logger.Infof("Computed entropy: %.2f", entropy)
	if entropy < 7.5 { // Threshold for high-quality entropy
		logger.Logger.Warnf("Entropy validation failed: %.2f", entropy)
		return entropy, ErrEntropyValidationFailed
	}

	logger.Logger.Infof("Entropy validation passed: %.2f", entropy)
	return entropy, nil
}

// QuantumSafeRandomBytes generates random bytes using quantum-safe primitives.
func QuantumSafeRandomBytes(data []byte) error {
	if len(data) == 0 {
		return ErrInvalidBuffer
	}

	if _, err := rand.Read(data); err != nil {
		logger.Logger.Errorf("Failed to generate quantum-safe random bytes: %v", err)
		return fmt.Errorf("failed to generate quantum-safe random bytes: %w", err)
	}

	logger.Logger.Infof("Generated %d quantum-safe random bytes", len(data))
	return nil
}

// ExampleUsage demonstrates random byte generation, entropy pooling, and validation.
func ExampleUsage() {
	// Generate random bytes
	randomBytes := make([]byte, 32)
	if err := RandomBytes(randomBytes); err != nil {
		logger.Logger.Fatalf("Random byte generation failed: %v", err)
	}

	// Add to entropy pool
	AddEntropy(randomBytes)

	// Retrieve from entropy pool
	entropy, err := GetEntropy(16)
	if err != nil {
		logger.Logger.Fatalf("Failed to retrieve entropy: %v", err)
	}
	logger.Logger.Infof("Retrieved entropy: %x", entropy)

	// Validate entropy
	if _, err := ValidateEntropy(randomBytes); err != nil {
		logger.Logger.Fatalf("Entropy validation failed: %v", err)
	}

	// Quantum-safe randomness
	quantumBytes := make([]byte, 32)
	if err := QuantumSafeRandomBytes(quantumBytes); err != nil {
		logger.Logger.Fatalf("Quantum-safe randomness generation failed: %v", err)
	}
	logger.Logger.Infof("Generated quantum-safe random bytes: %x", quantumBytes)
}

// Constants for random algorithms.
const (
	OQS_RAND_alg_system = "system"
	OQS_RAND_alg_custom = "custom"
)

type DummyKeyManager struct{}

func (dkm *DummyKeyManager) StoreKey(key []byte) error {
	logger.Logger.Infof("Storing key securely (DummyKeyManager)")
	return nil
}

func (dkm *DummyKeyManager) LoadKey() ([]byte, error) {
	logger.Logger.Infof("Loading key securely (DummyKeyManager)")
	return nil, errors.New("key not found")
}

// Global variables for key management and current random algorithm
var (
	keyManager       KeyManager      = &DummyKeyManager{}
	currentAlgorithm RandomAlgorithm = defaultRandomAlgorithm
	algorithmMutex   sync.RWMutex    // Ensures thread-safe access to currentAlgorithm
	entropyPool      []byte          // Entropy pool for randomness
	poolMutex        sync.Mutex      // Mutex for entropy pool
)

// defaultRandomAlgorithm uses the crypto/rand package to generate random bytes.
func defaultRandomAlgorithm(b []byte) (int, error) {
	n, err := rand.Read(b)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// SwitchAlgorithm switches the random number generator to the specified algorithm.
func SwitchAlgorithm(algorithm string, customFunc ...RandomAlgorithm) OQS_STATUS {
	algorithmMutex.Lock()
	defer algorithmMutex.Unlock()

	switch algorithm {
	case OQS_RAND_alg_system:
		currentAlgorithm = defaultRandomAlgorithm
		logger.Logger.Infof("Switched to system RNG")
		return OQS_SUCCESS
	case OQS_RAND_alg_custom:
		if len(customFunc) == 0 || customFunc[0] == nil {
			logger.Logger.Errorf("Custom RNG function not provided or is nil")
			return OQS_ERROR
		}
		currentAlgorithm = customFunc[0]
		logger.Logger.Infof("Switched to custom RNG")
		return OQS_SUCCESS
	default:
		logger.Logger.Errorf("Unsupported RNG algorithm: %s", algorithm)
		return OQS_ERROR
	}
}

// RandomBytes fills the given slice with random bytes.
func RandomBytes(randomArray []byte) error {
	if len(randomArray) == 0 {
		return ErrInvalidBuffer
	}

	algorithmMutex.RLock()
	alg := currentAlgorithm
	algorithmMutex.RUnlock()

	if alg == nil {
		return ErrAlgorithmNotSelected
	}

	_, err := alg(randomArray)
	if err != nil {
		logger.Logger.Errorf("RandomBytes failed: %v", err)
		return ErrRandomBytesGeneration
	}

	logger.Logger.Infof("RandomBytes generated successfully, bytes: %d", len(randomArray))
	return nil
}

// AddEntropy adds data to the entropy pool.
func AddEntropy(data []byte) {
	poolMutex.Lock()
	entropyPool = append(entropyPool, data...)
	poolMutex.Unlock()
	logger.Logger.Infof("Added %d bytes to the entropy pool", len(data))
}

// GetEntropy retrieves random bytes from the entropy pool.
func GetEntropy(size int) ([]byte, error) {
	poolMutex.Lock()
	defer poolMutex.Unlock()

	if len(entropyPool) < size {
		return nil, ErrKeyNotFound
	}

	entropy := entropyPool[:size]
	entropyPool = entropyPool[size:]
	logger.Logger.Infof("Retrieved %d bytes from the entropy pool", size)
	return entropy, nil
}

// ValidateEntropy computes the Shannon entropy of the data.
func ValidateEntropy(data []byte) (float64, error) {
	if len(data) == 0 {
		return 0, ErrInvalidBuffer
	}

	byteCounts := make(map[byte]int)
	for _, b := range data {
		byteCounts[b]++
	}

	entropy := 0.0
	total := float64(len(data))
	for _, count := range byteCounts {
		p := float64(count) / total
		entropy -= p * math.Log2(p)
	}

	logger.Logger.Infof("Computed entropy: %.2f", entropy)
	if entropy < 7.5 { // Threshold for high-quality entropy
		logger.Logger.Warnf("Entropy validation failed: %.2f", entropy)
		return entropy, ErrEntropyValidationFailed
	}

	logger.Logger.Infof("Entropy validation passed: %.2f", entropy)
	return entropy, nil
}

// QuantumSafeRandomBytes generates random bytes using quantum-safe primitives.
func QuantumSafeRandomBytes(data []byte) error {
	if len(data) == 0 {
		return ErrInvalidBuffer
	}

	if _, err := rand.Read(data); err != nil {
		logger.Logger.Errorf("Failed to generate quantum-safe random bytes: %v", err)
		return fmt.Errorf("failed to generate quantum-safe random bytes: %w", err)
	}

	logger.Logger.Infof("Generated %d quantum-safe random bytes", len(data))
	return nil
}

// ExampleUsage demonstrates random byte generation, entropy pooling, and validation.
func ExampleUsage() {
	// Generate random bytes
	randomBytes := make([]byte, 32)
	if err := RandomBytes(randomBytes); err != nil {
		logger.Logger.Fatalf("Random byte generation failed: %v", err)
	}

	// Add to entropy pool
	AddEntropy(randomBytes)

	// Retrieve from entropy pool
	entropy, err := GetEntropy(16)
	if err != nil {
		logger.Logger.Fatalf("Failed to retrieve entropy: %v", err)
	}
	logger.Logger.Infof("Retrieved entropy: %x", entropy)

	// Validate entropy
	if _, err := ValidateEntropy(randomBytes); err != nil {
		logger.Logger.Fatalf("Entropy validation failed: %v", err)
	}

	// Quantum-safe randomness
	quantumBytes := make([]byte, 32)
	if err := QuantumSafeRandomBytes(quantumBytes); err != nil {
		logger.Logger.Fatalf("Quantum-safe randomness generation failed: %v", err)
	}
	logger.Logger.Infof("Generated quantum-safe random bytes: %x", quantumBytes)
}

// Constants for random algorithms.
const (
	OQS_RAND_alg_system = "system"
	OQS_RAND_alg_custom = "custom"
)
