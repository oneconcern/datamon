package cafs

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/davecgh/go-spew/spew"
	lru "github.com/hashicorp/golang-lru"
	"github.com/oneconcern/datamon/internal/rand"
	"github.com/oneconcern/datamon/pkg/dlogger"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func keyFromFile(t testing.TB, pth string) Key {
	rhash := readTextFile(t, pth)
	rkey, err := KeyFromString(string(rhash))
	require.NoError(t, err)
	return rkey
}

func readTextFile(t testing.TB, pth string) []byte {
	v, err := ioutil.ReadFile(pth)
	require.NoError(t, err)
	return v
}

func TestChunkReader_SmallOnly(t *testing.T) {
	blobs := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(destDir, "cafs")))
	for _, tf := range testFiles(destDir) {
		if tf.Parts > 1 {
			continue
		}
		verifyChunkReader(t, blobs, tf)
		verifyChunkReaderAt(t, blobs, tf)
	}
}

func verifyChunkReader(t testing.TB, blobs storage.Store, tf testFile) {
	rkey := keyFromFile(t, tf.RootHash)
	rdr, err := newReader(blobs, rkey, leafSize,
		ReaderPrefix(""),
	)
	require.NoError(t, err)
	defer rdr.Close()

	actual, err := ioutil.ReadAll(rdr)
	require.NoError(t, err)

	expected := readTextFile(t, tf.Original)
	require.Equal(t, len(expected), len(actual))
	require.Equal(t, expected, actual)
}

func verifyChunkReaderAt(t testing.TB, blobs storage.Store, tf testFile, opts ...ReaderOption) {
	rkey := keyFromFile(t, tf.RootHash)
	offset := int64(11)
	size := 2 * int(leafSize)

	r, err := newReader(blobs, rkey, leafSize, opts...)
	require.NoErrorf(t, err, "did not expect newReader to fail, but got: %v", err)

	// assert that the reader is actually destroyable, when the gc eventually reclaims it
	// (terminates prefetching routines)
	defer r.(*chunkReader).destroy()

	rdr, ok := r.(io.ReaderAt)
	require.True(t, ok)

	expected := readTextFile(t, tf.Original)

	// single ReadAt
	b := make([]byte, size)
	n, err := rdr.ReadAt(b, offset)
	require.NoErrorf(t, filterEOF(err), "did not expect error but got: %v", err)
	assertReadAtContent(t, size, tf.Original, expected, b, n, offset)

	// parallel ReadAt on same reader, with different offsets
	type result struct {
		err    error
		buffer []byte
		count  int
		offset int64
	}
	resC := make(chan result, 10) // do not block on results
	go func(resC chan<- result) {
		var wg sync.WaitGroup
		for _, off := range []int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9} {
			wg.Add(1)
			go func(offset int64, res chan<- result, wg *sync.WaitGroup) {
				defer wg.Done()
				buffer := make([]byte, size)
				n, err := rdr.ReadAt(buffer, offset)
				res <- result{
					err:    err,
					buffer: buffer,
					count:  n,
					offset: offset,
				}
			}(off, resC, &wg)
		}
		wg.Wait()
		close(resC)
	}(resC)

	// collect results and assert read data again reference
	for res := range resC {
		if !assert.NoErrorf(t, filterEOF(res.err),
			"ReadAt on %s (offset: %d): did not expect error but got: %v", tf.Original, res.offset, res.err) {
			continue
		}
		assertReadAtContent(t, size, tf.Original, expected, res.buffer, res.count, res.offset)
	}
}

func filterEOF(err error) error {
	if err == nil || strings.Contains(err.Error(), "EOF") {
		return nil
	}
	return err
}

func assertReadAtContent(t testing.TB, size int, name string, expected, received []byte, n int, offset int64) {
	var count int
	// truncate tested data to the first 2*leafSize bytes
	if len(expected) > size {
		count = size
	} else {
		count = len(expected) - int(offset)
	}
	if !assert.Equalf(t, count, n, "expected to ReadAt %d bytes, got: %d", count, n) {
		return
	}
	expectedBytes := expected[int(offset) : int(offset)+count]
	if !assert.EqualValuesf(t, expectedBytes, received[:n],
		"expected ReadAt to match expectation on %s, with offset %d", name, offset) {
		if os.Getenv("DEBUG_TEST") == "" {
			return
		}
		// dump buffers for debug
		var diff int
		for i := range expectedBytes {
			if expected[i] != received[i] {
				diff = i
				break
			}
		}
		_ = ioutil.WriteFile("test-dump.out", []byte(fmt.Sprintf(`file: %s
			size=%d
			offset=%d
			received bytes=%d
			differ at: %d
			full: %s
			expected=%s
			actual=%s`, name, len(expected), offset, n, diff,
			expected,
			spew.Sdump(expectedBytes), spew.Sdump(received[:n]))), 0600)
	}
}

