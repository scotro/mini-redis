package server

import (
	"sort"
	"testing"

	"github.com/scotro/mini-redis/internal/resp"
	"github.com/scotro/mini-redis/internal/store"
)

// createTestServer creates a server for testing set commands.
func createTestServer(t *testing.T) *Server {
	t.Helper()
	st := store.New()
	cfg := Config{Port: 0}
	srv := New(st, cfg)
	t.Cleanup(func() {
		st.Close()
	})
	return srv
}

// makeArgs creates a slice of resp.Value from strings.
func makeArgs(args ...string) []resp.Value {
	result := make([]resp.Value, len(args))
	for i, arg := range args {
		result[i] = resp.Value{Type: resp.TypeBulkString, Str: arg}
	}
	return result
}

func TestHandleSAdd(t *testing.T) {
	ResetSetStore()
	srv := createTestServer(t)

	tests := []struct {
		name       string
		args       []string
		wantType   byte
		wantNum    int
		wantErr    bool
	}{
		{"add single member", []string{"myset", "a"}, resp.TypeInteger, 1, false},
		{"add multiple members", []string{"myset2", "a", "b", "c"}, resp.TypeInteger, 3, false},
		{"add duplicate", []string{"myset", "a"}, resp.TypeInteger, 0, false},
		{"add mixed new and existing", []string{"myset", "a", "b"}, resp.TypeInteger, 1, false},
		{"wrong number of args", []string{"myset"}, resp.TypeError, 0, true},
		{"no args", []string{}, resp.TypeError, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := srv.handleSAdd(makeArgs(tt.args...))

			if tt.wantErr {
				if result.Type != resp.TypeError {
					t.Errorf("Expected error, got type %c", result.Type)
				}
				return
			}

			if result.Type != tt.wantType {
				t.Errorf("Type = %c, want %c", result.Type, tt.wantType)
			}
			if result.Num != tt.wantNum {
				t.Errorf("Num = %d, want %d", result.Num, tt.wantNum)
			}
		})
	}
}

func TestHandleSAddWrongType(t *testing.T) {
	ResetSetStore()
	srv := createTestServer(t)

	// Set a string key first
	srv.store.Set("stringkey", "value")

	// Try to SADD to the string key
	result := srv.handleSAdd(makeArgs("stringkey", "member"))

	if result.Type != resp.TypeError {
		t.Errorf("Expected error type, got %c", result.Type)
	}
	if result.Str != "WRONGTYPE Operation against a key holding the wrong kind of value" {
		t.Errorf("Unexpected error message: %s", result.Str)
	}
}

func TestHandleSRem(t *testing.T) {
	ResetSetStore()
	srv := createTestServer(t)

	// Setup
	setStore.SAdd("myset", "a", "b", "c", "d")

	tests := []struct {
		name     string
		args     []string
		wantNum  int
		wantErr  bool
	}{
		{"remove existing", []string{"myset", "a"}, 1, false},
		{"remove multiple", []string{"myset", "b", "c"}, 2, false},
		{"remove non-existing", []string{"myset", "x", "y"}, 0, false},
		{"remove from non-existing key", []string{"nokey", "a"}, 0, false},
		{"wrong args", []string{"myset"}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := srv.handleSRem(makeArgs(tt.args...))

			if tt.wantErr {
				if result.Type != resp.TypeError {
					t.Errorf("Expected error, got type %c", result.Type)
				}
				return
			}

			if result.Type != resp.TypeInteger {
				t.Errorf("Type = %c, want integer", result.Type)
			}
			if result.Num != tt.wantNum {
				t.Errorf("Num = %d, want %d", result.Num, tt.wantNum)
			}
		})
	}
}

func TestHandleSRemWrongType(t *testing.T) {
	ResetSetStore()
	srv := createTestServer(t)

	srv.store.Set("stringkey", "value")

	result := srv.handleSRem(makeArgs("stringkey", "member"))

	if result.Type != resp.TypeError {
		t.Errorf("Expected error type, got %c", result.Type)
	}
}

