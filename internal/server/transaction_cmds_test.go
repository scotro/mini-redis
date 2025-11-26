package server

import (
	"strings"
	"testing"

	"github.com/scotro/mini-redis/internal/resp"
	"github.com/scotro/mini-redis/internal/transaction"
)

func newTestTransactionHandler() (*TransactionHandler, *transaction.MemoryVersionTracker) {
	vt := transaction.NewMemoryVersionTracker()
	return NewTransactionHandler(vt), vt
}

func TestHandleMulti(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*TransactionHandler)
		args     []string
		wantType byte
		wantStr  string
	}{
		{
			name:     "multi success",
			setup:    func(h *TransactionHandler) {},
			args:     []string{},
			wantType: resp.TypeSimpleString,
			wantStr:  "OK",
		},
		{
			name: "nested multi error",
			setup: func(h *TransactionHandler) {
				h.HandleMulti(makeArgs())
			},
			args:     []string{},
			wantType: resp.TypeError,
			wantStr:  "ERR MULTI calls can not be nested",
		},
		{
			name:     "wrong number of arguments",
			setup:    func(h *TransactionHandler) {},
			args:     []string{"extra"},
			wantType: resp.TypeError,
			wantStr:  "ERR wrong number of arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestTransactionHandler()
			tt.setup(h)

			result := h.HandleMulti(makeArgs(tt.args...))

			if result.Type != tt.wantType {
				t.Errorf("HandleMulti() type = %c, want %c", result.Type, tt.wantType)
			}

			if !strings.Contains(result.Str, tt.wantStr) {
				t.Errorf("HandleMulti() str = %q, want to contain %q", result.Str, tt.wantStr)
			}
		})
	}
}

func TestHandleExec(t *testing.T) {
	okResponse := resp.Value{Type: resp.TypeSimpleString, Str: "OK"}
	valueResponse := resp.Value{Type: resp.TypeBulkString, Str: "value1"}

	mockExecutor := func(cmd string, args []string) (resp.Value, error) {
		switch cmd {
		case "SET":
			return okResponse, nil
		case "GET":
			return valueResponse, nil
		default:
			return resp.Value{Type: resp.TypeSimpleString, Str: "OK"}, nil
		}
	}

	tests := []struct {
		name      string
		setup     func(*TransactionHandler, *transaction.MemoryVersionTracker)
		args      []string
		wantType  byte
		wantLen   int
		wantNull  bool
		wantErr   string
	}{
		{
			name:     "exec without multi",
			setup:    func(h *TransactionHandler, vt *transaction.MemoryVersionTracker) {},
			args:     []string{},
			wantType: resp.TypeError,
			wantErr:  "ERR EXEC without MULTI",
		},
		{
			name: "exec empty queue",
			setup: func(h *TransactionHandler, vt *transaction.MemoryVersionTracker) {
				h.HandleMulti(makeArgs())
			},
			args:     []string{},
			wantType: resp.TypeArray,
			wantLen:  0,
		},
		{
			name: "exec with queued commands",
			setup: func(h *TransactionHandler, vt *transaction.MemoryVersionTracker) {
				h.HandleMulti(makeArgs())
				h.QueueCommand("SET", []string{"key1", "value1"})
				h.QueueCommand("GET", []string{"key1"})
			},
			args:     []string{},
			wantType: resp.TypeArray,
			wantLen:  2,
		},
		{
			name: "exec with watch - no changes",
			setup: func(h *TransactionHandler, vt *transaction.MemoryVersionTracker) {
				vt.SetVersion("key1", 1)
				h.HandleWatch(makeArgs("key1"))
				h.HandleMulti(makeArgs())
				h.QueueCommand("SET", []string{"key1", "value1"})
			},
			args:     []string{},
			wantType: resp.TypeArray,
			wantLen:  1,
		},
		{
			name: "exec with watch - key changed",
			setup: func(h *TransactionHandler, vt *transaction.MemoryVersionTracker) {
				vt.SetVersion("key1", 1)
				h.HandleWatch(makeArgs("key1"))
				vt.IncrementVersion("key1") // Simulate another client changing the key
				h.HandleMulti(makeArgs())
				h.QueueCommand("SET", []string{"key1", "value1"})
			},
			args:     []string{},
			wantType: resp.TypeArray,
			wantNull: true,
		},
		{
			name: "wrong number of arguments",
			setup: func(h *TransactionHandler, vt *transaction.MemoryVersionTracker) {
				h.HandleMulti(makeArgs())
			},
			args:     []string{"extra"},
			wantType: resp.TypeError,
			wantErr:  "ERR wrong number of arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, vt := newTestTransactionHandler()
			tt.setup(h, vt)

			result := h.HandleExec(makeArgs(tt.args...), mockExecutor)

			if result.Type != tt.wantType {
				t.Errorf("HandleExec() type = %c, want %c", result.Type, tt.wantType)
			}

			if tt.wantType == resp.TypeArray {
				if tt.wantNull && !result.Null {
					t.Error("HandleExec() expected null array")
				}
				if !tt.wantNull && len(result.Array) != tt.wantLen {
					t.Errorf("HandleExec() array len = %d, want %d", len(result.Array), tt.wantLen)
				}
			}

			if tt.wantType == resp.TypeError && !strings.Contains(result.Str, tt.wantErr) {
				t.Errorf("HandleExec() error = %q, want to contain %q", result.Str, tt.wantErr)
			}
		})
	}
}

