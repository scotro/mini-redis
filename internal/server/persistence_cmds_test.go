package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scotro/mini-redis/internal/persistence"
	"github.com/scotro/mini-redis/internal/resp"
	"github.com/scotro/mini-redis/internal/store"
)

func TestPersistenceHandler_HandleSave(t *testing.T) {
	tmpDir := t.TempDir()
	snapshotPath := filepath.Join(tmpDir, "test.rdb")

	stringStore := store.New()
	defer stringStore.Close()

	stringStore.Set("key1", "value1")
	stringStore.Set("key2", "value2")

	stores := persistence.Stores{
		Strings: store.AsSnapshottable(stringStore),
	}
	manager := persistence.NewManager(snapshotPath, stores)
	handler := NewPersistenceHandler(manager)

	// Test SAVE command
	result := handler.HandleSave([]resp.Value{})

	if result.Type != resp.TypeSimpleString {
		t.Errorf("Expected TypeSimpleString, got %v", result.Type)
	}
	if result.Str != "OK" {
		t.Errorf("Expected OK, got %s", result.Str)
	}

	// Verify file was created
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Errorf("Snapshot file not created: %v", err)
	}

	// Test with wrong number of arguments
	result = handler.HandleSave([]resp.Value{{Type: resp.TypeBulkString, Str: "extra"}})
	if result.Type != resp.TypeError {
		t.Error("Expected error for wrong number of arguments")
	}
}

func TestPersistenceHandler_HandleBGSave(t *testing.T) {
	tmpDir := t.TempDir()
	snapshotPath := filepath.Join(tmpDir, "test.rdb")

	stringStore := store.New()
	defer stringStore.Close()

	stringStore.Set("key1", "value1")

	stores := persistence.Stores{
		Strings: store.AsSnapshottable(stringStore),
	}
	manager := persistence.NewManager(snapshotPath, stores)
	handler := NewPersistenceHandler(manager)

	// Test BGSAVE command
	result := handler.HandleBGSave([]resp.Value{})

	if result.Type != resp.TypeSimpleString {
		t.Errorf("Expected TypeSimpleString, got %v", result.Type)
	}
	if result.Str != "Background saving started" {
		t.Errorf("Expected 'Background saving started', got %s", result.Str)
	}

	// Wait for background save to complete
	if err := manager.WaitForSave(); err != nil {
		t.Errorf("WaitForSave failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Errorf("Snapshot file not created: %v", err)
	}

	// Test with wrong number of arguments
	result = handler.HandleBGSave([]resp.Value{{Type: resp.TypeBulkString, Str: "extra"}})
	if result.Type != resp.TypeError {
		t.Error("Expected error for wrong number of arguments")
	}
}

func TestPersistenceHandler_BGSaveInProgress(t *testing.T) {
	tmpDir := t.TempDir()
	snapshotPath := filepath.Join(tmpDir, "test.rdb")

	stringStore := store.New()
	defer stringStore.Close()

	stores := persistence.Stores{
		Strings: store.AsSnapshottable(stringStore),
	}
	manager := persistence.NewManager(snapshotPath, stores)
	handler := NewPersistenceHandler(manager)

	// Start first BGSAVE
	result := handler.HandleBGSave([]resp.Value{})
	if result.Type != resp.TypeSimpleString {
		t.Fatal("First BGSAVE should succeed")
	}

	// Try second BGSAVE while first is in progress
	// Note: This might be timing-dependent, so we check if it's either OK (if first finished) or error
	result = handler.HandleBGSave([]resp.Value{})
	if result.Type == resp.TypeError && result.Str != "ERR Background save already in progress" {
		t.Errorf("Expected 'Background save already in progress' error, got %s", result.Str)
	}

	// Wait for any save to complete
	manager.WaitForSave()
}

// TestPersistence_Integration tests the complete save/load cycle
func TestPersistence_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	snapshotPath := filepath.Join(tmpDir, "integration.rdb")

	// Phase 1: Create stores, add data, save
	{
		stringStore := store.New()
		listStore := store.NewListStore()
		hashStore := store.NewHashStore()
		setStore := store.NewSetStore()

		// Add various data types
		stringStore.Set("string1", "hello")
		stringStore.SetWithTTL("string2", "world", 1*time.Hour)

		listStore.RPush("mylist", "a", "b", "c")

		hashStore.HSet("myhash", "field1", "val1", "field2", "val2")

		setStore.SAdd("myset", "member1", "member2")

		stores := persistence.Stores{
			Strings: store.AsSnapshottable(stringStore),
			Lists:   store.AsSnapshottable(listStore),
			Hashes:  store.AsSnapshottable(hashStore),
			Sets:    store.AsSnapshottable(setStore),
		}
		manager := persistence.NewManager(snapshotPath, stores)
		handler := NewPersistenceHandler(manager)

		// Save
		result := handler.HandleSave([]resp.Value{})
		if result.Str != "OK" {
			t.Fatalf("SAVE failed: %s", result.Str)
		}

		stringStore.Close()
	}

	// Phase 2: Create fresh stores, load from snapshot, verify data
	{
		stringStore := store.New()
		listStore := store.NewListStore()
		hashStore := store.NewHashStore()
		setStore := store.NewSetStore()

		stores := persistence.Stores{
			Strings: store.AsSnapshottable(stringStore),
			Lists:   store.AsSnapshottable(listStore),
			Hashes:  store.AsSnapshottable(hashStore),
			Sets:    store.AsSnapshottable(setStore),
		}
		manager := persistence.NewManager(snapshotPath, stores)

		// Load
		result, err := manager.Load()
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		// Verify counts
		if result.TotalKeys() != 5 {
			t.Errorf("Expected 5 total keys, got %d", result.TotalKeys())
		}

		// Verify string data
		if val, ok := stringStore.Get("string1"); !ok || val != "hello" {
			t.Error("string1 not restored correctly")
		}
		if val, ok := stringStore.Get("string2"); !ok || val != "world" {
			t.Error("string2 not restored correctly")
		}
		if _, hasTTL := stringStore.TTL("string2"); !hasTTL {
			t.Error("string2 TTL not preserved")
		}

		// Verify list data
		list := listStore.LRange("mylist", 0, -1)
		if len(list) != 3 || list[0] != "a" || list[1] != "b" || list[2] != "c" {
			t.Errorf("mylist not restored correctly: %v", list)
		}

		// Verify hash data
		hashVal, ok := hashStore.HGet("myhash", "field1")
		if !ok || hashVal != "val1" {
			t.Error("myhash.field1 not restored correctly")
		}
		if hashStore.HLen("myhash") != 2 {
			t.Error("myhash length incorrect")
		}

		// Verify set data
		if !setStore.SIsMember("myset", "member1") {
			t.Error("myset.member1 not restored")
		}
		if !setStore.SIsMember("myset", "member2") {
			t.Error("myset.member2 not restored")
		}
		if setStore.SCard("myset") != 2 {
			t.Error("myset cardinality incorrect")
		}

		stringStore.Close()
	}
}
