package cafs

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"fmt"

	lru "github.com/hashicorp/golang-lru"

	"go.uber.org/zap"

	"github.com/minio/blake2b-simd"
	"github.com/oneconcern/datamon/pkg/storage"
)

type Reader interface {
	io.ReadCloser
	io.ReaderAt
}

type ReaderOption func(reader *chunkReader)

func TruncateLeaf(t bool) ReaderOption {
	return func(reader *chunkReader) {
		reader.leafTruncation = t
	}
}

func Keys(keys []Key) ReaderOption {
	return func(reader *chunkReader) {
		reader.keys = keys
	}
}

func VerifyHash(t bool) ReaderOption {
	return func(reader *chunkReader) {
		reader.verifyHash = t
	}
}

func SetCache(lru *lru.Cache) ReaderOption {
	return func(reader *chunkReader) {
		reader.lru = lru
	}
}

func SetLeafPool(leafPool *leafFreelist) ReaderOption {
	return func(reader *chunkReader) {
		reader.leafPool = leafPool
	}
}

func ConcurrentChunkWrites(concurrentChunkWrites int) ReaderOption {
	return func(reader *chunkReader) {
		reader.concurrentChunkWrites = concurrentChunkWrites
	}
}

func newReader(blobs storage.Store, hash Key, leafSize uint32, prefix string,
	zl *zap.Logger,
	opts ...ReaderOption) (Reader, error) {
	c := &chunkReader{
		fs:                    blobs,
		hash:                  hash,
		leafSize:              leafSize,
		currLeaf:              make([]byte, 0),
		concurrentChunkWrites: 3,
		zl: zl,
	}

	for _, apply := range opts {
		apply(c)
	}
	var err error
	if c.keys == nil {
		// ??? distinguish these two functions?
		if c.verifyHash {
			c.keys, err = LeafsForHash(blobs, hash, leafSize, prefix)
		} else {
			c.keys, err = leafsForHashInternVerify(blobs, hash, leafSize, prefix)
		}
		if err != nil {
			return nil, err
		}

	}
	return c, nil
}

type chunkReader struct {
	fs       storage.Store
	leafSize uint32
	hash     Key
	prefix   string
	keys     []Key
	idx      int

	rdr                   io.ReadCloser
	readSoFar             int
	lastChunk             bool
	leafTruncation        bool
	currLeaf              []byte
	verifyHash            bool
	lru                   *lru.Cache
	leafPool *leafFreelist
	zl *zap.Logger
	concurrentChunkWrites int
}

func (r *chunkReader) Close() error {
	if r.rdr != nil {
		return r.rdr.Close()
	}
	return nil
}

type cafsWriterAt struct {
	written int64
	w       io.WriterAt
	offset  int64
}

func (cw *cafsWriterAt) Write(p []byte) (n int, err error) {
	written, err := cw.w.WriteAt(p, cw.offset+cw.written) // io.WriteAt is expected to be thread safe
	cw.written += int64(written)
	return written, err
}

type serialReader struct {
	reader io.Reader
}

func (s *serialReader) Read(data []byte) (int, error) {
	return s.reader.Read(data)
}

func (r *chunkReader) WriteTo(writer io.Writer) (n int64, err error) {
	// WriteAt
	w, ok := writer.(io.WriterAt)
	if !ok {
		sR := &serialReader{ //Warp reader to avoid io.Copy from calling WriteTo in a loop.
			reader: r,
		}
		return io.Copy(writer, sR)
	}

	errC := make(chan error, len(r.keys))
	writtenC := make(chan int64, len(r.keys))
	var wg sync.WaitGroup
	if len(r.keys) == 0 {
		return 0, nil
	}
	// Start a go routine for each key and give the offset to write at.
	concurrentChunkWrites := r.concurrentChunkWrites
	if concurrentChunkWrites < 1 {
		concurrentChunkWrites = 1
	}
	concurrencyControl := make(chan struct{}, concurrentChunkWrites)
	for index, key := range r.keys {
		wg.Add(1)
		var truncation uint32
		if r.leafTruncation {
			truncation = 32 * 1024 // Buffer size used by io.Copy
		}
		i := int64(index) * int64(r.leafSize-truncation)
		concurrencyControl <- struct{}{}
		go func(writeAt int64, writer io.WriterAt, key Key, cafs storage.Store, wg *sync.WaitGroup) {
			defer func() {
				<-concurrencyControl
				wg.Done()
			}()
			rdr, err := cafs.Get(context.Background(), key.StringWithPrefix(r.prefix)) // thread safe
			if err != nil {
				errC <- err
				return
			}
			w := &cafsWriterAt{
				w:      writer,
				offset: writeAt,
			}
			written, err := io.Copy(w, rdr) // io.WriteAt is expected to be thread safe.
			if err != nil {
				errC <- err
				return
			}
			writtenC <- written
		}(i, w, key, r.fs, &wg)
	}
	var count int
	var written int64
	wg.Wait()
	for {
		select {
		case w := <-writtenC:
			count++
			written += w
			if count == len(r.keys) {
				return written, nil
			}
		case errC := <-errC:
			return 0, errC
		}
	}

}

func calculateKeyAndOffset(off int64, leafSize uint32) (index int64, offset int64) {
	index = off / int64(leafSize)
	offset = off % int64(leafSize)
	return
}

