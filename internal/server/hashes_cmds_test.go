package server

import (
	"strings"
	"testing"

	"github.com/scotro/mini-redis/internal/resp"
	"github.com/scotro/mini-redis/internal/store"
)

func newTestHashCommands() (*HashCommands, store.HashStore, store.Store) {
	hashStore := store.NewHashStore()
	stringStore := store.New()
	return NewHashCommands(hashStore, stringStore), hashStore, stringStore
}

func makeArgs(strs ...string) []resp.Value {
	args := make([]resp.Value, len(strs))
	for i, s := range strs {
		args[i] = resp.Value{Type: resp.TypeBulkString, Str: s}
	}
	return args
}

func TestHandleHSet(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(h *HashCommands, hs store.HashStore, ss store.Store)
		args        []string
		wantType    byte
		wantNum     int
		wantErr     string
	}{
		{
			name:     "set single field",
			args:     []string{"myhash", "field1", "value1"},
			wantType: resp.TypeInteger,
			wantNum:  1,
		},
		{
			name:     "set multiple fields",
			args:     []string{"myhash", "field1", "value1", "field2", "value2"},
			wantType: resp.TypeInteger,
			wantNum:  2,
		},
		{
			name: "update existing field returns 0",
			setup: func(h *HashCommands, hs store.HashStore, ss store.Store) {
				hs.HSet("myhash", "field1", "value1")
			},
			args:     []string{"myhash", "field1", "newvalue"},
			wantType: resp.TypeInteger,
			wantNum:  0,
		},
		{
			name:     "wrong number of arguments - too few",
			args:     []string{"myhash"},
			wantType: resp.TypeError,
			wantErr:  "ERR wrong number of arguments",
		},
		{
			name:     "wrong number of arguments - odd pairs",
			args:     []string{"myhash", "field1"},
			wantType: resp.TypeError,
			wantErr:  "ERR wrong number of arguments",
		},
		{
			name: "wrongtype error",
			setup: func(h *HashCommands, hs store.HashStore, ss store.Store) {
				ss.Set("myhash", "stringvalue")
			},
			args:     []string{"myhash", "field1", "value1"},
			wantType: resp.TypeError,
			wantErr:  "WRONGTYPE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, hs, ss := newTestHashCommands()
			defer ss.Close()

			if tt.setup != nil {
				tt.setup(h, hs, ss)
			}

			result := h.HandleHSet(makeArgs(tt.args...))

			if result.Type != tt.wantType {
				t.Errorf("HandleHSet() type = %c, want %c", result.Type, tt.wantType)
			}

			if tt.wantType == resp.TypeInteger && result.Num != tt.wantNum {
				t.Errorf("HandleHSet() num = %d, want %d", result.Num, tt.wantNum)
			}

			if tt.wantType == resp.TypeError && !strings.Contains(result.Str, tt.wantErr) {
				t.Errorf("HandleHSet() error = %q, want to contain %q", result.Str, tt.wantErr)
			}
		})
	}
}

func TestHandleHGet(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(h *HashCommands, hs store.HashStore, ss store.Store)
		args     []string
		wantType byte
		wantStr  string
		wantNull bool
		wantErr  string
	}{
		{
			name: "get existing field",
			setup: func(h *HashCommands, hs store.HashStore, ss store.Store) {
				hs.HSet("myhash", "field1", "value1")
			},
			args:     []string{"myhash", "field1"},
			wantType: resp.TypeBulkString,
			wantStr:  "value1",
		},
		{
			name:     "get non-existent key",
			args:     []string{"myhash", "field1"},
			wantType: resp.TypeBulkString,
			wantNull: true,
		},
		{
			name: "get non-existent field",
			setup: func(h *HashCommands, hs store.HashStore, ss store.Store) {
				hs.HSet("myhash", "field1", "value1")
			},
			args:     []string{"myhash", "field2"},
			wantType: resp.TypeBulkString,
			wantNull: true,
		},
		{
			name:     "wrong number of arguments",
			args:     []string{"myhash"},
			wantType: resp.TypeError,
			wantErr:  "ERR wrong number of arguments",
		},
		{
			name: "wrongtype error",
			setup: func(h *HashCommands, hs store.HashStore, ss store.Store) {
				ss.Set("myhash", "stringvalue")
			},
			args:     []string{"myhash", "field1"},
			wantType: resp.TypeError,
			wantErr:  "WRONGTYPE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, hs, ss := newTestHashCommands()
			defer ss.Close()

			if tt.setup != nil {
				tt.setup(h, hs, ss)
			}

			result := h.HandleHGet(makeArgs(tt.args...))

			if result.Type != tt.wantType {
				t.Errorf("HandleHGet() type = %c, want %c", result.Type, tt.wantType)
			}

			if tt.wantType == resp.TypeBulkString {
				if tt.wantNull && !result.Null {
					t.Errorf("HandleHGet() expected null")
				}
				if !tt.wantNull && result.Str != tt.wantStr {
					t.Errorf("HandleHGet() str = %q, want %q", result.Str, tt.wantStr)
				}
			}

			if tt.wantType == resp.TypeError && !strings.Contains(result.Str, tt.wantErr) {
				t.Errorf("HandleHGet() error = %q, want to contain %q", result.Str, tt.wantErr)
			}
		})
	}
}

