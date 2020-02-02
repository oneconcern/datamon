// Copyright © 2018 One Concern

package storage

import (
	"context"
	"io"
	"time"
)

//go:generate moq -out ./mockstorage/store.go -pkg mockstorage . Store

const (
	// Adding these to make code more readable when looking at Put Call

	// NoOverWrite does not accept clobbering of objects in store
	NoOverWrite = true

	// OverWrite accepts clobbering of objects in store
	OverWrite = false
)

// Attributes supported by object on this store
type Attributes struct {
	Created time.Time
	Updated time.Time
	Owner   string
}

// Store implementations know how to fetch and write entries from a and a K/V store.
//
// Typically this is something file system-like. Examples are S3, local FS, NFS, ...
// Implementations of this interface are assumed to be fairly simple.
type Store interface {
	String() string
	Has(context.Context, string) (bool, error)
	Get(context.Context, string) (io.ReadCloser, error)
	GetAttr(context.Context, string) (Attributes, error)
	GetAt(context.Context, string) (io.ReaderAt, error)
	Touch(context.Context, string) error
	Put(context.Context, string, io.Reader, bool) error
	Delete(context.Context, string) error
	Clear(context.Context) error

	// Keys returns all keys known to the store.
	// Depending on the implementation, some limit may exist on the maximum number of such returned keys
	Keys(context.Context) ([]string, error)

	// KeyPrefix provides a paginated key iterator using "pageToken" as the next starting point
	KeysPrefix(ctx context.Context, pageToken string, prefix string, delimiter string, count int) ([]string, string, error)
}

// StoreCRC knows how to update an object with a computed CRC checksum
type StoreCRC interface {
	PutCRC(context.Context, string, io.Reader, bool, uint32) error
}

// PipeIO copies data from a reader to a writer using io.Pipe
func PipeIO(writer io.Writer, reader io.Reader) (n int64, err error) {
	pr, pw := io.Pipe()
	errC := make(chan error, 1)
	go func() {
		defer pw.Close()
		_, err = io.Copy(pw, reader)
		if err != nil {
			errC <- err
		}
		close(errC)
	}()
	written, err := io.Copy(writer, pr)
	select {
	case err = <-errC:
		return 0, err
	default:
	}
	return written, err
}