func testLru(t *testing.T, size int, evict func(interface{}, interface{})) (*lru.Cache, *sync.Mutex) {
	var (
		lr  *lru.Cache
		err error
	)
	if evict != nil {
		lr, err = lru.NewWithEvict(size, evict)
	} else {
		lr, err = lru.New(size)
	}
	require.NoError(t, err)
	return lr, &sync.Mutex{}
}

func TestChunkReader_All(t *testing.T) {
	var l *zap.Logger
	if os.Getenv("DEBUG_TEST") != "" {
		l = dlogger.MustGetLogger("debug")
	} else {
		l = dlogger.MustGetLogger("info")
	}
	sl := newLeafFreelist(leafSize, 50)
	fl := newLeafFreelist(MaxLeafSize, 50)

	// a set of options to exercise chunkReaders under various conditions
	optSets := [][]ReaderOption{
		{
			ReaderLogger(l),
			SetCache(testLru(t, DefaultCacheSize, nil)),
			ReaderPrefix(""),
			ReaderPrefetch(0),
		},
		{
			ReaderLogger(l),
			ReaderPrefetch(1),
			ReaderVerifyHash(true),
		},
		{
			ReaderLogger(l),
			SetCache(testLru(t, DefaultCacheSize, nil)),
			SetLeafPool(fl),
			ReaderPrefetch(0),
		},
		{
			ReaderLogger(l),
			SetCache(testLru(t, 20, func(_ interface{}, lruVal interface{}) {
				fl.Release(lruVal.(LeafBuffer))
			})),
			SetLeafPool(fl),
			ReaderPrefetch(3),
		},
		{
			ReaderLogger(l),
			SetCache(testLru(t, 20, func(_ interface{}, lruVal interface{}) {
				sl.Release(lruVal.(LeafBuffer))
			})),
			SetLeafPool(sl),
			ReaderPrefetch(1),
		},
		{
			ReaderLogger(l),
			SetCache(testLru(t, 50, nil)),
			SetLeafPool(sl),
			ReaderPrefetch(4),
		},
		{
			ReaderLogger(l),
			SetCache(testLru(t, 2, func(_ interface{}, lruVal interface{}) {
				fl.Release(lruVal.(LeafBuffer))
			})),
			SetLeafPool(fl),
			ReaderPrefetch(1),
			ReaderVerifyHash(true),
		},
	}

	blobs := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(destDir, "cafs")))
	for _, toPin := range testFiles(destDir) {
		tf := toPin
		for i, optsToPin := range optSets {
			opts := optsToPin
			t.Run(tf.Original+"-"+strconv.Itoa(i), func(t *testing.T) {
				t.Parallel()
				verifyChunkReader(t, blobs, tf)
				verifyChunkReaderAt(t, blobs, tf, opts...)
			})
		}
	}
}

type fakeStore struct {
	chunks map[string][]byte
	storage.Store
}

func (f *fakeStore) Get(ctx context.Context, name string) (io.ReadCloser, error) {
	return &fakeReader{
		data: f.chunks[name],
	}, nil
}

type fakeReader struct {
	data      []byte
	readSoFar int
}

func (r *fakeReader) Read(data []byte) (int, error) {
	if len(r.data) <= r.readSoFar {
		return 0, io.EOF
	}
	read := copy(data, r.data[r.readSoFar:])
	r.readSoFar += read
	return read, nil
}

func (r *fakeReader) Close() error {
	return nil
}

type fakeWriter struct {
	data    []byte
	written int
}

func (w *fakeWriter) Write(p []byte) (int, error) {
	written := copy(w.data[w.written:], p)
	w.written += written
	return written, nil
}

type fakeWriteAt struct {
	data    []byte
	written int
	mux     sync.Mutex
}

