package cafs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/oneconcern/datamon/pkg/storage"

	"github.com/minio/blake2b-simd"
)

const (
	maxGoRoutinesPerPut = 100
)

// Writer interface for a content addressable FS
type Writer interface {
	io.WriteCloser
	Flush() (Key, []byte, error)
}

type fsWriter struct {
	fs            storage.Store
	leafSize      uint32
	leafs         []Key
	buf           []byte
	offset        uint32
	flushed       uint32
	pather        func(string) string
	prefix        string
	count         uint64
	flushChan     chan blobFlush
	errC          chan error
	maxGoRoutines chan struct{}
	wg            sync.WaitGroup
}

func (w *fsWriter) Write(p []byte) (n int, err error) {
	for desired := uint32(len(p)); desired > 0; {
		ofd := w.offset + desired
		if ofd == w.leafSize { // sizes line up, flush and continue
			w.wg.Add(1)
			w.count++ // next leaf
			w.maxGoRoutines <- struct{}{}
			go pFlush(
				false,
				w.buf,
				int(w.offset),
				w.prefix,
				w.leafSize,
				w.count,
				w.flushChan,
				w.errC,
				w.maxGoRoutines,
				w.pather,
				w.fs,
				&w.wg,
			)
			w.buf = make([]byte, w.leafSize) // new buffer
			w.offset = 0                     // new offset for new buffer
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

type blobFlush struct {
	count uint64
	key   Key
}

func pFlush(
	isLastNode bool,
	buffer []byte,
	endOffset int,
	prefix string,
	leafSize uint32,
	count uint64,
	flushChan chan blobFlush,
	errC chan error,
	maxGoRoutines chan struct{},
	pather func(string) string,
	destination storage.Store,
	wg *sync.WaitGroup,
) {
	done := func() {
		wg.Done()
		<-maxGoRoutines
	}
	// Calculate hash value
	hasher, err := blake2b.New(&blake2b.Config{
		Size: blake2b.Size,
		Tree: &blake2b.Tree{
			Fanout:        0,
			MaxDepth:      2,
			LeafSize:      leafSize,
			NodeOffset:    count,
			NodeDepth:     0,
			InnerHashSize: blake2b.Size,
			IsLastNode:    isLastNode,
		},
	})
	if err != nil {
		errC <- err
		done()
		return
	}
	_, err = hasher.Write(buffer[:endOffset])
	if err != nil {
		errC <- fmt.Errorf("flush segment hash: %v", err)
		done()
		return
	}

	leafKey, err := NewKey(hasher.Sum(nil))
	if err != nil {
		errC <- fmt.Errorf("flush key segment: %v", err)
		done()
		return
	}

	// Write the blob
	if pather == nil {
		// w.pather = func(lks string) string { return filepath.Join(lks[:3], lks[3:6], lks[6:]) }
		pather = func(lks string) string { return prefix + lks }
	}
	found, _ := destination.Has(context.TODO(), pather(leafKey.String()))
	if !found {
		err = destination.Put(context.TODO(), pather(leafKey.String()), bytes.NewReader(buffer[:endOffset]), storage.OverWrite)
		if err != nil {
			errC <- fmt.Errorf("write segment file: %v", err)
			done()
			return
		}
		fmt.Printf("pUploading blob:%s, bytes:%d\n", leafKey.String(), endOffset)
	} else {
		fmt.Printf("pDuplicate blob:%s, bytes:%d\n", leafKey.String(), endOffset)
	}
	flushChan <- blobFlush{
		count: count,
		key:   leafKey,
	}
	done()
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

	if w.pather == nil {
		// w.pather = func(lks string) string { return filepath.Join(lks[:3], lks[3:6], lks[6:]) }
		w.pather = func(lks string) string { return w.prefix + lks }
	}
	found, _ := w.fs.Has(context.TODO(), w.pather(leafKey.String()))
	if !found {
		err = w.fs.Put(context.TODO(), w.pather(leafKey.String()), bytes.NewReader(w.buf[:w.offset]), storage.OverWrite)
		if err != nil {
			return 0, fmt.Errorf("write segment file: %v", err)
		}
		fmt.Printf("Uploading blob:%s, bytes:%d\n", leafKey.String(), w.offset)
	} else {
		fmt.Printf("Duplicate blob:%s, bytes:%d\n", leafKey.String(), w.offset)
	}

	n := int(w.offset)
	w.offset = 0
	w.leafs = append(w.leafs, leafKey)
	return n, nil
}

func (w *fsWriter) Flush() (Key, []byte, error) {
	w.leafs = make([]Key, 0, w.count)
	if w.count > 0 {
		w.wg.Wait()
		w.leafs = make([]Key, w.count)
		for {
			select {
			case bf := <-w.flushChan:
				w.count--
				w.leafs[bf.count-1] = bf.key
				if w.count == 0 {
					break
				}
			case err := <-w.errC:
				return Key{}, nil, err

			default:
			}
			if w.count == 0 {
				break
			}
		}
	}
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
