// Copyright © 2018 One Concern

package storage

import (
	"context"
	"io"
	"strconv"
	"time"
)

//go:generate moq -out ./mockstorage/store.go -pkg mockstorage . Store
//go:generate moq -out ./mockstorage/store_versioned.go -pkg mockstorage . StoreVersioned Store

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
	// Has this object in the store?
	Has(context.Context, string) (bool, error)
	// Get this object's backing bytes.
	Get(context.Context, string) (io.ReadCloser, error)
	// GetAttr looks up the Attributes of this object.
	GetAttr(context.Context, string) (Attributes, error)
	// Get this object's backing bytes.
	GetAt(context.Context, string) (io.ReaderAt, error)
	// Touch, like the usual *nix verb, touch(1), changes the object's modify time.
	// Unlike touch(1), it does _not_ create an object if it doesn't exist.
	Touch(context.Context, string) error
	// Put writes bytes to a named object in the store.
	Put(context.Context, string, io.Reader, bool) error
	// Delete removes the specified object from the store.
	// ??? design affordances re. versions? e.g. --
	// - opt to remove all versons
	// - opt (or different function elsewhere) to rollback to previous version?
	Delete(context.Context, string) error
	// ??? what's the intent of this function, again?
	Clear(context.Context) error

	// Keys returns all keys known to the store.
	// Depending on the implementation, some limit on the maximum number
	// of such returned keys may exist.
	Keys(context.Context) ([]string, error)

	// KeyPrefix provides a paginated key iterator using "pageToken" as the next starting point
	KeysPrefix(
		ctx context.Context,
		pageToken string,
		prefix string,
		delimiter string,
		count int,
	) ([]string, string, error)
}

// StoreCRC knows how to update an object with a computed CRC checksum
type StoreCRC interface {
	PutCRC(context.Context, string, io.Reader, bool, uint32) error
}

const (
	// GcsSentinelVersion happens to be the sentinel value used by the google-cloud-go library,
	// not a constant used by the GCS api.  Yet it is so named for consistency with datamon
	// internal api and because, for cleaner seperation of concerns, the datamon codebase
	// does not pre-suppose that this google-cloud-go-internal value will not change
	// sometime in the future, although we wouldn't mind if the google-cloud-go library
	// made the constant publicly visible.
	GcsSentinelVersion = -1
)

type Version struct {
	gcsGeneration int64
}

func (version *Version) String() string {
	return strconv.FormatInt(version.gcsGeneration, 10)
}

func (version *Version) GcsVersion() int64 {
	if version.gcsGeneration != 0 {
		return version.gcsGeneration
	}
	return GcsSentinelVersion
}

type StoreVersioned interface {
	// KeyVersions returns all versions of a given key
	// operating assumptions:
	// * versions are strings (likely true)
	// * paging is unnecessary (likely false)
	KeyVersions(context.Context, string) ([]Version, error)
}

func NewVersionGcs(gcsGeneration int64) Version {
	return Version{
		gcsGeneration: gcsGeneration,
	}
}

type Settings struct {
	Version Version
}

///

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
