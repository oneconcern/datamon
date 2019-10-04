// Package status declares error constants returned by the variou
// implementations of the Store interface.
package status

import "errors"

var (
	// Sentinel errors returned by implementations of interfaces defined by storage

	// ErrNotExists indicates that the fetched object does not exist on storage
	ErrNotExists = errors.New("object doesn't exist")
)