func TestHandleDiscard(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*TransactionHandler)
		args     []string
		wantType byte
		wantStr  string
	}{
		{
			name:     "discard without multi",
			setup:    func(h *TransactionHandler) {},
			args:     []string{},
			wantType: resp.TypeError,
			wantStr:  "ERR DISCARD without MULTI",
		},
		{
			name: "discard success",
			setup: func(h *TransactionHandler) {
				h.HandleMulti(makeArgs())
			},
			args:     []string{},
			wantType: resp.TypeSimpleString,
			wantStr:  "OK",
		},
		{
			name: "discard with queued commands",
			setup: func(h *TransactionHandler) {
				h.HandleMulti(makeArgs())
				h.QueueCommand("SET", []string{"key1", "value1"})
			},
			args:     []string{},
			wantType: resp.TypeSimpleString,
			wantStr:  "OK",
		},
		{
			name: "wrong number of arguments",
			setup: func(h *TransactionHandler) {
				h.HandleMulti(makeArgs())
			},
			args:     []string{"extra"},
			wantType: resp.TypeError,
			wantStr:  "ERR wrong number of arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestTransactionHandler()
			tt.setup(h)

			result := h.HandleDiscard(makeArgs(tt.args...))

			if result.Type != tt.wantType {
				t.Errorf("HandleDiscard() type = %c, want %c", result.Type, tt.wantType)
			}

			if !strings.Contains(result.Str, tt.wantStr) {
				t.Errorf("HandleDiscard() str = %q, want to contain %q", result.Str, tt.wantStr)
			}
		})
	}
}

func TestHandleWatch(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*TransactionHandler, *transaction.MemoryVersionTracker)
		args     []string
		wantType byte
		wantStr  string
	}{
		{
			name: "watch single key",
			setup: func(h *TransactionHandler, vt *transaction.MemoryVersionTracker) {
				vt.SetVersion("key1", 1)
			},
			args:     []string{"key1"},
			wantType: resp.TypeSimpleString,
			wantStr:  "OK",
		},
		{
			name: "watch multiple keys",
			setup: func(h *TransactionHandler, vt *transaction.MemoryVersionTracker) {
				vt.SetVersion("key1", 1)
				vt.SetVersion("key2", 2)
			},
			args:     []string{"key1", "key2"},
			wantType: resp.TypeSimpleString,
			wantStr:  "OK",
		},
		{
			name: "watch inside multi error",
			setup: func(h *TransactionHandler, vt *transaction.MemoryVersionTracker) {
				h.HandleMulti(makeArgs())
			},
			args:     []string{"key1"},
			wantType: resp.TypeError,
			wantStr:  "ERR WATCH inside MULTI is not allowed",
		},
		{
			name:     "wrong number of arguments",
			setup:    func(h *TransactionHandler, vt *transaction.MemoryVersionTracker) {},
			args:     []string{},
			wantType: resp.TypeError,
			wantStr:  "ERR wrong number of arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, vt := newTestTransactionHandler()
			tt.setup(h, vt)

			result := h.HandleWatch(makeArgs(tt.args...))

			if result.Type != tt.wantType {
				t.Errorf("HandleWatch() type = %c, want %c", result.Type, tt.wantType)
			}

			if !strings.Contains(result.Str, tt.wantStr) {
				t.Errorf("HandleWatch() str = %q, want to contain %q", result.Str, tt.wantStr)
			}
		})
	}
}

func TestHandleUnwatch(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*TransactionHandler, *transaction.MemoryVersionTracker)
		args     []string
		wantType byte
		wantStr  string
	}{
		{
			name:     "unwatch without watch",
			setup:    func(h *TransactionHandler, vt *transaction.MemoryVersionTracker) {},
			args:     []string{},
			wantType: resp.TypeSimpleString,
			wantStr:  "OK",
		},
		{
			name: "unwatch with watched keys",
			setup: func(h *TransactionHandler, vt *transaction.MemoryVersionTracker) {
				vt.SetVersion("key1", 1)
				h.HandleWatch(makeArgs("key1"))
			},
			args:     []string{},
			wantType: resp.TypeSimpleString,
			wantStr:  "OK",
		},
		{
			name:     "wrong number of arguments",
			setup:    func(h *TransactionHandler, vt *transaction.MemoryVersionTracker) {},
			args:     []string{"extra"},
			wantType: resp.TypeError,
			wantStr:  "ERR wrong number of arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, vt := newTestTransactionHandler()
			tt.setup(h, vt)

			result := h.HandleUnwatch(makeArgs(tt.args...))

			if result.Type != tt.wantType {
				t.Errorf("HandleUnwatch() type = %c, want %c", result.Type, tt.wantType)
			}

			if !strings.Contains(result.Str, tt.wantStr) {
				t.Errorf("HandleUnwatch() str = %q, want to contain %q", result.Str, tt.wantStr)
			}
		})
	}
}

