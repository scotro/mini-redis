package transaction

import "errors"

// Transaction errors matching Redis error messages.
var (
	ErrNestedMulti        = errors.New("ERR MULTI calls can not be nested")
	ErrExecWithoutMulti   = errors.New("ERR EXEC without MULTI")
	ErrDiscardWithoutMulti = errors.New("ERR DISCARD without MULTI")
	ErrWatchInsideMulti   = errors.New("ERR WATCH inside MULTI is not allowed")
	ErrNotInMulti         = errors.New("ERR not in MULTI")
)
