// Copyright © 2018 One Concern

package storage

import (
	"context"
	"io"
	"time"
)

//go:generate moq -out ./mockstorage/store.go -pkg mockstorage . Store

const MaxObjectSizeInMemory = 2 * 1024 * 1024 * 1024 // 2 gigs

const (
	// Adding these to make code more readable when looking at Put Call
	NoOverWrite = true
	OverWrite   = false
)

type NewKey = bool

type Attributes struct {
	Created time.Time
	Updated time.Time
	Owner   string
}

// Store implementations know how to write entries to a K/V model.Store.
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
	Put(context.Context, string, io.Reader, NewKey) error
	Delete(context.Context, string) error
	Keys(context.Context) ([]string, error)
	Clear(context.Context) error
	KeysPrefix(ctx context.Context, pageToken string, prefix string, delimiter string, count int) ([]string, string, error)
}

type StoreCRC interface {
	PutCRC(context.Context, string, io.Reader, bool, uint32) error
}

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