func TestInTransaction(t *testing.T) {
	h, _ := newTestTransactionHandler()

	if h.InTransaction() {
		t.Error("InTransaction() = true before MULTI, want false")
	}

	h.HandleMulti(makeArgs())
	if !h.InTransaction() {
		t.Error("InTransaction() = false after MULTI, want true")
	}

	h.HandleDiscard(makeArgs())
	if h.InTransaction() {
		t.Error("InTransaction() = true after DISCARD, want false")
	}
}

func TestQueueCommand(t *testing.T) {
	h, _ := newTestTransactionHandler()

	// Queue without MULTI should fail (but we handle this gracefully)
	result := h.QueueCommand("SET", []string{"key", "value"})
	if result.Type != resp.TypeError {
		t.Errorf("QueueCommand() without MULTI type = %c, want error", result.Type)
	}

	// Queue after MULTI should succeed
	h.HandleMulti(makeArgs())
	result = h.QueueCommand("SET", []string{"key", "value"})
	if result.Type != resp.TypeSimpleString || result.Str != "QUEUED" {
		t.Errorf("QueueCommand() after MULTI = %v, want QUEUED", result)
	}

	result = h.QueueCommand("GET", []string{"key"})
	if result.Type != resp.TypeSimpleString || result.Str != "QUEUED" {
		t.Errorf("QueueCommand() second command = %v, want QUEUED", result)
	}
}

func TestIsTransactionCommand(t *testing.T) {
	transactionCommands := []string{"MULTI", "EXEC", "DISCARD", "WATCH", "UNWATCH"}
	regularCommands := []string{"GET", "SET", "DEL", "LPUSH", "HSET", "SADD"}

	for _, cmd := range transactionCommands {
		if !IsTransactionCommand(cmd) {
			t.Errorf("IsTransactionCommand(%q) = false, want true", cmd)
		}
	}

	for _, cmd := range regularCommands {
		if IsTransactionCommand(cmd) {
			t.Errorf("IsTransactionCommand(%q) = true, want false", cmd)
		}
	}
}

func TestTransactionFlow(t *testing.T) {
	// Test a full transaction flow: WATCH -> MULTI -> QUEUE -> EXEC
	h, vt := newTestTransactionHandler()

	// Set initial version
	vt.SetVersion("counter", 5)

	// WATCH the key
	result := h.HandleWatch(makeArgs("counter"))
	if result.Type != resp.TypeSimpleString || result.Str != "OK" {
		t.Errorf("WATCH failed: %v", result)
	}

	// Start transaction
	result = h.HandleMulti(makeArgs())
	if result.Type != resp.TypeSimpleString || result.Str != "OK" {
		t.Errorf("MULTI failed: %v", result)
	}

	// Queue commands
	result = h.QueueCommand("SET", []string{"counter", "10"})
	if result.Type != resp.TypeSimpleString || result.Str != "QUEUED" {
		t.Errorf("Queue SET failed: %v", result)
	}

	result = h.QueueCommand("GET", []string{"counter"})
	if result.Type != resp.TypeSimpleString || result.Str != "QUEUED" {
		t.Errorf("Queue GET failed: %v", result)
	}

	// Execute - should succeed since key wasn't modified
	mockExecutor := func(cmd string, args []string) (resp.Value, error) {
		return resp.Value{Type: resp.TypeSimpleString, Str: "OK"}, nil
	}

	result = h.HandleExec(makeArgs(), mockExecutor)
	if result.Type != resp.TypeArray {
		t.Errorf("EXEC failed: %v", result)
	}
	if result.Null {
		t.Error("EXEC returned null (WATCH failed) but key wasn't modified")
	}
	if len(result.Array) != 2 {
		t.Errorf("EXEC returned %d results, want 2", len(result.Array))
	}
}

func TestTransactionFlowWithModifiedKey(t *testing.T) {
	// Test WATCH failure when key is modified
	h, vt := newTestTransactionHandler()

	// Set initial version
	vt.SetVersion("counter", 5)

	// WATCH the key
	h.HandleWatch(makeArgs("counter"))

	// Simulate another client modifying the key
	vt.IncrementVersion("counter")

	// Start transaction
	h.HandleMulti(makeArgs())
	h.QueueCommand("SET", []string{"counter", "10"})

	// Execute - should fail because key was modified
	mockExecutor := func(cmd string, args []string) (resp.Value, error) {
		t.Error("Executor should not be called when WATCH fails")
		return resp.Value{}, nil
	}

	result := h.HandleExec(makeArgs(), mockExecutor)
	if result.Type != resp.TypeArray {
		t.Errorf("EXEC type = %c, want array", result.Type)
	}
	if !result.Null {
		t.Error("EXEC should return null array when WATCH fails")
	}
}
