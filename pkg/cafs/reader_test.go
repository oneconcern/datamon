package cafs

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/oneconcern/trumpet/pkg/blob"
	"github.com/oneconcern/trumpet/pkg/blob/localfs"
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

func verifyChunkReader(t testing.TB, blobs blob.Store, tf testFile) {
	rkey := keyFromFile(t, tf.RootHash)

	rdr, err := newReader(blobs, rkey, leafSize)
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
