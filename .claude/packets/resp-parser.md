# Work Packet: resp-parser

**Objective:** Implement RESP (Redis Serialization Protocol) parser for encoding/decoding Redis commands

**Branch:** `feature/resp-parser`
**Worktree:** `worktrees/agent-resp-parser`

## Acceptance Criteria

- [ ] Parse RESP simple strings (+OK\r\n)
- [ ] Parse RESP errors (-ERR message\r\n)
- [ ] Parse RESP integers (:1000\r\n)
- [ ] Parse RESP bulk strings ($5\r\nhello\r\n)
- [ ] Parse RESP arrays (*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n)
- [ ] Serialize Go values back to RESP format
- [ ] Unit tests for all parse/serialize functions
- [ ] No lint errors

## Context

**Key Files:**
`internal/resp/`

**Background:**
RESP is the protocol Redis uses for client-server communication. We need to parse incoming commands and serailize responses.

## Boundaries

**In Scope:**
`internal/resp/` only

**Out of Scope:**
- TCP networking
- data storage
- command handling

## Interface Contract
Export these for other agents:
```go
// Value represents a RESP value
type Value struct {
    Type  byte     // '+', '-', ':', '$', '*'
    Str   string
    Num   int
    Array []Value
}

func Parse(reader *bufio.Reader) (Value, error)
func (v Value) Serialize() []byte
```

## Signal Protocol

**Signal BLOCKED when:**
- Unclear about RESP edge cases

**Signal DONE when:**
- All acceptance criteria met
