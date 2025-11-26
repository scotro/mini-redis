package transaction

import (
	"testing"

	"github.com/scotro/mini-redis/internal/resp"
)

// mockExecutor creates a command executor that returns simple responses
func mockExecutor(responses map[string]resp.Value) CommandExecutor {
	return func(cmd string, args []string) (resp.Value, error) {
		if r, ok := responses[cmd]; ok {
			return r, nil
		}
		return resp.Value{Type: resp.TypeSimpleString, Str: "OK"}, nil
	}
}

// simpleVersionGetter returns a version getter from a map
func simpleVersionGetter(versions map[string]int64) VersionGetter {
	return func(key string) int64 {
		return versions[key]
	}
}

func TestTransaction_Begin(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Transaction)
		wantErr error
	}{
		{
			name:    "begin new transaction",
			setup:   func(tx *Transaction) {},
			wantErr: nil,
		},
		{
			name: "nested multi error",
			setup: func(tx *Transaction) {
				tx.Begin()
			},
			wantErr: ErrNestedMulti,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := New()
			tt.setup(tx)

			err := tx.Begin()
			if err != tt.wantErr {
				t.Errorf("Begin() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransaction_InTransaction(t *testing.T) {
	tx := New()

	if tx.InTransaction() {
		t.Error("InTransaction() = true before Begin, want false")
	}

	tx.Begin()
	if !tx.InTransaction() {
		t.Error("InTransaction() = false after Begin, want true")
	}

	tx.Discard()
	if tx.InTransaction() {
		t.Error("InTransaction() = true after Discard, want false")
	}
}

func TestTransaction_Queue(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Transaction)
		cmd     string
		args    []string
		wantErr error
		wantLen int
	}{
		{
			name:    "queue without multi",
			setup:   func(tx *Transaction) {},
			cmd:     "SET",
			args:    []string{"key", "value"},
			wantErr: ErrNotInMulti,
			wantLen: 0,
		},
		{
			name: "queue single command",
			setup: func(tx *Transaction) {
				tx.Begin()
			},
			cmd:     "SET",
			args:    []string{"key", "value"},
			wantErr: nil,
			wantLen: 1,
		},
		{
			name: "queue multiple commands",
			setup: func(tx *Transaction) {
				tx.Begin()
				tx.Queue("SET", []string{"key1", "value1"})
			},
			cmd:     "GET",
			args:    []string{"key1"},
			wantErr: nil,
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := New()
			tt.setup(tx)

			err := tx.Queue(tt.cmd, tt.args)
			if err != tt.wantErr {
				t.Errorf("Queue() error = %v, want %v", err, tt.wantErr)
			}

			if tx.QueueLength() != tt.wantLen {
				t.Errorf("QueueLength() = %d, want %d", tx.QueueLength(), tt.wantLen)
			}
		})
	}
}

func TestTransaction_Exec(t *testing.T) {
	okResponse := resp.Value{Type: resp.TypeSimpleString, Str: "OK"}
	valueResponse := resp.Value{Type: resp.TypeBulkString, Str: "value1"}

	tests := []struct {
		name       string
		setup      func(*Transaction)
		executor   CommandExecutor
		versions   map[string]int64
		wantErr    error
		wantLen    int
		wantNil    bool
	}{
		{
			name:    "exec without multi",
			setup:   func(tx *Transaction) {},
			wantErr: ErrExecWithoutMulti,
		},
		{
			name: "exec empty queue",
			setup: func(tx *Transaction) {
				tx.Begin()
			},
			executor: mockExecutor(nil),
			wantLen:  0,
		},
		{
			name: "exec with queued commands",
			setup: func(tx *Transaction) {
				tx.Begin()
				tx.Queue("SET", []string{"key1", "value1"})
				tx.Queue("GET", []string{"key1"})
			},
			executor: mockExecutor(map[string]resp.Value{
				"SET": okResponse,
				"GET": valueResponse,
			}),
			wantLen: 2,
		},
		{
			name: "exec with watch - no changes",
			setup: func(tx *Transaction) {
				tx.Watch(simpleVersionGetter(map[string]int64{"key1": 1}), "key1")
				tx.Begin()
				tx.Queue("SET", []string{"key1", "value1"})
			},
			executor: mockExecutor(nil),
			versions: map[string]int64{"key1": 1},
			wantLen:  1,
		},
		{
			name: "exec with watch - key changed",
			setup: func(tx *Transaction) {
				tx.Watch(simpleVersionGetter(map[string]int64{"key1": 1}), "key1")
				tx.Begin()
				tx.Queue("SET", []string{"key1", "value1"})
			},
			executor: mockExecutor(nil),
			versions: map[string]int64{"key1": 2}, // Version changed
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := New()
			tt.setup(tx)

			var getVersion VersionGetter
			if tt.versions != nil {
				getVersion = simpleVersionGetter(tt.versions)
			}

			results, err := tx.Exec(tt.executor, getVersion)

			if err != tt.wantErr {
				t.Errorf("Exec() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantNil && results != nil {
				t.Error("Exec() results not nil, want nil (WATCH failed)")
			}

			if !tt.wantNil && results != nil && len(results) != tt.wantLen {
				t.Errorf("Exec() results len = %d, want %d", len(results), tt.wantLen)
			}

			// After exec, should not be in transaction
			if err == nil && tx.InTransaction() {
				t.Error("InTransaction() = true after Exec, want false")
			}
		})
	}
}

