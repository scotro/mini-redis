package store

import (
	"sort"
	"testing"
)

func TestHSet(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(s HashStore)
		key         string
		fieldValues []string
		wantNew     int
		wantLen     int
	}{
		{
			name:        "set single field on new key",
			key:         "myhash",
			fieldValues: []string{"field1", "value1"},
			wantNew:     1,
			wantLen:     1,
		},
		{
			name:        "set multiple fields on new key",
			key:         "myhash",
			fieldValues: []string{"field1", "value1", "field2", "value2"},
			wantNew:     2,
			wantLen:     2,
		},
		{
			name: "update existing field",
			setup: func(s HashStore) {
				s.HSet("myhash", "field1", "value1")
			},
			key:         "myhash",
			fieldValues: []string{"field1", "newvalue"},
			wantNew:     0,
			wantLen:     1,
		},
		{
			name: "mix of new and existing fields",
			setup: func(s HashStore) {
				s.HSet("myhash", "field1", "value1")
			},
			key:         "myhash",
			fieldValues: []string{"field1", "newvalue", "field2", "value2"},
			wantNew:     1,
			wantLen:     2,
		},
		{
			name:        "odd number of arguments returns 0",
			key:         "myhash",
			fieldValues: []string{"field1"},
			wantNew:     0,
			wantLen:     0,
		},
		{
			name:        "empty arguments returns 0",
			key:         "myhash",
			fieldValues: []string{},
			wantNew:     0,
			wantLen:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewHashStore()
			if tt.setup != nil {
				tt.setup(s)
			}

			got := s.HSet(tt.key, tt.fieldValues...)
			if got != tt.wantNew {
				t.Errorf("HSet() = %d, want %d", got, tt.wantNew)
			}

			if gotLen := s.HLen(tt.key); gotLen != tt.wantLen {
				t.Errorf("HLen() after HSet = %d, want %d", gotLen, tt.wantLen)
			}
		})
	}
}

