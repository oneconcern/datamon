package cafs

import (
	"bytes"
	"context"
	"io"

	"github.com/oneconcern/trumpet/pkg/blob"
	"github.com/oneconcern/trumpet/pkg/blob/localfs"

	units "github.com/docker/go-units"
)

func Backend(store blob.Store) Option {
	return func(w *defaultFs) {
		w.fs = store
	}
}

// LeafSize configuration for the blake2b hashes
func LeafSize(sz uint32) Option {
	return func(w *defaultFs) {
		w.leafSize = sz
	}
}

// Option to configure content addressable FS components
type Option func(*defaultFs)

// Fs implementations provide content-addressable filesystem operations
type Fs interface {
	Get(context.Context, Key) (io.ReadCloser, error)
	Put(context.Context, io.Reader) (Key, error)
}

// New creates a new file system operations instance for a repository
func New(opts ...Option) (Fs, error) {
	f := &defaultFs{
		fs:       localfs.New(nil),
		leafSize: uint32(5 * units.MiB),
	}

	for _, apply := range opts {
		apply(f)
	}
	return f, nil
}

type defaultFs struct {
	fs       blob.Store
	leafSize uint32
}

func (d *defaultFs) Put(ctx context.Context, src io.Reader) (Key, error) {
	w := d.writer()
	defer w.Close()

	_, err := io.Copy(w, src)
	if err != nil {
		return Key{}, err
	}

	key, keys, err := w.Flush()
	if err != nil {
		return Key{}, err
	}
	if err = w.Close(); err != nil {
		return Key{}, err
	}

	if err := d.fs.Put(ctx, key.String(), bytes.NewReader(append(keys, key[:]...))); err != nil {
		return Key{}, err
	}

	return key, nil
}

func (d *defaultFs) Get(ctx context.Context, hash Key) (io.ReadCloser, error) {
	return newReader(d.fs, hash.String(), d.leafSize)
}

func (d *defaultFs) writer() Writer {
	return &fsWriter{
		fs:       d.fs,
		leafSize: d.leafSize,
		buf:      make([]byte, d.leafSize),
	}
}
