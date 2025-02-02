package ghostshell

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"

	// Importing the local post-quantum secure packages
	"ghostshell/ghostshell/oqs/sig"
)

// ExtensionsManager manages the loading and verification of trusted extensions.
type ExtensionsManager struct {
	logger           *zap.Logger
	mutex            sync.Mutex
	loadedExtensions []string
	publicKey        []byte // Should be initialized with the actual public key
	signature        []byte // Should be initialized with the actual signature
	signatureScheme  *sig.Scheme
}

// NewExtensionsManager initializes and returns a new instance of ExtensionsManager.
func NewExtensionsManager() (*ExtensionsManager, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize zap logger: %w", err)
	}

	logger.Info("Initializing Extensions Manager with post-quantum security.")

	// Initialize the signature scheme (Dilithium3)
	scheme, err := sig.NewScheme("Dilithium3")
	if err != nil {
		logger.Error("Failed to initialize signature scheme", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize signature scheme: %w", err)
	}

	// Placeholder public key and signature
	// In a real-world scenario, these should be loaded from a secure source
	publicKey := make([]byte, scheme.PublicKeyLength())
	signature := make([]byte, scheme.SignatureLength())

	// Initialize ExtensionsManager
	manager := &ExtensionsManager{
		logger:           logger,
		loadedExtensions: []string{},
		publicKey:        publicKey,
		signature:        signature,
		signatureScheme:  scheme,
	}

	return manager, nil
}

// Shutdown cleans up resources used by ExtensionsManager.
func (em *ExtensionsManager) Shutdown() {
	em.logger.Info("Cleaning up Extensions Manager.")
	// Any necessary cleanup can be performed here
	// For example, securely wiping loaded extensions if needed
	_ = em.logger.Sync()
}

// LoadTrustedExtensions loads and verifies extensions from the specified directory.
func (em *ExtensionsManager) LoadTrustedExtensions(extensionsDir string) error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	em.logger.Info("Loading trusted extensions", zap.String("directory", extensionsDir))

	// Check if the directory exists
	if _, err := os.Stat(extensionsDir); os.IsNotExist(err) {
		em.logger.Error("Extensions directory does not exist", zap.String("directory", extensionsDir))
		return fmt.Errorf("directory does not exist: %s", extensionsDir)
	}

	// Iterate over the directory entries
	files, err := ioutil.ReadDir(extensionsDir)
	if err != nil {
		em.logger.Error("Failed to read extensions directory", zap.Error(err))
		return fmt.Errorf("failed to read directory %s: %w", extensionsDir, err)
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".ext" {
			extensionPath := filepath.Join(extensionsDir, file.Name())
			em.logger.Info("Processing extension", zap.String("file", extensionPath))

			content, err := ioutil.ReadFile(extensionPath)
			if err != nil {
				em.logger.Error("Failed to read extension file", zap.String("file", extensionPath), zap.Error(err))
				continue
			}

			if em.verifyExtensionSignature(string(content)) {
				em.loadedExtensions = append(em.loadedExtensions, file.Name())
				em.logger.Info("Extension verified and loaded", zap.String("file", file.Name()))
			} else {
				em.logger.Warn("Extension failed verification", zap.String("file", file.Name()))
			}
		}
	}

	return nil
}

// verifyExtensionSignature verifies the signature of the extension content.
func (em *ExtensionsManager) verifyExtensionSignature(extensionContent string) bool {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	em.logger.Info("Verifying extension signature.")

	// Verify the signature using the signature scheme
	valid, err := em.signatureScheme.Verify(
		[]byte(extensionContent),
		em.signature,
		em.publicKey,
	)
	if err != nil {
		em.logger.Error("Signature verification failed", zap.Error(err))
		return false
	}

	if valid {
		em.logger.Info("Signature verified successfully.")
		return true
	}

	em.logger.Warn("Signature verification failed.")
	return false
}

// AddExtension adds and verifies a single extension from the specified path.
func (em *ExtensionsManager) AddExtension(extensionPath string) bool {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	em.logger.Info("Adding extension", zap.String("path", extensionPath))

	// Check if the file exists and has the correct extension
	if _, err := os.Stat(extensionPath); os.IsNotExist(err) || filepath.Ext(extensionPath) != ".ext" {
		em.logger.Error("Invalid extension file", zap.String("file", extensionPath))
		return false
	}

	// Read the extension file
	content, err := ioutil.ReadFile(extensionPath)
	if err != nil {
		em.logger.Error("Failed to read extension file", zap.String("file", extensionPath), zap.Error(err))
		return false
	}

	// Verify the signature
	if em.verifyExtensionSignature(string(content)) {
		filename := filepath.Base(extensionPath)
		em.loadedExtensions = append(em.loadedExtensions, filename)
		em.logger.Info("Extension successfully added", zap.String("file", filename))
		return true
	}

	em.logger.Warn("Failed to add extension. Signature verification failed.", zap.String("file", extensionPath))
	return false
}

// ListExtensions returns a slice of loaded extension filenames.
func (em *ExtensionsManager) ListExtensions() []string {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	em.logger.Info("Listing loaded extensions.")
	return append([]string(nil), em.loadedExtensions...) // Return a copy to prevent external modification
}