func chunkReaderReadAtHelperReadLeaf(r *chunkReader, k Key) (
	*leafBuffer,
	error) {

	rdr, e := r.fs.Get(context.Background(), k.StringWithPrefix(r.prefix))
	if e != nil {
		return nil, e
	}

	lb := r.leafPool.get()
	var n int
	for {
		m, e := rdr.Read(lb.slice[n:cap(lb.slice)])
		if m < 0 {
			panic(fmt.Errorf("read negative number of bytes"))
		}
		n += m
		lb.slice = lb.slice[:n]
		if e == io.EOF {
			return lb, nil // e is EOF, so return nil explicitly
		}
		if e != nil {
			r.leafPool.put(lb)
			return lb, e
		}
	}
}

func (r *chunkReader) ReadAt(p []byte, off int64) (n int, err error) {

	// Calculate first key and offset.
	index, offset := calculateKeyAndOffset(off, r.leafSize)
	if index >= int64(len(r.keys)) {
		return 0, nil
	}

	var read int

	addToCache := func(key Key, buffer *leafBuffer) {
		if buffer == nil {
			r.zl.Info("attempted to add nil buffer to cache")
			return
		}
		r.zl.Info("adding element to lru cache")
		alreadyContained, evictionOnAdd := r.lru.ContainsOrAdd(key.StringWithPrefix(r.prefix), buffer)
		if evictionOnAdd {
			r.zl.Info("lru cache evict on add")
		}
		if alreadyContained {
			r.zl.Info("lru cache add key already contained in cache")
			r.leafPool.put(buffer)
		}
	}

	// seekAhead makes sure the next blob in CAFS is already in memory when the reader reaches that blob.
	seekAhead := func() {
		nextIndex := index + 1
		if nextIndex >= int64(len(r.keys)) { return }
		if r.lru.Contains(r.keys[nextIndex].StringWithPrefix(r.prefix)) {
			return
		}
		go func(i int64) {
			k := r.keys[i]
			b, err := chunkReaderReadAtHelperReadLeaf(r, k)
			if err != nil {
				return
			}
			addToCache(k, b)
		}(nextIndex)
	}

	for {
		// Fetch Blob
		var buffer *leafBuffer
		key := r.keys[index]
		if b, ok := r.lru.Get(key.StringWithPrefix(r.prefix)); ok {
			buffer = b.(*leafBuffer)
		} else {
			buffer, err = chunkReaderReadAtHelperReadLeaf(r, key)
			if err != nil {
				return
			}
			addToCache(key, buffer)
		}
		seekAhead()

		// Read first leaf
		read = copy(p[n:], buffer.slice[offset:])
		n += read
		index++
		offset = 0
		if err != nil && !strings.Contains(err.Error(), "EOF") {
			return
		}
		if (len(p) == n) || (index >= int64(len(r.keys))) {
			return
		}
	}
}

func (r *chunkReader) Read(data []byte) (int, error) {
	bytesToRead := len(data)

	if r.lastChunk && r.rdr == nil {
		return 0, io.EOF
	}
	for {
		key := r.keys[r.idx]
		if r.rdr == nil {
			rdr, err := r.fs.Get(context.Background(), key.StringWithPrefix(r.prefix))
			if err != nil {
				return r.readSoFar, err
			}
			r.rdr = rdr
		}

		n, errRead := r.rdr.Read(data[r.readSoFar:])
		r.currLeaf = append(r.currLeaf, data[r.readSoFar:r.readSoFar+n]...)
		if errRead != nil {
			//#nosec
			r.rdr.Close()
			r.readSoFar += n
			if errRead == io.EOF { // we reached the end of the stream for this key
				r.idx++
				r.rdr = nil
				r.lastChunk = r.idx == len(r.keys)
				if r.verifyHash {
					nodeOffset := r.idx
					isLastNode := false
					/* ??? what? */
					if r.lastChunk && uint32(len(r.currLeaf)) != r.leafSize {
						nodeOffset--
						isLastNode = true
					}
					hasher, err := blake2b.New(&blake2b.Config{
						Size: blake2b.Size,
						Tree: &blake2b.Tree{
							Fanout:        0,
							MaxDepth:      2,
							LeafSize:      r.leafSize,
							NodeOffset:    uint64(nodeOffset),
							NodeDepth:     0,
							InnerHashSize: blake2b.Size,
							IsLastNode:    isLastNode,
						},
					})
					if err != nil {
						return 0, err
					}
					if _, err = hasher.Write(r.currLeaf); err != nil {
						return 0, err
					}
					leafKey, err := NewKey(hasher.Sum(nil))
					if err != nil {
						return 0, err
					}
					if key != leafKey {
						return 0, errors.New("hash verification failed")
					}
				}
				if r.lastChunk { // this was the last chunk, so also EOF for this hash
					if n == bytesToRead {
						return n, nil
					}
					return r.readSoFar, io.EOF
				}
				// move on to the next key
				r.currLeaf = make([]byte, 0)
				continue
			}
			return n, errRead
		}
		// we filled up the entire byte slice but still have data remaining in the reader,
		// we should move on to receive the next buffer
		r.readSoFar += n
		if r.readSoFar >= bytesToRead {
			r.readSoFar = 0
			// return without error
			return bytesToRead, nil
		}
	}
}
