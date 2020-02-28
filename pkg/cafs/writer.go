package cafs

import (
	"bytes"
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"sync/atomic"

	"github.com/oneconcern/datamon/pkg/dlogger"
	"github.com/oneconcern/datamon/pkg/metrics"
	"github.com/oneconcern/datamon/pkg/storage"
	"go.uber.org/zap"
)

// Writer interface for a content addressable FS with WriterCloser and Flush capabilities.
//
// Flushing returns the final root hash and leaf keys computed on the written data.
type Writer interface {
	io.WriteCloser
	Flush() (Key, []byte, error)
}

var _ Writer = &fsWriter{}

type fsWriter struct {
	store               storage.Store    // CAFS backing store
	prefix              string           // Prefix for store paths
	leafSize            uint32           // Size of chunks
	leaves              []Key            // List of keys backing a file
	buf                 []byte           // Buffer stage a chunk == leafSize
	offset              int              // till where buffer is used
	flushed             uint32           // writer has been flushed to store
	pather              func(Key) string // pathing logic
	count               uint64           // total number of blob writes
	flushChan           chan blobFlush   // channel for parallel writes
	errC                chan error       // channel for errors during parallel writes
	flushThreadDoneChan chan struct{}
	maxGoRoutines       chan struct{} // Max number of concurrent writes
	blobFlushes         []blobFlush
	errors              []error
	l                   *zap.Logger

	metrics.Enable
	m *M
}

func defaultFsWriter(blobs storage.Store, leafSize uint32) *fsWriter {
	return &fsWriter{
		store:               blobs,
		leafSize:            leafSize,
		buf:                 make([]byte, leafSize),
		flushChan:           make(chan blobFlush),
		errC:                make(chan error),
		flushThreadDoneChan: make(chan struct{}),
		blobFlushes:         make([]blobFlush, 0),
		errors:              make([]error, 0),
		l:                   dlogger.MustGetLogger("info"),
	}
}

func newWriter(blobs storage.Store, leafSize uint32, opts ...WriterOption) Writer {
	w := defaultFsWriter(blobs, leafSize)
	for _, apply := range opts {
		apply(w)
	}
	if w.maxGoRoutines == nil {
		// default flush concurrency
		w.maxGoRoutines = make(chan struct{}, 1)
	}
	if w.pather == nil {
		// default prefix path logic
		w.pather = func(lks Key) string { return lks.StringWithPrefix(w.prefix) }
	}

	if w.MetricsEnabled() {
		w.m = w.EnsureMetrics("cafs", &M{}).(*M)
	}

	go w.flushThread()
	return w
}

func (w *fsWriter) Write(p []byte) (n int, err error) {
	written := 0
	for {
		if written == len(p) {
			return len(p), nil
		}
		// Copy p to w.buf
		writable := len(w.buf) - w.offset
		if len(p) < writable {
			writable = len(p)
		}
		c := copy(w.buf[w.offset:], p[written:writable])
		w.offset += c
		written += c
		if w.offset == len(w.buf) { // sizes line up, flush and continue
			w.count++ // next leaf
			w.maxGoRoutines <- struct{}{}
			go pFlush(
				false,
				w.buf,
				w.leafSize,
				w.count,
				w.flushChan,
				w.errC,
				w.maxGoRoutines,
				w.writeBlob,
				w.l,
			)
			w.buf = make([]byte, w.leafSize) // new buffer
			w.offset = 0                     // new offset for new buffer
			continue
		}
	}
}

type blobFlush struct {
	count uint64
	key   Key
}

