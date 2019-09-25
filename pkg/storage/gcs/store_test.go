package gcs

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"testing"

	gcsStorage "cloud.google.com/go/storage"
	"github.com/oneconcern/datamon/internal"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

const (
	longPath = "this/is/a/long/path/to/an/object/the/object/is/under/this/path/list/with/prefix/please/"
)

func gen(i int) string {
	return longPath + fmt.Sprint(i)
}

func setup(t testing.TB, numOfObjects int) (storage.Store, func()) {

	ctx := context.Background()

	bucket := "datamontest-" + internal.RandStringBytesMaskImprSrc(15)
	log.Printf("Created bucket %s ", bucket)

	client, err := gcsStorage.NewClient(context.TODO(), option.WithScopes(gcsStorage.ScopeFullControl))
	require.NoError(t, err)
	err = client.Bucket(bucket).Create(ctx, "onec-co", nil)
	require.NoError(t, err)

	gcs, err := New(context.TODO(), bucket, "") // Use GOOGLE_APPLICATION_CREDENTIALS env variable
	require.NoError(t, err)
	for i := 0; i < numOfObjects; i++ {
		// Use path as payload
		err = gcs.Put(ctx, gen(i), bytes.NewBufferString(gen(i)), storage.IfNotPresent)
		require.NoError(t, err)
	}

	cleanup := func() {
		for i := 0; i < numOfObjects; i++ {
			err = gcs.Delete(ctx, gen(i))
			require.NoError(t, err)
		}
		log.Printf("Delete bucket %s ", bucket)
		err = client.Bucket(bucket).Delete(ctx)
		require.NoError(t, err)
	}

	return gcs, cleanup
}

func TestGCSGet(t *testing.T) {
	ctx := context.Background()
	count := 20
	gcs, cleanup := setup(t, count)
	defer cleanup()
	for i := 0; i < count; i++ {
		rdr, err := gcs.Get(ctx, gen(i))
		require.NoError(t, err)

		b, err := ioutil.ReadAll(rdr)
		require.NoError(t, err)
		assert.Equal(t, gen(i), string(b))

		// ReadAt: buffer larger than length
		start := 1
		end := len(gen(i))
		rdrAt, err := gcs.GetAt(ctx, gen(i))
		require.NoError(t, err)
		p := make([]byte, 2*end)
		n, err := rdrAt.ReadAt(p, int64(start))
		require.NoError(t, err)
		assert.Equal(t, len(gen(i))-start, n)
		assert.Equal(t, gen(i)[start:], string(p[:n]))
		for m := n; m < len(p); m++ {
			assert.Equal(t, uint8(0x00), p[m])
		}

		// ReadAt: buffer smaller than length
		start = 2
		end = len(gen(i)) - 1
		rdrAt, err = gcs.GetAt(ctx, gen(i))
		require.NoError(t, err)
		p = make([]byte, end-start)
		n, err = rdrAt.ReadAt(p, int64(start))
		require.NoError(t, err)
		assert.Equal(t, end-start, n)
		assert.Equal(t, gen(i)[start:end], string(p[:n]))
		for m := n; m < len(p); m++ {
			assert.Equal(t, uint8(0x00), p[m])
		}

		// ReadAt: buffer zero
		start = 0
		rdrAt, err = gcs.GetAt(ctx, gen(i))
		require.NoError(t, err)
		p = make([]byte, 0)
		n, err = rdrAt.ReadAt(p, int64(start))
		require.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, 0, len(p))

		// ReadAt: offset beyond length
		start = len(gen(i)) + 1
		end = len(gen(i)) + 2
		rdrAt, err = gcs.GetAt(ctx, gen(i))
		require.NoError(t, err)
		p = make([]byte, end-start)
		n, err = rdrAt.ReadAt(p, int64(start))
		require.NotNil(t, err)
		require.Equal(t, 0, n)
		require.True(t, strings.Contains(err.Error(), "InvalidRange"))
		for m := 0; m < len(p); m++ {
			assert.Equal(t, uint8(0x00), p[m])
		}
	}
}

func TestHas(t *testing.T) {
	ctx := context.Background()
	count := 2
	gcs, cleanup := setup(t, count)
	defer cleanup()

	for i := 0; i < count; i++ {

		has, err := gcs.Has(ctx, gen(i))
		require.NoError(t, err)
		require.True(t, has)

	}
	has, err := gcs.Has(ctx, gen(count+1))
	require.NoError(t, err)
	require.False(t, has)
}

func TestPut(t *testing.T) {
	ctx := context.Background()
	count := 3
	gcs, cleanup := setup(t, 0)
	defer cleanup()
	for i := 0; i < count; i++ {
		err := gcs.Put(ctx, gen(i), bytes.NewBufferString(gen(i)), storage.IfNotPresent)
		require.NoError(t, err)

		rdr, err := gcs.Get(ctx, gen(i))
		require.NoError(t, err)

		b, err := ioutil.ReadAll(rdr)
		require.NoError(t, err)
		assert.Equal(t, gen(i), string(b))

		err = gcs.Delete(ctx, gen(i))
		require.NoError(t, err)
	}
}

func TestKey(t *testing.T) {
	ctx := context.Background()

	gcs, cleanup := setup(t, 2)
	defer cleanup()

	key, err := gcs.Keys(ctx)
	require.NoError(t, err)

	assert.Len(t, key, 2)

}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	count := 10
	gcs, cleanup := setup(t, 0)
	defer cleanup()

	for i := 0; i < count; i++ {
		err := gcs.Put(ctx, gen(i), bytes.NewBufferString(gen(i)), storage.IfNotPresent)
		require.NoError(t, err)
	}
	for i := 0; i < count-1; i++ {
		err := gcs.Delete(ctx, gen(i))
		require.NoError(t, err)
	}
	keys, err := gcs.Keys(ctx)
	assert.NoError(t, err)
	assert.Len(t, keys, 1)
	err = gcs.Delete(ctx, gen(count-1))
	require.NoError(t, err)
	keys, err = gcs.Keys(ctx)
	assert.NoError(t, err)
	assert.Len(t, keys, 0)
}
