// Package status exports errors produced by the fuse package.
package status

import (
	"github.com/oneconcern/datamon/pkg/errors"
)

var (
	// ErrCafsKey indicates an invalid hash string which couldn't be transformed in a valid hash key for the content-addressable FS (i;e. the value is too short)
	ErrCafsKey = errors.New("failed to create cafs key")

	// ErrReadAt is an error while performing a ReadAt operation on a bundle
	ErrReadAt = errors.New("error in bundle ReadAt")
)
