package store

import (
	"sort"
	"sync"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	s := New()
	defer s.Close()

	if s == nil {
		t.Fatal("New() returned nil")
	}
}

func TestSetAndGet(t *testing.T) {
	s := New()
	defer s.Close()

	tests := []struct {
		key   string
		value string
	}{
		{"foo", "bar"},
		{"hello", "world"},
		{"empty", ""},
		{"key with spaces", "value with spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			s.Set(tt.key, tt.value)

			got, ok := s.Get(tt.key)
			if !ok {
				t.Errorf("Get(%q) returned ok=false, want ok=true", tt.key)
			}
			if got != tt.value {
				t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.value)
			}
		})
	}
}

func TestGetNonExistent(t *testing.T) {
	s := New()
	defer s.Close()

	got, ok := s.Get("nonexistent")
	if ok {
		t.Error("Get(nonexistent) returned ok=true, want ok=false")
	}
	if got != "" {
		t.Errorf("Get(nonexistent) = %q, want empty string", got)
	}
}

func TestSetOverwrite(t *testing.T) {
	s := New()
	defer s.Close()

	s.Set("key", "value1")
	s.Set("key", "value2")

	got, ok := s.Get("key")
	if !ok {
		t.Error("Get after overwrite returned ok=false")
	}
	if got != "value2" {
		t.Errorf("Get after overwrite = %q, want %q", got, "value2")
	}
}

func TestDelete(t *testing.T) {
	s := New()
	defer s.Close()

	s.Set("key", "value")

	deleted := s.Delete("key")
	if !deleted {
		t.Error("Delete returned false for existing key")
	}

	_, ok := s.Get("key")
	if ok {
		t.Error("Get after Delete returned ok=true")
	}
}

func TestDeleteNonExistent(t *testing.T) {
	s := New()
	defer s.Close()

	deleted := s.Delete("nonexistent")
	if deleted {
		t.Error("Delete returned true for non-existent key")
	}
}

func TestKeys(t *testing.T) {
	s := New()
	defer s.Close()

	s.Set("a", "1")
	s.Set("b", "2")
	s.Set("c", "3")

	keys := s.Keys()
	sort.Strings(keys)

	expected := []string{"a", "b", "c"}
	if len(keys) != len(expected) {
		t.Fatalf("Keys() returned %d keys, want %d", len(keys), len(expected))
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("Keys()[%d] = %q, want %q", i, key, expected[i])
		}
	}
}

func TestKeysEmpty(t *testing.T) {
	s := New()
	defer s.Close()

	keys := s.Keys()
	if len(keys) != 0 {
		t.Errorf("Keys() on empty store returned %d keys, want 0", len(keys))
	}
}

func TestSetWithTTL(t *testing.T) {
	s := New()
	defer s.Close()

	s.SetWithTTL("key", "value", 100*time.Millisecond)

	// Should exist immediately
	got, ok := s.Get("key")
	if !ok {
		t.Error("Get immediately after SetWithTTL returned ok=false")
	}
	if got != "value" {
		t.Errorf("Get immediately after SetWithTTL = %q, want %q", got, "value")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	_, ok = s.Get("key")
	if ok {
		t.Error("Get after TTL expiration returned ok=true")
	}
}

func TestSetWithTTLKeys(t *testing.T) {
	s := New()
	defer s.Close()

	s.SetWithTTL("expires", "value", 50*time.Millisecond)
	s.Set("permanent", "value")

	// Both keys should exist initially
	keys := s.Keys()
	if len(keys) != 2 {
		t.Errorf("Keys() initially returned %d keys, want 2", len(keys))
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	keys = s.Keys()
	if len(keys) != 1 {
		t.Errorf("Keys() after expiration returned %d keys, want 1", len(keys))
	}
	if len(keys) == 1 && keys[0] != "permanent" {
		t.Errorf("Keys() after expiration = %v, want [permanent]", keys)
	}
}

func TestTTL(t *testing.T) {
	s := New()
	defer s.Close()

	// Key without TTL
	s.Set("no-ttl", "value")
	ttl, ok := s.TTL("no-ttl")
	if ok {
		t.Errorf("TTL(no-ttl) returned ok=true, ttl=%v", ttl)
	}

	// Key with TTL
	s.SetWithTTL("with-ttl", "value", 500*time.Millisecond)
	ttl, ok = s.TTL("with-ttl")
	if !ok {
		t.Error("TTL(with-ttl) returned ok=false")
	}
	// TTL should be roughly 500ms (allow some tolerance)
	if ttl < 400*time.Millisecond || ttl > 500*time.Millisecond {
		t.Errorf("TTL(with-ttl) = %v, want ~500ms", ttl)
	}

	// Non-existent key
	_, ok = s.TTL("nonexistent")
	if ok {
		t.Error("TTL(nonexistent) returned ok=true")
	}
}

func TestTTLExpired(t *testing.T) {
	s := New()
	defer s.Close()

	s.SetWithTTL("key", "value", 50*time.Millisecond)

	time.Sleep(100 * time.Millisecond)

	_, ok := s.TTL("key")
	if ok {
		t.Error("TTL after expiration returned ok=true")
	}
}

func TestBackgroundCleanup(t *testing.T) {
	s := New()
	defer s.Close()

	// Set multiple keys with short TTL
	for i := 0; i < 100; i++ {
		s.SetWithTTL("key"+string(rune('0'+i%10)), "value", 50*time.Millisecond)
	}

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)

	keys := s.Keys()
	if len(keys) != 0 {
		t.Errorf("After cleanup, Keys() returned %d keys, want 0", len(keys))
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := New()
	defer s.Close()

	var wg sync.WaitGroup
	numGoroutines := 100
	numOps := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := "key" + string(rune('0'+id%10))
				s.Set(key, "value")
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := "key" + string(rune('0'+id%10))
				s.Get(key)
			}
		}(i)
	}

	// Concurrent deletes
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := "key" + string(rune('0'+id%10))
				s.Delete(key)
			}
		}(i)
	}

	// Concurrent Keys() calls
	for i := 0; i < numGoroutines/4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				s.Keys()
			}
		}()
	}

	wg.Wait()
}

func TestConcurrentTTL(t *testing.T) {
	s := New()
	defer s.Close()

	var wg sync.WaitGroup
	numGoroutines := 50
	numOps := 50

	// Concurrent SetWithTTL
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := "key" + string(rune('0'+id%10))
				s.SetWithTTL(key, "value", 100*time.Millisecond)
			}
		}(i)
	}

	// Concurrent TTL checks
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := "key" + string(rune('0'+id%10))
				s.TTL(key)
			}
		}(i)
	}

	wg.Wait()
}

func TestCloseIdempotent(t *testing.T) {
	s := New()

	// Should not panic when called multiple times
	s.Close()
	s.Close()
	s.Close()
}
