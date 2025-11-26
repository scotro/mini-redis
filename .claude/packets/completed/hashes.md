# Work Packet: hashes

**Objective:** Implement Redis hash data type with HSET, HGET, HDEL, HGETALL, HKEYS, and HLEN commands

**Branch:** `feature/hashes`
**Worktree:** `worktrees/agent-hashes`

## Acceptance Criteria

- [ ] HSET key field value [field value ...] - Set fields, return count of new fields added
- [ ] HGET key field - Get value of field (null if missing)
- [ ] HDEL key field [field ...] - Delete fields, return count deleted
- [ ] HGETALL key - Return all field-value pairs as array
- [ ] HKEYS key - Return all field names
- [ ] HLEN key - Return number of fields (0 if missing)
- [ ] WRONGTYPE error when operating on non-hash keys
- [ ] All existing tests pass
- [ ] New tests cover all commands and edge cases
- [ ] No lint errors introduced

## Context

**Key Files:**
- `internal/store/store.go` - Existing string store interface (reference only)
- `internal/server/server.go` - Existing command dispatch (reference only)

**Background:**
Redis hashes are maps of field-value pairs, useful for representing objects. HSET can set multiple fields at once. HGETALL returns alternating field/value entries in the response array.

**Technical Constraints:**
- Use map[string]string for field storage
- Must be thread-safe (use sync.RWMutex)
- Must track key types to return WRONGTYPE errors

## Boundaries

**In Scope:**
- `internal/store/hashes.go` - Hash storage implementation
- `internal/store/hashes_test.go` - Hash storage tests
- `internal/server/hashes_cmds.go` - Hash command handlers
- `internal/server/hashes_cmds_test.go` - Command handler tests

**Out of Scope:**
- Do NOT modify `internal/store/store.go`
- Do NOT modify `internal/server/server.go`
- Do NOT implement list or set commands (other agents)

## Interface Contracts

Create these types and methods (will be integrated later):

```go
// internal/store/hashes.go

type HashStore interface {
    HSet(key string, fieldValues ...string) int  // returns count of NEW fields
    HGet(key, field string) (string, bool)
    HDel(key string, fields ...string) int
    HGetAll(key string) map[string]string
    HKeys(key string) []string
    HLen(key string) int
    // For type checking during integration
    KeyType(key string) string  // returns "hash", "none", etc.
}
```

```go
// internal/server/hashes_cmds.go

// Handlers follow this pattern (using resp helpers from server.go):
func (s *Server) handleHSet(args []resp.Value) resp.Value
func (s *Server) handleHGet(args []resp.Value) resp.Value
func (s *Server) handleHDel(args []resp.Value) resp.Value
func (s *Server) handleHGetAll(args []resp.Value) resp.Value
func (s *Server) handleHKeys(args []resp.Value) resp.Value
func (s *Server) handleHLen(args []resp.Value) resp.Value
```

## Signal Protocol

**Signal BLOCKED when:**
- Need clarification on HSET return value (new fields vs total fields)
- Unsure how to handle type conflicts with existing string keys
- Tests reveal issues in code outside boundaries

**Signal DONE when:**
- All acceptance criteria met
- All tests passing
- Ready for integration review

## Notes

- HSET returns count of NEW fields added (not total fields, not updated fields)
- HGETALL returns flat array: [field1, value1, field2, value2, ...]
- Empty hashes should be automatically deleted (Redis behavior)
- Field order in HGETALL and HKEYS is not guaranteed (map iteration order)
