// Package transaction implements Redis transaction support with MULTI/EXEC/DISCARD/WATCH.
package transaction

import (
	"sync"

	"github.com/scotro/mini-redis/internal/resp"
)

// QueuedCommand represents a command queued during a transaction.
type QueuedCommand struct {
	Name string
	Args []string
}

// CommandExecutor is a function that executes a command and returns the result.
// This is passed in during EXEC to execute queued commands.
type CommandExecutor func(cmd string, args []string) (resp.Value, error)

// VersionGetter is a function that retrieves the current version of a key.
type VersionGetter func(key string) int64

// Transaction manages the state for a Redis transaction.
// Each connection should have its own Transaction instance.
type Transaction struct {
	mu       sync.Mutex
	inMulti  bool
	queue    []QueuedCommand
	watching map[string]int64 // key -> version at watch time
}

// New creates a new Transaction.
func New() *Transaction {
	return &Transaction{
		watching: make(map[string]int64),
	}
}

// Begin starts a transaction (MULTI command).
// Returns an error if already in a transaction.
func (t *Transaction) Begin() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.inMulti {
		return ErrNestedMulti
	}

	t.inMulti = true
	t.queue = nil
	return nil
}

// InTransaction returns true if currently in MULTI mode.
func (t *Transaction) InTransaction() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.inMulti
}

// Queue adds a command to the transaction queue.
// Returns an error if not in a transaction.
func (t *Transaction) Queue(cmd string, args []string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.inMulti {
		return ErrNotInMulti
	}

	t.queue = append(t.queue, QueuedCommand{
		Name: cmd,
		Args: args,
	})
	return nil
}

// QueueLength returns the number of queued commands.
func (t *Transaction) QueueLength() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.queue)
}

// Exec executes all queued commands atomically and returns the results.
// Returns nil results if watched keys have changed.
// After EXEC, the transaction is reset and watched keys are cleared.
func (t *Transaction) Exec(executor CommandExecutor, getVersion VersionGetter) ([]resp.Value, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.inMulti {
		return nil, ErrExecWithoutMulti
	}

	// Check if any watched keys have changed
	if !t.checkWatchLocked(getVersion) {
		// Reset state
		t.resetLocked()
		// Return empty array (nil array in Redis terms means WATCH failed)
		return nil, nil
	}

	// Execute all queued commands
	results := make([]resp.Value, len(t.queue))
	for i, qc := range t.queue {
		result, err := executor(qc.Name, qc.Args)
		if err != nil {
			// In Redis, individual command errors are returned in the results array
			// but don't abort the entire transaction
			results[i] = resp.Value{Type: resp.TypeError, Str: err.Error()}
		} else {
			results[i] = result
		}
	}

	// Reset state
	t.resetLocked()

	return results, nil
}

// Discard aborts the transaction and clears the queue.
// Returns an error if not in a transaction.
func (t *Transaction) Discard() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.inMulti {
		return ErrDiscardWithoutMulti
	}

	t.resetLocked()
	return nil
}

// Watch records the current version of keys for optimistic locking.
// Must be called before MULTI.
// Returns an error if already in a transaction.
func (t *Transaction) Watch(getVersion VersionGetter, keys ...string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.inMulti {
		return ErrWatchInsideMulti
	}

	for _, key := range keys {
		t.watching[key] = getVersion(key)
	}
	return nil
}

// Unwatch clears all watched keys.
func (t *Transaction) Unwatch() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.watching = make(map[string]int64)
}

// IsWatching returns true if any keys are being watched.
func (t *Transaction) IsWatching() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.watching) > 0
}

// WatchedKeys returns a copy of the currently watched keys and their versions.
func (t *Transaction) WatchedKeys() map[string]int64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	result := make(map[string]int64, len(t.watching))
	for k, v := range t.watching {
		result[k] = v
	}
	return result
}

// CheckWatch verifies that no watched keys have been modified.
// Returns true if all watched keys have the same version as when they were watched.
func (t *Transaction) CheckWatch(getVersion VersionGetter) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.checkWatchLocked(getVersion)
}

// checkWatchLocked is the internal version that assumes the lock is held.
func (t *Transaction) checkWatchLocked(getVersion VersionGetter) bool {
	if getVersion == nil {
		return true
	}

	for key, watchedVersion := range t.watching {
		currentVersion := getVersion(key)
		if currentVersion != watchedVersion {
			return false
		}
	}
	return true
}

// resetLocked resets the transaction state. Assumes lock is held.
func (t *Transaction) resetLocked() {
	t.inMulti = false
	t.queue = nil
	t.watching = make(map[string]int64)
}