func TestHGet(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(s HashStore)
		key       string
		field     string
		wantValue string
		wantOk    bool
	}{
		{
			name: "get existing field",
			setup: func(s HashStore) {
				s.HSet("myhash", "field1", "value1")
			},
			key:       "myhash",
			field:     "field1",
			wantValue: "value1",
			wantOk:    true,
		},
		{
			name: "get non-existent field",
			setup: func(s HashStore) {
				s.HSet("myhash", "field1", "value1")
			},
			key:       "myhash",
			field:     "nonexistent",
			wantValue: "",
			wantOk:    false,
		},
		{
			name:      "get from non-existent key",
			key:       "nonexistent",
			field:     "field1",
			wantValue: "",
			wantOk:    false,
		},
		{
			name: "get updated value",
			setup: func(s HashStore) {
				s.HSet("myhash", "field1", "value1")
				s.HSet("myhash", "field1", "value2")
			},
			key:       "myhash",
			field:     "field1",
			wantValue: "value2",
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewHashStore()
			if tt.setup != nil {
				tt.setup(s)
			}

			gotValue, gotOk := s.HGet(tt.key, tt.field)
			if gotValue != tt.wantValue {
				t.Errorf("HGet() value = %q, want %q", gotValue, tt.wantValue)
			}
			if gotOk != tt.wantOk {
				t.Errorf("HGet() ok = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestHDel(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(s HashStore)
		key         string
		fields      []string
		wantDeleted int
		wantLen     int
	}{
		{
			name: "delete single field",
			setup: func(s HashStore) {
				s.HSet("myhash", "field1", "value1", "field2", "value2")
			},
			key:         "myhash",
			fields:      []string{"field1"},
			wantDeleted: 1,
			wantLen:     1,
		},
		{
			name: "delete multiple fields",
			setup: func(s HashStore) {
				s.HSet("myhash", "field1", "value1", "field2", "value2", "field3", "value3")
			},
			key:         "myhash",
			fields:      []string{"field1", "field2"},
			wantDeleted: 2,
			wantLen:     1,
		},
		{
			name: "delete non-existent field",
			setup: func(s HashStore) {
				s.HSet("myhash", "field1", "value1")
			},
			key:         "myhash",
			fields:      []string{"nonexistent"},
			wantDeleted: 0,
			wantLen:     1,
		},
		{
			name:        "delete from non-existent key",
			key:         "nonexistent",
			fields:      []string{"field1"},
			wantDeleted: 0,
			wantLen:     0,
		},
		{
			name: "delete all fields removes key",
			setup: func(s HashStore) {
				s.HSet("myhash", "field1", "value1")
			},
			key:         "myhash",
			fields:      []string{"field1"},
			wantDeleted: 1,
			wantLen:     0,
		},
		{
			name: "mix of existing and non-existent fields",
			setup: func(s HashStore) {
				s.HSet("myhash", "field1", "value1", "field2", "value2")
			},
			key:         "myhash",
			fields:      []string{"field1", "nonexistent"},
			wantDeleted: 1,
			wantLen:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewHashStore()
			if tt.setup != nil {
				tt.setup(s)
			}

			got := s.HDel(tt.key, tt.fields...)
			if got != tt.wantDeleted {
				t.Errorf("HDel() = %d, want %d", got, tt.wantDeleted)
			}

			if gotLen := s.HLen(tt.key); gotLen != tt.wantLen {
				t.Errorf("HLen() after HDel = %d, want %d", gotLen, tt.wantLen)
			}
		})
	}
}

func TestHGetAll(t *testing.T) {
	tests := []struct {
		name  string
		setup func(s HashStore)
		key   string
		want  map[string]string
	}{
		{
			name: "get all from hash",
			setup: func(s HashStore) {
				s.HSet("myhash", "field1", "value1", "field2", "value2")
			},
			key:  "myhash",
			want: map[string]string{"field1": "value1", "field2": "value2"},
		},
		{
			name:  "get all from non-existent key",
			key:   "nonexistent",
			want:  map[string]string{},
		},
		{
			name: "get all from empty hash after deletes",
			setup: func(s HashStore) {
				s.HSet("myhash", "field1", "value1")
				s.HDel("myhash", "field1")
			},
			key:  "myhash",
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewHashStore()
			if tt.setup != nil {
				tt.setup(s)
			}

			got := s.HGetAll(tt.key)
			if len(got) != len(tt.want) {
				t.Errorf("HGetAll() length = %d, want %d", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("HGetAll()[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestHKeys(t *testing.T) {
	tests := []struct {
		name  string
		setup func(s HashStore)
		key   string
		want  []string
	}{
		{
			name: "get keys from hash",
			setup: func(s HashStore) {
				s.HSet("myhash", "field1", "value1", "field2", "value2")
			},
			key:  "myhash",
			want: []string{"field1", "field2"},
		},
		{
			name:  "get keys from non-existent key",
			key:   "nonexistent",
			want:  []string{},
		},
		{
			name: "get keys after partial delete",
			setup: func(s HashStore) {
				s.HSet("myhash", "field1", "value1", "field2", "value2")
				s.HDel("myhash", "field1")
			},
			key:  "myhash",
			want: []string{"field2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewHashStore()
			if tt.setup != nil {
				tt.setup(s)
			}

			got := s.HKeys(tt.key)
			sort.Strings(got)
			sort.Strings(tt.want)

			if len(got) != len(tt.want) {
				t.Errorf("HKeys() length = %d, want %d", len(got), len(tt.want))
			}
			for i, v := range tt.want {
				if i >= len(got) || got[i] != v {
					t.Errorf("HKeys()[%d] = %q, want %q", i, got[i], v)
				}
			}
		})
	}
}

func TestHLen(t *testing.T) {
	tests := []struct {
		name  string
		setup func(s HashStore)
		key   string
		want  int
	}{
		{
			name:  "len of non-existent key",
			key:   "nonexistent",
			want:  0,
		},
		{
			name: "len of hash with one field",
			setup: func(s HashStore) {
				s.HSet("myhash", "field1", "value1")
			},
			key:  "myhash",
			want: 1,
		},
		{
			name: "len of hash with multiple fields",
			setup: func(s HashStore) {
				s.HSet("myhash", "field1", "value1", "field2", "value2", "field3", "value3")
			},
			key:  "myhash",
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewHashStore()
			if tt.setup != nil {
				tt.setup(s)
			}

			got := s.HLen(tt.key)
			if got != tt.want {
				t.Errorf("HLen() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestHashStoreKeyType(t *testing.T) {
	tests := []struct {
		name  string
		setup func(s HashStore)
		key   string
		want  string
	}{
		{
			name:  "type of non-existent key",
			key:   "nonexistent",
			want:  "none",
		},
		{
			name: "type of hash key",
			setup: func(s HashStore) {
				s.HSet("myhash", "field1", "value1")
			},
			key:  "myhash",
			want: "hash",
		},
		{
			name: "type after hash is deleted",
			setup: func(s HashStore) {
				s.HSet("myhash", "field1", "value1")
				s.HDel("myhash", "field1")
			},
			key:  "myhash",
			want: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewHashStore()
			if tt.setup != nil {
				tt.setup(s)
			}

			got := s.KeyType(tt.key)
			if got != tt.want {
				t.Errorf("KeyType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHGetAllReturnsCopy(t *testing.T) {
	s := NewHashStore()
	s.HSet("myhash", "field1", "value1")

	result := s.HGetAll("myhash")
	result["field1"] = "modified"

	value, ok := s.HGet("myhash", "field1")
	if !ok || value != "value1" {
		t.Errorf("HGetAll() did not return a copy, original was modified")
	}
}