func TestHandleSMembers(t *testing.T) {
	ResetSetStore()
	srv := createTestServer(t)

	// Empty set
	result := srv.handleSMembers(makeArgs("nonexistent"))
	if result.Type != resp.TypeArray {
		t.Errorf("Type = %c, want array", result.Type)
	}
	if len(result.Array) != 0 {
		t.Errorf("Empty set returned %d members, want 0", len(result.Array))
	}

	// Set with members
	setStore.SAdd("myset", "c", "a", "b")
	result = srv.handleSMembers(makeArgs("myset"))

	if result.Type != resp.TypeArray {
		t.Errorf("Type = %c, want array", result.Type)
	}
	if len(result.Array) != 3 {
		t.Errorf("Got %d members, want 3", len(result.Array))
	}

	// Extract and sort members
	members := make([]string, len(result.Array))
	for i, v := range result.Array {
		members[i] = v.Str
	}
	sort.Strings(members)

	expected := []string{"a", "b", "c"}
	for i, m := range members {
		if m != expected[i] {
			t.Errorf("Member[%d] = %q, want %q", i, m, expected[i])
		}
	}
}

func TestHandleSMembersWrongArgs(t *testing.T) {
	ResetSetStore()
	srv := createTestServer(t)

	result := srv.handleSMembers(makeArgs())
	if result.Type != resp.TypeError {
		t.Errorf("Expected error, got type %c", result.Type)
	}

	result = srv.handleSMembers(makeArgs("key1", "key2"))
	if result.Type != resp.TypeError {
		t.Errorf("Expected error, got type %c", result.Type)
	}
}

func TestHandleSMembersWrongType(t *testing.T) {
	ResetSetStore()
	srv := createTestServer(t)

	srv.store.Set("stringkey", "value")

	result := srv.handleSMembers(makeArgs("stringkey"))
	if result.Type != resp.TypeError {
		t.Errorf("Expected error type, got %c", result.Type)
	}
}

func TestHandleSIsMember(t *testing.T) {
	ResetSetStore()
	srv := createTestServer(t)

	setStore.SAdd("myset", "a", "b", "c")

	tests := []struct {
		name    string
		args    []string
		wantNum int
		wantErr bool
	}{
		{"member exists", []string{"myset", "a"}, 1, false},
		{"member not exists", []string{"myset", "d"}, 0, false},
		{"key not exists", []string{"nokey", "a"}, 0, false},
		{"wrong args - one", []string{"myset"}, 0, true},
		{"wrong args - three", []string{"myset", "a", "b"}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := srv.handleSIsMember(makeArgs(tt.args...))

			if tt.wantErr {
				if result.Type != resp.TypeError {
					t.Errorf("Expected error, got type %c", result.Type)
				}
				return
			}

			if result.Type != resp.TypeInteger {
				t.Errorf("Type = %c, want integer", result.Type)
			}
			if result.Num != tt.wantNum {
				t.Errorf("Num = %d, want %d", result.Num, tt.wantNum)
			}
		})
	}
}

func TestHandleSIsMemberWrongType(t *testing.T) {
	ResetSetStore()
	srv := createTestServer(t)

	srv.store.Set("stringkey", "value")

	result := srv.handleSIsMember(makeArgs("stringkey", "member"))
	if result.Type != resp.TypeError {
		t.Errorf("Expected error type, got %c", result.Type)
	}
}

func TestHandleSCard(t *testing.T) {
	ResetSetStore()
	srv := createTestServer(t)

	// Non-existent key
	result := srv.handleSCard(makeArgs("nonexistent"))
	if result.Type != resp.TypeInteger || result.Num != 0 {
		t.Errorf("SCard(nonexistent) = %d, want 0", result.Num)
	}

	// Set with members
	setStore.SAdd("myset", "a", "b", "c")
	result = srv.handleSCard(makeArgs("myset"))
	if result.Type != resp.TypeInteger || result.Num != 3 {
		t.Errorf("SCard(myset) = %d, want 3", result.Num)
	}
}

