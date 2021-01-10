package context

import (
	"errors"
)

var (
	// ErrConfigPointerIsNil returns if pointer to config is nil
	ErrConfigPointerIsNil = errors.New("pointer to config is nil")
	// ErrLoggerPointerIsNil returns if pointer to logger is nil
	ErrLoggerPointerIsNil = errors.New("pointer to logger is nil")
)
