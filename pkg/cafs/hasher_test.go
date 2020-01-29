package cafs

import (
	"bytes"
	"context"
	"encoding/hex"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeafHashes_Single(t *testing.T) {
	blobs := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(destDir, "cafs")))
	for _, toPin := range testFiles(destDir) {
		tf := toPin
		if tf.Parts > 1 {
			continue
		}
		t.Run(tf.Original+"-hashes-single", func(t *testing.T) {
			t.Parallel()

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
			if len(cafsbk) < KeySize || len(orig) < KeySize {
				require.EqualError(t, err, "provided data is too short to contain a key")
			} else {
				require.EqualError(t, err, "the last hash in the file is not the root key")
			}
		})
	}
}

func TestLeafHashes_Multi(t *testing.T) {
	blobs := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(destDir, "cafs")))
	for _, toPin := range testFiles(destDir) {
		tf := toPin
		t.Run(tf.Original+"-hashes-multi", func(t *testing.T) {
			t.Parallel()

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
		})
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

func TestLeafHashes_EdgeCases(t *testing.T) {
	_, err := KeyFromBytes([]byte{}, leafSize, 1, false)
	assert.NoError(t, err)

	_, err = KeyFromBytes(nil, leafSize, 1, false)
	assert.NoError(t, err)

	key0 := bytes.Repeat([]byte("a"), KeySize-1)
	key1 := bytes.Repeat([]byte("a"), KeySize)
	key2 := bytes.Repeat([]byte("b"), KeySize)

	_, err = verifiedKeys(key0, leafSize)
	assert.Error(t, err)

	toVerify := key1
	_, err = verifiedKeys(toVerify, leafSize)
	assert.Error(t, err)

	toVerify = append(toVerify, key2...)
	_, err = verifiedKeys(toVerify, leafSize)
	assert.Error(t, err)

	root, err := rootHash([]Key{MustNewKey(key1), MustNewKey(key2)}, leafSize)
	require.NoError(t, err)

	toVerify = append(toVerify, root[:]...)
	_, err = verifiedKeys(toVerify, leafSize)
	assert.NoError(t, err)

	res := UnverifiedLeafKeys(toVerify, leafSize)
	assert.Len(t, res, 3)

	_ = copy(toVerify, key1)
	toVerify = append(toVerify, key2...)
	toVerify = append(toVerify, root[:len(root)-4]...)
	_, err = verifiedKeys(toVerify, leafSize)
	assert.Error(t, err)

	_ = copy(toVerify, key1)
	toVerify = append(toVerify, key0...)
	_, err = verifiedKeys(toVerify, leafSize)
	assert.Error(t, err)

	key0 = []byte("b")
	key1 = bytes.Repeat([]byte("a"), KeySize)
	toVerify = bytes.Join([][]byte{key1, key0}, []byte("x"))

	_, err = verifiedKeys(toVerify, leafSize)
	assert.Error(t, err)

	_, err = leaves(toVerify, leafSize)
	assert.Error(t, err)

	assert.Panics(t, func() {
		_ = UnverifiedLeafKeys(toVerify, leafSize)
	})
}
