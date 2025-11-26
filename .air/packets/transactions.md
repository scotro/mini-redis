# Packet: transactions

**Objective:** Implement Redis transactions with MULTI, EXEC, DISCARD, and WATCH commands.

## Boundaries

**In scope:**
- `internal/transaction/` (new package - create all files here)
- `internal/server/transaction_cmds.go` (new file for transaction commands)

**Out of scope:**
- `internal/server/server.go` (do NOT modify - integration will happen later)
- `internal/store/` (transactions wrap commands, don't modify stores directly)
- `internal/persistence/` (another agent's work)
- `internal/pubsub/` (another agent's work)

## Context

Redis transactions work as follows:
1. `MULTI` - Start a transaction, enter "queuing mode"
2. Commands are queued (not executed), server responds with `+QUEUED`
3. `EXEC` - Execute all queued commands atomically, return array of results
4. `DISCARD` - Abort transaction, clear queue
5. `WATCH key [key ...]` - Optimistic locking, EXEC fails if watched keys changed

Follow existing patterns:
- Thread-safety with `sync.RWMutex`
- Interface-based design for testability
- Table-driven tests

## Implementation Requirements

### 1. Transaction Package (`internal/transaction/`)

**Core types:**
```go
type Transaction struct {
    mu        sync.Mutex
    inMulti   bool
    queue     []QueuedCommand
    watching  map[string]int64  // key -> version at watch time
}

type QueuedCommand struct {
    Name string
    Args []string
}

// CommandExecutor is passed in to execute commands during EXEC
type CommandExecutor func(cmd string, args []string) (resp.Value, error)
```

**Methods:**
- `Begin()` - Start transaction (MULTI)
- `Queue(cmd string, args []string)` - Add command to queue
- `Exec(executor CommandExecutor) ([]resp.Value, error)` - Execute all queued commands
- `Discard()` - Clear queue and exit transaction mode
- `Watch(keys ...string, getVersion func(key string) int64)` - Watch keys for changes
- `CheckWatch(getVersion func(key string) int64) bool` - Check if watched keys changed
- `InTransaction() bool` - Check if in MULTI mode

### 2. Version Tracking

For WATCH to work, we need to track key versions. Create an interface that the store can implement:

```go
type VersionTracker interface {
    GetVersion(key string) int64
}
```

For now, implement a simple in-memory version tracker in the transaction package that can be used for testing. The actual store integration will happen later.

### 3. Commands (`internal/server/transaction_cmds.go`)

Create a `TransactionHandler` struct:
- `MULTI` - Start transaction, return OK
- `EXEC` - Execute queued commands, return array of results (or nil if WATCH failed)
- `DISCARD` - Abort transaction, return OK
- `WATCH key [key ...]` - Watch keys, return OK (must be called before MULTI)
- `UNWATCH` - Clear all watched keys, return OK

**Error cases:**
- `MULTI` when already in MULTI → `ERR MULTI calls can not be nested`
- `EXEC` without MULTI → `ERR EXEC without MULTI`
- `DISCARD` without MULTI → `ERR DISCARD without MULTI`
- `WATCH` inside MULTI → `ERR WATCH inside MULTI is not allowed`

### 4. Command Queuing Behavior

When in transaction mode, most commands should be queued:
- Return `+QUEUED\r\n` instead of executing
- MULTI, EXEC, DISCARD, WATCH, UNWATCH are NOT queued (execute immediately)

## Acceptance Criteria

- [ ] `MULTI` starts a transaction
- [ ] Commands after MULTI are queued and return QUEUED
- [ ] `EXEC` executes all queued commands and returns results array
- [ ] `DISCARD` clears the queue and exits transaction mode
- [ ] `WATCH` tracks key versions for optimistic locking
- [ ] `EXEC` returns nil (empty array) if watched keys were modified
- [ ] Proper error messages for invalid command sequences
- [ ] Unit tests for transaction package
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

## Notes

- Transactions are per-connection state, not global
- WATCH must be called BEFORE MULTI
- After EXEC or DISCARD, watched keys are automatically unwatched
- The CommandExecutor pattern allows testing without a real server
- For atomic execution, the executor should hold appropriate locks
- Error in one queued command doesn't abort others (Redis behavior) - return EXECABORT only for syntax errors caught at queue time
