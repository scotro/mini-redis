package store

import (
	"sync"
	"testing"
)

func TestLPush(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		values   []string
		wantLen  int
		wantList []string
	}{
		{
			name:     "push single value to new list",
			key:      "mylist",
			values:   []string{"a"},
			wantLen:  1,
			wantList: []string{"a"},
		},
		{
			name:     "push multiple values to new list",
			key:      "mylist",
			values:   []string{"a", "b", "c"},
			wantLen:  3,
			wantList: []string{"c", "b", "a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewListStore()
			gotLen := s.LPush(tt.key, tt.values...)
			if gotLen != tt.wantLen {
				t.Errorf("LPush() = %d, want %d", gotLen, tt.wantLen)
			}

			gotList := s.LRange(tt.key, 0, -1)
			if len(gotList) != len(tt.wantList) {
				t.Errorf("LRange() len = %d, want %d", len(gotList), len(tt.wantList))
				return
			}
			for i, v := range gotList {
				if v != tt.wantList[i] {
					t.Errorf("LRange()[%d] = %q, want %q", i, v, tt.wantList[i])
				}
			}
		})
	}
}

func TestLPushToExistingList(t *testing.T) {
	s := NewListStore()
	s.RPush("mylist", "x", "y")
	gotLen := s.LPush("mylist", "a", "b")
	if gotLen != 4 {
		t.Errorf("LPush() = %d, want 4", gotLen)
	}

	gotList := s.LRange("mylist", 0, -1)
	want := []string{"b", "a", "x", "y"}
	if len(gotList) != len(want) {
		t.Errorf("LRange() len = %d, want %d", len(gotList), len(want))
		return
	}
	for i, v := range gotList {
		if v != want[i] {
			t.Errorf("LRange()[%d] = %q, want %q", i, v, want[i])
		}
	}
}

func TestRPush(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		values   []string
		wantLen  int
		wantList []string
	}{
		{
			name:     "push single value to new list",
			key:      "mylist",
			values:   []string{"a"},
			wantLen:  1,
			wantList: []string{"a"},
		},
		{
			name:     "push multiple values to new list",
			key:      "mylist",
			values:   []string{"a", "b", "c"},
			wantLen:  3,
			wantList: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewListStore()
			gotLen := s.RPush(tt.key, tt.values...)
			if gotLen != tt.wantLen {
				t.Errorf("RPush() = %d, want %d", gotLen, tt.wantLen)
			}

			gotList := s.LRange(tt.key, 0, -1)
			if len(gotList) != len(tt.wantList) {
				t.Errorf("LRange() len = %d, want %d", len(gotList), len(tt.wantList))
				return
			}
			for i, v := range gotList {
				if v != tt.wantList[i] {
					t.Errorf("LRange()[%d] = %q, want %q", i, v, tt.wantList[i])
				}
			}
		})
	}
}

func TestRPushToExistingList(t *testing.T) {
	s := NewListStore()
	s.RPush("mylist", "a", "b")
	gotLen := s.RPush("mylist", "c", "d")
	if gotLen != 4 {
		t.Errorf("RPush() = %d, want 4", gotLen)
	}

	gotList := s.LRange("mylist", 0, -1)
	want := []string{"a", "b", "c", "d"}
	if len(gotList) != len(want) {
		t.Errorf("LRange() len = %d, want %d", len(gotList), len(want))
		return
	}
	for i, v := range gotList {
		if v != want[i] {
			t.Errorf("LRange()[%d] = %q, want %q", i, v, want[i])
		}
	}
}

