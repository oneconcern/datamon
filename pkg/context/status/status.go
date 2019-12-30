// Package status defines errors for datamon context
package status

import (
	"github.com/oneconcern/datamon/pkg/errors"
)

var (
	// datamon context errors

	// ErrInitMetadata indicates that we could not initialize the metadata store for this context
	ErrInitMetadata = errors.New("failed to initialize metadata store")

	// ErrInitBlob indicates that we could not initialize the blob store for this context
	ErrInitBlob = errors.New("failed to initialize blob store")

	// ErrInitVMetadata indicates that we could not initialize the versioned metadata for this context
	ErrInitVMetadata = errors.New("failed to initialize vmetadata store")

	// ErrInitWAL indicates that we could not initialize the write-ahead-log for this context
	ErrInitWAL = errors.New("failed to initialize wal store")

	// ErrInitRLog indicates that we could not initialize the read log for this context
	ErrInitRLog = errors.New("failed to initialize read log store")
)
