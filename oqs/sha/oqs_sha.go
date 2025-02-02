// OQS-GO
// Aaron Stovall

package sha

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"hash"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/sha3"
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
	logger.Infof("Logger initialized with file: %s", logFileName1)
}

// Constants for hash sizes
const (
	SHA3_256Size = 32 // SHA3-256 produces a 32-byte hash
	SHA3_384Size = 48 // SHA3-384 produces a 48-byte hash
	SHA3_512Size = 64 // SHA3-512 produces a 64-byte hash
)

// SHAContext structure for SHA operations.
type SHAContext struct {
	Algorithm string
	State     hash.Hash
	DataLen   uint64
}

// Global variables for key management
var (
	hmacKey      []byte
	hmacKeyMutex sync.RWMutex // Ensures thread-safe access to hmacKey
)

// SHA3_256 computes the SHA3-256 hash of the input.
func SHA3_256Hash(output []byte, input []byte) {
	hash := sha3.Sum256(input)
	copy(output, hash[:])
	logger.Infof("SHA3-256: Computed %d-byte hash for %d bytes of input", SHA3_256Size, len(input))
}

// SHA3_384 computes the SHA3-384 hash of the input.
func SHA3_384Hash(output []byte, input []byte) {
	hash := sha3.Sum384(input)
	copy(output, hash[:])
	logger.Infof("SHA3-384: Computed %d-byte hash for %d bytes of input", SHA3_384Size, len(input))
}

// SHA3_512 computes the SHA3-512 hash of the input.
func SHA3_512Hash(output []byte, input []byte) {
	hash := sha3.Sum512(input)
	copy(output, hash[:])
	logger.Infof("SHA3-512: Computed %d-byte hash for %d bytes of input", SHA3_512Size, len(input))
}

// HMACSHA3_256 computes the HMAC for SHA3-256.
func HMACSHA3_256(key []byte, data []byte, output []byte) error {
	if len(output) < SHA3_256Size {
		return errors.New("output buffer is too small")
	}
	mac := hmac.New(sha3.New256, key)
	if _, err := mac.Write(data); err != nil {
		logger.Errorf("HMACSHA3_256 write failed: %v", err)
		return errors.New("HMAC generation failed")
	}
	copy(output, mac.Sum(nil))
	logger.Infof("HMAC-SHA3-256: Computed %d-byte HMAC", SHA3_256Size)
	return nil
}

// HMACSHA3_512 computes the HMAC for SHA3-512.
func HMACSHA3_512(key []byte, data []byte, output []byte) error {
	if len(output) < SHA3_512Size {
		return errors.New("output buffer is too small")
	}
	mac := hmac.New(sha3.New512, key)
	if _, err := mac.Write(data); err != nil {
		logger.Errorf("HMACSHA3_512 write failed: %v", err)
		return errors.New("HMAC generation failed")
	}
	copy(output, mac.Sum(nil))
	logger.Infof("HMAC-SHA3-512: Computed %d-byte HMAC", SHA3_512Size)
	return nil
}

// SHA224 computes the SHA-224 hash of the input.
func SHA224Hash(output []byte, input []byte) {
	hash := sha256.Sum224(input)
	copy(output, hash[:])
	logger.Infof("SHA-224: Computed %d-byte hash for %d bytes of input", sha256.Size224, len(input))
}

// SHA384 computes the SHA-384 hash of the input.
func SHA384Hash(output []byte, input []byte) {
	hash := sha512.Sum384(input)
	copy(output, hash[:])
	logger.Infof("SHA-384: Computed %d-byte hash for %d bytes of input", sha512.Size384, len(input))
}

// SHAContextInitializer initializes the hash context based on the algorithm.
func SHAContextInitializer(ctx *SHAContext, algorithm string) error {
	switch algorithm {
	case "SHA-256":
		ctx.State = sha256.New()
	case "SHA-224":
		ctx.State = sha256.New224()
	case "SHA-512":
		ctx.State = sha512.New()
	case "SHA-384":
		ctx.State = sha512.New384()
	case "SHA3-256":
		ctx.State = sha3.New256()
	case "SHA3-512":
		ctx.State = sha3.New512()
	default:
		return fmt.Errorf("unsupported algorithm: %s", algorithm)
	}
	ctx.Algorithm = algorithm
	ctx.DataLen = 0
	logger.Infof("%s context initialized", algorithm)
	return nil
}

// HashValidator compares two hashes for equality.
func HashValidator(hash1, hash2 []byte) bool {
	isEqual := hmac.Equal(hash1, hash2)
	if isEqual {
		logger.Infof("Hash validation successful")
	} else {
		logger.Warnf("Hash validation failed")
	}
	return isEqual
}

// RollingHash computes a rolling hash for data streams or sliding windows.
func RollingHash(data []byte, windowSize int) ([]uint64, error) {
	if windowSize <= 0 || len(data) < windowSize {
		return nil, errors.New("invalid window size")
	}

	hashes := make([]uint64, len(data)-windowSize+1)
	var hash uint64
	const base uint64 = 257
	const mod uint64 = 1e9 + 7

	// Compute the hash for the first window
	for i := 0; i < windowSize; i++ {
		hash = (hash*base + uint64(data[i])) % mod
	}
	hashes[0] = hash

	// Compute rolling hashes
	power := uint64(1)
	for i := 0; i < windowSize-1; i++ {
		power = (power * base) % mod
	}

	for i := 1; i < len(hashes); i++ {
		hash = (hash*base - uint64(data[i-1])*power + uint64(data[i+windowSize-1])) % mod
		if hash < 0 {
			hash += mod
		}
		hashes[i] = hash
	}

	logger.Infof("Rolling hashes computed for window size %d", windowSize)
	return hashes, nil
}
