// Package server provides transaction command handlers for the Redis server.
package server

import (
	"github.com/scotro/mini-redis/internal/resp"
	"github.com/scotro/mini-redis/internal/transaction"
)

// TransactionHandler handles Redis transaction commands (MULTI, EXEC, DISCARD, WATCH, UNWATCH).
// This struct is designed to be integrated with Server during the integration phase.
type TransactionHandler struct {
	tx             *transaction.Transaction
	versionTracker transaction.VersionTracker
}

// NewTransactionHandler creates a new TransactionHandler.
// The versionTracker should be shared with the store for WATCH to work correctly.
func NewTransactionHandler(versionTracker transaction.VersionTracker) *TransactionHandler {
	return &TransactionHandler{
		tx:             transaction.New(),
		versionTracker: versionTracker,
	}
}

// Transaction returns the underlying transaction, allowing the server to check state.
func (h *TransactionHandler) Transaction() *transaction.Transaction {
	return h.tx
}

// HandleMulti handles the MULTI command.
// MULTI
// Returns OK if successful, error if already in a transaction.
func (h *TransactionHandler) HandleMulti(args []resp.Value) resp.Value {
	if len(args) != 0 {
		return respError("ERR wrong number of arguments for 'multi' command")
	}

	if err := h.tx.Begin(); err != nil {
		return respError(err.Error())
	}

	return respSimpleString("OK")
}

// HandleExec handles the EXEC command.
// EXEC
// Returns an array of results from all queued commands, or nil if WATCH failed.
func (h *TransactionHandler) HandleExec(args []resp.Value, executor transaction.CommandExecutor) resp.Value {
	if len(args) != 0 {
		return respError("ERR wrong number of arguments for 'exec' command")
	}

	var getVersion transaction.VersionGetter
	if h.versionTracker != nil {
		getVersion = h.versionTracker.GetVersion
	}

	results, err := h.tx.Exec(executor, getVersion)
	if err != nil {
		return respError(err.Error())
	}

	// nil results means WATCH failed
	if results == nil {
		return resp.Value{Type: resp.TypeArray, Null: true}
	}

	return resp.Value{Type: resp.TypeArray, Array: results}
}

// HandleDiscard handles the DISCARD command.
// DISCARD
// Returns OK if successful, error if not in a transaction.
func (h *TransactionHandler) HandleDiscard(args []resp.Value) resp.Value {
	if len(args) != 0 {
		return respError("ERR wrong number of arguments for 'discard' command")
	}

	if err := h.tx.Discard(); err != nil {
		return respError(err.Error())
	}

	return respSimpleString("OK")
}

// HandleWatch handles the WATCH command.
// WATCH key [key ...]
// Returns OK if successful, error if called inside a transaction.
func (h *TransactionHandler) HandleWatch(args []resp.Value) resp.Value {
	if len(args) < 1 {
		return respError("ERR wrong number of arguments for 'watch' command")
	}

	keys := make([]string, len(args))
	for i, arg := range args {
		keys[i] = arg.Str
	}

	var getVersion transaction.VersionGetter
	if h.versionTracker != nil {
		getVersion = h.versionTracker.GetVersion
	} else {
		// Default to returning 0 for all keys if no version tracker
		getVersion = func(key string) int64 { return 0 }
	}

	if err := h.tx.Watch(getVersion, keys...); err != nil {
		return respError(err.Error())
	}

	return respSimpleString("OK")
}

// HandleUnwatch handles the UNWATCH command.
// UNWATCH
// Returns OK. Clears all watched keys.
func (h *TransactionHandler) HandleUnwatch(args []resp.Value) resp.Value {
	if len(args) != 0 {
		return respError("ERR wrong number of arguments for 'unwatch' command")
	}

	h.tx.Unwatch()
	return respSimpleString("OK")
}

// InTransaction returns true if currently in MULTI mode.
// This allows the server to decide whether to queue commands or execute them.
func (h *TransactionHandler) InTransaction() bool {
	return h.tx.InTransaction()
}

// QueueCommand queues a command during a transaction.
// Returns a QUEUED response if successful.
func (h *TransactionHandler) QueueCommand(cmd string, args []string) resp.Value {
	if err := h.tx.Queue(cmd, args); err != nil {
		return respError(err.Error())
	}
	return respSimpleString("QUEUED")
}

// IsTransactionCommand returns true if the command should be executed immediately
// even during a transaction (MULTI, EXEC, DISCARD, WATCH, UNWATCH).
func IsTransactionCommand(cmd string) bool {
	switch cmd {
	case "MULTI", "EXEC", "DISCARD", "WATCH", "UNWATCH":
		return true
	default:
		return false
	}
}
