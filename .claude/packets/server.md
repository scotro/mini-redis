# Work Packet: server

**Objective:** Implement TCP server with Redis command handling

**Branch:** `feature/server
**Worktree:** `worktrees/agent-server

## Acceptance Criteria

- [ ] TCP server listening on configurable port (default 6379)
- [ ] Handle concurrent client connections
- [ ] Command handlers: PING, ECHO, GET, SET, DEL, EXPIRE, TTL
- [ ] SET supports EX option: SET key value EX seconds
- [ ] Proper error responses for invalid commands
- [ ] Integration tests (connect, send command, verify response)
- [ ] No lint errors

## Context

**Key Files:**
- `internal/server/`
- `cmd/mini-redis/`

**Background:**
The server accepts TCP connections, parses RESP commands using the resp package, executes them against the store, and returns the RESP responses.

## Boundaries

**In Scope:**
- `internal/server/`
- `cmd/mini-redis/main.go`

**Out of Scope:**
**Soft dependency on resp-parser and store:** Start with mock implementations, integrate real ones when available.

## Interface Contract
Assume these interfaces exist (from other agents):
```go
// From resp package
type Value struct { Type byte; Str string; Num int; Array []Value }
func Parse(reader *bufio.Reader) (Value, error)
func (v Value) Serialize() []byte

// From store package
type Store interface {
    Get(key string) (string, bool)
    Set(key string, value string)
    SetWithTTL(key string, value string, ttl time.Duration)
    Delete(key string) bool
    TTL(key string) (time.Duration, bool)
}
```

## Signal Protocol

**Signal BLOCKED when:**
- Need a decision on command behavior, interface changes needed

**Signal DONE when:**
- Server runs, handles all commands, tests pass
