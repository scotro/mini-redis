// Package server provides hash command handlers for the Redis server.
package server

import (
	"github.com/scotro/mini-redis/internal/resp"
	"github.com/scotro/mini-redis/internal/store"
)

// HashCommands handles Redis hash commands.
// This struct is designed to be integrated with Server during the integration phase.
type HashCommands struct {
	hashStore   store.HashStore
	stringStore store.Store // For type checking against string keys
}

// NewHashCommands creates a new HashCommands handler.
func NewHashCommands(hashStore store.HashStore, stringStore store.Store) *HashCommands {
	return &HashCommands{
		hashStore:   hashStore,
		stringStore: stringStore,
	}
}

// wrongTypeError is the standard Redis error for type mismatches.
const wrongTypeError = "WRONGTYPE Operation against a key holding the wrong kind of value"

// checkKeyType returns an error response if the key exists in the string store.
func (h *HashCommands) checkKeyType(key string) *resp.Value {
	if _, exists := h.stringStore.Get(key); exists {
		err := respError(wrongTypeError)
		return &err
	}
	return nil
}

// HandleHSet handles the HSET command.
// HSET key field value [field value ...]
// Returns the number of fields that were added (not updated).
func (h *HashCommands) HandleHSet(args []resp.Value) resp.Value {
	if len(args) < 3 {
		return respError("ERR wrong number of arguments for 'hset' command")
	}

	// Must have an even number of field-value pairs after the key
	if (len(args)-1)%2 != 0 {
		return respError("ERR wrong number of arguments for 'hset' command")
	}

	key := args[0].Str

	// Check for type conflict with string keys
	if err := h.checkKeyType(key); err != nil {
		return *err
	}

	// Extract field-value pairs
	fieldValues := make([]string, len(args)-1)
	for i := 1; i < len(args); i++ {
		fieldValues[i-1] = args[i].Str
	}

	count := h.hashStore.HSet(key, fieldValues...)
	return respInteger(count)
}

// HandleHGet handles the HGET command.
// HGET key field
// Returns the value of field, or nil if field or key doesn't exist.
func (h *HashCommands) HandleHGet(args []resp.Value) resp.Value {
	if len(args) != 2 {
		return respError("ERR wrong number of arguments for 'hget' command")
	}

	key := args[0].Str
	field := args[1].Str

	// Check for type conflict with string keys
	if err := h.checkKeyType(key); err != nil {
		return *err
	}

	value, exists := h.hashStore.HGet(key, field)
	if !exists {
		return respNullBulkString()
	}
	return respBulkString(value)
}

// HandleHDel handles the HDEL command.
// HDEL key field [field ...]
// Returns the number of fields that were removed.
func (h *HashCommands) HandleHDel(args []resp.Value) resp.Value {
	if len(args) < 2 {
		return respError("ERR wrong number of arguments for 'hdel' command")
	}

	key := args[0].Str

	// Check for type conflict with string keys
	if err := h.checkKeyType(key); err != nil {
		return *err
	}

	fields := make([]string, len(args)-1)
	for i := 1; i < len(args); i++ {
		fields[i-1] = args[i].Str
	}

	count := h.hashStore.HDel(key, fields...)
	return respInteger(count)
}

// HandleHGetAll handles the HGETALL command.
// HGETALL key
// Returns all fields and values as an array [field1, value1, field2, value2, ...].
func (h *HashCommands) HandleHGetAll(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return respError("ERR wrong number of arguments for 'hgetall' command")
	}

	key := args[0].Str

	// Check for type conflict with string keys
	if err := h.checkKeyType(key); err != nil {
		return *err
	}

	hash := h.hashStore.HGetAll(key)

	// Build the flat array response
	array := make([]resp.Value, 0, len(hash)*2)
	for field, value := range hash {
		array = append(array, respBulkString(field), respBulkString(value))
	}

	return resp.Value{Type: resp.TypeArray, Array: array}
}

// HandleHKeys handles the HKEYS command.
// HKEYS key
// Returns all field names as an array.
func (h *HashCommands) HandleHKeys(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return respError("ERR wrong number of arguments for 'hkeys' command")
	}

	key := args[0].Str

	// Check for type conflict with string keys
	if err := h.checkKeyType(key); err != nil {
		return *err
	}

	keys := h.hashStore.HKeys(key)

	array := make([]resp.Value, len(keys))
	for i, k := range keys {
		array[i] = respBulkString(k)
	}

	return resp.Value{Type: resp.TypeArray, Array: array}
}

// HandleHLen handles the HLEN command.
// HLEN key
// Returns the number of fields in the hash, or 0 if key doesn't exist.
func (h *HashCommands) HandleHLen(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return respError("ERR wrong number of arguments for 'hlen' command")
	}

	key := args[0].Str

	// Check for type conflict with string keys
	if err := h.checkKeyType(key); err != nil {
		return *err
	}

	length := h.hashStore.HLen(key)
	return respInteger(length)
}
