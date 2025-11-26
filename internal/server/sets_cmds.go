// Package server implements Redis set command handlers.
package server

import (
	"github.com/scotro/mini-redis/internal/resp"
	"github.com/scotro/mini-redis/internal/store"
)

// setStore is the package-level SetStore used by set command handlers.
// This will be properly integrated with Server during the integration phase.
var setStore *store.MemorySetStore

func init() {
	setStore = store.NewSetStore()
}

// GetSetStore returns the package-level SetStore for testing.
func GetSetStore() *store.MemorySetStore {
	return setStore
}

// ResetSetStore resets the package-level SetStore (for testing).
func ResetSetStore() {
	setStore = store.NewSetStore()
}

// handleSAdd handles the SADD command.
// SADD key member [member ...]
// Returns the number of new members added.
func (s *Server) handleSAdd(args []resp.Value) resp.Value {
	if len(args) < 2 {
		return respError("ERR wrong number of arguments for 'sadd' command")
	}

	key := args[0].Str

	// Check for type conflict with string store
	if _, exists := s.store.Get(key); exists {
		return respError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	members := make([]string, len(args)-1)
	for i, arg := range args[1:] {
		members[i] = arg.Str
	}

	added := s.setStore.SAdd(key, members...)
	return respInteger(added)
}

// handleSRem handles the SREM command.
// SREM key member [member ...]
// Returns the number of members removed.
func (s *Server) handleSRem(args []resp.Value) resp.Value {
	if len(args) < 2 {
		return respError("ERR wrong number of arguments for 'srem' command")
	}

	key := args[0].Str

	// Check for type conflict with string store
	if _, exists := s.store.Get(key); exists {
		return respError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	members := make([]string, len(args)-1)
	for i, arg := range args[1:] {
		members[i] = arg.Str
	}

	removed := s.setStore.SRem(key, members...)
	return respInteger(removed)
}

// handleSMembers handles the SMEMBERS command.
// SMEMBERS key
// Returns all members of the set.
func (s *Server) handleSMembers(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return respError("ERR wrong number of arguments for 'smembers' command")
	}

	key := args[0].Str

	// Check for type conflict with string store
	if _, exists := s.store.Get(key); exists {
		return respError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	members := s.setStore.SMembers(key)
	array := make([]resp.Value, len(members))
	for i, m := range members {
		array[i] = respBulkString(m)
	}

	return resp.Value{Type: resp.TypeArray, Array: array}
}

// handleSIsMember handles the SISMEMBER command.
// SISMEMBER key member
// Returns 1 if member exists, 0 otherwise.
func (s *Server) handleSIsMember(args []resp.Value) resp.Value {
	if len(args) != 2 {
		return respError("ERR wrong number of arguments for 'sismember' command")
	}

	key := args[0].Str
	member := args[1].Str

	// Check for type conflict with string store
	if _, exists := s.store.Get(key); exists {
		return respError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	if s.setStore.SIsMember(key, member) {
		return respInteger(1)
	}
	return respInteger(0)
}

// handleSCard handles the SCARD command.
// SCARD key
// Returns the cardinality (size) of the set.
func (s *Server) handleSCard(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return respError("ERR wrong number of arguments for 'scard' command")
	}

	key := args[0].Str

	// Check for type conflict with string store
	if _, exists := s.store.Get(key); exists {
		return respError("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	return respInteger(s.setStore.SCard(key))
}

// handleSInter handles the SINTER command.
// SINTER key [key ...]
// Returns the intersection of all given sets.
func (s *Server) handleSInter(args []resp.Value) resp.Value {
	if len(args) < 1 {
		return respError("ERR wrong number of arguments for 'sinter' command")
	}

	keys := make([]string, len(args))
	for i, arg := range args {
		key := arg.Str
		// Check for type conflict with string store
		if _, exists := s.store.Get(key); exists {
			return respError("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
		keys[i] = key
	}

	members := s.setStore.SInter(keys...)
	array := make([]resp.Value, len(members))
	for i, m := range members {
		array[i] = respBulkString(m)
	}

	return resp.Value{Type: resp.TypeArray, Array: array}
}
