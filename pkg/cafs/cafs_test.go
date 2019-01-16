package cafs

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func assertReaderOriginal(t testing.TB, original string, rdr io.ReadCloser) {
	defer rdr.Close()

	b, err := ioutil.ReadAll(rdr)
	require.NoError(t, err)
	require.NoError(t, rdr.Close())

	expected := readTextFile(t, original)
	actual := string(b)
	require.Equal(t, len(expected), len(actual))
	require.Equal(t, expected, actual)
}

func TestCAFS_Get(t *testing.T) {
	blobs := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(destDir, "cafs")))
	fs, err := New(
		LeafSize(leafSize),
		Backend(blobs),
	)
	require.NoError(t, err)

	for _, tf := range testFiles(destDir) {
		rhash := readTextFile(t, tf.RootHash)
		rkey, err := KeyFromString(rhash)
		require.NoError(t, err)

		rdr, err := fs.Get(context.Background(), rkey)
		require.NoError(t, err)
		assertReaderOriginal(t, tf.Original, rdr)
	}
}

func TestCAFS_Put(t *testing.T) {
	td, err := ioutil.TempDir("", "tpt-cafs-put")
	require.NoError(t, err)
	defer os.RemoveAll(td)

	files := testFiles(td)
	g, _, fs, err := setupTestData(td, files)
	require.NoError(t, err)

	orig := filepath.Join(td, "original", "test-cas-put")
	require.NoError(t, GenerateFile(orig, 512*1024, g.leafSize))

	f, err := os.Open(orig)
	require.NoError(t, err)
	defer f.Close()

	written, rk, _, err := fs.Put(context.Background(), f)
	require.NoError(t, err)
	fileInfo, err := f.Stat()
	require.NoError(t, err)
	require.Equal(t, fileInfo.Size(), written)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	rdr, err := fs.Get(context.Background(), rk)
	require.NoError(t, err)

	assertReaderOriginal(t, orig, rdr)
}

func TestCAFS_Delete(t *testing.T) {
	td, err := ioutil.TempDir("", "tpt-cafs-delete")
	require.NoError(t, err)
	defer os.RemoveAll(td)

	files := testFiles(td)
	_, blobs, fs, err := setupTestData(td, files)
	require.NoError(t, err)

	// tenparts file
	tf := files[3]
	rkey := keyFromFile(t, tf.RootHash)

	keys, err := LeafsForHash(blobs, rkey, leafSize, "")
	require.NoError(t, err)

	for _, k := range keys {
		has, err := blobs.Has(context.Background(), k.String())
		require.NoError(t, err)
		require.True(t, has)
	}

	require.NoError(t, fs.Delete(context.Background(), rkey))

	for _, k := range keys {
		has, err := blobs.Has(context.Background(), k.String())
		require.NoError(t, err)
		require.False(t, has)
	}
}

func TestCAFS_Has_AllPresent(t *testing.T) {
	blobs := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(destDir, "cafs")))
	fs, err := New(
		LeafSize(leafSize),
		Backend(blobs),
	)
	require.NoError(t, err)

	files := testFiles(destDir)

	rkeys := make(map[Key]struct{})
	lkeys := make(map[Key]struct{})
	allKeys := make(map[Key]struct{})

	for _, tf := range files {

		k := keyFromFile(t, tf.RootHash)
		rkeys[k] = struct{}{}
		allKeys[k] = struct{}{}

		keys, err := LeafsForHash(blobs, k, leafSize, "")
		require.NoError(t, err)
		for _, kk := range keys {
			allKeys[kk] = struct{}{}
			lkeys[kk] = struct{}{}
		}
	}

	// test for root keys
	for rkey := range rkeys {
		has, missing, err := fs.Has(context.Background(), rkey)
		require.NoError(t, err)
		require.Nil(t, missing)
		require.True(t, has)
	}

	// test for leaf key
	for lkey := range lkeys {
		has, missing, err := fs.Has(context.Background(), lkey)
		require.NoError(t, err)
		require.Nil(t, missing)
		require.True(t, has)
	}

	// test for root keys, and only root keys
	for rkey := range rkeys {
		has, missing, err := fs.Has(context.Background(), rkey, HasOnlyRoots())
		require.NoError(t, err)
		require.Nil(t, missing)
		require.True(t, has)
	}

	// test for lkeys with only roots, should all be false
	for lkey := range lkeys {
		has, missing, err := fs.Has(context.Background(), lkey, HasOnlyRoots())
		require.NoError(t, err)
		require.Nil(t, missing)
		require.False(t, has)
	}

	// test for root keys, and only root keys
	for rkey := range rkeys {
		has, missing, err := fs.Has(context.Background(), rkey, HasGatherIncomplete())
		require.NoError(t, err)
		require.Nil(t, missing)
		require.True(t, has)
	}

	// test for lkeys with only roots, should all be false
	for lkey := range lkeys {
		has, missing, err := fs.Has(context.Background(), lkey, HasGatherIncomplete())
		require.NoError(t, err)
		require.Nil(t, missing)
		require.False(t, has)
	}

}

