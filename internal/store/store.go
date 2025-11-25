// Package store provides key-value storage with TTL support.
// This is a mock implementation for server development.
// Will be replaced by the real implementation from the store agent.
package store

import (
	"sync"
	"time"
)

// Store defines the interface for key-value storage.
type Store interface {
	Get(key string) (string, bool)
	Set(key string, value string)
	SetWithTTL(key string, value string, ttl time.Duration)
	Delete(key string) bool
	TTL(key string) (time.Duration, bool)
	Expire(key string, ttl time.Duration) bool
}

// entry represents a stored value with optional expiration.
type entry struct {
	value     string
	expiresAt time.Time
	hasTTL    bool
}

// MemoryStore is an in-memory implementation of Store.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]entry
}

// New creates a new MemoryStore.
func New() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]entry),
	}
}

// Get retrieves a value by key. Returns false if key doesn't exist or is expired.
func (s *MemoryStore) Get(key string) (string, bool) {
	s.mu.RLock()
	e, exists := s.data[key]
	s.mu.RUnlock()

	if !exists {
		return "", false
	}

	if e.hasTTL && time.Now().After(e.expiresAt) {
		s.Delete(key)
		return "", false
	}

	return e.value, true
}

// Set stores a value with no expiration.
func (s *MemoryStore) Set(key string, value string) {
	s.mu.Lock()
	s.data[key] = entry{value: value, hasTTL: false}
	s.mu.Unlock()
}

// SetWithTTL stores a value with a time-to-live duration.
func (s *MemoryStore) SetWithTTL(key string, value string, ttl time.Duration) {
	s.mu.Lock()
	s.data[key] = entry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
		hasTTL:    true,
	}
	s.mu.Unlock()
}

// Delete removes a key. Returns true if key existed.
func (s *MemoryStore) Delete(key string) bool {
	s.mu.Lock()
	_, exists := s.data[key]
	delete(s.data, key)
	s.mu.Unlock()
	return exists
}

// TTL returns the remaining time-to-live for a key.
// Returns (duration, true) if key exists and has TTL.
// Returns (-1, true) if key exists but has no TTL.
// Returns (0, false) if key doesn't exist.
func (s *MemoryStore) TTL(key string) (time.Duration, bool) {
	s.mu.RLock()
	e, exists := s.data[key]
	s.mu.RUnlock()

	if !exists {
		return 0, false
	}

	if e.hasTTL {
		if time.Now().After(e.expiresAt) {
			s.Delete(key)
			return 0, false
		}
		return time.Until(e.expiresAt), true
	}

	return -1, true // key exists but no TTL
}

// Expire sets a TTL on an existing key. Returns true if key exists.
func (s *MemoryStore) Expire(key string, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, exists := s.data[key]
	if !exists {
		return false
	}

	if e.hasTTL && time.Now().After(e.expiresAt) {
		delete(s.data, key)
		return false
	}

	s.data[key] = entry{
		value:     e.value,
		expiresAt: time.Now().Add(ttl),
		hasTTL:    true,
	}
	return true
}
