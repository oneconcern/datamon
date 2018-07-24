package localfs

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/oneconcern/trumpet/pkg/blob"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHas(t *testing.T) {
	bs := setupStore(t)

	has, err := bs.Has("sixteentons")
	require.NoError(t, err)
	require.True(t, has)

	has, err = bs.Has("seventeentons")
	require.NoError(t, err)
	require.True(t, has)

	has, err = bs.Has("fifteentons")
	require.NoError(t, err)
	require.False(t, has)
}

func TestGet(t *testing.T) {
	bs := setupStore(t)

	rdr, err := bs.Get("sixteentons")
	require.NoError(t, err)
	b, err := ioutil.ReadAll(rdr)
	require.NoError(t, err)
	assert.Equal(t, "this is the text", string(b))

	rdr, err = bs.Get("seventeentons")
	require.NoError(t, err)
	b, err = ioutil.ReadAll(rdr)
	require.NoError(t, err)
	assert.Equal(t, "this is the text for another thing", string(b))
}

func TestKeys(t *testing.T) {
	bs := setupStore(t)

	keys, err := bs.Keys()
	require.NoError(t, err)
	require.Len(t, keys, 2)
}

func TestDelete(t *testing.T) {
	bs := setupStore(t)

	require.NoError(t, bs.Delete("seventeentons"))
	k, _ := bs.Keys()
	assert.Len(t, k, 1)
}

func TestClear(t *testing.T) {
	bs := setupStore(t)

	require.NoError(t, bs.Clear())
	k, _ := bs.Keys()
	require.Empty(t, k)
}

func TestPut(t *testing.T) {
	bs := setupStore(t)

	content := bytes.NewBufferString("here we go once again")
	err := bs.Put("eighteentons", content)
	require.NoError(t, err)

	rdr, err := bs.Get("eighteentons")
	require.NoError(t, err)
	b, err := ioutil.ReadAll(rdr)
	require.NoError(t, err)
	require.NoError(t, rdr.Close())

	assert.Equal(t, "here we go once again", string(b))

	k, _ := bs.Keys()
	assert.Len(t, k, 3)
}

func setupStore(t testing.TB) blob.Store {
	t.Helper()

	fs := afero.NewMemMapFs()
	f, err := fs.Create("si/xt/eentons")
	require.NoError(t, err)
	_, err = f.WriteString("this is the text")
	require.NoError(t, err)
	f.Close()

	ff, err := fs.Create("se/ve/nteentons")
	require.NoError(t, err)
	_, err = ff.WriteString("this is the text for another thing")
	require.NoError(t, err)
	ff.Close()

	return New(fs)
}
