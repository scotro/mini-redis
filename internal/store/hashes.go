// Package store provides hash data type storage for Redis-like operations.
package store

import (
	"sync"
)

// HashStore defines the interface for hash operations.
type HashStore interface {
	HSet(key string, fieldValues ...string) int
	HGet(key, field string) (string, bool)
	HDel(key string, fields ...string) int
	HGetAll(key string) map[string]string
	HKeys(key string) []string
	HLen(key string) int
	KeyType(key string) string
}

// MemoryHashStore is a thread-safe in-memory implementation of HashStore.
type MemoryHashStore struct {
	mu     sync.RWMutex
	hashes map[string]map[string]string
}

// NewHashStore creates a new HashStore.
func NewHashStore() HashStore {
	return &MemoryHashStore{
		hashes: make(map[string]map[string]string),
	}
}

// HSet sets fields in the hash stored at key.
// Returns the number of NEW fields that were added (not updated).
func (s *MemoryHashStore) HSet(key string, fieldValues ...string) int {
	if len(fieldValues) < 2 || len(fieldValues)%2 != 0 {
		return 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	hash, exists := s.hashes[key]
	if !exists {
		hash = make(map[string]string)
		s.hashes[key] = hash
	}

	newFields := 0
	for i := 0; i < len(fieldValues); i += 2 {
		field := fieldValues[i]
		value := fieldValues[i+1]
		if _, fieldExists := hash[field]; !fieldExists {
			newFields++
		}
		hash[field] = value
	}

	return newFields
}

// HGet returns the value of field in the hash stored at key.
// Returns (value, true) if the field exists, ("", false) otherwise.
func (s *MemoryHashStore) HGet(key, field string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hash, exists := s.hashes[key]
	if !exists {
		return "", false
	}

	value, fieldExists := hash[field]
	return value, fieldExists
}

// HDel removes the specified fields from the hash stored at key.
// Returns the number of fields that were removed.
func (s *MemoryHashStore) HDel(key string, fields ...string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash, exists := s.hashes[key]
	if !exists {
		return 0
	}

	deleted := 0
	for _, field := range fields {
		if _, fieldExists := hash[field]; fieldExists {
			delete(hash, field)
			deleted++
		}
	}

	// Auto-delete empty hashes (Redis behavior)
	if len(hash) == 0 {
		delete(s.hashes, key)
	}

	return deleted
}

// HGetAll returns all fields and values of the hash stored at key.
// Returns an empty map if the key doesn't exist.
func (s *MemoryHashStore) HGetAll(key string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hash, exists := s.hashes[key]
	if !exists {
		return make(map[string]string)
	}

	// Return a copy to avoid data races
	result := make(map[string]string, len(hash))
	for field, value := range hash {
		result[field] = value
	}
	return result
}

// HKeys returns all field names in the hash stored at key.
// Returns an empty slice if the key doesn't exist.
func (s *MemoryHashStore) HKeys(key string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hash, exists := s.hashes[key]
	if !exists {
		return []string{}
	}

	keys := make([]string, 0, len(hash))
	for field := range hash {
		keys = append(keys, field)
	}
	return keys
}

// HLen returns the number of fields in the hash stored at key.
// Returns 0 if the key doesn't exist.
func (s *MemoryHashStore) HLen(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hash, exists := s.hashes[key]
	if !exists {
		return 0
	}
	return len(hash)
}

// KeyType returns the type of the key.
// Returns "hash" if the key exists in this store, "none" otherwise.
func (s *MemoryHashStore) KeyType(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, exists := s.hashes[key]; exists {
		return "hash"
	}
	return "none"
}
