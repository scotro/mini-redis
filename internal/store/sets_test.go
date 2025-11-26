package store

import (
	"sort"
	"sync"
	"testing"
)

func TestNewSetStore(t *testing.T) {
	s := NewSetStore()
	if s == nil {
		t.Fatal("NewSetStore() returned nil")
	}
}

func TestSAdd(t *testing.T) {
	s := NewSetStore()

	tests := []struct {
		name     string
		key      string
		members  []string
		expected int
	}{
		{"add single member", "set1", []string{"a"}, 1},
		{"add multiple members", "set2", []string{"a", "b", "c"}, 3},
		{"add duplicate in same call", "set3", []string{"a", "a", "a"}, 1},
		{"add empty members", "set4", []string{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.SAdd(tt.key, tt.members...)
			if got != tt.expected {
				t.Errorf("SAdd(%q, %v) = %d, want %d", tt.key, tt.members, got, tt.expected)
			}
		})
	}
}

func TestSAddDuplicates(t *testing.T) {
	s := NewSetStore()

	// First add
	added := s.SAdd("myset", "a", "b", "c")
	if added != 3 {
		t.Errorf("First SAdd returned %d, want 3", added)
	}

	// Add duplicates and new
	added = s.SAdd("myset", "a", "b", "d")
	if added != 1 {
		t.Errorf("Second SAdd returned %d, want 1 (only 'd' is new)", added)
	}

	// Add only duplicates
	added = s.SAdd("myset", "a", "b", "c", "d")
	if added != 0 {
		t.Errorf("Third SAdd returned %d, want 0 (all duplicates)", added)
	}
}

func TestSRem(t *testing.T) {
	s := NewSetStore()

	// Setup
	s.SAdd("myset", "a", "b", "c", "d")

	tests := []struct {
		name     string
		members  []string
		expected int
	}{
		{"remove single existing", []string{"a"}, 1},
		{"remove multiple existing", []string{"b", "c"}, 2},
		{"remove non-existing", []string{"x", "y"}, 0},
		{"remove mixed", []string{"d", "z"}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.SRem("myset", tt.members...)
			if got != tt.expected {
				t.Errorf("SRem(myset, %v) = %d, want %d", tt.members, got, tt.expected)
			}
		})
	}
}

func TestSRemNonExistentKey(t *testing.T) {
	s := NewSetStore()

	removed := s.SRem("nonexistent", "a", "b")
	if removed != 0 {
		t.Errorf("SRem on non-existent key returned %d, want 0", removed)
	}
}

func TestSRemAutoDelete(t *testing.T) {
	s := NewSetStore()

	s.SAdd("myset", "a", "b")
	s.SRem("myset", "a", "b")

	// Set should be auto-deleted when empty
	if s.KeyType("myset") != "none" {
		t.Error("Empty set was not auto-deleted")
	}
}

func TestSMembers(t *testing.T) {
	s := NewSetStore()

	// Empty set
	members := s.SMembers("nonexistent")
	if len(members) != 0 {
		t.Errorf("SMembers(nonexistent) returned %d members, want 0", len(members))
	}

	// Set with members
	s.SAdd("myset", "c", "a", "b")
	members = s.SMembers("myset")
	sort.Strings(members)

	expected := []string{"a", "b", "c"}
	if len(members) != len(expected) {
		t.Fatalf("SMembers returned %d members, want %d", len(members), len(expected))
	}
	for i, m := range members {
		if m != expected[i] {
			t.Errorf("SMembers()[%d] = %q, want %q", i, m, expected[i])
		}
	}
}

func TestSIsMember(t *testing.T) {
	s := NewSetStore()

	s.SAdd("myset", "a", "b", "c")

	tests := []struct {
		member   string
		expected bool
	}{
		{"a", true},
		{"b", true},
		{"c", true},
		{"d", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.member, func(t *testing.T) {
			got := s.SIsMember("myset", tt.member)
			if got != tt.expected {
				t.Errorf("SIsMember(myset, %q) = %v, want %v", tt.member, got, tt.expected)
			}
		})
	}
}

func TestSIsMemberNonExistentKey(t *testing.T) {
	s := NewSetStore()

	if s.SIsMember("nonexistent", "a") {
		t.Error("SIsMember on non-existent key returned true")
	}
}

