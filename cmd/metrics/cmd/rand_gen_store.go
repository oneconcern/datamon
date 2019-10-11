package cmd

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"time"

	"github.com/oneconcern/datamon/pkg/storage"
)

// internal.RandStringBytesMaskImprSrc(15)

/**
 * this is a storage.Store implementation that generates random or patterned data on read,
 * the sort of data that's useful for gathering metrics.
 */

type readerAtReadCloser struct {
	rAt io.ReaderAt
	off int64
}

func (r *readerAtReadCloser) Close() error {
	return nil
}

func (r *readerAtReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadAt(p, r.off)
	r.off += int64(n)
	return n, err
}

func (r *readerAtReadCloser) ReadAt(p []byte, off int64) (int, error) {
	return r.rAt.ReadAt(p, off)
}

type byteFunc func(int64) (byte, error)

type byteFuncReaderAt struct {
	byteFunc byteFunc
}

func (r *byteFuncReaderAt) ReadAt(p []byte, off int64) (int, error) {
	var i int
	if off < 0 {
		return 0, errors.New("can't read at negative offset")
	}
	for ; i < len(p); i++ {
		b, err := r.byteFunc(off)
		if err != nil {
			return i, err
		}
		p[i] = b
	}
	return i, nil
}

func randByteFunc(max int64) byteFunc {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return func(off int64) (byte, error) {
		if off > max {
			return 0, io.EOF
		}
		return byte(r.Uint32()), nil
	}
}

func zeroOneChunkByteFunc(chunkBytes int64, max int64) byteFunc {
	return func(off int64) (byte, error) {
		if off > max {
			return 0, io.EOF
		}
		intraChunkOff := off % (2 * chunkBytes)
		if intraChunkOff < chunkBytes {
			return 0, nil
		}
		return 0xFF, nil
	}
}

func repeatingStripesByteFunc(stripe []byte, max int64) byteFunc {
	return func(off int64) (byte, error) {
		if off > max {
			return 0, io.EOF
		}
		return stripe[off%int64(len(stripe))], nil
	}
}

type genType uint8

const (
	genTypeRand = iota
	genTypeZeroOneChunks
	genTypeRepeatingStripes
)

type genStore struct {
	keyset     map[string]bool
	genType    genType
	max        int64
	chunkBytes int64
	stripe     []byte
}

func (gs genStore) String() string {
	return "random generator store"
}

func (gs genStore) Has(ctx context.Context, key string) (bool, error) {
	return gs.keyset[key], nil
}

func (gs genStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	rAt, err := gs.GetAt(ctx, key)
	if err != nil {
		return nil, err
	}
	return &readerAtReadCloser{rAt: rAt}, nil
}

func (gs genStore) GetAt(ctx context.Context, key string) (io.ReaderAt, error) {
	var fn byteFunc
	switch gs.genType {
	case genTypeRand:
		fn = randByteFunc(gs.max)
	case genTypeZeroOneChunks:
		if gs.chunkBytes < 1 {
			return nil, errors.New("genStore invalid")
		}
		fn = zeroOneChunkByteFunc(gs.chunkBytes, gs.max)
	case genTypeRepeatingStripes:
		if gs.stripe == nil || len(gs.stripe) == 0 {
			return nil, errors.New("genStore invalid")
		}
		fn = repeatingStripesByteFunc(gs.stripe, gs.max)
	}
	return &byteFuncReaderAt{byteFunc: fn}, nil
}

func (gs genStore) Put(ctx context.Context, key string, source io.Reader, exclusive bool) error {
	return errors.New("can't modify the generative store")
}

func (gs genStore) Delete(ctx context.Context, key string) error {
	return errors.New("can't modify the generative store")
}

func (gs genStore) Keys(ctx context.Context) ([]string, error) {
	keys := make([]string, 0)
	for k := range gs.keyset {
		keys = append(keys, k)
	}
	return keys, nil
}

func (gs genStore) KeysPrefix(ctx context.Context, token, prefix, delimiter string, count int) ([]string, string, error) {
	return nil, "", errors.New("unimplemented")
}

func (gs genStore) Clear(ctx context.Context) error {
	return errors.New("unimplemented")
}

func newGenStoreHBuildKeyset(keys []string) map[string]bool {
	ks := make(map[string]bool)
	for _, k := range keys {
		ks[k] = true
	}
	return ks
}

// nolint:deadcode,unused
func newGenStoreRand(keys []string, max int64) storage.Store {
	return genStore{
		keyset:  newGenStoreHBuildKeyset(keys),
		genType: genTypeRand,
		max:     max,
	}
}

// nolint:deadcode,unused
func newGenStoreZeroOneChunks(keys []string, max int64, chunkBytes int64) storage.Store {
	return genStore{
		keyset:     newGenStoreHBuildKeyset(keys),
		genType:    genTypeZeroOneChunks,
		max:        max,
		chunkBytes: chunkBytes,
	}
}

func newGenStoreRepeatingStripes(keys []string, max int64, stripe []byte) storage.Store {
	return genStore{
		keyset:  newGenStoreHBuildKeyset(keys),
		genType: genTypeRepeatingStripes,
		max:     max,
		stripe:  stripe,
	}
}
