package ghost

import (
	"fmt"
	"sync"

	"go.uber.org/zap"

	// Importing the local post-quantum secure packages

	"ghostshell/ghostshell/oqs/sig"
)

// CryptoUtils provides cryptographic functionalities using post-quantum algorithms.
type CryptoUtils struct {
	logger *zap.Logger
	once   sync.Once
}

// NewCryptoUtils initializes and returns a new instance of CryptoUtils.
// It ensures that the OQS library is initialized only once.
func NewCryptoUtils() (*CryptoUtils, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize zap logger: %w", err)
	}

	// Initialize OQS library
	// Assuming oqs_sig.Init() corresponds to OQS_init()
	if err := sig.Init(); err != nil {
		logger.Error("Failed to initialize OQS library", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize OQS library: %w", err)
	}

	logger.Info("OQS library initialized successfully.")

	return &CryptoUtils{
		logger: logger,
	}, nil
}

// Shutdown cleans up the crypto context by destroying the OQS library.
func (c *CryptoUtils) Shutdown() {
	c.once.Do(func() {
		// Assuming oqs_sig.Destroy() corresponds to OQS_destroy()
		if err := sig.Destroy(); err != nil {
			c.logger.Error("Failed to destroy OQS library", zap.Error(err))
		} else {
			c.logger.Info("OQS library destroyed successfully.")
		}

		// Sync the logger to flush any pending logs
		_ = c.logger.Sync()
	})
}

// GenerateKeyPair generates a post-quantum key pair for the specified algorithm.
// It returns the public key, secret key, and an error if any.
func (c *CryptoUtils) GenerateKeyPair(alg string) ([]byte, []byte, error) {
	c.logger.Info("Generating key pair", zap.String("algorithm", alg))

	// Create a new signature scheme instance
	scheme, err := sig.NewScheme(alg)
	if err != nil {
		c.logger.Error("Unsupported signature algorithm", zap.String("algorithm", alg), zap.Error(err))
		return nil, nil, fmt.Errorf("unsupported signature algorithm: %s", alg)
	}
	defer scheme.Free()

	// Generate key pair
	publicKey, secretKey, err := scheme.Keypair()
	if err != nil {
		c.logger.Error("Failed to generate key pair", zap.String("algorithm", alg), zap.Error(err))
		return nil, nil, fmt.Errorf("failed to generate key pair for algorithm %s: %w", alg, err)
	}

	c.logger.Info("Key pair generated successfully", zap.String("algorithm", alg))
	return publicKey, secretKey, nil
}

// SignMessage signs a message using the provided secret key and algorithm.
// It returns the signature and an error if any.
func (c *CryptoUtils) SignMessage(message, secretKey []byte, alg string) ([]byte, error) {
	c.logger.Info("Signing message", zap.String("algorithm", alg))

	// Create a new signature scheme instance
	scheme, err := sig.NewScheme(alg)
	if err != nil {
		c.logger.Error("Unsupported signature algorithm", zap.String("algorithm", alg), zap.Error(err))
		return nil, fmt.Errorf("unsupported signature algorithm: %s", alg)
	}
	defer scheme.Free()

	// Sign the message
	signature, err := scheme.Sign(message, secretKey)
	if err != nil {
		c.logger.Error("Failed to sign message", zap.String("algorithm", alg), zap.Error(err))
		return nil, fmt.Errorf("failed to sign message with algorithm %s: %w", alg, err)
	}

	c.logger.Info("Message signed successfully", zap.String("algorithm", alg))
	return signature, nil
}

// VerifySignature verifies the signature of a message using the provided public key and algorithm.
// It returns true if the signature is valid, false otherwise, along with an error if any.
func (c *CryptoUtils) VerifySignature(message, signature, publicKey []byte, alg string) (bool, error) {
	c.logger.Info("Verifying signature", zap.String("algorithm", alg))

	// Create a new signature scheme instance
	scheme, err := sig.NewScheme(alg)
	if err != nil {
		c.logger.Error("Unsupported signature algorithm", zap.String("algorithm", alg), zap.Error(err))
		return false, fmt.Errorf("unsupported signature algorithm: %s", alg)
	}
	defer scheme.Free()

	// Verify the signature
	valid, err := scheme.Verify(message, signature, publicKey)
	if err != nil {
		c.logger.Error("Failed to verify signature", zap.String("algorithm", alg), zap.Error(err))
		return false, fmt.Errorf("failed to verify signature with algorithm %s: %w", alg, err)
	}

	if valid {
		c.logger.Info("Signature verification succeeded", zap.String("algorithm", alg))
	} else {
		c.logger.Warn("Signature verification failed", zap.String("algorithm", alg))
	}

	return valid, nil
}

// SecureWipe securely wipes the sensitive data from memory.
// It attempts to overwrite the data slice with zeros.
// Note: Due to Go's garbage collection, complete memory wiping cannot be guaranteed.
func (c *CryptoUtils) SecureWipe(data []byte) {
	c.logger.Info("Securely wiping sensitive data")

	if data == nil {
		c.logger.Warn("Attempted to wipe nil data slice")
		return
	}

	for i := range data {
		data[i] = 0
	}

	// Additional measures can be implemented if needed, such as using packages that prevent
	// the compiler from optimizing away the wiping operation.
}