func TestLPop(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(s ListStore)
		key       string
		wantValue string
		wantOk    bool
		wantLen   int
	}{
		{
			name:      "pop from non-existent list",
			setup:     func(s ListStore) {},
			key:       "mylist",
			wantValue: "",
			wantOk:    false,
			wantLen:   0,
		},
		{
			name: "pop from single-element list",
			setup: func(s ListStore) {
				s.RPush("mylist", "a")
			},
			key:       "mylist",
			wantValue: "a",
			wantOk:    true,
			wantLen:   0,
		},
		{
			name: "pop from multi-element list",
			setup: func(s ListStore) {
				s.RPush("mylist", "a", "b", "c")
			},
			key:       "mylist",
			wantValue: "a",
			wantOk:    true,
			wantLen:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewListStore()
			tt.setup(s)

			gotValue, gotOk := s.LPop(tt.key)
			if gotValue != tt.wantValue {
				t.Errorf("LPop() value = %q, want %q", gotValue, tt.wantValue)
			}
			if gotOk != tt.wantOk {
				t.Errorf("LPop() ok = %v, want %v", gotOk, tt.wantOk)
			}

			gotLen := s.LLen(tt.key)
			if gotLen != tt.wantLen {
				t.Errorf("LLen() after LPop = %d, want %d", gotLen, tt.wantLen)
			}
		})
	}
}

func TestRPop(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(s ListStore)
		key       string
		wantValue string
		wantOk    bool
		wantLen   int
	}{
		{
			name:      "pop from non-existent list",
			setup:     func(s ListStore) {},
			key:       "mylist",
			wantValue: "",
			wantOk:    false,
			wantLen:   0,
		},
		{
			name: "pop from single-element list",
			setup: func(s ListStore) {
				s.RPush("mylist", "a")
			},
			key:       "mylist",
			wantValue: "a",
			wantOk:    true,
			wantLen:   0,
		},
		{
			name: "pop from multi-element list",
			setup: func(s ListStore) {
				s.RPush("mylist", "a", "b", "c")
			},
			key:       "mylist",
			wantValue: "c",
			wantOk:    true,
			wantLen:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewListStore()
			tt.setup(s)

			gotValue, gotOk := s.RPop(tt.key)
			if gotValue != tt.wantValue {
				t.Errorf("RPop() value = %q, want %q", gotValue, tt.wantValue)
			}
			if gotOk != tt.wantOk {
				t.Errorf("RPop() ok = %v, want %v", gotOk, tt.wantOk)
			}

			gotLen := s.LLen(tt.key)
			if gotLen != tt.wantLen {
				t.Errorf("LLen() after RPop = %d, want %d", gotLen, tt.wantLen)
			}
		})
	}
}