func TestHandleSCardWrongArgs(t *testing.T) {
	ResetSetStore()
	srv := createTestServer(t)

	result := srv.handleSCard(makeArgs())
	if result.Type != resp.TypeError {
		t.Errorf("Expected error, got type %c", result.Type)
	}

	result = srv.handleSCard(makeArgs("key1", "key2"))
	if result.Type != resp.TypeError {
		t.Errorf("Expected error, got type %c", result.Type)
	}
}

func TestHandleSCardWrongType(t *testing.T) {
	ResetSetStore()
	srv := createTestServer(t)

	srv.store.Set("stringkey", "value")

	result := srv.handleSCard(makeArgs("stringkey"))
	if result.Type != resp.TypeError {
		t.Errorf("Expected error type, got %c", result.Type)
	}
}

func TestHandleSInter(t *testing.T) {
	ResetSetStore()
	srv := createTestServer(t)

	setStore.SAdd("set1", "a", "b", "c", "d")
	setStore.SAdd("set2", "b", "c", "d", "e")
	setStore.SAdd("set3", "c", "d", "e", "f")

	tests := []struct {
		name     string
		keys     []string
		expected []string
	}{
		{"single key", []string{"set1"}, []string{"a", "b", "c", "d"}},
		{"two keys", []string{"set1", "set2"}, []string{"b", "c", "d"}},
		{"three keys", []string{"set1", "set2", "set3"}, []string{"c", "d"}},
		{"with non-existent", []string{"set1", "nonexistent"}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := srv.handleSInter(makeArgs(tt.keys...))

			if result.Type != resp.TypeArray {
				t.Errorf("Type = %c, want array", result.Type)
				return
			}

			members := make([]string, len(result.Array))
			for i, v := range result.Array {
				members[i] = v.Str
			}
			sort.Strings(members)
			sort.Strings(tt.expected)

			if len(members) != len(tt.expected) {
				t.Errorf("Got %d members, want %d: %v", len(members), len(tt.expected), members)
				return
			}

			for i, m := range members {
				if m != tt.expected[i] {
					t.Errorf("Member[%d] = %q, want %q", i, m, tt.expected[i])
				}
			}
		})
	}
}

func TestHandleSInterWrongArgs(t *testing.T) {
	ResetSetStore()
	srv := createTestServer(t)

	result := srv.handleSInter(makeArgs())
	if result.Type != resp.TypeError {
		t.Errorf("Expected error, got type %c", result.Type)
	}
}

func TestHandleSInterWrongType(t *testing.T) {
	ResetSetStore()
	srv := createTestServer(t)

	setStore.SAdd("set1", "a", "b")
	srv.store.Set("stringkey", "value")

	result := srv.handleSInter(makeArgs("set1", "stringkey"))
	if result.Type != resp.TypeError {
		t.Errorf("Expected error type, got %c", result.Type)
	}
}

func TestSetCommandsWithEmptyString(t *testing.T) {
	ResetSetStore()
	srv := createTestServer(t)

	// Add empty string as member
	result := srv.handleSAdd(makeArgs("myset", ""))
	if result.Type != resp.TypeInteger || result.Num != 1 {
		t.Errorf("SAdd with empty string failed")
	}

	// Check membership
	result = srv.handleSIsMember(makeArgs("myset", ""))
	if result.Type != resp.TypeInteger || result.Num != 1 {
		t.Errorf("SIsMember with empty string = %d, want 1", result.Num)
	}

	// Get members
	result = srv.handleSMembers(makeArgs("myset"))
	if len(result.Array) != 1 || result.Array[0].Str != "" {
		t.Errorf("SMembers did not return empty string member")
	}
}

func TestSetAutoDeleteOnEmpty(t *testing.T) {
	ResetSetStore()
	srv := createTestServer(t)

	// Add and remove all members
	srv.handleSAdd(makeArgs("myset", "a", "b"))
	srv.handleSRem(makeArgs("myset", "a", "b"))

	// Set should be auto-deleted
	result := srv.handleSCard(makeArgs("myset"))
	if result.Num != 0 {
		t.Errorf("SCard after removing all members = %d, want 0", result.Num)
	}

	result = srv.handleSMembers(makeArgs("myset"))
	if len(result.Array) != 0 {
		t.Errorf("SMembers after removing all members returned %d, want 0", len(result.Array))
	}
}
