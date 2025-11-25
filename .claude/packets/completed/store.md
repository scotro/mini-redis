# Work Packet: store

**Objective:** Implement thread-safe in-memory key-value store with TTL support
**Branch:** `feature/store` 
**Worktree:** `worktrees/agent-store`

## Acceptance Criteria
- [ ] Thread-safe Get/Set/Delete operations
- [ ] TTL support (keys expire after duration)
- [ ] Background goroutine for expired key cleanup
- [ ] Keys() method for iteration
- [ ] Unit tests with race detection (`go test -race`)
- [ ] No lint errors

## Context 
**Key Files:** `internal/store/`
**Background:** The store holds all Redis data in memory. Must be safe for concurrent access from multiple client connections

## Boundaries
**In Scope:** `internal/store/` only

**Out of Scope:** RESP parsing, TCP networking, command handling

## Interface Contract
Export these for other agents: 
```go                        
type Store interface { 
    Get(key string) (string, bool) 
    Set(key string, value string) 
    SetWithTTL(key string, value string, ttl time.Duration)
    Delete(key string) bool
    Keys() []string 
    TTL(key string) (time.Duration, bool)  // returns remaining TTL
}

func New() Store  
```

## Signal Protocol 

Signal BLOCKED when: Unsure about TTL precision requirements 
Signal DONE when: All acceptance criteria met, tests passing with -race