func TestTransaction_Discard(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Transaction)
		wantErr error
	}{
		{
			name:    "discard without multi",
			setup:   func(tx *Transaction) {},
			wantErr: ErrDiscardWithoutMulti,
		},
		{
			name: "discard with empty queue",
			setup: func(tx *Transaction) {
				tx.Begin()
			},
			wantErr: nil,
		},
		{
			name: "discard with queued commands",
			setup: func(tx *Transaction) {
				tx.Begin()
				tx.Queue("SET", []string{"key1", "value1"})
				tx.Queue("GET", []string{"key1"})
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := New()
			tt.setup(tx)

			err := tx.Discard()
			if err != tt.wantErr {
				t.Errorf("Discard() error = %v, want %v", err, tt.wantErr)
			}

			if err == nil {
				if tx.InTransaction() {
					t.Error("InTransaction() = true after Discard, want false")
				}
				if tx.QueueLength() != 0 {
					t.Errorf("QueueLength() = %d after Discard, want 0", tx.QueueLength())
				}
			}
		})
	}
}

func TestTransaction_Watch(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Transaction)
		keys     []string
		versions map[string]int64
		wantErr  error
		wantKeys int
	}{
		{
			name:     "watch single key",
			setup:    func(tx *Transaction) {},
			keys:     []string{"key1"},
			versions: map[string]int64{"key1": 1},
			wantKeys: 1,
		},
		{
			name:     "watch multiple keys",
			setup:    func(tx *Transaction) {},
			keys:     []string{"key1", "key2", "key3"},
			versions: map[string]int64{"key1": 1, "key2": 2, "key3": 3},
			wantKeys: 3,
		},
		{
			name: "watch inside multi error",
			setup: func(tx *Transaction) {
				tx.Begin()
			},
			keys:     []string{"key1"},
			versions: map[string]int64{"key1": 1},
			wantErr:  ErrWatchInsideMulti,
		},
		{
			name: "watch accumulates keys",
			setup: func(tx *Transaction) {
				tx.Watch(simpleVersionGetter(map[string]int64{"key1": 1}), "key1")
			},
			keys:     []string{"key2"},
			versions: map[string]int64{"key1": 1, "key2": 2},
			wantKeys: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := New()
			tt.setup(tx)

			err := tx.Watch(simpleVersionGetter(tt.versions), tt.keys...)
			if err != tt.wantErr {
				t.Errorf("Watch() error = %v, want %v", err, tt.wantErr)
			}

			if err == nil {
				watched := tx.WatchedKeys()
				if len(watched) != tt.wantKeys {
					t.Errorf("WatchedKeys() len = %d, want %d", len(watched), tt.wantKeys)
				}
			}
		})
	}
}

func TestTransaction_Unwatch(t *testing.T) {
	tx := New()
	tx.Watch(simpleVersionGetter(map[string]int64{"key1": 1}), "key1", "key2")

	if !tx.IsWatching() {
		t.Error("IsWatching() = false after Watch, want true")
	}

	tx.Unwatch()

	if tx.IsWatching() {
		t.Error("IsWatching() = true after Unwatch, want false")
	}

	watched := tx.WatchedKeys()
	if len(watched) != 0 {
		t.Errorf("WatchedKeys() len = %d after Unwatch, want 0", len(watched))
	}
}

func TestTransaction_CheckWatch(t *testing.T) {
	tests := []struct {
		name         string
		watchVersion map[string]int64
		checkVersion map[string]int64
		want         bool
	}{
		{
			name:         "no watched keys",
			watchVersion: nil,
			checkVersion: nil,
			want:         true,
		},
		{
			name:         "versions unchanged",
			watchVersion: map[string]int64{"key1": 1, "key2": 2},
			checkVersion: map[string]int64{"key1": 1, "key2": 2},
			want:         true,
		},
		{
			name:         "one version changed",
			watchVersion: map[string]int64{"key1": 1, "key2": 2},
			checkVersion: map[string]int64{"key1": 1, "key2": 3},
			want:         false,
		},
		{
			name:         "all versions changed",
			watchVersion: map[string]int64{"key1": 1, "key2": 2},
			checkVersion: map[string]int64{"key1": 5, "key2": 6},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := New()

			if tt.watchVersion != nil {
				for key := range tt.watchVersion {
					tx.Watch(simpleVersionGetter(tt.watchVersion), key)
				}
			}

			result := tx.CheckWatch(simpleVersionGetter(tt.checkVersion))
			if result != tt.want {
				t.Errorf("CheckWatch() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestTransaction_ExecClearsWatch(t *testing.T) {
	tx := New()
	versions := map[string]int64{"key1": 1}

	tx.Watch(simpleVersionGetter(versions), "key1")
	if !tx.IsWatching() {
		t.Error("IsWatching() = false after Watch")
	}

	tx.Begin()
	tx.Exec(mockExecutor(nil), simpleVersionGetter(versions))

	if tx.IsWatching() {
		t.Error("IsWatching() = true after Exec, want false")
	}
}

func TestTransaction_DiscardClearsWatch(t *testing.T) {
	tx := New()
	versions := map[string]int64{"key1": 1}

	tx.Watch(simpleVersionGetter(versions), "key1")
	if !tx.IsWatching() {
		t.Error("IsWatching() = false after Watch")
	}

	tx.Begin()
	tx.Discard()

	if tx.IsWatching() {
		t.Error("IsWatching() = true after Discard, want false")
	}
}
