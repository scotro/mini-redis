package persistence

import "errors"

// ErrSaveInProgress is returned when attempting to start a background save while one is already running.
var ErrSaveInProgress = errors.New("background save already in progress")

// ErrNoSnapshot is returned when attempting to load a snapshot that doesn't exist.
var ErrNoSnapshot = errors.New("no snapshot file found")
