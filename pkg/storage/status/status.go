// Package status declares error constants returned by the various
// implementations of the Store interface.
//
// NOTE: such constants are located in a separate package to avoid
// creating undue cyclical dependencies between pkg/store and one
// of its implementions.
package status

import "errors"

var (
	// Sentinel errors returned by implementations of interfaces defined by storage

	// ErrNotExists indicates that the fetched object does not exist on storage
	ErrNotExists = errors.New("object doesn't exist")
)
