// Package status exports errors produced by the core package.
package status

import (
	"github.com/oneconcern/datamon/pkg/errors"
)

var (
	// ErrInterrupted signals that the current background processing has been interrupted
	ErrInterrupted = errors.New("background processing interrupted")

	// ErrNotFound indicates an object was not found
	ErrNotFound = errors.New("not found")

	// ErrUnexpectedUpdate indicates an update operation was attempted on some immutable store
	ErrUnexpectedUpdate = errors.New("unexpected update")

	// ErrConfigContext indicates an error while attempting to retrieve contexts from a remote config store
	ErrConfigContext = errors.New("error retrieving contexts from config store")
)
