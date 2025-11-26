# Packet: pubsub

**Objective:** Implement Redis Pub/Sub with SUBSCRIBE, UNSUBSCRIBE, PUBLISH, and PSUBSCRIBE commands.

## Boundaries

**In scope:**
- `internal/pubsub/` (new package - create all files here)
- `internal/server/pubsub_cmds.go` (new file for pub/sub commands)

**Out of scope:**
- `internal/server/server.go` (do NOT modify - integration will happen later)
- `internal/store/` (pub/sub doesn't use the store layer)
- `internal/persistence/` (another agent's work)
- `internal/server/transaction_cmds.go` (another agent's work)

## Context

Pub/Sub in Redis is a messaging system where:
- Clients SUBSCRIBE to channels and receive messages
- Other clients PUBLISH messages to channels
- PSUBSCRIBE allows pattern-based subscriptions (e.g., `news.*`)

Follow existing patterns:
- Thread-safety with `sync.RWMutex`
- Interface-based design for testability
- Table-driven tests

## Implementation Requirements

### 1. PubSub Package (`internal/pubsub/`)

**Core types:**
```go
type PubSub struct {
    mu       sync.RWMutex
    channels map[string]map[*Subscriber]struct{}  // channel -> subscribers
    patterns map[string]map[*Subscriber]struct{}  // pattern -> subscribers
}

type Subscriber struct {
    ID       string
    Messages chan Message
}

type Message struct {
    Type    string  // "message", "pmessage", "subscribe", "unsubscribe"
    Channel string
    Pattern string  // for pmessage
    Payload string
}
```

**Methods:**
- `Subscribe(sub *Subscriber, channels ...string) int` - returns subscription count
- `Unsubscribe(sub *Subscriber, channels ...string) int`
- `PSubscribe(sub *Subscriber, patterns ...string) int`
- `PUnsubscribe(sub *Subscriber, patterns ...string) int`
- `Publish(channel, message string) int` - returns number of clients that received the message

### 2. Pattern Matching

For PSUBSCRIBE, support Redis glob patterns:
- `*` matches any sequence of characters
- `?` matches any single character
- `[abc]` matches any character in brackets

Use `path.Match` or implement simple glob matching.

### 3. Commands (`internal/server/pubsub_cmds.go`)

Create a `PubSubHandler` struct:
- `SUBSCRIBE channel [channel ...]` - Subscribe to channels
- `UNSUBSCRIBE [channel ...]` - Unsubscribe from channels
- `PUBLISH channel message` - Publish message, returns receiver count
- `PSUBSCRIBE pattern [pattern ...]` - Subscribe to patterns
- `PUNSUBSCRIBE [pattern ...]` - Unsubscribe from patterns

**Important:** SUBSCRIBE/PSUBSCRIBE put the connection into "subscription mode" where only SUBSCRIBE, UNSUBSCRIBE, PSUBSCRIBE, PUNSUBSCRIBE, PING, and QUIT are valid.

### 4. Message Format

SUBSCRIBE/UNSUBSCRIBE responses:
```
*3\r\n$9\r\nsubscribe\r\n$7\r\nmychan\r\n:1\r\n
```

Published messages to subscribers:
```
*3\r\n$7\r\nmessage\r\n$7\r\nmychan\r\n$5\r\nhello\r\n
```

## Acceptance Criteria

- [ ] `SUBSCRIBE` subscribes client to one or more channels
- [ ] `PUBLISH` delivers messages to all subscribed clients
- [ ] `PSUBSCRIBE` subscribes to pattern-matched channels
- [ ] `UNSUBSCRIBE` and `PUNSUBSCRIBE` work correctly
- [ ] Multiple clients can subscribe to the same channel
- [ ] Pattern matching works with `*`, `?`, `[abc]`
- [ ] Unit tests for pubsub package
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

## Notes

- Subscribers need a buffered channel to avoid blocking publishers
- Use a reasonable buffer size (e.g., 100 messages)
- If a subscriber's buffer is full, drop the message (Redis behavior)
- The handler will need access to a per-connection Subscriber instance
- For now, create the handler to work with a Subscriber passed in; connection integration happens later