func TestHandleHDel(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(h *HashCommands, hs store.HashStore, ss store.Store)
		args     []string
		wantType byte
		wantNum  int
		wantErr  string
	}{
		{
			name: "delete single field",
			setup: func(h *HashCommands, hs store.HashStore, ss store.Store) {
				hs.HSet("myhash", "field1", "value1", "field2", "value2")
			},
			args:     []string{"myhash", "field1"},
			wantType: resp.TypeInteger,
			wantNum:  1,
		},
		{
			name: "delete multiple fields",
			setup: func(h *HashCommands, hs store.HashStore, ss store.Store) {
				hs.HSet("myhash", "field1", "value1", "field2", "value2")
			},
			args:     []string{"myhash", "field1", "field2"},
			wantType: resp.TypeInteger,
			wantNum:  2,
		},
		{
			name:     "delete from non-existent key",
			args:     []string{"myhash", "field1"},
			wantType: resp.TypeInteger,
			wantNum:  0,
		},
		{
			name:     "wrong number of arguments",
			args:     []string{"myhash"},
			wantType: resp.TypeError,
			wantErr:  "ERR wrong number of arguments",
		},
		{
			name: "wrongtype error",
			setup: func(h *HashCommands, hs store.HashStore, ss store.Store) {
				ss.Set("myhash", "stringvalue")
			},
			args:     []string{"myhash", "field1"},
			wantType: resp.TypeError,
			wantErr:  "WRONGTYPE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, hs, ss := newTestHashCommands()
			defer ss.Close()

			if tt.setup != nil {
				tt.setup(h, hs, ss)
			}

			result := h.HandleHDel(makeArgs(tt.args...))

			if result.Type != tt.wantType {
				t.Errorf("HandleHDel() type = %c, want %c", result.Type, tt.wantType)
			}

			if tt.wantType == resp.TypeInteger && result.Num != tt.wantNum {
				t.Errorf("HandleHDel() num = %d, want %d", result.Num, tt.wantNum)
			}

			if tt.wantType == resp.TypeError && !strings.Contains(result.Str, tt.wantErr) {
				t.Errorf("HandleHDel() error = %q, want to contain %q", result.Str, tt.wantErr)
			}
		})
	}
}

func TestHandleHGetAll(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(h *HashCommands, hs store.HashStore, ss store.Store)
		args       []string
		wantType   byte
		wantLen    int
		wantFields map[string]string
		wantErr    string
	}{
		{
			name: "get all fields",
			setup: func(h *HashCommands, hs store.HashStore, ss store.Store) {
				hs.HSet("myhash", "field1", "value1", "field2", "value2")
			},
			args:       []string{"myhash"},
			wantType:   resp.TypeArray,
			wantLen:    4, // 2 fields * 2 (field + value)
			wantFields: map[string]string{"field1": "value1", "field2": "value2"},
		},
		{
			name:     "get all from non-existent key",
			args:     []string{"myhash"},
			wantType: resp.TypeArray,
			wantLen:  0,
		},
		{
			name:     "wrong number of arguments",
			args:     []string{"myhash", "extra"},
			wantType: resp.TypeError,
			wantErr:  "ERR wrong number of arguments",
		},
		{
			name: "wrongtype error",
			setup: func(h *HashCommands, hs store.HashStore, ss store.Store) {
				ss.Set("myhash", "stringvalue")
			},
			args:     []string{"myhash"},
			wantType: resp.TypeError,
			wantErr:  "WRONGTYPE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, hs, ss := newTestHashCommands()
			defer ss.Close()

			if tt.setup != nil {
				tt.setup(h, hs, ss)
			}

			result := h.HandleHGetAll(makeArgs(tt.args...))

			if result.Type != tt.wantType {
				t.Errorf("HandleHGetAll() type = %c, want %c", result.Type, tt.wantType)
			}

			if tt.wantType == resp.TypeArray {
				if len(result.Array) != tt.wantLen {
					t.Errorf("HandleHGetAll() array len = %d, want %d", len(result.Array), tt.wantLen)
				}

				// Verify field-value pairs
				if tt.wantFields != nil {
					gotFields := make(map[string]string)
					for i := 0; i < len(result.Array); i += 2 {
						gotFields[result.Array[i].Str] = result.Array[i+1].Str
					}
					for k, v := range tt.wantFields {
						if gotFields[k] != v {
							t.Errorf("HandleHGetAll() field %q = %q, want %q", k, gotFields[k], v)
						}
					}
				}
			}

			if tt.wantType == resp.TypeError && !strings.Contains(result.Str, tt.wantErr) {
				t.Errorf("HandleHGetAll() error = %q, want to contain %q", result.Str, tt.wantErr)
			}
		})
	}
}

