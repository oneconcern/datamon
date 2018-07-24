package blob

import (
	"io"
)

// Store implementations know how to write entries to a K/V store.Store.
//
// Typically this is something file system-like. Examples are S3, local FS, NFS, ...
// Implementations of this interface are assumed to be fairly simple.
type Store interface {
	Has(string) (bool, error)
	Get(string) (io.ReadCloser, error)
	Put(string, io.Reader) error
	Delete(string) error
	Keys() ([]string, error)
	Clear() error
}
