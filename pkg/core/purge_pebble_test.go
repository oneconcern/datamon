package core

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPebbleMerger(t *testing.T) {
	kvStore, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(kvStore)
	}()

	db, err := makeKVPebble(kvStore, nil)
	require.NoError(t, err)

	require.NoError(t, db.Set([]byte("abc"), []byte("X")))
	require.NoError(t, db.SetIfNotExists([]byte("abc"), []byte("Y")))
	require.NoError(t, db.SetIfNotExists([]byte("abc"), []byte("Z")))

	val, err := db.Get([]byte("abc"))
	require.NoError(t, err)
	require.Equal(t, []byte("X"), val)

	require.NoError(t, db.SetIfNotExists([]byte("xyz"), []byte("Y")))
	val, err = db.Get([]byte("xyz"))
	require.NoError(t, err)
	require.Equal(t, []byte("Y"), val)
}
