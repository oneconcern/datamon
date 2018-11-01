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
	Put(context.Context, string, io.Reader) error
	Delete(context.Context, string) error
	Keys(context.Context) ([]string, error)
	Clear(context.Context) error
}


func ReadTee(ctx context.Context, sStore Store, source string,  dStore Store, destination string) ([]byte, error) {
	reader, err := sStore.Get(ctx, source)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	object, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	err = dStore.Put(ctx,destination, bytes.NewReader(object))
	if err != nil {
		return nil, err
	}
	return object, err
}
