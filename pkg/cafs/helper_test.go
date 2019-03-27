package cafs

import (
	"context"
	"encoding/hex"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/oneconcern/datamon/pkg/storage"

	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func mustBytes(res []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return res
}

const (
	destDir = "../../testdata"

	leafSize uint32 = 1.5 * 1024 * 1024
)

type testFile struct {
	Original string
	RootHash string
	Parts    int
	Size     int
}

func testFiles(destDir string) []testFile {
	return []testFile{
		{
			Original: filepath.Join(destDir, "original", "small"),
			RootHash: filepath.Join(destDir, "roots", "small"),
			Parts:    1,
			Size:     int(leafSize - 512),
		},
		{
			Original: filepath.Join(destDir, "original", "twoparts"),
			RootHash: filepath.Join(destDir, "roots", "twoparts"),
			Parts:    2,
			Size:     int(leafSize + leafSize/2),
		},
		{
			Original: filepath.Join(destDir, "original", "fourparts"),
			RootHash: filepath.Join(destDir, "roots", "fourparts"),
			Parts:    4,
			Size:     int(3*leafSize + leafSize/2),
		},
		{
			Original: filepath.Join(destDir, "original", "tenparts"),
			RootHash: filepath.Join(destDir, "roots", "tenparts"),
			Parts:    10,
			Size:     int(10*leafSize - 512),
		},
		{
			Original: filepath.Join(destDir, "original", "exact10"),
			RootHash: filepath.Join(destDir, "roots", "exact10"),
			Parts:    10,
			Size:     int(10 * leafSize),
		},
		{
			Original: filepath.Join(destDir, "original", "under10"),
			RootHash: filepath.Join(destDir, "roots", "under10"),
			Parts:    10,
			Size:     int(10*leafSize - 1),
		},
		{
			Original: filepath.Join(destDir, "original", "over10"),
			RootHash: filepath.Join(destDir, "roots", "over10"),
			Parts:    11,
			Size:     int(10*leafSize + 1),
		},
		{
			Original: filepath.Join(destDir, "original", "onetwoeigth-not-tree-root"),
			RootHash: filepath.Join(destDir, "roots", "onetwoeigth-not-tree-root"),
			Parts:    1,
			Size:     128,
		},
		{
			Original: filepath.Join(destDir, "original", "fivetwelve-not-tree-root"),
			RootHash: filepath.Join(destDir, "roots", "fivetwelve-not-tree-root"),
			Parts:    1,
			Size:     512,
		},
	}
}

func TestMain(m *testing.M) {
	if _, _, _, err := setupTestData(destDir, testFiles(destDir)); err != nil {
		log.Fatalln(err)
	}
	os.Exit(m.Run())
}

func setupTestData(dir string, files []testFile) (*testDataGenerator, storage.Store, Fs, error) {
	os.RemoveAll(dir)
	blobs := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(dir, "cafs")))
	fs, err := New(
		LeafSize(leafSize),
		Backend(blobs),
		LeafTruncation(false),
	)
	if err != nil {
		return nil, nil, nil, err
	}

	g := &testDataGenerator{destDir: dir, leafSize: leafSize, fs: fs}
	if err = g.Initialize(); err != nil {
		return nil, nil, nil, err
	}
	return g, blobs, fs, g.Generate(files)
}

type testDataGenerator struct {
	destDir  string
	leafSize uint32
	fs       Fs
}

func (t *testDataGenerator) Initialize() error {
	if err := os.MkdirAll(filepath.Join(t.destDir, "original"), 0700); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(t.destDir, "cafs"), 0700); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(t.destDir, "roots"), 0700); err != nil {
		return err
	}
	return nil
}

func (t *testDataGenerator) Generate(files []testFile) error {
	for _, ps := range files {
		if err := GenerateFile(ps.Original, ps.Size, t.leafSize); err != nil {
			return err
		}

		if err := GenerateCAFSFile(ps.Original, t.fs, t.destDir); err != nil {
			return err
		}
	}
	return nil
}

