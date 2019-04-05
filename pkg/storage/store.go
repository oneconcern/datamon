// Copyright © 2018 One Concern

package storage

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
)

type errString string

const MaxObjectSizeInMemory = 2 * 1024 * 1024 * 1024 // 2 gigs
func (e errString) Error() string                    { return string(e) }

const (
	IfNotPresent = true
	OverWrite    = false
)

const (
	ErrNotFound     errString = "not found"
	ErrForbidden    errString = "forbidden"
	ErrNotSupported errString = "not supported"
	ErrExists       errString = "exists already"
	ErrObjectTooBig errString = "object too big to be read into memory"
)

// Store implementations know how to write entries to a K/V model.Store.
//
// Typically this is something file system-like. Examples are S3, local FS, NFS, ...
// Implementations of this interface are assumed to be fairly simple.
type Store interface {
	String() string
	Has(context.Context, string) (bool, error)
	Get(context.Context, string) (io.ReadCloser, error)
	GetAt(context.Context, string) (io.ReaderAt, error)
	Put(context.Context, string, io.Reader, bool) error
	Delete(context.Context, string) error
	Keys(context.Context) ([]string, error)
	Clear(context.Context) error
	KeysPrefix(ctx context.Context, pageToken string, prefix string, delimiter string, count int) ([]string, string, error)
}

type StoreCRC interface {
	PutCRC(context.Context, string, io.Reader, bool, uint32) error
}

func ReadTee(ctx context.Context, sStore Store, source string, dStore Store, destination string) ([]byte, error) {
	reader, err := sStore.Get(ctx, source)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	object, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	err = dStore.Put(ctx, destination, bytes.NewReader(object), IfNotPresent)
	if err != nil {
		return nil, err
	}
	return object, err
}

func PipeIO(writer io.Writer, reader io.ReadCloser) (n int64, err error) {
	pr, pw := io.Pipe()
	errC := make(chan error, 1)
	go func() {
		defer pw.Close()
		_, err := io.Copy(pw, reader)
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
