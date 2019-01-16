package cafs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/oneconcern/datamon/pkg/storage"

	"github.com/minio/blake2b-simd"
)

// Writer interface for a content addressable FS
type Writer interface {
	io.WriteCloser
	Flush() (Key, []byte, error)
}

type fsWriter struct {
	fs       storage.Store
	leafSize uint32
	leafs    []Key
	buf      []byte
	offset   uint32
	flushed  uint32
	pather   func(string) string
	prefix   string
}

func (w *fsWriter) Write(p []byte) (n int, err error) {
	for desired := uint32(len(p)); desired > 0; {
		ofd := w.offset + desired
		if ofd == w.leafSize { // sizes line up, flush and continue
			w.flush(false)
			continue
		}

		if ofd < w.leafSize {
			copy(w.buf[w.offset:], p[uint32(len(p))-desired:])
			w.offset += desired
			desired = 0
		} else {
			actual := w.leafSize - w.offset
			copy(w.buf[w.offset:], p[uint32(len(p))-desired:uint32(len(p))-desired+actual])
			desired -= actual
			w.offset += actual
		}
	}
	return len(p), nil
}

func (w *fsWriter) flush(isLastNode bool) (int, error) {
	hasher, err := blake2b.New(&blake2b.Config{
		Size: blake2b.Size,
		Tree: &blake2b.Tree{
			Fanout:        0,
			MaxDepth:      2,
			LeafSize:      w.leafSize,
			NodeOffset:    uint64(len(w.leafs)),
			NodeDepth:     0,
			InnerHashSize: blake2b.Size,
			IsLastNode:    isLastNode,
		},
	})
	if err != nil {
		return 0, err
	}

	_, err = hasher.Write(w.buf[:w.offset])
	if err != nil {
		return 0, fmt.Errorf("flush segment hash: %v", err)
	}

	leafKey, err := NewKey(hasher.Sum(nil))
	if err != nil {
		return 0, fmt.Errorf("flush key segment: %v", err)
	}
	w.leafs = append(w.leafs, leafKey)

	if w.pather == nil {
		// w.pather = func(lks string) string { return filepath.Join(lks[:3], lks[3:6], lks[6:]) }
		w.pather = func(lks string) string { return w.prefix + lks }
	}
	found, _ := w.fs.Has(context.TODO(), w.pather(leafKey.String()))
	if !found {
		err = w.fs.Put(context.TODO(), w.pather(leafKey.String()), bytes.NewReader(w.buf[:w.offset]))
		if err != nil {
			return 0, fmt.Errorf("write segment file: %v", err)
		}
		n := int(w.offset)
		w.offset = 0
		return n, nil
	} else {
		fmt.Printf("Duplicate blob:%s, bytes:%d\n", leafKey.String(), w.offset)
	}
	return 0, nil
}

func (w *fsWriter) Flush() (Key, []byte, error) {
	atomic.StoreUint32(&w.flushed, 1)

	_, err := w.flush(true)
	if err != nil {
		return Key{}, nil, err
	}

	rhash, err := RootHash(w.leafs, w.leafSize)
	if err != nil {
		return Key{}, nil, fmt.Errorf("flush make root hash: %v", err)
	}

	leafHashes := make([]byte, len(w.leafs)*KeySize)
	for i, leaf := range w.leafs {
		offset := KeySize * i
		copy(leafHashes[offset:offset+KeySize], leaf[:])
	}
	return rhash, leafHashes, nil
}

func (w *fsWriter) Close() error {
	if !atomic.CompareAndSwapUint32(&w.flushed, 1, 0) {
		return fmt.Errorf("stream closed without being flushed")
	}
	return nil
}
