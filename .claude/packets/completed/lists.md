# Work Packet: lists

**Objective:** Implement Redis list data type with LPUSH, RPUSH, LPOP, RPOP, LRANGE, and LLEN commands

**Branch:** `feature/lists`
**Worktree:** `worktrees/agent-lists`

## Acceptance Criteria

- [ ] LPUSH key value [value ...] - Prepend values to list, return new length
- [ ] RPUSH key value [value ...] - Append values to list, return new length
- [ ] LPOP key - Remove and return first element (null if empty/missing)
- [ ] RPOP key - Remove and return last element (null if empty/missing)
- [ ] LRANGE key start stop - Return range of elements (supports negative indices)
- [ ] LLEN key - Return length of list (0 if missing)
- [ ] WRONGTYPE error when operating on non-list keys
- [ ] All existing tests pass
- [ ] New tests cover all commands and edge cases
- [ ] No lint errors introduced

## Context

**Key Files:**
- `internal/store/store.go` - Existing string store interface (reference only)
- `internal/server/server.go` - Existing command dispatch (reference only)

**Background:**
Redis lists are ordered collections of strings. They support push/pop from both ends (making them usable as stacks or queues) and indexed access. In real Redis, lists are stored separately from strings with type checking.

**Technical Constraints:**
- Use slices for storage (not linked lists) for simplicity
- Must be thread-safe (use sync.RWMutex)
- Must track key types to return WRONGTYPE errors

## Boundaries

**In Scope:**
- `internal/store/lists.go` - List storage implementation
- `internal/store/lists_test.go` - List storage tests
- `internal/server/lists_cmds.go` - List command handlers
- `internal/server/lists_cmds_test.go` - Command handler tests

**Out of Scope:**
- Do NOT modify `internal/store/store.go`
- Do NOT modify `internal/server/server.go`
- Do NOT implement hash or set commands (other agents)

## Interface Contracts

Create these types and methods (will be integrated later):

```go
// internal/store/lists.go

type ListStore interface {
    LPush(key string, values ...string) int
    RPush(key string, values ...string) int
    LPop(key string) (string, bool)
    RPop(key string) (string, bool)
    LRange(key string, start, stop int) []string
    LLen(key string) int
    // For type checking during integration
    KeyType(key string) string  // returns "list", "none", etc.
}
```

```go
// internal/server/lists_cmds.go

// Handlers follow this pattern (using resp helpers from server.go):
func (s *Server) handleLPush(args []resp.Value) resp.Value
func (s *Server) handleRPush(args []resp.Value) resp.Value
func (s *Server) handleLPop(args []resp.Value) resp.Value
func (s *Server) handleRPop(args []resp.Value) resp.Value
func (s *Server) handleLRange(args []resp.Value) resp.Value
func (s *Server) handleLLen(args []resp.Value) resp.Value
```

## Signal Protocol

**Signal BLOCKED when:**
- Need clarification on LRANGE negative index behavior
- Unsure how to handle type conflicts with existing string keys
- Tests reveal issues in code outside boundaries

**Signal DONE when:**
- All acceptance criteria met
- All tests passing
- Ready for integration review

## Notes

- LRANGE uses 0-based indices; -1 means last element, -2 second to last, etc.
- LRANGE is inclusive on both ends: LRANGE key 0 -1 returns all elements
- Empty lists should be automatically deleted (Redis behavior)
