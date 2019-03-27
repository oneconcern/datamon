package cafs

import (
	"context"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/oneconcern/datamon/internal"

	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func keyFromFile(t testing.TB, pth string) Key {
	rhash := readTextFile(t, pth)
	rkey, err := KeyFromString(rhash)
	require.NoError(t, err)
	return rkey
}

func readTextFile(t testing.TB, pth string) string {
	v, err := ioutil.ReadFile(pth)
	if err != nil {
		require.NoError(t, err)
	}
	return string(v)
}

func TestChunkReader_SmallOnly(t *testing.T) {
	blobs := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(destDir, "cafs")))
	for _, tf := range testFiles(destDir) {
		if tf.Parts > 1 {
			continue
		}
		verifyChunkReader(t, blobs, tf)
	}
}

func verifyChunkReader(t testing.TB, blobs storage.Store, tf testFile) {
	rkey := keyFromFile(t, tf.RootHash)

	rdr, err := newReader(blobs, rkey, leafSize, "")
	require.NoError(t, err)
	defer rdr.Close()

	b, err := ioutil.ReadAll(rdr)
	require.NoError(t, err)

	expected := readTextFile(t, tf.Original)
	actual := string(b)
	require.Equal(t, len(expected), len(actual))
	require.Equal(t, expected, actual)
}

func TestChunkReader_All(t *testing.T) {
	blobs := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(destDir, "cafs")))
	for _, tf := range testFiles(destDir) {
		verifyChunkReader(t, blobs, tf)
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
	fakeStore := fakeStore{
		chunks: make(map[string][]byte, 2),
	}
	fakeStore.chunks[keyStr1] = internal.RandBytesMaskImprSrc(64 * 1024)
	fakeStore.chunks[keyStr2] = internal.RandBytesMaskImprSrc(64 * 1024)
	key, err := KeyFromString(rKeyStr)
	require.NoError(t, err)
	key1, err := KeyFromString(keyStr1)
	require.NoError(t, err)
	key2, err := KeyFromString(keyStr2)
	require.NoError(t, err)
	keys := []Key{key1, key2}
	reader, err := newReader(&fakeStore, key, 64*1024, "",
		TruncateLeaf(false),
		Keys(keys),
	)
	require.NoError(t, err)
	rWriteTo, ok := reader.(io.WriterTo)
	require.True(t, ok)
	fakeWriterAt := &fakeWriteAt{
		data: make([]byte, 2*64*1024),
	}
	written, err := rWriteTo.WriteTo(fakeWriterAt)
	require.NoError(t, err)
	require.Equal(t, written, int64(2*64*1024))
	require.Equal(t, fakeStore.chunks[keyStr1], fakeWriterAt.data[:64*1024])
	require.Equal(t, fakeStore.chunks[keyStr2], fakeWriterAt.data[64*1024:])
	// Pass writer with write at. make sure data read from reader is written to writerAt
	fakeWriter := &fakeWriter{
		data: make([]byte, 2*64*1024),
	}
	written, err = rWriteTo.WriteTo(fakeWriter)
	require.NoError(t, err)
	require.Equal(t, written, int64(2*64*1024))
	require.Equal(t, fakeStore.chunks[keyStr1], fakeWriter.data[:64*1024])
	require.Equal(t, fakeStore.chunks[keyStr2], fakeWriter.data[64*1024:])
	// Set truncation on and verify.
}
