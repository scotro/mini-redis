package transaction

import (
	"sync"
	"testing"
)

func TestMemoryVersionTracker_GetVersion(t *testing.T) {
	vt := NewMemoryVersionTracker()

	// Non-existent key should return 0
	if v := vt.GetVersion("key1"); v != 0 {
		t.Errorf("GetVersion(non-existent) = %d, want 0", v)
	}

	// After increment, should return 1
	vt.IncrementVersion("key1")
	if v := vt.GetVersion("key1"); v != 1 {
		t.Errorf("GetVersion(after increment) = %d, want 1", v)
	}
}

func TestMemoryVersionTracker_IncrementVersion(t *testing.T) {
	vt := NewMemoryVersionTracker()

	// Multiple increments
	vt.IncrementVersion("key1")
	vt.IncrementVersion("key1")
	vt.IncrementVersion("key1")

	if v := vt.GetVersion("key1"); v != 3 {
		t.Errorf("GetVersion(after 3 increments) = %d, want 3", v)
	}

	// Different keys are independent
	vt.IncrementVersion("key2")
	if v := vt.GetVersion("key2"); v != 1 {
		t.Errorf("GetVersion(key2) = %d, want 1", v)
	}
	if v := vt.GetVersion("key1"); v != 3 {
		t.Errorf("GetVersion(key1) = %d, want 3 (unchanged)", v)
	}
}

func TestMemoryVersionTracker_DeleteVersion(t *testing.T) {
	vt := NewMemoryVersionTracker()

	vt.IncrementVersion("key1")
	vt.IncrementVersion("key1")

	if v := vt.GetVersion("key1"); v != 2 {
		t.Errorf("GetVersion(before delete) = %d, want 2", v)
	}

	vt.DeleteVersion("key1")

	if v := vt.GetVersion("key1"); v != 0 {
		t.Errorf("GetVersion(after delete) = %d, want 0", v)
	}

	// Delete non-existent key should not panic
	vt.DeleteVersion("non-existent")
}

func TestMemoryVersionTracker_SetVersion(t *testing.T) {
	vt := NewMemoryVersionTracker()

	vt.SetVersion("key1", 100)
	if v := vt.GetVersion("key1"); v != 100 {
		t.Errorf("GetVersion(after SetVersion) = %d, want 100", v)
	}

	vt.IncrementVersion("key1")
	if v := vt.GetVersion("key1"); v != 101 {
		t.Errorf("GetVersion(after SetVersion+Increment) = %d, want 101", v)
	}
}

func TestMemoryVersionTracker_Concurrent(t *testing.T) {
	vt := NewMemoryVersionTracker()
	var wg sync.WaitGroup
	iterations := 100
	goroutines := 10

	// Concurrent increments
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				vt.IncrementVersion("key1")
			}
		}()
	}

	// Concurrent reads
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				vt.GetVersion("key1")
			}
		}()
	}

	wg.Wait()

	expected := int64(goroutines * iterations)
	if v := vt.GetVersion("key1"); v != expected {
		t.Errorf("GetVersion(after concurrent increments) = %d, want %d", v, expected)
	}
}

func TestMemoryVersionTracker_ImplementsInterface(t *testing.T) {
	var _ VersionTracker = NewMemoryVersionTracker()
}
