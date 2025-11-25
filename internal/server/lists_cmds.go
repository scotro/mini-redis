// Package server contains list command handlers for the Redis server.
package server

import (
	"strconv"

	"github.com/scotro/mini-redis/internal/resp"
	"github.com/scotro/mini-redis/internal/store"
)

// ListCommandHandler handles list-related Redis commands.
// This is designed to be integrated into the main Server struct.
type ListCommandHandler struct {
	listStore store.ListStore
	// stringStore is used for type checking against existing string keys
	stringStore store.Store
}

// NewListCommandHandler creates a new handler with the given stores.
func NewListCommandHandler(listStore store.ListStore, stringStore store.Store) *ListCommandHandler {
	return &ListCommandHandler{
		listStore:   listStore,
		stringStore: stringStore,
	}
}

// checkType verifies that an operation can be performed on a key.
// Returns a WRONGTYPE error if the key exists but is not a list.
func (h *ListCommandHandler) checkType(key string) *resp.Value {
	// Check if key exists as a string
	if _, exists := h.stringStore.Get(key); exists {
		errResp := respError("WRONGTYPE Operation against a key holding the wrong kind of value")
		return &errResp
	}
	return nil
}

// HandleLPush handles the LPUSH command.
// LPUSH key value [value ...]
// Prepends values to a list. Returns the length of the list after the push.
func (h *ListCommandHandler) HandleLPush(args []resp.Value) resp.Value {
	if len(args) < 2 {
		return respError("ERR wrong number of arguments for 'lpush' command")
	}

	key := args[0].Str

	// Type check
	if errResp := h.checkType(key); errResp != nil {
		return *errResp
	}

	values := make([]string, len(args)-1)
	for i, arg := range args[1:] {
		values[i] = arg.Str
	}

	length := h.listStore.LPush(key, values...)
	return respInteger(length)
}

// HandleRPush handles the RPUSH command.
// RPUSH key value [value ...]
// Appends values to a list. Returns the length of the list after the push.
func (h *ListCommandHandler) HandleRPush(args []resp.Value) resp.Value {
	if len(args) < 2 {
		return respError("ERR wrong number of arguments for 'rpush' command")
	}

	key := args[0].Str

	// Type check
	if errResp := h.checkType(key); errResp != nil {
		return *errResp
	}

	values := make([]string, len(args)-1)
	for i, arg := range args[1:] {
		values[i] = arg.Str
	}

	length := h.listStore.RPush(key, values...)
	return respInteger(length)
}

// HandleLPop handles the LPOP command.
// LPOP key
// Removes and returns the first element of the list.
func (h *ListCommandHandler) HandleLPop(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return respError("ERR wrong number of arguments for 'lpop' command")
	}

	key := args[0].Str

	// Type check
	if errResp := h.checkType(key); errResp != nil {
		return *errResp
	}

	value, ok := h.listStore.LPop(key)
	if !ok {
		return respNullBulkString()
	}
	return respBulkString(value)
}

// HandleRPop handles the RPOP command.
// RPOP key
// Removes and returns the last element of the list.
func (h *ListCommandHandler) HandleRPop(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return respError("ERR wrong number of arguments for 'rpop' command")
	}

	key := args[0].Str

	// Type check
	if errResp := h.checkType(key); errResp != nil {
		return *errResp
	}

	value, ok := h.listStore.RPop(key)
	if !ok {
		return respNullBulkString()
	}
	return respBulkString(value)
}

// HandleLRange handles the LRANGE command.
// LRANGE key start stop
// Returns elements from start to stop (inclusive). Supports negative indices.
func (h *ListCommandHandler) HandleLRange(args []resp.Value) resp.Value {
	if len(args) != 3 {
		return respError("ERR wrong number of arguments for 'lrange' command")
	}

	key := args[0].Str

	// Type check
	if errResp := h.checkType(key); errResp != nil {
		return *errResp
	}

	start, err := strconv.Atoi(args[1].Str)
	if err != nil {
		return respError("ERR value is not an integer or out of range")
	}

	stop, err := strconv.Atoi(args[2].Str)
	if err != nil {
		return respError("ERR value is not an integer or out of range")
	}

	elements := h.listStore.LRange(key, start, stop)

	array := make([]resp.Value, len(elements))
	for i, elem := range elements {
		array[i] = respBulkString(elem)
	}

	return resp.Value{Type: resp.TypeArray, Array: array}
}

// HandleLLen handles the LLEN command.
// LLEN key
// Returns the length of the list.
func (h *ListCommandHandler) HandleLLen(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return respError("ERR wrong number of arguments for 'llen' command")
	}

	key := args[0].Str

	// Type check - for LLEN, we return 0 if key doesn't exist or is a list
	// We only error if it's a different type
	if errResp := h.checkType(key); errResp != nil {
		return *errResp
	}

	length := h.listStore.LLen(key)
	return respInteger(length)
}
