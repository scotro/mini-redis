package server

import (
	"testing"

	"github.com/scotro/mini-redis/internal/resp"
	"github.com/scotro/mini-redis/internal/store"
)

func newTestListHandler() *ListCommandHandler {
	return NewListCommandHandler(store.NewListStore(), store.New())
}

func makeListArgs(strs ...string) []resp.Value {
	args := make([]resp.Value, len(strs))
	for i, s := range strs {
		args[i] = resp.Value{Type: resp.TypeBulkString, Str: s}
	}
	return args
}

func TestHandleLPush(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantType byte
		wantNum  int
		wantErr  bool
	}{
		{
			name:     "single value",
			args:     []string{"mylist", "a"},
			wantType: resp.TypeInteger,
			wantNum:  1,
		},
		{
			name:     "multiple values",
			args:     []string{"mylist", "a", "b", "c"},
			wantType: resp.TypeInteger,
			wantNum:  3,
		},
		{
			name:     "no values",
			args:     []string{"mylist"},
			wantType: resp.TypeError,
			wantErr:  true,
		},
		{
			name:     "no arguments",
			args:     []string{},
			wantType: resp.TypeError,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestListHandler()
			got := h.HandleLPush(makeListArgs(tt.args...))

			if got.Type != tt.wantType {
				t.Errorf("HandleLPush() type = %c, want %c", got.Type, tt.wantType)
			}
			if !tt.wantErr && got.Num != tt.wantNum {
				t.Errorf("HandleLPush() num = %d, want %d", got.Num, tt.wantNum)
			}
		})
	}
}

func TestHandleLPushOrder(t *testing.T) {
	h := newTestListHandler()

	// LPUSH mylist a b c should result in [c, b, a]
	h.HandleLPush(makeListArgs("mylist", "a", "b", "c"))

	result := h.HandleLRange(makeListArgs("mylist", "0", "-1"))
	if result.Type != resp.TypeArray {
		t.Fatalf("Expected array, got %c", result.Type)
	}

	want := []string{"c", "b", "a"}
	if len(result.Array) != len(want) {
		t.Fatalf("Expected %d elements, got %d", len(want), len(result.Array))
	}

	for i, v := range result.Array {
		if v.Str != want[i] {
			t.Errorf("Element %d = %q, want %q", i, v.Str, want[i])
		}
	}
}

func TestHandleRPush(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantType byte
		wantNum  int
		wantErr  bool
	}{
		{
			name:     "single value",
			args:     []string{"mylist", "a"},
			wantType: resp.TypeInteger,
			wantNum:  1,
		},
		{
			name:     "multiple values",
			args:     []string{"mylist", "a", "b", "c"},
			wantType: resp.TypeInteger,
			wantNum:  3,
		},
		{
			name:     "no values",
			args:     []string{"mylist"},
			wantType: resp.TypeError,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestListHandler()
			got := h.HandleRPush(makeListArgs(tt.args...))

			if got.Type != tt.wantType {
				t.Errorf("HandleRPush() type = %c, want %c", got.Type, tt.wantType)
			}
			if !tt.wantErr && got.Num != tt.wantNum {
				t.Errorf("HandleRPush() num = %d, want %d", got.Num, tt.wantNum)
			}
		})
	}
}

func TestHandleLPop(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(h *ListCommandHandler)
		args     []string
		wantType byte
		wantStr  string
		wantNull bool
		wantErr  bool
	}{
		{
			name:     "pop from non-existent list",
			setup:    func(h *ListCommandHandler) {},
			args:     []string{"mylist"},
			wantType: resp.TypeBulkString,
			wantNull: true,
		},
		{
			name: "pop from list with elements",
			setup: func(h *ListCommandHandler) {
				h.HandleRPush(makeListArgs("mylist", "a", "b", "c"))
			},
			args:     []string{"mylist"},
			wantType: resp.TypeBulkString,
			wantStr:  "a",
		},
		{
			name:     "wrong number of arguments",
			setup:    func(h *ListCommandHandler) {},
			args:     []string{},
			wantType: resp.TypeError,
			wantErr:  true,
		},
		{
			name:     "too many arguments",
			setup:    func(h *ListCommandHandler) {},
			args:     []string{"mylist", "extra"},
			wantType: resp.TypeError,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestListHandler()
			tt.setup(h)

			got := h.HandleLPop(makeListArgs(tt.args...))

			if got.Type != tt.wantType {
				t.Errorf("HandleLPop() type = %c, want %c", got.Type, tt.wantType)
			}
			if !tt.wantErr && !tt.wantNull && got.Str != tt.wantStr {
				t.Errorf("HandleLPop() str = %q, want %q", got.Str, tt.wantStr)
			}
			if tt.wantNull && !got.Null {
				t.Errorf("HandleLPop() null = false, want true")
			}
		})
	}
}

