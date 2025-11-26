package persistence

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scotro/mini-redis/internal/store"
)

func TestManager_SaveAndLoad(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	snapshotPath := filepath.Join(tmpDir, "test.rdb")

	// Create stores with test data
	stringStore := store.New()
	listStore := store.NewListStore()
	hashStore := store.NewHashStore()
	setStore := store.NewSetStore()

	// Add test data
	stringStore.Set("key1", "value1")
	stringStore.Set("key2", "value2")
	stringStore.SetWithTTL("key3", "value3", 1*time.Hour)

	listStore.RPush("list1", "a", "b", "c")
	listStore.LPush("list2", "x", "y")

	hashStore.HSet("hash1", "field1", "val1", "field2", "val2")

	setStore.SAdd("set1", "member1", "member2", "member3")

	// Create manager and save
	stores := Stores{
		Strings: store.AsSnapshottable(stringStore),
		Lists:   store.AsSnapshottable(listStore),
		Hashes:  store.AsSnapshottable(hashStore),
		Sets:    store.AsSnapshottable(setStore),
	}
	manager := NewManager(snapshotPath, stores)

	if err := manager.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify file exists
	if !manager.Exists() {
		t.Fatal("Snapshot file should exist after save")
	}

	// Create new stores and load
	stringStore2 := store.New()
	listStore2 := store.NewListStore()
	hashStore2 := store.NewHashStore()
	setStore2 := store.NewSetStore()

	stores2 := Stores{
		Strings: store.AsSnapshottable(stringStore2),
		Lists:   store.AsSnapshottable(listStore2),
		Hashes:  store.AsSnapshottable(hashStore2),
		Sets:    store.AsSnapshottable(setStore2),
	}
	manager2 := NewManager(snapshotPath, stores2)

	result, err := manager2.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify load result counts
	if result.StringKeys != 3 {
		t.Errorf("Expected 3 string keys, got %d", result.StringKeys)
	}
	if result.ListKeys != 2 {
		t.Errorf("Expected 2 list keys, got %d", result.ListKeys)
	}
	if result.HashKeys != 1 {
		t.Errorf("Expected 1 hash key, got %d", result.HashKeys)
	}
	if result.SetKeys != 1 {
		t.Errorf("Expected 1 set key, got %d", result.SetKeys)
	}
	if result.TotalKeys() != 7 {
		t.Errorf("Expected 7 total keys, got %d", result.TotalKeys())
	}

	// Verify string data
	if val, ok := stringStore2.Get("key1"); !ok || val != "value1" {
		t.Errorf("String key1 not restored correctly: got %q, ok=%v", val, ok)
	}
	if val, ok := stringStore2.Get("key2"); !ok || val != "value2" {
		t.Errorf("String key2 not restored correctly: got %q, ok=%v", val, ok)
	}
	if val, ok := stringStore2.Get("key3"); !ok || val != "value3" {
		t.Errorf("String key3 not restored correctly: got %q, ok=%v", val, ok)
	}

	// Verify TTL was preserved (should still have TTL)
	if _, hasTTL := stringStore2.TTL("key3"); !hasTTL {
		t.Error("TTL for key3 was not preserved")
	}

	// Verify list data
	list1 := listStore2.LRange("list1", 0, -1)
	if len(list1) != 3 || list1[0] != "a" || list1[1] != "b" || list1[2] != "c" {
		t.Errorf("List list1 not restored correctly: got %v", list1)
	}

	// Verify hash data
	hashVal, ok := hashStore2.HGet("hash1", "field1")
	if !ok || hashVal != "val1" {
		t.Errorf("Hash hash1.field1 not restored correctly: got %q, ok=%v", hashVal, ok)
	}

	// Verify set data
	if !setStore2.SIsMember("set1", "member1") {
		t.Error("Set set1 member1 not restored")
	}
	if !setStore2.SIsMember("set1", "member2") {
		t.Error("Set set1 member2 not restored")
	}

	// Cleanup
	stringStore.Close()
	stringStore2.Close()
}

func TestManager_BackgroundSave(t *testing.T) {
	tmpDir := t.TempDir()
	snapshotPath := filepath.Join(tmpDir, "test.rdb")

	stringStore := store.New()
	defer stringStore.Close()

	stringStore.Set("key1", "value1")

	stores := Stores{
		Strings: store.AsSnapshottable(stringStore),
	}
	manager := NewManager(snapshotPath, stores)

	// Start background save
	if err := manager.BackgroundSave(); err != nil {
		t.Fatalf("BackgroundSave() failed: %v", err)
	}

	// Should be marked as saving
	if !manager.IsSaving() {
		t.Error("IsSaving() should return true during background save")
	}

	// Attempting another BGSAVE should fail
	if err := manager.BackgroundSave(); err != ErrSaveInProgress {
		t.Errorf("Expected ErrSaveInProgress, got %v", err)
	}

	// Wait for completion
	if err := manager.WaitForSave(); err != nil {
		t.Fatalf("WaitForSave() returned error: %v", err)
	}

	// Verify file was created
	if !manager.Exists() {
		t.Error("Snapshot file should exist after background save")
	}
}

func TestManager_LoadNoSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	snapshotPath := filepath.Join(tmpDir, "nonexistent.rdb")

	stores := Stores{}
	manager := NewManager(snapshotPath, stores)

	if manager.Exists() {
		t.Error("Exists() should return false for nonexistent file")
	}

	_, err := manager.Load()
	if err != ErrNoSnapshot {
		t.Errorf("Expected ErrNoSnapshot, got %v", err)
	}
}

func TestManager_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	snapshotPath := filepath.Join(tmpDir, "test.rdb")

	stringStore := store.New()
	defer stringStore.Close()

	stringStore.Set("key1", "value1")

	stores := Stores{
		Strings: store.AsSnapshottable(stringStore),
	}
	manager := NewManager(snapshotPath, stores)

	if err := manager.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify temp file doesn't exist (was renamed)
	tmpPath := snapshotPath + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("Temp file should not exist after save")
	}

	// Verify main file exists
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Errorf("Snapshot file should exist: %v", err)
	}
}

func TestManager_ExpiredKeysNotSaved(t *testing.T) {
	tmpDir := t.TempDir()
	snapshotPath := filepath.Join(tmpDir, "test.rdb")

	stringStore := store.New()
	defer stringStore.Close()

	stringStore.Set("persistent", "value1")
	stringStore.SetWithTTL("expired", "value2", 1*time.Millisecond)

	// Wait for key to expire
	time.Sleep(10 * time.Millisecond)

	stores := Stores{
		Strings: store.AsSnapshottable(stringStore),
	}
	manager := NewManager(snapshotPath, stores)

	if err := manager.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Load into fresh store
	stringStore2 := store.New()
	defer stringStore2.Close()

	stores2 := Stores{
		Strings: store.AsSnapshottable(stringStore2),
	}
	manager2 := NewManager(snapshotPath, stores2)

	result, err := manager2.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Only persistent key should be loaded
	if result.StringKeys != 1 {
		t.Errorf("Expected 1 string key (expired excluded), got %d", result.StringKeys)
	}

	if _, ok := stringStore2.Get("persistent"); !ok {
		t.Error("Persistent key should be restored")
	}
	if _, ok := stringStore2.Get("expired"); ok {
		t.Error("Expired key should not be restored")
	}
}

func TestManager_Path(t *testing.T) {
	manager := NewManager("/path/to/dump.rdb", Stores{})
	if manager.Path() != "/path/to/dump.rdb" {
		t.Errorf("Expected path /path/to/dump.rdb, got %s", manager.Path())
	}
}
