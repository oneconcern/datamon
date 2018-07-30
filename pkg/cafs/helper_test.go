package cafs

import (
	"context"
	"encoding/hex"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/oneconcern/trumpet/pkg/blob"

	"github.com/oneconcern/trumpet"
	"github.com/oneconcern/trumpet/pkg/blob/localfs"
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

var testFiles = []testFile{
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
		Parts:    11,
		Size:     int(10*leafSize - 512),
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

func TestMain(m *testing.M) {
	if _, _, _, err := setupTestData(destDir, testFiles); err != nil {
		log.Fatalln(err)
	}
	if err := exec.Command("sync").Run(); err != nil {
		log.Fatalln(err)
	}
	os.Exit(m.Run())
}

func setupTestData(dir string, files []testFile) (string, blob.Store, Fs, error) {
	os.RemoveAll(dir)

	blobs := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(dir, "cafs")))
	fs, err := New(
		LeafSize(leafSize),
		Backend(blobs),
	)
	if err != nil {
		return "", nil, nil, err
	}

	g := &testDataGenerator{destDir: dir, leafSize: leafSize, fs: fs}
	if err = g.Initialize(); err != nil {
		return "", nil, nil, err
	}
	return destDir, blobs, fs, g.Generate(files)
}

type testDataGenerator struct {
	destDir  string
	leafSize uint32
	fs       Fs
}

func (t *testDataGenerator) Initialize() error {
	if err := os.MkdirAll(filepath.Join(destDir, "original"), 0700); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(destDir, "cafs"), 0700); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(destDir, "roots"), 0700); err != nil {
		return err
	}
	return nil
}

func (t *testDataGenerator) generateFile(tgt string, size int) error {
	f, err := os.Create(tgt)
	if err != nil {
		return err
	}
	defer f.Close()

	if size <= int(leafSize) { // small single chunk file
		_, err := f.WriteString(trumpet.RandStringBytesMaskImprSrc(size))
		if err != nil {
			return err
		}
		return f.Close()
	}

	var parts = size / int(t.leafSize)
	for i := 0; i < parts; i++ {
		_, err := f.WriteString(trumpet.RandStringBytesMaskImprSrc(int(t.leafSize)))
		if err != nil {
			return err
		}
	}
	remaining := size - (parts * int(t.leafSize))
	if remaining > 0 {
		_, err := f.WriteString(trumpet.RandStringBytesMaskImprSrc(remaining))
		if err != nil {
			return err
		}
	}
	return f.Close()
}

func (t *testDataGenerator) generateCAFSFile(src string) error {
	fsrc, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fsrc.Close()

	key, err := t.fs.Put(context.Background(), fsrc)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(t.destDir, "roots", filepath.Base(src)), []byte(key.String()), 0600)
}

func (t *testDataGenerator) Generate(files []testFile) error {
	for _, ps := range files {
		if err := t.generateFile(ps.Original, ps.Size); err != nil {
			return err
		}

		if err := t.generateCAFSFile(ps.Original); err != nil {
			return err
		}
	}
	return nil
}

func TestLeafHashes_Single(t *testing.T) {
	blobs := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(destDir, "cafs")))
	for _, tf := range testFiles {
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

		cafsbk := hex.EncodeToString(cafsb[:KeySize])
		lrdr, err := blobs.Get(context.Background(), cafsbk)
		require.NoError(t, err)
		cafslb, err := ioutil.ReadAll(lrdr)
		lrdr.Close()
		require.NoError(t, err)
		require.Equal(t, orig, cafslb)

		keys, err := LeafKeys(string(rhash), cafsb, leafSize)
		require.NoError(t, err)
		require.Len(t, keys, 1)

		_, err = LeafKeys(cafsbk, orig, leafSize)
		require.Error(t, err)
		require.EqualError(t, err, "the last hash in the file is not the checksum")
	}
}

func TestLeafHashes_Multi(t *testing.T) {
	blobs := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(destDir, "cafs")))
	for _, tf := range testFiles {

		rhash := readTextFile(t, tf.RootHash)
		rrdr, err := blobs.Get(context.Background(), rhash)
		require.NoError(t, err)

		b, err := ioutil.ReadAll(rrdr)
		require.NoError(t, err)
		rrdr.Close()

		keys, err := LeafKeys(rhash, b, leafSize)
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
		wrong   = []byte("ffahncyloc4ws8a3fu6je52qoynhi9vselmyxnusrrg6m2yv2mg3np0puazh2xql")
		correct = mustBytes(hex.DecodeString(leafHash + rootHash))
	)

	_, err := RootHash([]Key{MustNewKey(leaf)}, leafSize)
	require.NoError(t, err)

	_, err = LeafKeys(rootHash, leaf, leafSize)
	require.Error(t, err)
	_, err = LeafKeys(leafHash, wrong, leafSize)
	require.Error(t, err)
	keys, err := LeafKeys(rootHash, correct, leafSize)
	require.NoError(t, err)
	require.Equal(t, MustNewKey(leaf), keys[0])
}
