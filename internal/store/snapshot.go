// Package store provides snapshot interfaces for persistence.
package store

import (
	"errors"
	"time"
)

// ErrInvalidSnapshotData is returned when snapshot data has an invalid type.
var ErrInvalidSnapshotData = errors.New("invalid snapshot data type")

// Snapshottable defines the interface for stores that can export/import their data.
type Snapshottable interface {
	ExportData() interface{}
	ImportData(data interface{}) error
}

// StringEntry represents a string entry for export/import.
type StringEntry struct {
	Value     string
	ExpiresAt int64 // Unix timestamp, 0 means no expiration
}

// StringSnapshot represents exported string store data.
type StringSnapshot struct {
	Data map[string]StringEntry
}

// ListSnapshot represents exported list store data.
type ListSnapshot struct {
	Data map[string][]string
}

// HashSnapshot represents exported hash store data.
type HashSnapshot struct {
	Data map[string]map[string]string
}

// SetSnapshot represents exported set store data.
type SetSnapshot struct {
	Data map[string][]string
}

// ExportData exports all string data for snapshotting.
func (s *memoryStore) ExportData() interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := StringSnapshot{
		Data: make(map[string]StringEntry, len(s.data)),
	}

	for key, e := range s.data {
		// Skip expired entries
		if e.isExpired() {
			continue
		}

		entry := StringEntry{
			Value: e.value,
		}
		if !e.expiresAt.IsZero() {
			entry.ExpiresAt = e.expiresAt.Unix()
		}
		snapshot.Data[key] = entry
	}

	return snapshot
}

// ImportData imports string data from a snapshot.
func (s *memoryStore) ImportData(data interface{}) error {
	snapshot, ok := data.(StringSnapshot)
	if !ok {
		return ErrInvalidSnapshotData
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, e := range snapshot.Data {
		entry := &entry{
			value: e.Value,
		}
		if e.ExpiresAt > 0 {
			expiresAt := time.Unix(e.ExpiresAt, 0)
			// Skip already expired entries
			if expiresAt.Before(now) {
				continue
			}
			entry.expiresAt = expiresAt
		}
		s.data[key] = entry
	}

	return nil
}

// ExportData exports all list data for snapshotting.
func (s *memoryListStore) ExportData() interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := ListSnapshot{
		Data: make(map[string][]string, len(s.data)),
	}

	for key, list := range s.data {
		// Make a copy of the list
		listCopy := make([]string, len(list))
		copy(listCopy, list)
		snapshot.Data[key] = listCopy
	}

	return snapshot
}

// ImportData imports list data from a snapshot.
func (s *memoryListStore) ImportData(data interface{}) error {
	snapshot, ok := data.(ListSnapshot)
	if !ok {
		return ErrInvalidSnapshotData
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for key, list := range snapshot.Data {
		listCopy := make([]string, len(list))
		copy(listCopy, list)
		s.data[key] = listCopy
	}

	return nil
}

// ExportData exports all hash data for snapshotting.
func (s *MemoryHashStore) ExportData() interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := HashSnapshot{
		Data: make(map[string]map[string]string, len(s.hashes)),
	}

	for key, hash := range s.hashes {
		hashCopy := make(map[string]string, len(hash))
		for field, value := range hash {
			hashCopy[field] = value
		}
		snapshot.Data[key] = hashCopy
	}

	return snapshot
}

// ImportData imports hash data from a snapshot.
func (s *MemoryHashStore) ImportData(data interface{}) error {
	snapshot, ok := data.(HashSnapshot)
	if !ok {
		return ErrInvalidSnapshotData
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for key, hash := range snapshot.Data {
		hashCopy := make(map[string]string, len(hash))
		for field, value := range hash {
			hashCopy[field] = value
		}
		s.hashes[key] = hashCopy
	}

	return nil
}

// ExportData exports all set data for snapshotting.
func (s *MemorySetStore) ExportData() interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := SetSnapshot{
		Data: make(map[string][]string, len(s.data)),
	}

	for key, set := range s.data {
		members := make([]string, 0, len(set))
		for member := range set {
			members = append(members, member)
		}
		snapshot.Data[key] = members
	}

	return snapshot
}

// ImportData imports set data from a snapshot.
func (s *MemorySetStore) ImportData(data interface{}) error {
	snapshot, ok := data.(SetSnapshot)
	if !ok {
		return ErrInvalidSnapshotData
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for key, members := range snapshot.Data {
		set := make(map[string]struct{}, len(members))
		for _, member := range members {
			set[member] = struct{}{}
		}
		s.data[key] = set
	}

	return nil
}

// AsSnapshottable type asserts any store to Snapshottable.
// Returns nil if the store doesn't implement Snapshottable.
func AsSnapshottable(s interface{}) Snapshottable {
	if snap, ok := s.(Snapshottable); ok {
		return snap
	}
	return nil
}
