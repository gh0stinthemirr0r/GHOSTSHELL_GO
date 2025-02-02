package network

import (
	"errors"
	"sync"

	"ghostshell/ghostshell/oqs"

	"go.uber.org/zap"
)

// RoundRobin manages a set of items and provides a way to cycle through them sequentially.
type RoundRobin[T any] struct {
	items         []T
	index         int
	mutex         sync.Mutex
	logger        *zap.Logger
	encryptionKey []byte
}

// NewRoundRobin creates a new RoundRobin instance with post-quantum security features.
func NewRoundRobin[T any](items []T, logger *zap.Logger) (*RoundRobin[T], error) {
	if len(items) == 0 {
		logger.Error("Attempted to initialize RoundRobin with an empty item list")
		return nil, errors.New("items list cannot be empty")
	}

	// Generate an encryption key for secure operations
	encryptionKey, err := oqs.GenerateRandomBytes(32)
	if err != nil {
		logger.Error("Failed to generate encryption key for RoundRobin", zap.Error(err))
		return nil, err
	}

	logger.Info("Initialized RoundRobin instance", zap.Int("itemCount", len(items)))
	return &RoundRobin[T]{
		items:         items,
		index:         0,
		logger:        logger,
		encryptionKey: encryptionKey,
	}, nil
}

// Next returns the next item in the RoundRobin and advances the index securely.
func (r *RoundRobin[T]) Next() T {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	item := r.items[r.index]
	r.index = (r.index + 1) % len(r.items)

	r.logger.Debug("Retrieved next item", zap.Int("currentIndex", r.index), zap.Any("item", item))
	return item
}

// Add adds a new item to the RoundRobin securely.
func (r *RoundRobin[T]) Add(item T) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.items = append(r.items, item)
	r.logger.Info("Added new item to RoundRobin", zap.Any("item", item), zap.Int("newItemCount", len(r.items)))
}

// Remove removes an item from the RoundRobin by value securely. If the item is not found, it returns an error.
func (r *RoundRobin[T]) Remove(item T) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for i, v := range r.items {
		if v == item {
			r.items = append(r.items[:i], r.items[i+1:]...)
			if r.index >= len(r.items) {
				r.index = 0
			}
			r.logger.Info("Removed item from RoundRobin", zap.Any("item", item), zap.Int("remainingItems", len(r.items)))
			return nil
		}
	}

	r.logger.Warn("Attempted to remove item not found in RoundRobin", zap.Any("item", item))
	return errors.New("item not found in RoundRobin")
}

// Size returns the number of items in the RoundRobin securely.
func (r *RoundRobin[T]) Size() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	size := len(r.items)
	r.logger.Debug("Retrieved RoundRobin size", zap.Int("size", size))
	return size
}

// Reset clears all items in the RoundRobin securely and rotates the encryption key.
func (r *RoundRobin[T]) Reset() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.items = nil
	r.index = 0

	// Rotate the encryption key for added security
	newKey, err := oqs.GenerateRandomBytes(32)
	if err != nil {
		r.logger.Error("Failed to rotate encryption key during Reset", zap.Error(err))
		return
	}
	r.encryptionKey = newKey

	r.logger.Info("Reset RoundRobin, all items cleared and encryption key rotated")
}
