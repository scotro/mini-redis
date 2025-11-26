# Work Packet: sets

**Objective:** Implement Redis set data type with SADD, SREM, SMEMBERS, SISMEMBER, SCARD, and SINTER commands

**Branch:** `feature/sets`
**Worktree:** `worktrees/agent-sets`

## Acceptance Criteria

- [ ] SADD key member [member ...] - Add members, return count of new members added
- [ ] SREM key member [member ...] - Remove members, return count removed
- [ ] SMEMBERS key - Return all members as array
- [ ] SISMEMBER key member - Return 1 if member exists, 0 otherwise
- [ ] SCARD key - Return cardinality (size) of set (0 if missing)
- [ ] SINTER key [key ...] - Return intersection of all sets
- [ ] WRONGTYPE error when operating on non-set keys
- [ ] All existing tests pass
- [ ] New tests cover all commands and edge cases
- [ ] No lint errors introduced

## Context

**Key Files:**
- `internal/store/store.go` - Existing string store interface (reference only)
- `internal/server/server.go` - Existing command dispatch (reference only)

**Background:**
Redis sets are unordered collections of unique strings. They support standard set operations like membership testing and intersection. Sets are commonly used for tagging, tracking unique items, and computing relationships.

**Technical Constraints:**
- Use map[string]struct{} for storage (memory-efficient set)
- Must be thread-safe (use sync.RWMutex)
- Must track key types to return WRONGTYPE errors

## Boundaries

**In Scope:**
- `internal/store/sets.go` - Set storage implementation
- `internal/store/sets_test.go` - Set storage tests
- `internal/server/sets_cmds.go` - Set command handlers
- `internal/server/sets_cmds_test.go` - Command handler tests

**Out of Scope:**
- Do NOT modify `internal/store/store.go`
- Do NOT modify `internal/server/server.go`
- Do NOT implement list or hash commands (other agents)

## Interface Contracts

Create these types and methods (will be integrated later):

```go
// internal/store/sets.go

type SetStore interface {
    SAdd(key string, members ...string) int  // returns count of NEW members
    SRem(key string, members ...string) int
    SMembers(key string) []string
    SIsMember(key, member string) bool
    SCard(key string) int
    SInter(keys ...string) []string
    // For type checking during integration
    KeyType(key string) string  // returns "set", "none", etc.
}
```

```go
// internal/server/sets_cmds.go

// Handlers follow this pattern (using resp helpers from server.go):
func (s *Server) handleSAdd(args []resp.Value) resp.Value
func (s *Server) handleSRem(args []resp.Value) resp.Value
func (s *Server) handleSMembers(args []resp.Value) resp.Value
func (s *Server) handleSIsMember(args []resp.Value) resp.Value
func (s *Server) handleSCard(args []resp.Value) resp.Value
func (s *Server) handleSInter(args []resp.Value) resp.Value
```

## Signal Protocol

**Signal BLOCKED when:**
- Need clarification on SINTER with non-existent keys (empty set?)
- Unsure how to handle type conflicts with existing string keys
- Tests reveal issues in code outside boundaries

**Signal DONE when:**
- All acceptance criteria met
- All tests passing
- Ready for integration review

## Notes

- SADD returns count of NEW members (duplicates don't count)
- SMEMBERS order is not guaranteed (set iteration order)
- SINTER with non-existent key returns empty set
- SINTER with single key returns that key's members
- Empty sets should be automatically deleted (Redis behavior)
