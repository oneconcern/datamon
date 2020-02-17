// Copyright © 2018 One Concern

package localfs

import (
	"bytes"
	"context"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHas(t *testing.T) {
	bs, cleanup := setupStore(t)
	defer cleanup()

	has, err := bs.Has(context.Background(), "sixteentons")
	require.NoError(t, err)
	require.True(t, has)

	has, err = bs.Has(context.Background(), "seventeentons")
	require.NoError(t, err)
	require.True(t, has)

	has, err = bs.Has(context.Background(), "fifteentons")
	require.NoError(t, err)
	require.False(t, has)
}

func TestGet(t *testing.T) {
	bs, cleanup := setupStore(t)
	defer cleanup()

	rdr, err := bs.Get(context.Background(), "sixteentons")
	require.NoError(t, err)
	b, err := ioutil.ReadAll(rdr)
	require.NoError(t, err)
	assert.Equal(t, "this is the text", string(b))

	rdr, err = bs.Get(context.Background(), "seventeentons")
	require.NoError(t, err)
	b, err = ioutil.ReadAll(rdr)
	require.NoError(t, err)
	assert.Equal(t, "this is the text for another thing", string(b))
}

func TestKeys(t *testing.T) {
	bs, cleanup := setupStore(t)
	defer cleanup()

	keys, err := bs.Keys(context.Background())
	require.NoError(t, err)
	require.Len(t, keys, 2)
}

func TestDelete(t *testing.T) {
	bs, cleanup := setupStore(t)
	defer cleanup()

	require.NoError(t, bs.Delete(context.Background(), "seventeentons"))
	k, _ := bs.Keys(context.Background())
	assert.Len(t, k, 1)
}

func TestClear(t *testing.T) {
	bs, cleanup := setupStore(t)
	defer cleanup()

	require.NoError(t, bs.Clear(context.Background()))
	k, _ := bs.Keys(context.Background())
	require.Empty(t, k)
}

func TestPut(t *testing.T) {
	bs, cleanup := setupStore(t)
	defer cleanup()

	content := bytes.NewBufferString("here we go once again")
	err := bs.Put(context.Background(), "eighteentons", content, storage.NoOverWrite)
	require.NoError(t, err)

	rdr, err := bs.Get(context.Background(), "eighteentons")
	require.NoError(t, err)
	b, err := ioutil.ReadAll(rdr)
	require.NoError(t, err)
	require.NoError(t, rdr.Close())

	assert.Equal(t, "here we go once again", string(b))

	k, _ := bs.Keys(context.Background())
	assert.Len(t, k, 3)
}

func setupStore(t testing.TB) (storage.Store, func()) {
	t.Helper()

	fs := afero.NewMemMapFs()
	f, err := fs.Create("sixteentons")
	require.NoError(t, err)
	_, err = f.WriteString("this is the text")
	require.NoError(t, err)
	f.Close()

	ff, err := fs.Create("seventeentons")
	require.NoError(t, err)
	_, err = ff.WriteString("this is the text for another thing")
	require.NoError(t, err)
	ff.Close()

	return New(fs), func() {}
}

func fakeFile(t testing.TB, fs afero.Fs, file string) {
	f, err := fs.Create(file)
	require.NoError(t, err)
	_, err = f.WriteString("this is the text")
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)
}