func TestSCard(t *testing.T) {
	s := NewSetStore()

	// Non-existent key
	if card := s.SCard("nonexistent"); card != 0 {
		t.Errorf("SCard(nonexistent) = %d, want 0", card)
	}

	// Empty after creation doesn't exist
	s.SAdd("myset", "a", "b", "c")
	if card := s.SCard("myset"); card != 3 {
		t.Errorf("SCard(myset) = %d, want 3", card)
	}

	s.SRem("myset", "a")
	if card := s.SCard("myset"); card != 2 {
		t.Errorf("SCard(myset) after remove = %d, want 2", card)
	}
}

func TestSInter(t *testing.T) {
	s := NewSetStore()

	s.SAdd("set1", "a", "b", "c", "d")
	s.SAdd("set2", "b", "c", "d", "e")
	s.SAdd("set3", "c", "d", "e", "f")

	tests := []struct {
		name     string
		keys     []string
		expected []string
	}{
		{"single key", []string{"set1"}, []string{"a", "b", "c", "d"}},
		{"two keys", []string{"set1", "set2"}, []string{"b", "c", "d"}},
		{"three keys", []string{"set1", "set2", "set3"}, []string{"c", "d"}},
		{"no keys", []string{}, []string{}},
		{"non-existent key", []string{"set1", "nonexistent"}, []string{}},
		{"only non-existent", []string{"nonexistent"}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.SInter(tt.keys...)
			sort.Strings(got)
			sort.Strings(tt.expected)

			if len(got) != len(tt.expected) {
				t.Fatalf("SInter(%v) returned %d members, want %d: got %v", tt.keys, len(got), len(tt.expected), got)
			}
			for i, m := range got {
				if m != tt.expected[i] {
					t.Errorf("SInter(%v)[%d] = %q, want %q", tt.keys, i, m, tt.expected[i])
				}
			}
		})
	}
}

func TestSInterEmptySets(t *testing.T) {
	s := NewSetStore()

	s.SAdd("set1", "a", "b")
	s.SAdd("empty")

	// After adding nothing, "empty" doesn't exist
	result := s.SInter("set1", "empty")
	if len(result) != 0 {
		t.Errorf("SInter with non-existent set returned %v, want empty", result)
	}
}

func TestKeyType(t *testing.T) {
	s := NewSetStore()

	// Non-existent
	if typ := s.KeyType("nonexistent"); typ != "none" {
		t.Errorf("KeyType(nonexistent) = %q, want %q", typ, "none")
	}

	// Set
	s.SAdd("myset", "a")
	if typ := s.KeyType("myset"); typ != "set" {
		t.Errorf("KeyType(myset) = %q, want %q", typ, "set")
	}
}

func TestConcurrentSetAccess(t *testing.T) {
	s := NewSetStore()

	var wg sync.WaitGroup
	numGoroutines := 50
	numOps := 100

	// Concurrent SAdd
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				s.SAdd("concurrent", "member"+string(rune('0'+id%10)))
			}
		}(i)
	}

	// Concurrent SIsMember
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				s.SIsMember("concurrent", "member"+string(rune('0'+id%10)))
			}
		}(i)
	}

	// Concurrent SMembers
	for i := 0; i < numGoroutines/4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				s.SMembers("concurrent")
			}
		}()
	}

	// Concurrent SCard
	for i := 0; i < numGoroutines/4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				s.SCard("concurrent")
			}
		}()
	}

	wg.Wait()
}

func TestConcurrentSetRemove(t *testing.T) {
	s := NewSetStore()

	var wg sync.WaitGroup
	numGoroutines := 50
	numOps := 100

	// Concurrent SAdd and SRem
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				member := "member" + string(rune('0'+j%10))
				s.SAdd("concurrent", member)
				s.SRem("concurrent", member)
			}
		}(i)
	}

	wg.Wait()
}

func TestConcurrentSInter(t *testing.T) {
	s := NewSetStore()

	// Setup some sets
	s.SAdd("set1", "a", "b", "c")
	s.SAdd("set2", "b", "c", "d")
	s.SAdd("set3", "c", "d", "e")

	var wg sync.WaitGroup
	numGoroutines := 50
	numOps := 100

	// Concurrent SInter
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				s.SInter("set1", "set2", "set3")
			}
		}()
	}

	// Concurrent modifications
	for i := 0; i < numGoroutines/4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				s.SAdd("set1", "temp")
				s.SRem("set1", "temp")
			}
		}(i)
	}

	wg.Wait()
}