func pFlush(
	isLastNode bool,
	buffer []byte,
	leafSize uint32,
	count uint64,
	flushChan chan blobFlush,
	errC chan<- error,
	maxGoRoutines chan struct{},
	blobWriter func([]byte, Key, uint64) error,
	l *zap.Logger,
) {
	defer func() {
		<-maxGoRoutines
	}()

	l.Debug("cafs writer computing leaf hash (partial flush)", zap.Uint64("leaf key index", count), zap.Bool("isLastNode", isLastNode))
	leafKey, err := KeyFromBytes(buffer, leafSize, count, isLastNode)
	if err != nil {
		errC <- err
		return
	}

	if err = blobWriter(buffer, leafKey, count); err != nil {
		errC <- err
		return
	}
	flushChan <- blobFlush{
		count: count,
		key:   leafKey,
	}
}

func (w *fsWriter) flush(isLastNode bool) (int, error) {
	if w.offset == 0 {
		return 0, nil
	}

	w.l.Debug("cafs writer computing leaf hash", zap.Int("leaf key index", len(w.leaves)), zap.Bool("isLastNode", isLastNode))
	leafKey, err := KeyFromBytes(w.buf[:w.offset], w.leafSize, uint64(len(w.leaves)), isLastNode)
	if err != nil {
		return 0, err
	}

	if err = w.writeBlob(w.buf[:w.offset], leafKey, uint64(w.offset)); err != nil {
		return 0, err
	}

	n := w.offset
	w.offset = 0
	w.leaves = append(w.leaves, leafKey)
	return n, nil
}

func (w *fsWriter) writeBlob(data []byte, key Key, n uint64) (err error) {
	found, _ := w.store.Has(context.TODO(), w.pather(key))
	if found {
		w.l.Info("Duplicate blob", zap.Stringer("key", key), zap.Uint64("offset", n))
		if w.MetricsEnabled() {
			w.m.Volume.Blobs.IncBlob("write")
			w.m.Volume.Blobs.IncDuplicate("write")
		}
		return nil
	}
	switch d := w.store.(type) {
	case storage.StoreCRC:
		crc := crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli))
		err = d.PutCRC(context.TODO(), w.pather(key), bytes.NewReader(data), storage.OverWrite, crc)
	default:
		err = w.store.Put(context.TODO(), w.pather(key), bytes.NewReader(data), storage.OverWrite)
	}
	if err != nil {
		return fmt.Errorf("write segment file: %s err:%w", w.pather(key), err)
	}
	w.l.Info("Uploading blob", zap.Stringer("key", key), zap.Int("chunk size", len(data)), zap.Uint64("offset", n))
	if w.MetricsEnabled() {
		w.m.Volume.Blobs.IncBlob("write")
		w.m.Volume.Blobs.Size(int64(len(data)), "write")
	}
	return
}

func (w *fsWriter) flushThread() {
	var err error
	var bf blobFlush
	notDone := true
	for notDone {
		select {
		case bf, notDone = <-w.flushChan:
			if notDone {
				w.blobFlushes = append(w.blobFlushes, bf)
			}
		case err = <-w.errC:
			w.errors = append(w.errors, err)
		}
	}
	w.flushThreadDoneChan <- struct{}{}
}

// don't Write() during Flush()
func (w *fsWriter) Flush() (Key, []byte, error) {
	for i := 0; i < cap(w.maxGoRoutines); i++ {
		w.maxGoRoutines <- struct{}{}
	}
	close(w.flushChan)
	<-w.flushThreadDoneChan
	if len(w.errors) != 0 {
		return Key{}, nil, w.errors[0]
	}
	w.leaves = make([]Key, len(w.blobFlushes))
	for _, bf := range w.blobFlushes {
		w.leaves[bf.count-1] = bf.key
	}
	atomic.StoreUint32(&w.flushed, 1)

	_, err := w.flush(true)
	if err != nil {
		return Key{}, nil, err
	}

	w.l.Debug("cafs writer computing root hash", zap.Int("leaf keys", len(w.leaves)))
	rhash, err := RootHash(w.leaves, w.leafSize)
	if err != nil {
		return Key{}, nil, fmt.Errorf("flush make root hash: %v", err)
	}

	leafHashes := make([]byte, len(w.leaves)*KeySize)
	for i, leaf := range w.leaves {
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
