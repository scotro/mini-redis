// Package persistence provides RDB-style snapshot persistence for mini-redis.
package persistence

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/scotro/mini-redis/internal/store"
)

// Snapshot represents the complete state of all stores at a point in time.
type Snapshot struct {
	Strings store.StringSnapshot
	Lists   store.ListSnapshot
	Hashes  store.HashSnapshot
	Sets    store.SetSnapshot
}

// Stores holds references to all the stores that can be snapshotted.
type Stores struct {
	Strings store.Snapshottable
	Lists   store.Snapshottable
	Hashes  store.Snapshottable
	Sets    store.Snapshottable
}

// Manager handles snapshot operations for mini-redis.
type Manager struct {
	mu       sync.Mutex
	path     string
	stores   Stores
	saving   bool
	saveDone chan error
}

// NewManager creates a new persistence manager.
func NewManager(path string, stores Stores) *Manager {
	return &Manager{
		path:   path,
		stores: stores,
	}
}

// Path returns the snapshot file path.
func (m *Manager) Path() string {
	return m.path
}

// Save synchronously saves all stores to disk.
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.saveInternal()
}

// saveInternal performs the actual save operation (must hold mutex).
func (m *Manager) saveInternal() error {
	snapshot := m.createSnapshot()
	return m.writeSnapshot(snapshot)
}

// createSnapshot gathers data from all stores.
func (m *Manager) createSnapshot() *Snapshot {
	snapshot := &Snapshot{}

	if m.stores.Strings != nil {
		if data, ok := m.stores.Strings.ExportData().(store.StringSnapshot); ok {
			snapshot.Strings = data
		}
	}

	if m.stores.Lists != nil {
		if data, ok := m.stores.Lists.ExportData().(store.ListSnapshot); ok {
			snapshot.Lists = data
		}
	}

	if m.stores.Hashes != nil {
		if data, ok := m.stores.Hashes.ExportData().(store.HashSnapshot); ok {
			snapshot.Hashes = data
		}
	}

	if m.stores.Sets != nil {
		if data, ok := m.stores.Sets.ExportData().(store.SetSnapshot); ok {
			snapshot.Sets = data
		}
	}

	return snapshot
}

// writeSnapshot writes the snapshot to disk atomically.
func (m *Manager) writeSnapshot(snapshot *Snapshot) error {
	// Write to temp file first for atomicity
	tmpPath := m.path + ".tmp"

	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(snapshot); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to encode snapshot: %w", err)
	}

	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to sync file: %w", err)
	}

	if err := file.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close file: %w", err)
	}

	// Atomically rename temp file to final path
	if err := os.Rename(tmpPath, m.path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// BackgroundSave starts a background save operation.
// Returns immediately. Use WaitForSave() to wait for completion.
func (m *Manager) BackgroundSave() error {
	m.mu.Lock()
	if m.saving {
		m.mu.Unlock()
		return ErrSaveInProgress
	}
	m.saving = true
	m.saveDone = make(chan error, 1)
	m.mu.Unlock()

	// Create snapshot while holding no locks
	// Note: Each store's ExportData is already thread-safe
	snapshot := m.createSnapshot()

	// Write to disk in goroutine
	go func() {
		m.mu.Lock()
		defer func() {
			m.saving = false
			m.mu.Unlock()
		}()

		err := m.writeSnapshot(snapshot)
		m.saveDone <- err
	}()

	return nil
}

// WaitForSave waits for a background save to complete and returns its result.
func (m *Manager) WaitForSave() error {
	m.mu.Lock()
	done := m.saveDone
	m.mu.Unlock()

	if done == nil {
		return nil
	}

	return <-done
}

// IsSaving returns true if a background save is in progress.
func (m *Manager) IsSaving() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saving
}

// Load reads a snapshot from disk and restores all stores.
func (m *Manager) Load() (*LoadResult, error) {
	file, err := os.Open(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoSnapshot
		}
		return nil, fmt.Errorf("failed to open snapshot file: %w", err)
	}
	defer file.Close()

	return m.LoadFrom(file)
}

// LoadFrom reads and restores a snapshot from the given reader.
func (m *Manager) LoadFrom(r io.Reader) (*LoadResult, error) {
	var snapshot Snapshot
	decoder := gob.NewDecoder(r)
	if err := decoder.Decode(&snapshot); err != nil {
		return nil, fmt.Errorf("failed to decode snapshot: %w", err)
	}

	result := &LoadResult{}

	if m.stores.Strings != nil {
		if err := m.stores.Strings.ImportData(snapshot.Strings); err != nil {
			return nil, fmt.Errorf("failed to restore strings: %w", err)
		}
		result.StringKeys = len(snapshot.Strings.Data)
	}

	if m.stores.Lists != nil {
		if err := m.stores.Lists.ImportData(snapshot.Lists); err != nil {
			return nil, fmt.Errorf("failed to restore lists: %w", err)
		}
		result.ListKeys = len(snapshot.Lists.Data)
	}

	if m.stores.Hashes != nil {
		if err := m.stores.Hashes.ImportData(snapshot.Hashes); err != nil {
			return nil, fmt.Errorf("failed to restore hashes: %w", err)
		}
		result.HashKeys = len(snapshot.Hashes.Data)
	}

	if m.stores.Sets != nil {
		if err := m.stores.Sets.ImportData(snapshot.Sets); err != nil {
			return nil, fmt.Errorf("failed to restore sets: %w", err)
		}
		result.SetKeys = len(snapshot.Sets.Data)
	}

	return result, nil
}

// LoadResult contains statistics about a loaded snapshot.
type LoadResult struct {
	StringKeys int
	ListKeys   int
	HashKeys   int
	SetKeys    int
}

// TotalKeys returns the total number of keys loaded.
func (r *LoadResult) TotalKeys() int {
	return r.StringKeys + r.ListKeys + r.HashKeys + r.SetKeys
}

// Exists returns true if a snapshot file exists.
func (m *Manager) Exists() bool {
	_, err := os.Stat(m.path)
	return err == nil
}
