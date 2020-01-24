package cafs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"

	lru "github.com/hashicorp/golang-lru"

	"github.com/oneconcern/datamon/pkg/dlogger"
	"github.com/oneconcern/datamon/pkg/storage"
)

const (
	defaultBufferTruncation uint32 = 32 * 1024 // default io.Copy buffer size
	defaultConcurrentWrite         = 3
	defaultFetchAhead              = 1
)

// Reader has the capability to Read, Close and ReadAt
type Reader interface {
	io.ReadCloser
	io.ReaderAt
}

var _ Reader = &chunkReader{}

type fetch struct {
	// fetch captures results from prefetching
	index int
	err   error
}

func defaultChunkReader(blobs storage.Store, hash Key, leafSize uint32) *chunkReader {
	return &chunkReader{
		fs:                    blobs,
		hash:                  hash,
		leafSize:              leafSize,
		currLeaf:              make([]byte, 0),
		concurrentChunkWrites: defaultConcurrentWrite,
		l:                     dlogger.MustGetLogger("info"),
		maxFetchAhead:         defaultFetchAhead,
	}
}

func newReader(blobs storage.Store, hash Key, leafSize uint32, opts ...ReaderOption) (Reader, error) {
	c := defaultChunkReader(blobs, hash, leafSize)
	for _, apply := range opts {
		apply(c)
	}

	var err error

	if c.lru == nil {
		// default LRU, no eviction handling
		c.lru, err = lru.New(BytesToBuffers(DefaultCacheSize, leafSize))
		if err != nil {
			return nil, err
		}
		c.lruLatch = &sync.Mutex{}
	}

	if c.leafTruncation {
		c.truncation = defaultBufferTruncation
	}

	if c.concurrentChunkWrites < 1 {
		c.concurrentChunkWrites = 1
	}

	if c.pather == nil {
		// default prefix path logic
		c.pather = func(lks Key) string { return lks.StringWithPrefix(c.prefix) }
	}

	if c.leafPool == nil {
		// nil instance: regular buffer allocation managed by gc
		var l *leafFreelist
		c.leafPool = l
	}

	c.readLeaf = readLeafFunc(c)
	c.addToCache = addToCacheFunc(c)
	c.seekAhead = seekAheadFunc(c)

	if c.keys == nil {
		c.l.Debug("cafs reader retrieving blob keys", zap.String("prefix", c.prefix))
		c.keys, err = LeavesForHash(blobs, hash, leafSize, c.prefix)
		if err != nil {
			return nil, err
		}
	}

	c.withPrefetch = c.maxFetchAhead > 0
	if c.withPrefetch {
		c.fetching = make(map[int]fetch, c.maxFetchAhead)
		c.fetchC = make(chan fetch)
		c.prefetchDoneC = make(chan struct{})

		var wg sync.WaitGroup
		wg.Add(1)
		go c.watchPrefetched(&wg, c.fetchC)

		// wind down prefetchers when the reader is gone
		runtime.SetFinalizer(c, func(obj interface{}) {
			r := obj.(*chunkReader)
			r.destroy()
		})
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
	concurrentChunkWrites int
	l                     *zap.Logger
	truncation            uint32
	withVerifyHash        bool
	readLeaf              func(Key, int, int, <-chan struct{}) (LeafBuffer, bool, error)
	seekAhead             func(int, int) bool
	pather                func(Key) string

	// caching
	addToCache func(Key, LeafBuffer)
	lru        *lru.Cache // TODO(fred): nice - factorize all this under a single FsCache interface
	lruLatch   sync.Locker
	leafPool   FreeList

	// prefetching
	withPrefetch  bool
	maxFetchAhead int
	fetching      map[int]fetch // a map of leaf indices currently being fetched by ReadAt
	fetchingLatch sync.Mutex
	fetchC        chan fetch
	fetcherWg     sync.WaitGroup
	prefetchDoneC chan struct{}

	// metrics
	fetched     uint64 // number of fetched leaf blocks
	requested   uint64 // number of fetch requests for leaf blocks
	wasted      uint64 // number of wasted fetched leaf blocks
	cacheHits   uint64
	cacheMisses uint64
}

// cafsWriterAt wraps an io.WriterAt and avoids conlicting method calls
type cafsWriterAt struct {
	written int64
	w       io.WriterAt
	offset  int64
}

// WriteTo writes data until there's no more data to write or when an error occurs.
// The return value n is the number of bytes written. Any error encountered during the write is also returned.
func (cw *cafsWriterAt) Write(p []byte) (n int, err error) {
	written, err := cw.w.WriteAt(p, cw.offset+cw.written) // io.WriteAt is expected to be thread safe
	cw.written += int64(written)
	return written, err
}

// serialReader wraps an io.Reader and avoids conlicting method calls
type serialReader struct {
	reader io.Reader
}

// Read reads up to len(p) bytes into p.
// It returns the number of bytes read (0 <= n <= len(p)) and any error encountered.
// If some data is available but not len(p) bytes, Read conventionally returns what is available instead of waiting for more.
func (s *serialReader) Read(data []byte) (int, error) {
	return s.reader.Read(data)
}

func (r *chunkReader) destroy() {
	if r.withPrefetch {
		r.l.Debug("winding down prefetchers")
		close(r.prefetchDoneC)
		// interrupt background prefetchers (avoids cache pollution by stale background prefetching goroutines)
		r.fetcherWg.Wait()
		close(r.fetchC)
	}
}

// Close this reader
func (r *chunkReader) Close() error {
	if r.rdr != nil {
		return r.rdr.Close()
	}
	return nil
}

// WriteTo writes data to w until there's no more data to write or when an error occurs.
// The return value n is the number of bytes written. Any error encountered during the write is also returned.
func (r *chunkReader) WriteTo(writer io.Writer) (n int64, err error) {
	// WriteAt
	w, ok := writer.(io.WriterAt)
	if !ok {
		sR := &serialReader{ //Wrap reader to avoid io.Copy from calling WriteTo in a loop.
			reader: r,
		}
		// TODO(fred): nice - io.CopyBuffer is probably better to get the copy working buffer aligned to leaf buffers
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
	concurrencyControl := make(chan struct{}, concurrentChunkWrites)
	for index, key := range r.keys {
		wg.Add(1)
		i := int64(index) * int64(r.leafSize-r.truncation)
		concurrencyControl <- struct{}{}
		go func(writeAt int64, writer io.WriterAt, key Key, cafs storage.Store, wg *sync.WaitGroup) {
			defer func() {
				<-concurrencyControl
				wg.Done()
			}()
			rdr, err := cafs.Get(context.Background(), r.pather(key)) // thread safe
			if err != nil {
				errC <- err
				return
			}
			w := &cafsWriterAt{
				w:      writer,
				offset: writeAt,
			}
			// TODO(fred): nice - io.CopyBuffer is probably better to get the copy working buffer aligned to leaf buffers
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

func (r *chunkReader) watchPrefetched(wg *sync.WaitGroup, fetchC <-chan fetch) {
	defer wg.Done()

	// collect prefetched leaf index
	for f := range fetchC {
		r.fetchingLatch.Lock()
		r.fetching[f.index] = f
		r.fetchingLatch.Unlock()
	}
}

func (r *chunkReader) doPrefetch(index, initiator int, logger *zap.Logger) (LeafBuffer, bool, error) {
	// doPrefetch is in charge of launching go routines to fetch-ahead of the current leaf whenever appropriate:
	// * fetching ahead is throttled by some parameter (default is 1 ahead)
	// * we request leaves ahead only once, unless they have been wiped from buffer cache (reduce wasted work)
	// * we cannot totally rule out wasted work because the buffer cache may not be able to retain all buffers
	//
	// A way to assert that there is no more wasted work than allowed by memory constraint:
	// $ DEBUG_TEST=1 go test -v -run Reader_All|grep -E '(wasted work)|(has read leaf)'|jq -c '{key: .key,msg: .msg}'|sort

	r.fetchingLatch.Lock()
	defer r.fetchingLatch.Unlock()

	if _, beingFetched := r.fetching[index+1]; !beingFetched {
		// starts immediately fetching next blob(s) in the background
		isRunning := r.seekAhead(index, initiator)
		if isRunning {
			// launched or about to be
			r.fetching[index+1] = fetch{}
		}
	}

	if f, beingFetched := r.fetching[index]; beingFetched {
		// the current leaf is being prefetched
		if index != initiator {
			// skip redundant prefetching
			return nil, false, nil
		}
		if f.err != nil {
			return nil, false, f.err
		}
		delete(r.fetching, index)

		r.lruLatch.Lock()
		b, ok := r.lru.Get(r.pather(r.keys[index]))
		if ok {
			lb := b.(LeafBuffer)
			lb.Pin()
			r.lruLatch.Unlock()
			// happy path: prefetching work is reused
			return lb, true, nil
		}
		r.lruLatch.Unlock()
		// previous work wasted: prefetch has completed but cache entry has been evicted: must restart
		logger.Debug("restart fetching wasted work")
		atomic.AddUint64(&r.wasted, 1)
	}

	// the current leaf is not being prefetched: carry on with read ops.

	// acquire a new buffer from the freelist: the buffer will be kept busy unless an error occurs
	lb := r.leafPool.Get()
	lb.Pin()
	r.fetching[index] = fetch{}
	return lb, false, nil
}

func calculateKeyAndOffset(off int64, leafSize uint32) (index int, offset int64) {
	index = int(off / int64(leafSize))
	offset = off % int64(leafSize)
	return
}

func readLeafFunc(r *chunkReader) func(Key, int, int, <-chan struct{}) (LeafBuffer, bool, error) {
	leafPoolLogger := r.l.With( // TODO(fred): nice - should replace debug logging by proper metrics instrumentation
		zap.Uint32("leaf size", r.leafSize),
		zap.Uint32("buffer size", r.leafPool.Size()),
	)
	return func(k Key, index, initiator int, doneC <-chan struct{}) (LeafBuffer, bool, error) {
		// readLeaf fetches an entire leaf from store
		logger := r.l.With(zap.String("prefix", r.prefix), zap.Stringer("key", k), zap.Int("index", index))
		logger.Debug("cafs reading leaf from store")
		rdr, err := r.fs.Get(context.Background(), r.pather(k))
		if err != nil {
			return nil, false, err
		}
		var (
			lb   LeafBuffer
			done bool
		)

		defer func() {
			if !done {
				logger.Debug("cafs has read leaf from store", zap.Error(err))
			}
		}()

		if !r.withPrefetch {
			// no prefetching
			lb = r.leafPool.Get()
			lb.Pin()
		} else {
			// launch background prefetchers as needed (this is a recursion).
			// The "done" status indicates that a prefetcher has completed
			// the retrieval of this list and extracted the buffer from the cache.
			lb, done, err = r.doPrefetch(index, initiator, logger)
			if err != nil {
				return nil, false, err
			}
			if lb == nil {
				// safeguard against nil returned to main loop
				if index == initiator {
					panic("programmer's error: buffer existence should be guaranteed")
				}
				// skip redundant fetch
				return nil, false, nil
			}
			if done {
				// short-circuit reads: work has already been done
				// hint the caller that this comes from cache (via prefetchers)
				return lb, true, nil
			}
		}

		if len(lb.Bytes()) > 0 {
			// safeguard against providing dirty buffers: a buffer should be either done and returned above, or clean
			panic(fmt.Sprintf("programmer's error: buffer should have 0 length, but has: %d", len(lb.Bytes())))
		}
		leafPoolLogger.Debug("cafs freelist pool",
			zap.Int("allocated buffers", r.leafPool.Buffers()),
			zap.Int("free buffers", r.leafPool.FreeBuffers()),
		)

		for { // TODO(fred): nice - this could be chunked further and parallelized using rdr.ReadAt
			select {
			case <-doneC:
				logger.Debug("interrupted readLeaf")
				return nil, false, nil
			default:
			}

			buffer := lb.Bytes()
			read, e := rdr.Read(buffer[len(buffer):cap(buffer)])
			if e != nil && e != io.EOF {
				// relinquish trashed buffer
				lb.Unpin()
				r.leafPool.Release(lb)
				return nil, false, e
			}

			if read < 0 {
				// safeguard against funny readers
				panic(fmt.Errorf("reader returned a negative number of bytes"))
			}

			_ = lb.Slice(0, len(buffer)+read)
			if e == io.EOF || read == 0 {
				break
			}
		}
		atomic.AddUint64(&r.fetched, 1)

		if r.withVerifyHash {
			logger.Debug("cafs reader ReadAt: hash verification")
			// NOTE: we follow the checksumming scheme adopted by the writer
			var (
				i      int
				isLast bool
			)
			if index+1 == len(r.keys) && uint32(len(lb.Bytes())) != r.leafSize {
				i = index
				isLast = true
			} else {
				i = index + 1
			}
			if err := r.verifyHash(k, lb.Bytes(), i, isLast); err != nil {
				return nil, false, err
			}
		}
		return lb, false, nil
	}
}

func addToCacheFunc(r *chunkReader) func(Key, LeafBuffer) {
	return func(key Key, buffer LeafBuffer) {
		if buffer == nil {
			return
		}
		alreadyContained, _ := r.lru.ContainsOrAdd(r.pather(key), buffer)
		if alreadyContained {
			// already in cache, relinquish freshly acquired buffer to pool
			r.leafPool.Release(buffer)
		}
	}
}

// seekAhead prefetches blobs ahead
func seekAheadFunc(r *chunkReader) func(int, int) bool {
	return func(index, initiator int) bool {
		nextIndex := index + 1
		if nextIndex >= len(r.keys) || nextIndex-initiator > r.maxFetchAhead {
			// don't attempt to fetch farther ahead than permitted
			return false
		}
		if r.lru.Contains(r.pather(r.keys[nextIndex])) {
			// don't attempt if already in cache
			return false
		}
		r.fetcherWg.Add(1)
		go func(i int, outputC chan<- fetch, doneC <-chan struct{}, wg *sync.WaitGroup) {
			defer wg.Done()
			k := r.keys[i]
			select {
			case <-doneC:
				r.l.Debug("interrupted seekAhead")
				return
			default:
			}
			r.l.Debug("prefetch started", zap.Int("new index", i), zap.Stringer("key", r.keys[i]))
			b, fromCache, err := r.readLeaf(k, i, initiator, r.prefetchDoneC)
			if err != nil {
				outputC <- fetch{err: err}
				return
			}
			if b == nil {
				// readLeaf has been skipped
				return
			}
			outputC <- fetch{index: i}
			b.Unpin()
			if !fromCache {
				r.addToCache(k, b)
			}
		}(nextIndex, r.fetchC, r.prefetchDoneC, &r.fetcherWg)

		// tells the caller prefetching is on the way
		return true
	}
}

func (r *chunkReader) ReadAt(data []byte, off int64) (readBytes int, err error) {
	bytesToRead := len(data)
	readLogger := r.l.With(zap.String("prefix", r.prefix))
	readLogger.Debug("Start cafs reader ReadAt", zap.Int("requested length", bytesToRead), zap.Int64("offset", off))
	defer func() {
		readLogger.Debug("End cafs reader ReadAt",
			zap.Int("requested length", bytesToRead),
			zap.Int64("offset", off),
			zap.Int("total read", readBytes),
			zap.Uint64("requested leaves", r.requested),
			zap.Uint64("fetched leaves", r.fetched),
			zap.Uint64("wasted fetched leaves", r.wasted),
			zap.Uint64("cache hits", r.cacheHits),
			zap.Uint64("cache misses", r.cacheMisses),
			zap.Error(err))
	}()

	// calculate first key and offset
	index, offset := calculateKeyAndOffset(off, r.leafSize)
	if index >= len(r.keys) {
		return 0, nil
	}

	for {
		// fetch leaf blobs
		var (
			buffer    LeafBuffer
			fromCache bool
		)
		key := r.keys[index]
		r.lruLatch.Lock()
		b, ok := r.lru.Get(r.pather(key))
		if ok {
			atomic.AddUint64(&r.cacheHits, 1)
			buffer = b.(LeafBuffer)
			// buffer is pinned: if the cache wants to relinquish this to the buffer pool (eviction)
			// actual buffer recycling will have to wait that we are done with writing this buffer.
			buffer.Pin()
			r.lruLatch.Unlock()
		} else {
			r.lruLatch.Unlock()
			// collects some metrics
			atomic.AddUint64(&r.cacheMisses, 1)
			atomic.AddUint64(&r.requested, 1)
			buffer, fromCache, err = r.readLeaf(key, index, index, r.prefetchDoneC) // readLeaf returns a pinned buffer
			if err != nil {
				return
			}
		}

		readBytes += copy(data[readBytes:], buffer.Bytes()[offset:])
		buffer.Unpin()

		if !ok && !fromCache {
			// we are done with this buffer: stash it to the cache for reuse
			r.addToCache(key, buffer)
		}

		if err != nil && !strings.Contains(err.Error(), "EOF") {
			return
		}

		index++
		offset = 0

		if (readBytes == bytesToRead) || (index >= len(r.keys)) {
			return
		}
	}
}

// Read some bytes. This implementation does not use caching or prefetching like ReadAt.
//
// NOTE: this does not use cache
// TODO(fred): great - this should be factorized with ReadAt
func (r *chunkReader) Read(data []byte) (int, error) {
	bytesToRead := len(data)
	r.l.Debug("cafs reader Read", zap.Int("length", bytesToRead))

	if r.lastChunk && r.rdr == nil {
		return 0, io.EOF
	}
	for {
		key := r.keys[r.idx]
		if r.rdr == nil {
			rdr, err := r.fs.Get(context.Background(), r.pather(key))
			if err != nil {
				return r.readSoFar, err
			}
			r.rdr = rdr
		}

		n, errRead := r.rdr.Read(data[r.readSoFar:])
		r.currLeaf = append(r.currLeaf, data[r.readSoFar:r.readSoFar+n]...)
		if errRead != nil {
			r.rdr.Close() // TODO(fred): nice - why are we ignoring errors here?
			r.readSoFar += n
			if errRead == io.EOF { // we reached the end of the stream for this key
				r.idx++
				r.rdr = nil
				r.lastChunk = r.idx == len(r.keys)
				if r.withVerifyHash {
					nodeOffset := r.idx
					isLastNode := false

					// NOTE: we follow the checksumming scheme adopted by the writer.
					// The writer behaves in a way a bit unexpected here: not only offets don't start at zero
					// as one might expect, but the last node is not flagged as the last one
					// when the content size is aligned with the leaf size.
					if r.lastChunk && uint32(len(r.currLeaf)) != r.leafSize {
						nodeOffset--
						isLastNode = true
					}
					r.l.Debug("cafs reader Read: hash verification", zap.Stringer("key", key))
					if err := r.verifyHash(key, r.currLeaf, nodeOffset, isLastNode); err != nil {
						return 0, err
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

func (r *chunkReader) verifyHash(key Key, data []byte, offset int, isLastNode bool) error {
	// NOTE(fred): caveats, following how the writer is working.
	//   * offset starts at 1, ... n
	//     This is inconsistent with the blake pkg documentation, which starts offset at 0
	//   * the isLastNode flag is not set when the data size is aligned with the leaf size.
	leafKey, err := KeyFromBytes(data, r.leafSize, uint64(offset), isLastNode)
	if err != nil {
		return err
	}
	if key != leafKey {
		r.l.Error("hash verification failed",
			zap.Stringer("key", key),
			zap.Int("keys", len(r.keys)),
			zap.Stringer("computed hash", leafKey),
			zap.Int("node offset", offset),
			zap.Bool("isLastNode", isLastNode),
		)
		return errors.New("hash verification failed")
	}
	return nil
}
