# Mini-Redis Roadmap

## Completed

- **Core Infrastructure**
  - RESP protocol parser (serialize/deserialize)
  - Thread-safe in-memory key-value store with TTL
  - TCP server with concurrent connection handling

- **String Commands**
  - PING, ECHO
  - GET, SET (with EX option), DEL
  - EXPIRE, TTL

---

## Tier 1: Data Types (Highly Parallel - 3 agents)

Dependency graph:
```
[lists] ───┐
[hashes] ──┼──→ [integration into server]
[sets] ────┘
```

### Lists
- **Commands**: LPUSH, RPUSH, LPOP, RPOP, LRANGE, LLEN
- **Complexity**: Moderate (doubly-linked list or slice)

### Hashes
- **Commands**: HSET, HGET, HDEL, HGETALL, HKEYS, HLEN
- **Complexity**: Simple (nested maps)

### Sets
- **Commands**: SADD, SREM, SMEMBERS, SISMEMBER, SCARD, SINTER
- **Complexity**: Moderate (map-based sets, intersection logic)

---

## Tier 2: Features (Parallel - 2-3 agents)

### Persistence
- RDB-style snapshots
- **Commands**: SAVE, BGSAVE
- Load snapshot on startup
- Configurable auto-save intervals

### Pub/Sub
- **Commands**: SUBSCRIBE, UNSUBSCRIBE, PUBLISH, PSUBSCRIBE
- Channel management
- Pattern matching subscriptions

### Transactions
- **Commands**: MULTI, EXEC, DISCARD, WATCH
- Command queuing
- Atomic execution

---

## Tier 3: Production Hardening

### Authentication & Security
- **Commands**: AUTH
- requirepass configuration
- Connection authentication state

### Configuration & Info
- **Commands**: CONFIG GET, CONFIG SET, INFO, DBSIZE, FLUSHDB, FLUSHALL
- Runtime configuration
- Server statistics

### Key Operations
- **Commands**: KEYS, EXISTS, RENAME, TYPE, RANDOMKEY
- Pattern matching for KEYS
- Key type tracking

### Atomic Counters
- **Commands**: INCR, INCRBY, DECR, DECRBY, INCRBYFLOAT
- String-to-integer conversion
- Overflow handling
