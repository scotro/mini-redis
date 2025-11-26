// Package server contains persistence command handlers for the Redis server.
package server

import (
	"github.com/scotro/mini-redis/internal/persistence"
	"github.com/scotro/mini-redis/internal/resp"
)

// PersistenceHandler handles persistence-related Redis commands (SAVE, BGSAVE).
type PersistenceHandler struct {
	manager *persistence.Manager
}

// NewPersistenceHandler creates a new handler with the given persistence manager.
func NewPersistenceHandler(manager *persistence.Manager) *PersistenceHandler {
	return &PersistenceHandler{
		manager: manager,
	}
}

// HandleSave handles the SAVE command.
// SAVE - Synchronously saves the dataset to disk.
// Returns OK on success.
func (h *PersistenceHandler) HandleSave(args []resp.Value) resp.Value {
	if len(args) != 0 {
		return respError("ERR wrong number of arguments for 'save' command")
	}

	if err := h.manager.Save(); err != nil {
		return respError("ERR " + err.Error())
	}

	return respSimpleString("OK")
}

// HandleBGSave handles the BGSAVE command.
// BGSAVE - Asynchronously saves the dataset to disk.
// Returns "Background saving started" on success.
func (h *PersistenceHandler) HandleBGSave(args []resp.Value) resp.Value {
	if len(args) != 0 {
		return respError("ERR wrong number of arguments for 'bgsave' command")
	}

	if err := h.manager.BackgroundSave(); err != nil {
		if err == persistence.ErrSaveInProgress {
			return respError("ERR Background save already in progress")
		}
		return respError("ERR " + err.Error())
	}

	return respSimpleString("Background saving started")
}