func TestCAFS_Has_SomeMissing(t *testing.T) {
	td, err := ioutil.TempDir("", "tpt-cafs-delete")
	require.NoError(t, err)
	defer os.RemoveAll(td)

	files := testFiles(td)
	_, blobs, fs, err := setupTestData(td, files)
	require.NoError(t, err)

	rkeys := make(map[Key]struct{})
	lkeys := make(map[Key]struct{})
	missing := make(map[Key]map[Key]struct{})
	for _, tf := range files {
		k := keyFromFile(t, tf.RootHash)
		rkeys[k] = struct{}{}
		if _, ok := missing[k]; !ok {
			missing[k] = make(map[Key]struct{})
		}

		keys, err := LeafsForHash(blobs, k, leafSize, "")
		require.NoError(t, err)
		for i, kk := range keys {
			lkeys[kk] = struct{}{}
			if i%2 != 0 {
				err := blobs.Delete(context.Background(), kk.String())
				require.NoError(t, err)
				missing[k][kk] = struct{}{}
			}
		}
	}

	// test for root keys, and only root keys
	for rkey := range rkeys {
		has, missing, err := fs.Has(context.Background(), rkey, HasOnlyRoots())
		require.NoError(t, err)
		require.Nil(t, missing)
		require.True(t, has)
	}

	// test for lkeys with only roots, should all be false
	for lkey := range lkeys {
		has, missing, err := fs.Has(context.Background(), lkey, HasOnlyRoots())
		require.NoError(t, err)
		require.Nil(t, missing)
		require.False(t, has)
	}

	// test for root keys, and only root keys
	for rkey := range rkeys {
		mks := missing[rkey]

		has, missing, err := fs.Has(context.Background(), rkey, HasGatherIncomplete())
		require.NoError(t, err)
		require.True(t, has)

		require.Equal(t, len(mks), len(missing))
		for _, m := range missing {
			delete(mks, m)
		}
		require.Empty(t, mks)
	}

	// test for lkeys with only roots, should all be false
	for lkey := range lkeys {
		has, missing, err := fs.Has(context.Background(), lkey, HasGatherIncomplete())
		require.NoError(t, err)
		require.Nil(t, missing)
		require.False(t, has)
	}
}

func TestCAFS_Keys(t *testing.T) {
	blobs := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(destDir, "cafs")))
	fs, err := New(
		LeafSize(leafSize),
		Backend(blobs),
	)
	require.NoError(t, err)

	files := testFiles(destDir)
	allKeys := make(map[Key]struct{})
	for _, tf := range files {
		k := keyFromFile(t, tf.RootHash)
		allKeys[k] = struct{}{}
		keys, e2 := LeafsForHash(blobs, k, leafSize, "")
		require.NoError(t, e2)
		for _, kk := range keys {
			allKeys[kk] = struct{}{}
		}
	}

	keys, err := fs.Keys(context.Background())
	require.NoError(t, err)
	require.Equal(t, len(allKeys), len(keys))

	for _, key := range keys {
		delete(allKeys, key)
	}

	require.Empty(t, allKeys)
}

func TestCAFS_RootKeys(t *testing.T) {
	blobs := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(destDir, "cafs")))
	fs, err := New(
		LeafSize(leafSize),
		Backend(blobs),
	)
	require.NoError(t, err)

	files := testFiles(destDir)
	allKeys := make(map[Key]struct{})
	for _, tf := range files {
		k := keyFromFile(t, tf.RootHash)
		allKeys[k] = struct{}{}
	}

	keys, err := fs.RootKeys(context.Background())
	require.NoError(t, err)
	require.Equal(t, len(allKeys), len(keys))

	for _, key := range keys {
		delete(allKeys, key)
	}

	require.Empty(t, allKeys)
}
func TestCAFS_Clear(t *testing.T) {
	td, err := ioutil.TempDir("", "tpt-cafs-clear")
	require.NoError(t, err)
	defer os.RemoveAll(td)

	files := testFiles(td)
	_, blobs, fs, err := setupTestData(td, files)
	require.NoError(t, err)

	keys, err := fs.Keys(context.Background())
	require.NoError(t, err)

	for _, k := range keys {
		has, err := blobs.Has(context.Background(), k.String())
		require.NoError(t, err)
		require.True(t, has)
	}

	require.NoError(t, fs.Clear(context.Background()))

	for _, k := range keys {
		has, err := blobs.Has(context.Background(), k.String())
		require.NoError(t, err)
		require.False(t, has)
	}
}
