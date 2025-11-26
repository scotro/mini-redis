package transaction

import "sync"

// VersionTracker tracks version numbers for keys to support optimistic locking.
type VersionTracker interface {
	// GetVersion returns the current version of a key.
	// Returns 0 for keys that don't exist.
	GetVersion(key string) int64

	// IncrementVersion increments the version of a key.
	// Should be called whenever a key is modified.
	IncrementVersion(key string)

	// DeleteVersion removes version tracking for a key.
	// Called when a key is deleted.
	DeleteVersion(key string)
}

// MemoryVersionTracker is a simple in-memory implementation of VersionTracker.
// This is used for testing. The actual store can implement VersionTracker directly.
type MemoryVersionTracker struct {
	mu       sync.RWMutex
	versions map[string]int64
}

// NewMemoryVersionTracker creates a new in-memory version tracker.
func NewMemoryVersionTracker() *MemoryVersionTracker {
	return &MemoryVersionTracker{
		versions: make(map[string]int64),
	}
}

// GetVersion returns the current version of a key.
func (v *MemoryVersionTracker) GetVersion(key string) int64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.versions[key]
}

// IncrementVersion increments the version of a key.
func (v *MemoryVersionTracker) IncrementVersion(key string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.versions[key]++
}

// DeleteVersion removes version tracking for a key.
func (v *MemoryVersionTracker) DeleteVersion(key string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.versions, key)
}

// SetVersion sets the version of a key to a specific value (useful for testing).
func (v *MemoryVersionTracker) SetVersion(key string, version int64) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.versions[key] = version
}
