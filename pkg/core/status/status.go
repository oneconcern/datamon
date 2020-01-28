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

	// ErrCafsKey indicates an invalid hash string which couldn't be transformed in a valid hash key for the content-addressable FS (i;e. the value is too short)
	ErrCafsKey = errors.New("failed to create cafs key")

	// ErrReadAt is an error while performing a ReadAt operation on a bundle
	ErrReadAt = errors.New("error in bundle ReadAt")
)
