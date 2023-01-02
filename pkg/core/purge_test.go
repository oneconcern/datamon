package core

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPurgeDBReader(t *testing.T) {
	kvStore, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(kvStore)
	}()

	db, erk := openKV(kvStore, defaultPurgeOptions(nil))
	require.NoError(t, erk)

	// prepare some keys in the store
	for _, key := range testKeys() {
		require.NoError(t,
			db.Set([]byte(key), []byte{}),
		)
	}

	ts := time.Now().UTC()
	r := newDBReader(context.Background(), db, ts, zap.NewNop(), 100)
	defer func() {
		_ = r.Close()
	}()

	t.Run("should read timestamp", func(t *testing.T) {
		b := make([]byte, 40)
		n, err := r.Read(b)
		require.NoError(t, err)
		require.Greater(t, n, 0)
		require.Equal(t, append([]byte(ts.Format(layout)), '\n'), b[:n])
	})

	t.Run("should read first key", func(t *testing.T) {
		b := make([]byte, 20)
		n, err := r.Read(b)
		require.NoError(t, err)
		require.Equal(t, 2, n)
		require.Equal(t, []byte{'A', '\n'}, b[:n])
	})

	t.Run("should read next key", func(t *testing.T) {
		b := make([]byte, 20)
		n, err := r.Read(b)
		require.NoError(t, err)
		require.Equal(t, 3, n)
		require.Equal(t, []byte{'B', 'C', '\n'}, b[:n])
	})

	t.Run("should split next key", func(t *testing.T) {
		b := make([]byte, 2)
		n, err := r.Read(b)
		require.NoError(t, err)
		require.Equal(t, 2, n)
		require.Equal(t, []byte{'D', 'E'}, b[:n])

		n, err = r.Read(b)
		require.NoError(t, err)
		require.Equal(t, 2, n)
		require.Equal(t, []byte{'F', '\n'}, b[:n])
	})

	t.Run("should split next key", func(t *testing.T) {
		b := make([]byte, 2)
		n, err := r.Read(b)
		require.NoError(t, err)
		require.Equal(t, 2, n)
		require.Equal(t, []byte{'G', 'H'}, b[:n])

		n, err = r.Read(b)
		require.NoError(t, err)
		require.Equal(t, 2, n)
		require.Equal(t, []byte{'I', 'J'}, b[:n])

		n, err = r.Read(b)
		require.NoError(t, err)
		require.Equal(t, 1, n)
		require.Equal(t, []byte{'\n'}, b[:n])
	})

	t.Run("should reach EOF", func(t *testing.T) {
		b := make([]byte, 2)
		n, err := r.Read(b)
		require.ErrorIs(t, err, io.EOF)
		require.Equal(t, 0, n)
	})

	t.Run("should track the count of keys read", func(t *testing.T) {
		require.Equal(t, uint64(4), r.Count())
	})
}

func testKeys() []string {
	return []string{
		"A",
		"BC",
		"DEF",
		"GHIJ",
	}
}
