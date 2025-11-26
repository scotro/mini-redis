# Work Packet: server-integration

**Objective:** Wire lists, hashes, and sets command handlers into the main server command dispatch

**Branch:** `feature/server-integration`
**Worktree:** `worktrees/agent-server-integration`

## Acceptance Criteria

- [ ] Server accepts ListStore, HashStore, and SetStore in constructor
- [ ] All list commands wired into executeCommand switch: LPUSH, RPUSH, LPOP, RPOP, LRANGE, LLEN
- [ ] All hash commands wired into executeCommand switch: HSET, HGET, HDEL, HGETALL, HKEYS, HLEN
- [ ] All set commands wired into executeCommand switch: SADD, SREM, SMEMBERS, SISMEMBER, SCARD, SINTER
- [ ] cmd/mini-redis/main.go updated to create and inject all stores
- [ ] All tests pass: `go test ./...`
- [ ] Server starts and all commands work via redis-cli
- [ ] No lint errors: `go vet ./...`

## Context

**Key Files:**
- `internal/server/server.go` - Main server, needs switch cases added
- `internal/server/lists_cmds.go` - List handlers (already implemented)
- `internal/server/hashes_cmds.go` - Hash handlers (already implemented)
- `internal/server/sets_cmds.go` - Set handlers (already implemented)
- `internal/store/lists.go` - ListStore interface and implementation
- `internal/store/hashes.go` - HashStore interface and implementation
- `internal/store/sets.go` - SetStore interface and implementation
- `cmd/mini-redis/main.go` - Entry point, needs to create stores

**Background:**
Three agents implemented list, hash, and set data types in parallel. Each created their own store implementation and command handlers. This packet wires them together into the main server so all commands are accessible.

**Technical Constraints:**
- Maintain backward compatibility with existing string commands
- Keep the Server struct clean - add store fields for each data type
- Follow existing patterns in server.go for command dispatch

## Boundaries

**In Scope:**
- Modify `internal/server/server.go` - add store fields, wire switch cases
- Modify `cmd/mini-redis/main.go` - create and inject stores
- Fix any compilation issues from integration

**Out of Scope:**
- Do NOT modify the store implementations (lists.go, hashes.go, sets.go)
- Do NOT modify the command handlers (lists_cmds.go, hashes_cmds.go, sets_cmds.go)
- Do NOT add new commands beyond what's already implemented

## Integration Pattern

The Server struct should look like:
```go
type Server struct {
    config     Config
    store      store.Store      // existing string store
    listStore  store.ListStore  // add this
    hashStore  store.HashStore  // add this
    setStore   store.SetStore   // add this
    listener   net.Listener
    wg         sync.WaitGroup
    quit       chan struct{}
}
```

The executeCommand switch should add cases:
```go
case "LPUSH":
    return s.handleLPush(args)
case "RPUSH":
    return s.handleRPush(args)
// ... etc for all new commands
```

## Signal Protocol

**Signal BLOCKED when:**
- Store interfaces don't match expected signatures
- Handler methods have different signatures than expected
- Circular import issues

**Signal DONE when:**
- All commands work via redis-cli
- All tests pass
- Ready for final review

## Verification Steps

After integration, test with:
```bash
# Build and run
go build -o mini-redis ./cmd/mini-redis && ./mini-redis &

# Test lists
redis-cli RPUSH mylist a b c
redis-cli LRANGE mylist 0 -1

# Test hashes
redis-cli HSET user:1 name Alice age 30
redis-cli HGETALL user:1

# Test sets
redis-cli SADD tags go redis
redis-cli SMEMBERS tags

# Cleanup
killall mini-redis
```