func TestHandleRPop(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(h *ListCommandHandler)
		args     []string
		wantType byte
		wantStr  string
		wantNull bool
		wantErr  bool
	}{
		{
			name:     "pop from non-existent list",
			setup:    func(h *ListCommandHandler) {},
			args:     []string{"mylist"},
			wantType: resp.TypeBulkString,
			wantNull: true,
		},
		{
			name: "pop from list with elements",
			setup: func(h *ListCommandHandler) {
				h.HandleRPush(makeListArgs("mylist", "a", "b", "c"))
			},
			args:     []string{"mylist"},
			wantType: resp.TypeBulkString,
			wantStr:  "c",
		},
		{
			name:     "wrong number of arguments",
			setup:    func(h *ListCommandHandler) {},
			args:     []string{},
			wantType: resp.TypeError,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestListHandler()
			tt.setup(h)

			got := h.HandleRPop(makeListArgs(tt.args...))

			if got.Type != tt.wantType {
				t.Errorf("HandleRPop() type = %c, want %c", got.Type, tt.wantType)
			}
			if !tt.wantErr && !tt.wantNull && got.Str != tt.wantStr {
				t.Errorf("HandleRPop() str = %q, want %q", got.Str, tt.wantStr)
			}
			if tt.wantNull && !got.Null {
				t.Errorf("HandleRPop() null = false, want true")
			}
		})
	}
}

func TestHandleLRange(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(h *ListCommandHandler)
		args     []string
		wantType byte
		want     []string
		wantErr  bool
	}{
		{
			name:     "range on non-existent list",
			setup:    func(h *ListCommandHandler) {},
			args:     []string{"mylist", "0", "-1"},
			wantType: resp.TypeArray,
			want:     []string{},
		},
		{
			name: "full range",
			setup: func(h *ListCommandHandler) {
				h.HandleRPush(makeListArgs("mylist", "a", "b", "c", "d", "e"))
			},
			args:     []string{"mylist", "0", "-1"},
			wantType: resp.TypeArray,
			want:     []string{"a", "b", "c", "d", "e"},
		},
		{
			name: "partial range",
			setup: func(h *ListCommandHandler) {
				h.HandleRPush(makeListArgs("mylist", "a", "b", "c", "d", "e"))
			},
			args:     []string{"mylist", "1", "3"},
			wantType: resp.TypeArray,
			want:     []string{"b", "c", "d"},
		},
		{
			name: "negative indices",
			setup: func(h *ListCommandHandler) {
				h.HandleRPush(makeListArgs("mylist", "a", "b", "c", "d", "e"))
			},
			args:     []string{"mylist", "-3", "-1"},
			wantType: resp.TypeArray,
			want:     []string{"c", "d", "e"},
		},
		{
			name:     "wrong number of arguments",
			setup:    func(h *ListCommandHandler) {},
			args:     []string{"mylist", "0"},
			wantType: resp.TypeError,
			wantErr:  true,
		},
		{
			name:     "invalid start index",
			setup:    func(h *ListCommandHandler) {},
			args:     []string{"mylist", "abc", "1"},
			wantType: resp.TypeError,
			wantErr:  true,
		},
		{
			name:     "invalid stop index",
			setup:    func(h *ListCommandHandler) {},
			args:     []string{"mylist", "0", "xyz"},
			wantType: resp.TypeError,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestListHandler()
			tt.setup(h)

			got := h.HandleLRange(makeListArgs(tt.args...))

			if got.Type != tt.wantType {
				t.Errorf("HandleLRange() type = %c, want %c", got.Type, tt.wantType)
				return
			}

			if tt.wantErr {
				return
			}

			if len(got.Array) != len(tt.want) {
				t.Errorf("HandleLRange() len = %d, want %d", len(got.Array), len(tt.want))
				return
			}

			for i, v := range got.Array {
				if v.Str != tt.want[i] {
					t.Errorf("HandleLRange()[%d] = %q, want %q", i, v.Str, tt.want[i])
				}
			}
		})
	}
}

