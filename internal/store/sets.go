// Package store provides set storage implementation.
package store

import (
	"sync"
)

// SetStore defines the interface for set operations.
type SetStore interface {
	SAdd(key string, members ...string) int
	SRem(key string, members ...string) int
	SMembers(key string) []string
	SIsMember(key, member string) bool
	SCard(key string) int
	SInter(keys ...string) []string
	KeyType(key string) string
}

// MemorySetStore is a thread-safe in-memory implementation of SetStore.
type MemorySetStore struct {
	mu   sync.RWMutex
	data map[string]map[string]struct{}
}

// NewSetStore creates a new MemorySetStore.
func NewSetStore() *MemorySetStore {
	return &MemorySetStore{
		data: make(map[string]map[string]struct{}),
	}
}

// SAdd adds members to a set. Returns the count of new members added.
func (s *MemorySetStore) SAdd(key string, members ...string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data[key] == nil {
		s.data[key] = make(map[string]struct{})
	}

	added := 0
	for _, member := range members {
		if _, exists := s.data[key][member]; !exists {
			s.data[key][member] = struct{}{}
			added++
		}
	}
	return added
}

// SRem removes members from a set. Returns the count of members removed.
func (s *MemorySetStore) SRem(key string, members ...string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	set, exists := s.data[key]
	if !exists {
		return 0
	}

	removed := 0
	for _, member := range members {
		if _, exists := set[member]; exists {
			delete(set, member)
			removed++
		}
	}

	// Auto-delete empty sets (Redis behavior)
	if len(set) == 0 {
		delete(s.data, key)
	}

	return removed
}

// SMembers returns all members of a set.
func (s *MemorySetStore) SMembers(key string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	set, exists := s.data[key]
	if !exists {
		return []string{}
	}

	members := make([]string, 0, len(set))
	for member := range set {
		members = append(members, member)
	}
	return members
}

// SIsMember returns true if member exists in the set.
func (s *MemorySetStore) SIsMember(key, member string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	set, exists := s.data[key]
	if !exists {
		return false
	}

	_, exists = set[member]
	return exists
}

// SCard returns the cardinality (size) of the set.
func (s *MemorySetStore) SCard(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	set, exists := s.data[key]
	if !exists {
		return 0
	}
	return len(set)
}

// SInter returns the intersection of all given sets.
// Returns empty set if any key doesn't exist.
func (s *MemorySetStore) SInter(keys ...string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(keys) == 0 {
		return []string{}
	}

	// Check if any key doesn't exist - if so, intersection is empty
	for _, key := range keys {
		if _, exists := s.data[key]; !exists {
			return []string{}
		}
	}

	// Find the smallest set for efficient iteration
	smallestIdx := 0
	smallestSize := len(s.data[keys[0]])
	for i, key := range keys {
		if len(s.data[key]) < smallestSize {
			smallestIdx = i
			smallestSize = len(s.data[key])
		}
	}

	// Iterate through smallest set and check membership in all others
	result := make([]string, 0)
	for member := range s.data[keys[smallestIdx]] {
		inAll := true
		for i, key := range keys {
			if i == smallestIdx {
				continue
			}
			if _, exists := s.data[key][member]; !exists {
				inAll = false
				break
			}
		}
		if inAll {
			result = append(result, member)
		}
	}

	return result
}

// KeyType returns the type of the key ("set" or "none").
func (s *MemorySetStore) KeyType(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, exists := s.data[key]; exists {
		return "set"
	}
	return "none"
}