func TestLeafHashes_Single(t *testing.T) {
	blobs := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(destDir, "cafs")))
	for _, tf := range testFiles(destDir) {
		if tf.Parts > 1 {
			continue
		}
		rhash, err := ioutil.ReadFile(tf.RootHash)
		require.NoError(t, err)
		rb, err := hex.DecodeString(string(rhash))
		require.NoError(t, err)

		orig, err := ioutil.ReadFile(tf.Original)
		require.NoError(t, err)

		rrdr, err := blobs.Get(context.Background(), string(rhash))
		require.NoError(t, err)

		cafsb, err := ioutil.ReadAll(rrdr)
		rrdr.Close()
		require.NoError(t, err)
		require.Equal(t, (tf.Parts+1)*KeySize, len(cafsb))
		require.Equal(t, rb, cafsb[tf.Parts*KeySize:])

		cafsbk, err := NewKey(cafsb[:KeySize])
		require.NoError(t, err)
		lrdr, err := blobs.Get(context.Background(), cafsbk.String())
		require.NoError(t, err)
		cafslb, err := ioutil.ReadAll(lrdr)
		lrdr.Close()
		require.NoError(t, err)
		require.Equal(t, orig, cafslb)

		rkey, err := KeyFromString(string(rhash))
		require.NoError(t, err)
		keys, err := LeafKeys(rkey, cafsb, leafSize)
		require.NoError(t, err)
		require.Len(t, keys, 1)

		_, err = LeafKeys(cafsbk, orig, leafSize)
		require.Error(t, err)
		require.EqualError(t, err, "the last hash in the file is not the checksum")
	}
}

func TestLeafHashes_Multi(t *testing.T) {
	blobs := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(destDir, "cafs")))
	for _, tf := range testFiles(destDir) {

		rhash := readTextFile(t, tf.RootHash)
		rrdr, err := blobs.Get(context.Background(), rhash)
		require.NoError(t, err)
		rkey, err := KeyFromString(rhash)
		require.NoError(t, err)

		b, err := ioutil.ReadAll(rrdr)
		require.NoError(t, err)
		rrdr.Close()

		keys, err := LeafKeys(rkey, b, leafSize)
		require.NoError(t, err)
		require.Len(t, keys, tf.Parts)
		for i, key := range keys {
			require.Equal(t, b[i*KeySize:i*KeySize+KeySize], key[:])

			has, err := blobs.Has(context.Background(), key.String())
			require.NoError(t, err)
			require.True(t, has)
		}
	}
}

func TestLeafHashes_Constants(t *testing.T) {
	const (
		leafHash = "907fd469f998570163a79d10cb30fb75e7733760e6d0f865f7e31db4f8e7cd6590fd7a3e70d522e310bf2476383face2f00a05bd0d5bedf1457cdfd0e28a04d6"
		rootHash = "878fd79c0e47f2c7d3c512eb352d41397bedf2677cd5a5e68d582f7864d2f5fe78216d7ce465880b629bd73ec9ccd999d3958d7f5955fbdcc8afc9a743eb0dd8"
	)

	var (
		leaf    = mustBytes(hex.DecodeString(leafHash))
		root    = mustBytes(hex.DecodeString(rootHash))
		wrong   = []byte("ffahncyloc4ws8a3fu6je52qoynhi9vselmyxnusrrg6m2yv2mg3np0puazh2xql")
		correct = mustBytes(hex.DecodeString(leafHash + rootHash))
	)

	_, err := RootHash([]Key{MustNewKey(leaf)}, leafSize)
	require.NoError(t, err)

	_, err = LeafKeys(MustNewKey(root), leaf, leafSize)
	require.Error(t, err)
	_, err = LeafKeys(MustNewKey(leaf), wrong, leafSize)
	require.Error(t, err)
	keys, err := LeafKeys(MustNewKey(root), correct, leafSize)
	require.NoError(t, err)
	require.Equal(t, MustNewKey(leaf), keys[0])
}
