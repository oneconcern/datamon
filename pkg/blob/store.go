package blob

import (
	"context"
	"io"
)

type errString string

func (e errString) Error() string { return string(e) }

const (
	ErrNotFound     errString = "not found"
	ErrForbidden    errString = "forbidden"
	ErrNotSupported errString = "not supported"
	ErrExists       errString = "exists already"
)

// Store implementations know how to write entries to a K/V store.Store.
//
// Typically this is something file system-like. Examples are S3, local FS, NFS, ...
// Implementations of this interface are assumed to be fairly simple.
type Store interface {
	String() string
	Has(context.Context, string) (bool, error)
	Get(context.Context, string) (io.ReadCloser, error)
	Put(context.Context, string, io.Reader) error
	Delete(context.Context, string) error
	Keys(context.Context) ([]string, error)
	Clear(context.Context) error
}
