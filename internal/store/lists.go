// Package store provides list data structure operations.
package store

import (
	"sync"
)

// ListStore defines the interface for list operations.
type ListStore interface {
	LPush(key string, values ...string) int
	RPush(key string, values ...string) int
	LPop(key string) (string, bool)
	RPop(key string) (string, bool)
	LRange(key string, start, stop int) []string
	LLen(key string) int
	KeyType(key string) string
}

// memoryListStore is a thread-safe in-memory implementation of ListStore.
type memoryListStore struct {
	mu   sync.RWMutex
	data map[string][]string
}

// NewListStore creates a new ListStore.
func NewListStore() ListStore {
	return &memoryListStore{
		data: make(map[string][]string),
	}
}

// LPush prepends one or more values to a list. Returns the new length of the list.
// Values are inserted at the head of the list, from left to right.
// So LPUSH mylist a b c will result in a list containing c, b, a.
func (s *memoryListStore) LPush(key string, values ...string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, exists := s.data[key]
	if !exists {
		list = make([]string, 0, len(values))
	}

	// Prepend values in reverse order so the rightmost value ends up at head
	// LPUSH key a b c should result in [c, b, a, ...existing...]
	newList := make([]string, len(values)+len(list))
	for i, v := range values {
		newList[len(values)-1-i] = v
	}
	copy(newList[len(values):], list)

	s.data[key] = newList
	return len(newList)
}

// RPush appends one or more values to a list. Returns the new length of the list.
func (s *memoryListStore) RPush(key string, values ...string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, exists := s.data[key]
	if !exists {
		list = make([]string, 0, len(values))
	}

	list = append(list, values...)
	s.data[key] = list
	return len(list)
}

// LPop removes and returns the first element of the list.
// Returns ("", false) if the list is empty or doesn't exist.
func (s *memoryListStore) LPop(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, exists := s.data[key]
	if !exists || len(list) == 0 {
		return "", false
	}

	value := list[0]
	list = list[1:]

	// Delete empty lists (Redis behavior)
	if len(list) == 0 {
		delete(s.data, key)
	} else {
		s.data[key] = list
	}

	return value, true
}

// RPop removes and returns the last element of the list.
// Returns ("", false) if the list is empty or doesn't exist.
func (s *memoryListStore) RPop(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, exists := s.data[key]
	if !exists || len(list) == 0 {
		return "", false
	}

	value := list[len(list)-1]
	list = list[:len(list)-1]

	// Delete empty lists (Redis behavior)
	if len(list) == 0 {
		delete(s.data, key)
	} else {
		s.data[key] = list
	}

	return value, true
}

// LRange returns the specified range of elements from the list.
// Start and stop are zero-based indices. Negative indices count from the end.
// The range is inclusive on both ends.
func (s *memoryListStore) LRange(key string, start, stop int) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list, exists := s.data[key]
	if !exists {
		return []string{}
	}

	length := len(list)

	// Convert negative indices to positive
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}

	// Clamp indices to valid range
	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}

	// Return empty if range is invalid
	if start > stop || start >= length {
		return []string{}
	}

	// Return a copy of the slice to prevent external modification
	result := make([]string, stop-start+1)
	copy(result, list[start:stop+1])
	return result
}

// LLen returns the length of the list. Returns 0 if the key doesn't exist.
func (s *memoryListStore) LLen(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list, exists := s.data[key]
	if !exists {
		return 0
	}
	return len(list)
}

// KeyType returns the type of the key. Returns "list" for list keys, "none" for non-existent keys.
func (s *memoryListStore) KeyType(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.data[key]
	if !exists {
		return "none"
	}
	return "list"
}
