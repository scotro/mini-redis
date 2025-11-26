package store

import (
	"testing"
	"time"
)

func TestStringStore_ExportImportData(t *testing.T) {
	// Create source store with data
	src := New().(*memoryStore)
	defer src.Close()

	src.Set("key1", "value1")
	src.Set("key2", "value2")
	src.SetWithTTL("key3", "value3", 1*time.Hour)

	// Export data
	exported := src.ExportData()
	snapshot, ok := exported.(StringSnapshot)
	if !ok {
		t.Fatal("ExportData did not return StringSnapshot")
	}

	// Verify exported data
	if len(snapshot.Data) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(snapshot.Data))
	}

	if snapshot.Data["key1"].Value != "value1" {
		t.Error("key1 value mismatch")
	}
	if snapshot.Data["key2"].Value != "value2" {
		t.Error("key2 value mismatch")
	}
	if snapshot.Data["key3"].Value != "value3" {
		t.Error("key3 value mismatch")
	}
	if snapshot.Data["key3"].ExpiresAt == 0 {
		t.Error("key3 should have expiration")
	}

	// Import into new store
	dst := New().(*memoryStore)
	defer dst.Close()

	if err := dst.ImportData(snapshot); err != nil {
		t.Fatalf("ImportData failed: %v", err)
	}

	// Verify imported data
	if val, ok := dst.Get("key1"); !ok || val != "value1" {
		t.Error("key1 not imported correctly")
	}
	if val, ok := dst.Get("key2"); !ok || val != "value2" {
		t.Error("key2 not imported correctly")
	}
	if val, ok := dst.Get("key3"); !ok || val != "value3" {
		t.Error("key3 not imported correctly")
	}
	if _, hasTTL := dst.TTL("key3"); !hasTTL {
		t.Error("key3 TTL not preserved")
	}
}

func TestStringStore_ImportInvalidData(t *testing.T) {
	s := New().(*memoryStore)
	defer s.Close()

	err := s.ImportData("invalid")
	if err != ErrInvalidSnapshotData {
		t.Errorf("Expected ErrInvalidSnapshotData, got %v", err)
	}
}

func TestStringStore_ExportSkipsExpired(t *testing.T) {
	s := New().(*memoryStore)
	defer s.Close()

	s.Set("persistent", "value")
	s.SetWithTTL("expired", "value", 1*time.Millisecond)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	exported := s.ExportData()
	snapshot := exported.(StringSnapshot)

	if len(snapshot.Data) != 1 {
		t.Errorf("Expected 1 entry (expired excluded), got %d", len(snapshot.Data))
	}
	if _, ok := snapshot.Data["persistent"]; !ok {
		t.Error("persistent key should be in snapshot")
	}
	if _, ok := snapshot.Data["expired"]; ok {
		t.Error("expired key should not be in snapshot")
	}
}

func TestListStore_ExportImportData(t *testing.T) {
	src := NewListStore().(*memoryListStore)

	src.RPush("list1", "a", "b", "c")
	src.LPush("list2", "x", "y")

	exported := src.ExportData()
	snapshot, ok := exported.(ListSnapshot)
	if !ok {
		t.Fatal("ExportData did not return ListSnapshot")
	}

	// Verify exported data
	if len(snapshot.Data) != 2 {
		t.Errorf("Expected 2 lists, got %d", len(snapshot.Data))
	}

	// Import into new store
	dst := NewListStore().(*memoryListStore)

	if err := dst.ImportData(snapshot); err != nil {
		t.Fatalf("ImportData failed: %v", err)
	}

	// Verify
	list1 := dst.LRange("list1", 0, -1)
	if len(list1) != 3 || list1[0] != "a" || list1[1] != "b" || list1[2] != "c" {
		t.Errorf("list1 not restored correctly: %v", list1)
	}
}

func TestListStore_ImportInvalidData(t *testing.T) {
	s := NewListStore().(*memoryListStore)

	err := s.ImportData("invalid")
	if err != ErrInvalidSnapshotData {
		t.Errorf("Expected ErrInvalidSnapshotData, got %v", err)
	}
}

func TestHashStore_ExportImportData(t *testing.T) {
	src := NewHashStore().(*MemoryHashStore)

	src.HSet("hash1", "field1", "val1", "field2", "val2")
	src.HSet("hash2", "foo", "bar")

	exported := src.ExportData()
	snapshot, ok := exported.(HashSnapshot)
	if !ok {
		t.Fatal("ExportData did not return HashSnapshot")
	}

	if len(snapshot.Data) != 2 {
		t.Errorf("Expected 2 hashes, got %d", len(snapshot.Data))
	}

	dst := NewHashStore().(*MemoryHashStore)

	if err := dst.ImportData(snapshot); err != nil {
		t.Fatalf("ImportData failed: %v", err)
	}

	val, ok := dst.HGet("hash1", "field1")
	if !ok || val != "val1" {
		t.Error("hash1.field1 not restored correctly")
	}
}

func TestHashStore_ImportInvalidData(t *testing.T) {
	s := NewHashStore().(*MemoryHashStore)

	err := s.ImportData("invalid")
	if err != ErrInvalidSnapshotData {
		t.Errorf("Expected ErrInvalidSnapshotData, got %v", err)
	}
}

func TestSetStore_ExportImportData(t *testing.T) {
	src := NewSetStore()

	src.SAdd("set1", "a", "b", "c")
	src.SAdd("set2", "x", "y")

	exported := src.ExportData()
	snapshot, ok := exported.(SetSnapshot)
	if !ok {
		t.Fatal("ExportData did not return SetSnapshot")
	}

	if len(snapshot.Data) != 2 {
		t.Errorf("Expected 2 sets, got %d", len(snapshot.Data))
	}

	dst := NewSetStore()

	if err := dst.ImportData(snapshot); err != nil {
		t.Fatalf("ImportData failed: %v", err)
	}

	if !dst.SIsMember("set1", "a") {
		t.Error("set1.a not restored")
	}
	if !dst.SIsMember("set1", "b") {
		t.Error("set1.b not restored")
	}
	if dst.SCard("set1") != 3 {
		t.Error("set1 size incorrect after restore")
	}
}

func TestSetStore_ImportInvalidData(t *testing.T) {
	s := NewSetStore()

	err := s.ImportData("invalid")
	if err != ErrInvalidSnapshotData {
		t.Errorf("Expected ErrInvalidSnapshotData, got %v", err)
	}
}

func TestAsSnapshottable(t *testing.T) {
	tests := []struct {
		name  string
		store interface{}
		isNil bool
	}{
		{"StringStore", New(), false},
		{"ListStore", NewListStore(), false},
		{"HashStore", NewHashStore(), false},
		{"SetStore", NewSetStore(), false},
		{"nil", nil, true},
		{"string", "not a store", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AsSnapshottable(tt.store)
			if tt.isNil && result != nil {
				t.Error("Expected nil, got non-nil")
			}
			if !tt.isNil && result == nil {
				t.Error("Expected non-nil, got nil")
			}
		})
	}

	// Cleanup stores
	New().(Store).Close()
}