func TestHandleHKeys(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(h *HashCommands, hs store.HashStore, ss store.Store)
		args     []string
		wantType byte
		wantKeys []string
		wantErr  string
	}{
		{
			name: "get keys",
			setup: func(h *HashCommands, hs store.HashStore, ss store.Store) {
				hs.HSet("myhash", "field1", "value1", "field2", "value2")
			},
			args:     []string{"myhash"},
			wantType: resp.TypeArray,
			wantKeys: []string{"field1", "field2"},
		},
		{
			name:     "get keys from non-existent key",
			args:     []string{"myhash"},
			wantType: resp.TypeArray,
			wantKeys: []string{},
		},
		{
			name:     "wrong number of arguments",
			args:     []string{"myhash", "extra"},
			wantType: resp.TypeError,
			wantErr:  "ERR wrong number of arguments",
		},
		{
			name: "wrongtype error",
			setup: func(h *HashCommands, hs store.HashStore, ss store.Store) {
				ss.Set("myhash", "stringvalue")
			},
			args:     []string{"myhash"},
			wantType: resp.TypeError,
			wantErr:  "WRONGTYPE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, hs, ss := newTestHashCommands()
			defer ss.Close()

			if tt.setup != nil {
				tt.setup(h, hs, ss)
			}

			result := h.HandleHKeys(makeArgs(tt.args...))

			if result.Type != tt.wantType {
				t.Errorf("HandleHKeys() type = %c, want %c", result.Type, tt.wantType)
			}

			if tt.wantType == resp.TypeArray {
				if len(result.Array) != len(tt.wantKeys) {
					t.Errorf("HandleHKeys() array len = %d, want %d", len(result.Array), len(tt.wantKeys))
				}

				// Check all expected keys are present (order not guaranteed)
				gotKeys := make(map[string]bool)
				for _, v := range result.Array {
					gotKeys[v.Str] = true
				}
				for _, k := range tt.wantKeys {
					if !gotKeys[k] {
						t.Errorf("HandleHKeys() missing key %q", k)
					}
				}
			}

			if tt.wantType == resp.TypeError && !strings.Contains(result.Str, tt.wantErr) {
				t.Errorf("HandleHKeys() error = %q, want to contain %q", result.Str, tt.wantErr)
			}
		})
	}
}

func TestHandleHLen(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(h *HashCommands, hs store.HashStore, ss store.Store)
		args     []string
		wantType byte
		wantNum  int
		wantErr  string
	}{
		{
			name: "get length",
			setup: func(h *HashCommands, hs store.HashStore, ss store.Store) {
				hs.HSet("myhash", "field1", "value1", "field2", "value2")
			},
			args:     []string{"myhash"},
			wantType: resp.TypeInteger,
			wantNum:  2,
		},
		{
			name:     "get length of non-existent key",
			args:     []string{"myhash"},
			wantType: resp.TypeInteger,
			wantNum:  0,
		},
		{
			name:     "wrong number of arguments",
			args:     []string{"myhash", "extra"},
			wantType: resp.TypeError,
			wantErr:  "ERR wrong number of arguments",
		},
		{
			name: "wrongtype error",
			setup: func(h *HashCommands, hs store.HashStore, ss store.Store) {
				ss.Set("myhash", "stringvalue")
			},
			args:     []string{"myhash"},
			wantType: resp.TypeError,
			wantErr:  "WRONGTYPE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, hs, ss := newTestHashCommands()
			defer ss.Close()

			if tt.setup != nil {
				tt.setup(h, hs, ss)
			}

			result := h.HandleHLen(makeArgs(tt.args...))

			if result.Type != tt.wantType {
				t.Errorf("HandleHLen() type = %c, want %c", result.Type, tt.wantType)
			}

			if tt.wantType == resp.TypeInteger && result.Num != tt.wantNum {
				t.Errorf("HandleHLen() num = %d, want %d", result.Num, tt.wantNum)
			}

			if tt.wantType == resp.TypeError && !strings.Contains(result.Str, tt.wantErr) {
				t.Errorf("HandleHLen() error = %q, want to contain %q", result.Str, tt.wantErr)
			}
		})
	}
}