func TestKeysPrefix(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := fs.MkdirAll("/a/b/c", 0777)
	require.NoError(t, err)
	err = fs.MkdirAll("/a/d", 0777)
	require.NoError(t, err)
	for i := 0; i < 10; i++ {
		fakeFile(t, fs, "/a/b/c/e"+strconv.Itoa(i))
		fakeFile(t, fs, "/a/d/f"+strconv.Itoa(i))
	}

	store := New(fs)

	var (
		keys []string
		next string
	)

	i := 0
	search := "/a"
	for keys, next, err = store.KeysPrefix(context.Background(), "", search, "", 3); next != ""; keys, next, err = store.KeysPrefix(context.Background(), next, search, "", 3) {
		require.NoError(t, err)
		assert.Len(t, keys, 3)
		i++
	}
	require.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Equal(t, i, 6)

	i = 0
	search = "/a/d/f"
	for keys, next, err = store.KeysPrefix(context.Background(), "", search, "", 4); next != ""; keys, next, err = store.KeysPrefix(context.Background(), next, search, "", 4) {
		require.NoError(t, err)
		assert.Len(t, keys, 4)
		i++
	}
	require.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Equal(t, i, 2)

	i = 0
	search = "a"
	for keys, next, err = store.KeysPrefix(context.Background(), "", search, "", 3); next != ""; keys, next, err = store.KeysPrefix(context.Background(), next, search, "", 3) {
		require.NoError(t, err)
		assert.Len(t, keys, 3)
		i++
	}
	require.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Equal(t, i, 6)

	i = 0
	search = "a/d"
	for keys, next, err = store.KeysPrefix(context.Background(), "", search, "", 100); next != ""; keys, next, err = store.KeysPrefix(context.Background(), next, search, "", 100) {
		i++
		t.Fail()
	}
	require.NoError(t, err)
	assert.Len(t, keys, 10)
	assert.Equal(t, i, 0)
}

func TestKeysPrefixWithDelimiter(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := fs.MkdirAll("/a/b-1/c", 0777)
	require.NoError(t, err)
	err = fs.MkdirAll("/a/d", 0777)
	require.NoError(t, err)
	for i := 0; i < 10; i++ {
		fakeFile(t, fs, "/a/b-1/c/e-"+strconv.Itoa(i)+"@z")
		fakeFile(t, fs, "/a/d/f-"+strconv.Itoa(i)+"@z")
		fakeFile(t, fs, "/a/b/g@"+strconv.Itoa(i))
	}

	store := New(fs)

	var (
		keys []string
		next string
	)

	// with unsuccessful search
	search := "/z"
	keys, next, err = store.KeysPrefix(context.Background(), "", search, "", 5)
	assert.Len(t, keys, 0)
	assert.Empty(t, next)
	assert.NoError(t, err)

	// with invalid next
	search = "/a"
	keys, next, err = store.KeysPrefix(context.Background(), "", search, "", 5)
	require.NotEmpty(t, next)
	assert.NoError(t, err)
	keys, next, err = store.KeysPrefix(context.Background(), "nowhere", search, "", 5)
	assert.Len(t, keys, 0)
	assert.Empty(t, next)
	assert.NoError(t, err)

	// with delimiters and deduplication
	i := 0
	search = "/a"
	for keys, next, err = store.KeysPrefix(context.Background(), "", search, "-", 5); next != ""; keys, next, err = store.KeysPrefix(context.Background(), next, search, "-", 5) {
		require.NoError(t, err)
		assert.Lenf(t, keys, 5, "got keys %v", keys)
		i++
	}
	require.NoError(t, err)
	assert.Lenf(t, keys, 2, "got keys %v", keys)
	assert.Equal(t, i, 2)

	i = 0
	search = "/a/b"
	for keys, next, err = store.KeysPrefix(context.Background(), "", search, "@", 5); next != ""; keys, next, err = store.KeysPrefix(context.Background(), next, search, "@", 5) {
		require.NoError(t, err)
		assert.Lenf(t, keys, 5, "got keys %v", keys)
		i++
	}
	require.NoError(t, err)
	assert.Equal(t, i, 2)

	i = 0
	search = "a/b"
	for keys, next, err = store.KeysPrefix(context.Background(), "", search, "@", 5); next != ""; keys, next, err = store.KeysPrefix(context.Background(), next, search, "@", 5) {
		require.NoError(t, err)
		assert.Lenf(t, keys, 5, "got keys %v", keys)
		for _, key := range keys {
			assert.False(t, strings.HasPrefix(key, "/"))
		}
		i++
	}
	require.NoError(t, err)
	assert.Lenf(t, keys, 1, "got keys %v", keys)
	assert.Equal(t, i, 2)
}