func (w *fakeWriteAt) Write(p []byte) (int, error) {
	written := copy(w.data, p)
	w.written += written
	return written, nil
}

func (w *fakeWriteAt) WriteAt(p []byte, off int64) (int, error) {
	w.mux.Lock()
	written := copy(w.data[off:], p)
	w.mux.Unlock()
	return written, nil
}

func TestWriteTo(t *testing.T) {
	k := strings.Repeat("0", 126)
	keyStr1 := k + "01"
	keyStr2 := k + "02"
	rKeyStr := k + "12"
	// Pass writer without write at. make sure data read from reader is written to writer
	testFakeStore := fakeStore{
		chunks: make(map[string][]byte, 2),
	}
	const chunkSize = 64 * 1024

	testFakeStore.chunks[keyStr1] = rand.Bytes(chunkSize)
	testFakeStore.chunks[keyStr2] = rand.Bytes(chunkSize)
	key, err := KeyFromString(rKeyStr)
	require.NoError(t, err)
	key1, err := KeyFromString(keyStr1)
	require.NoError(t, err)
	key2, err := KeyFromString(keyStr2)
	require.NoError(t, err)
	keys := []Key{key1, key2}
	reader, err := newReader(&testFakeStore, key, chunkSize,
		ReaderPrefix(""),
		TruncateLeaf(false),
		Keys(keys),
	)
	require.NoError(t, err)
	rWriteTo, ok := reader.(io.WriterTo)
	require.True(t, ok)
	fakeWriterAt := &fakeWriteAt{
		data: make([]byte, 2*chunkSize),
	}
	written, err := rWriteTo.WriteTo(fakeWriterAt)
	require.NoError(t, err)
	require.Equal(t, written, int64(2*chunkSize))
	require.Equal(t, testFakeStore.chunks[keyStr1], fakeWriterAt.data[:chunkSize])
	require.Equal(t, testFakeStore.chunks[keyStr2], fakeWriterAt.data[chunkSize:])
	// Pass writer with write at. make sure data read from reader is written to writerAt
	fakeWriter := &fakeWriter{
		data: make([]byte, 2*chunkSize),
	}
	written, err = rWriteTo.WriteTo(fakeWriter)
	require.NoError(t, err)
	require.Equal(t, written, int64(2*chunkSize))
	require.Equal(t, testFakeStore.chunks[keyStr1], fakeWriter.data[:chunkSize])
	require.Equal(t, testFakeStore.chunks[keyStr2], fakeWriter.data[chunkSize:])
	// TODO: Set truncation on and verify.
}

type oiCalculatorTests struct {
	inOffset  int64
	leafSize  uint32
	outIndex  int64
	outOffset int64
}

func TestOffsetIndexCalculator(t *testing.T) {
	tests := []oiCalculatorTests{
		{
			inOffset:  0,
			leafSize:  1,
			outIndex:  0,
			outOffset: 0,
		},
		{
			inOffset:  1,
			leafSize:  1,
			outIndex:  1,
			outOffset: 0,
		},
		{
			inOffset:  2,
			leafSize:  1,
			outIndex:  2,
			outOffset: 0,
		},
		{
			inOffset:  0,
			leafSize:  leafSize,
			outIndex:  0,
			outOffset: 0,
		},
		{
			inOffset:  int64(leafSize) - 1,
			leafSize:  leafSize,
			outIndex:  0,
			outOffset: int64(leafSize) - 1,
		},
		{
			inOffset:  int64(leafSize),
			leafSize:  leafSize,
			outIndex:  1,
			outOffset: 0,
		},
		{
			inOffset:  int64(leafSize) * 10,
			leafSize:  leafSize,
			outIndex:  10, // 0-10 => 11th leaf
			outOffset: 0,
		},
		{
			inOffset:  int64(leafSize)*10 - 1,
			leafSize:  leafSize,
			outIndex:  9,                   // 0-9 => 10th leaf
			outOffset: int64(leafSize) - 1, // last byte
		},
	}
	for i, test := range tests {
		index, offset := calculateKeyAndOffset(test.inOffset, test.leafSize)
		require.Equal(t, test.outIndex, int64(index), "Test number: "+strconv.Itoa(i))
		require.Equal(t, test.outOffset, offset, "Test number "+strconv.Itoa(i))
	}
}
