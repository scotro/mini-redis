// Package store provides a thread-safe in-memory key-value store with TTL support.
package store

import (
	"sync"
	"time"
)

// Store defines the interface for the key-value store.
type Store interface {
	Get(key string) (string, bool)
	Set(key string, value string)
	SetWithTTL(key string, value string, ttl time.Duration)
	Delete(key string) bool
	Keys() []string
	TTL(key string) (time.Duration, bool)
	Close()
}

// entry holds a value and its optional expiration time.
type entry struct {
	value     string
	expiresAt time.Time // zero value means no expiration
}

// isExpired returns true if the entry has expired.
func (e *entry) isExpired() bool {
	if e.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(e.expiresAt)
}

// memoryStore is a thread-safe in-memory implementation of Store.
type memoryStore struct {
	mu      sync.RWMutex
	data    map[string]*entry
	done    chan struct{}
	stopped bool
}

// New creates a new Store with background cleanup.
func New() Store {
	s := &memoryStore{
		data: make(map[string]*entry),
		done: make(chan struct{}),
	}
	go s.cleanupLoop()
	return s
}

// cleanupLoop periodically removes expired keys.
func (s *memoryStore) cleanupLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.removeExpired()
		}
	}
}

// removeExpired deletes all expired entries.
func (s *memoryStore) removeExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, e := range s.data {
		if e.isExpired() {
			delete(s.data, key)
		}
	}
}

// Get retrieves a value by key. Returns false if key doesn't exist or is expired.
func (s *memoryStore) Get(key string) (string, bool) {
	s.mu.RLock()
	e, exists := s.data[key]
	s.mu.RUnlock()

	if !exists {
		return "", false
	}

	if e.isExpired() {
		// Lazily delete expired key
		s.Delete(key)
		return "", false
	}

	return e.value, true
}

// Set stores a key-value pair with no expiration.
func (s *memoryStore) Set(key string, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = &entry{
		value: value,
	}
}

// SetWithTTL stores a key-value pair that expires after the given duration.
func (s *memoryStore) SetWithTTL(key string, value string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = &entry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a key from the store. Returns true if the key existed.
func (s *memoryStore) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.data[key]
	if exists {
		delete(s.data, key)
	}
	return exists
}

// Keys returns all non-expired keys in the store.
func (s *memoryStore) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.data))
	for key, e := range s.data {
		if !e.isExpired() {
			keys = append(keys, key)
		}
	}
	return keys
}

// TTL returns the remaining time-to-live for a key.
// Returns (0, false) if the key doesn't exist or has no TTL.
// Returns (duration, true) if the key has a TTL set.
func (s *memoryStore) TTL(key string) (time.Duration, bool) {
	s.mu.RLock()
	e, exists := s.data[key]
	s.mu.RUnlock()

	if !exists {
		return 0, false
	}

	if e.expiresAt.IsZero() {
		return 0, false
	}

	if e.isExpired() {
		return 0, false
	}

	remaining := time.Until(e.expiresAt)
	if remaining < 0 {
		return 0, false
	}

	return remaining, true
}

// Close stops the background cleanup goroutine.
func (s *memoryStore) Close() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	s.mu.Unlock()

	close(s.done)
}
