package gcs

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"testing"

	gcsStorage "cloud.google.com/go/storage"
	"github.com/oneconcern/datamon/internal"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

const (
	testObject1 = "test-object-1"
	testObject2 = "test-object-2"
	testObject3 = "test-object-3"

	testObject1Content = "gcs test-object-1"
	testObject2Content = "gcs test-object-2"
	testObject3Content = "gcs test-object-3"
)

func setup(t testing.TB) (storage.Store, func()) {

	ctx := context.Background()

	bucket := "datamontest-" + internal.RandStringBytesMaskImprSrc(15)
	log.Printf("Created bucket %s ", bucket)

	client, err := gcsStorage.NewClient(context.TODO(), option.WithScopes(gcsStorage.ScopeFullControl))
	require.NoError(t, err)
	err = client.Bucket(bucket).Create(ctx, "onec-co", nil)
	require.NoError(t, err)

	gcs, err := New(bucket, "") // Use GOOGLE_APPLICATION_CREDENTIALS env variable
	require.NoError(t, err)

	err = gcs.Put(ctx, testObject1, bytes.NewBufferString(testObject1Content), storage.IfNotPresent)
	require.NoError(t, err)

	err = gcs.Put(ctx, testObject2, bytes.NewBufferString(testObject2Content), storage.IfNotPresent)
	require.NoError(t, err)

	cleanup := func() {
		err = gcs.Delete(ctx, testObject1)
		require.NoError(t, err)

		err = gcs.Delete(ctx, testObject2)
		require.NoError(t, err)

		log.Printf("Delete bucket %s ", bucket)
		err = client.Bucket(bucket).Delete(ctx)
		require.NoError(t, err)
	}

	return gcs, cleanup
}

func TestGCSGet(t *testing.T) {
	ctx := context.Background()

	gcs, cleanup := setup(t)
	defer cleanup()

	rdr, err := gcs.Get(ctx, testObject1)
	require.NoError(t, err)

	b, err := ioutil.ReadAll(rdr)
	require.NoError(t, err)
	assert.Equal(t, testObject1Content, string(b))

	rdr, err = gcs.Get(ctx, testObject2)
	require.NoError(t, err)

	b, err = ioutil.ReadAll(rdr)
	require.NoError(t, err)
	assert.Equal(t, testObject2Content, string(b))
}

func TestHas(t *testing.T) {
	ctx := context.Background()

	gcs, cleanup := setup(t)
	defer cleanup()

	has, err := gcs.Has(ctx, testObject1)
	require.NoError(t, err)
	require.True(t, has)

	has, err = gcs.Has(ctx, testObject2)
	require.NoError(t, err)
	require.True(t, has)
}

func TestPut(t *testing.T) {
	ctx := context.Background()

	gcs, cleanup := setup(t)
	defer cleanup()

	err := gcs.Put(ctx, testObject3, bytes.NewBufferString(testObject3Content), storage.IfNotPresent)
	require.NoError(t, err)

	rdr, err := gcs.Get(ctx, testObject3)
	require.NoError(t, err)

	b, err := ioutil.ReadAll(rdr)
	require.NoError(t, err)
	assert.Equal(t, testObject3Content, string(b))

	err = gcs.Delete(ctx, testObject3)
	require.NoError(t, err)
}

func TestKey(t *testing.T) {
	ctx := context.Background()

	gcs, cleanup := setup(t)
	defer cleanup()

	key, err := gcs.Keys(ctx)
	require.NoError(t, err)

	assert.Len(t, key, 2)

}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	gcs, cleanup := setup(t)
	defer cleanup()

	err := gcs.Put(ctx, testObject3, bytes.NewBufferString(testObject3Content), storage.IfNotPresent)
	require.NoError(t, err)

	err = gcs.Delete(ctx, testObject3)
	require.NoError(t, err)

	keys, err := gcs.Keys(ctx)
	assert.NoError(t, err)
	assert.Len(t, keys, 2)
}