func TestLRange(t *testing.T) {
	tests := []struct {
		name  string
		setup func(s ListStore)
		key   string
		start int
		stop  int
		want  []string
	}{
		{
			name:  "range on non-existent list",
			setup: func(s ListStore) {},
			key:   "mylist",
			start: 0,
			stop:  -1,
			want:  []string{},
		},
		{
			name: "full range with negative index",
			setup: func(s ListStore) {
				s.RPush("mylist", "a", "b", "c", "d", "e")
			},
			key:   "mylist",
			start: 0,
			stop:  -1,
			want:  []string{"a", "b", "c", "d", "e"},
		},
		{
			name: "partial range from start",
			setup: func(s ListStore) {
				s.RPush("mylist", "a", "b", "c", "d", "e")
			},
			key:   "mylist",
			start: 0,
			stop:  2,
			want:  []string{"a", "b", "c"},
		},
		{
			name: "partial range in middle",
			setup: func(s ListStore) {
				s.RPush("mylist", "a", "b", "c", "d", "e")
			},
			key:   "mylist",
			start: 1,
			stop:  3,
			want:  []string{"b", "c", "d"},
		},
		{
			name: "range with negative start",
			setup: func(s ListStore) {
				s.RPush("mylist", "a", "b", "c", "d", "e")
			},
			key:   "mylist",
			start: -3,
			stop:  -1,
			want:  []string{"c", "d", "e"},
		},
		{
			name: "range with both negative indices",
			setup: func(s ListStore) {
				s.RPush("mylist", "a", "b", "c", "d", "e")
			},
			key:   "mylist",
			start: -4,
			stop:  -2,
			want:  []string{"b", "c", "d"},
		},
		{
			name: "range exceeds list length",
			setup: func(s ListStore) {
				s.RPush("mylist", "a", "b", "c")
			},
			key:   "mylist",
			start: 0,
			stop:  100,
			want:  []string{"a", "b", "c"},
		},
		{
			name: "start exceeds list length",
			setup: func(s ListStore) {
				s.RPush("mylist", "a", "b", "c")
			},
			key:   "mylist",
			start: 100,
			stop:  200,
			want:  []string{},
		},
		{
			name: "start greater than stop",
			setup: func(s ListStore) {
				s.RPush("mylist", "a", "b", "c")
			},
			key:   "mylist",
			start: 2,
			stop:  0,
			want:  []string{},
		},
		{
			name: "single element range",
			setup: func(s ListStore) {
				s.RPush("mylist", "a", "b", "c")
			},
			key:   "mylist",
			start: 1,
			stop:  1,
			want:  []string{"b"},
		},
		{
			name: "negative start before beginning",
			setup: func(s ListStore) {
				s.RPush("mylist", "a", "b", "c")
			},
			key:   "mylist",
			start: -100,
			stop:  1,
			want:  []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewListStore()
			tt.setup(s)

			got := s.LRange(tt.key, tt.start, tt.stop)
			if len(got) != len(tt.want) {
				t.Errorf("LRange() len = %d, want %d (got: %v, want: %v)", len(got), len(tt.want), got, tt.want)
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("LRange()[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestLLen(t *testing.T) {
	tests := []struct {
		name  string
		setup func(s ListStore)
		key   string
		want  int
	}{
		{
			name:  "len of non-existent list",
			setup: func(s ListStore) {},
			key:   "mylist",
			want:  0,
		},
		{
			name: "len of list with elements",
			setup: func(s ListStore) {
				s.RPush("mylist", "a", "b", "c")
			},
			key:  "mylist",
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewListStore()
			tt.setup(s)

			got := s.LLen(tt.key)
			if got != tt.want {
				t.Errorf("LLen() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestListStoreKeyType(t *testing.T) {
	tests := []struct {
		name  string
		setup func(s ListStore)
		key   string
		want  string
	}{
		{
			name:  "non-existent key",
			setup: func(s ListStore) {},
			key:   "mylist",
			want:  "none",
		},
		{
			name: "existing list key",
			setup: func(s ListStore) {
				s.RPush("mylist", "a")
			},
			key:  "mylist",
			want: "list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewListStore()
			tt.setup(s)

			got := s.KeyType(tt.key)
			if got != tt.want {
				t.Errorf("KeyType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEmptyListDeletion(t *testing.T) {
	s := NewListStore()
	s.RPush("mylist", "a")

	_, ok := s.LPop("mylist")
	if !ok {
		t.Fatal("LPop() should return ok=true")
	}

	// List should be deleted after becoming empty
	if s.KeyType("mylist") != "none" {
		t.Error("Empty list should be deleted")
	}

	// Same test with RPop
	s.RPush("mylist2", "a")
	_, ok = s.RPop("mylist2")
	if !ok {
		t.Fatal("RPop() should return ok=true")
	}

	if s.KeyType("mylist2") != "none" {
		t.Error("Empty list should be deleted after RPop")
	}
}

func TestListConcurrentAccess(t *testing.T) {
	s := NewListStore()
	var wg sync.WaitGroup

	// Concurrent pushes
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			s.LPush("mylist", "left")
		}()
		go func() {
			defer wg.Done()
			s.RPush("mylist", "right")
		}()
	}
	wg.Wait()

	if got := s.LLen("mylist"); got != 200 {
		t.Errorf("LLen() = %d, want 200", got)
	}

	// Concurrent pops
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			s.LPop("mylist")
		}()
		go func() {
			defer wg.Done()
			s.RPop("mylist")
		}()
	}
	wg.Wait()

	if got := s.LLen("mylist"); got != 0 {
		t.Errorf("LLen() after pops = %d, want 0", got)
	}
}
