# Packet: persistence

**Objective:** Implement RDB-style snapshot persistence with SAVE/BGSAVE commands and automatic loading on startup.

## Boundaries

**In scope:**
- `internal/persistence/` (new package - create all files here)
- `internal/server/persistence_cmds.go` (new file for SAVE, BGSAVE commands)
- `internal/store/snapshot.go` (new file - add Snapshot/Restore methods to stores)
- `cmd/mini-redis/main.go` (add snapshot loading on startup, pass persistence to server)

**Out of scope:**
- `internal/server/server.go` (do NOT modify - integration will happen later)
- `internal/server/lists_cmds.go`, `hashes_cmds.go`, `sets_cmds.go`
- `internal/pubsub/` (another agent's work)
- `internal/server/transaction_cmds.go` (another agent's work)

## Context

The codebase has four stores that need to be serialized:
- `Store` (strings with TTL) in `internal/store/store.go`
- `ListStore` in `internal/store/lists.go`
- `HashStore` in `internal/store/hashes.go`
- `SetStore` in `internal/store/sets.go`

Follow the existing patterns:
- Thread-safety with `sync.RWMutex`
- Interface-based design for testability
- Table-driven tests

## Implementation Requirements

### 1. Persistence Package (`internal/persistence/`)

Create a snapshot manager that can:
- Serialize all store data to a binary/gob format
- Save to `dump.rdb` (or configurable path)
- Load snapshot and restore all stores
- Support background saves (BGSAVE) without blocking

### 2. Store Snapshot Interface (`internal/store/snapshot.go`)

Add methods to export/import store data:
```go
type Snapshottable interface {
    ExportData() interface{}
    ImportData(data interface{}) error
}
```

### 3. Commands (`internal/server/persistence_cmds.go`)

Create a `PersistenceHandler` struct (following the pattern in `lists_cmds.go`):
- `SAVE` - Synchronous snapshot to disk, returns OK
- `BGSAVE` - Background snapshot, returns "Background saving started"

### 4. Startup Loading (`cmd/mini-redis/main.go`)

- Check for existing `dump.rdb` on startup
- Load and restore all stores before starting server
- Log "Loading snapshot..." and "Loaded N keys"

## Acceptance Criteria

- [ ] `SAVE` command writes all data to disk
- [ ] `BGSAVE` command starts background save without blocking
- [ ] Server loads existing snapshot on startup
- [ ] Snapshot includes: strings (with TTL), lists, hashes, sets
- [ ] Unit tests for persistence package
- [ ] Integration test: save, restart, verify data persists
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

## Notes

- Use Go's `encoding/gob` for serialization (simple and effective)
- For BGSAVE, use a goroutine with a mutex to prevent concurrent saves
- Store TTLs as absolute Unix timestamps, recalculate remaining TTL on load
- Auto-save intervals are stretch goal - focus on SAVE/BGSAVE first