func TestHandleLLen(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(h *ListCommandHandler)
		args     []string
		wantType byte
		wantNum  int
		wantErr  bool
	}{
		{
			name:     "len of non-existent list",
			setup:    func(h *ListCommandHandler) {},
			args:     []string{"mylist"},
			wantType: resp.TypeInteger,
			wantNum:  0,
		},
		{
			name: "len of list with elements",
			setup: func(h *ListCommandHandler) {
				h.HandleRPush(makeListArgs("mylist", "a", "b", "c"))
			},
			args:     []string{"mylist"},
			wantType: resp.TypeInteger,
			wantNum:  3,
		},
		{
			name:     "wrong number of arguments",
			setup:    func(h *ListCommandHandler) {},
			args:     []string{},
			wantType: resp.TypeError,
			wantErr:  true,
		},
		{
			name:     "too many arguments",
			setup:    func(h *ListCommandHandler) {},
			args:     []string{"mylist", "extra"},
			wantType: resp.TypeError,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestListHandler()
			tt.setup(h)

			got := h.HandleLLen(makeListArgs(tt.args...))

			if got.Type != tt.wantType {
				t.Errorf("HandleLLen() type = %c, want %c", got.Type, tt.wantType)
			}
			if !tt.wantErr && got.Num != tt.wantNum {
				t.Errorf("HandleLLen() num = %d, want %d", got.Num, tt.wantNum)
			}
		})
	}
}

func TestWrongTypeError(t *testing.T) {
	// Create a handler with a string store that has a key
	listStore := store.NewListStore()
	stringStore := store.New()
	defer stringStore.Close()

	stringStore.Set("stringkey", "value")

	h := NewListCommandHandler(listStore, stringStore)

	tests := []struct {
		name    string
		handler func(args []resp.Value) resp.Value
		args    []string
	}{
		{
			name:    "LPUSH on string key",
			handler: h.HandleLPush,
			args:    []string{"stringkey", "value"},
		},
		{
			name:    "RPUSH on string key",
			handler: h.HandleRPush,
			args:    []string{"stringkey", "value"},
		},
		{
			name:    "LPOP on string key",
			handler: h.HandleLPop,
			args:    []string{"stringkey"},
		},
		{
			name:    "RPOP on string key",
			handler: h.HandleRPop,
			args:    []string{"stringkey"},
		},
		{
			name:    "LRANGE on string key",
			handler: h.HandleLRange,
			args:    []string{"stringkey", "0", "-1"},
		},
		{
			name:    "LLEN on string key",
			handler: h.HandleLLen,
			args:    []string{"stringkey"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.handler(makeListArgs(tt.args...))

			if got.Type != resp.TypeError {
				t.Errorf("Expected error type, got %c", got.Type)
			}
			if got.Str != "WRONGTYPE Operation against a key holding the wrong kind of value" {
				t.Errorf("Expected WRONGTYPE error, got %q", got.Str)
			}
		})
	}
}

func TestStackBehavior(t *testing.T) {
	h := newTestListHandler()

	// Use LPUSH and LPOP as a stack (LIFO)
	h.HandleLPush(makeListArgs("stack", "1", "2", "3"))

	// Should pop in order: 3, 2, 1
	expected := []string{"3", "2", "1"}
	for _, want := range expected {
		got := h.HandleLPop(makeListArgs("stack"))
		if got.Str != want {
			t.Errorf("LPOP = %q, want %q", got.Str, want)
		}
	}

	// Stack should be empty now
	got := h.HandleLPop(makeListArgs("stack"))
	if !got.Null {
		t.Errorf("Expected null after emptying stack, got %q", got.Str)
	}
}

func TestQueueBehavior(t *testing.T) {
	h := newTestListHandler()

	// Use RPUSH and LPOP as a queue (FIFO)
	h.HandleRPush(makeListArgs("queue", "1", "2", "3"))

	// Should pop in order: 1, 2, 3
	expected := []string{"1", "2", "3"}
	for _, want := range expected {
		got := h.HandleLPop(makeListArgs("queue"))
		if got.Str != want {
			t.Errorf("LPOP = %q, want %q", got.Str, want)
		}
	}

	// Queue should be empty now
	got := h.HandleLPop(makeListArgs("queue"))
	if !got.Null {
		t.Errorf("Expected null after emptying queue, got %q", got.Str)
	}
}
