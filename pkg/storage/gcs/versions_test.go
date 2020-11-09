package gcs

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strconv"
	"sync"
	"testing"

	gcsStorage "cloud.google.com/go/storage"
	"github.com/oneconcern/datamon/internal/rand"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

func setupVersions(t testing.TB, numOfObjects, numOfVersions int) (*gcs, func()) {
	ctx := context.Background()
	bucket := "deleteme-datamontest-" + rand.LetterString(15)

	client, err := gcsStorage.NewClient(context.TODO(), option.WithScopes(gcsStorage.ScopeFullControl))
	require.NoError(t, err)

	require.NoError(t,
		client.Bucket(bucket).Create(ctx, "onec-co",
			&gcsStorage.BucketAttrs{
				VersioningEnabled: true,
			}),
		"Failed to create bucket:"+bucket)
	t.Logf("Created versioned bucket %s ", bucket)

	gcsStore, err := New(context.TODO(), bucket, "") // Use GOOGLE_APPLICATION_CREDENTIALS env variable
	require.NoError(t, err, "failed to create gcs client")
	gcsV := gcsStore.(*gcs)
	wg := sync.WaitGroup{}
	gcsPut := func(path string) error {
		buf := bytes.NewBufferString(path)
		return gcsV.Put(ctx, path, buf, storage.OverWrite)
	}
	create := func(i int, wg *sync.WaitGroup) {
		defer wg.Done()
		path := constStringWithIndex(i)
		e := gcsPut(path)
		require.NoError(t, e, "Index at: "+fmt.Sprint(i))
	}
	for i := 0; i < numOfObjects; i++ {
		for j := 0; j < numOfVersions; j++ {
			wg.Add(1)
			go create(i, &wg)
		}
	}
	wg.Wait()

	cleanup := func() {
		delete := func(key string, wg *sync.WaitGroup) {
			defer wg.Done()
			versions, e := gcsV.KeyVersions(ctx, key)
			require.NoError(t, e, "couldn't list versions:"+key)
			t.Logf("have versions %v\n", versions)
			for _, version := range versions {
				t.Logf("deleting version %q\n", version)
				gen, _ := strconv.ParseInt(version, 10, 64)
				require.NoError(t,
					gcsV.client.Bucket(bucket).Object(key).Generation(gen).Delete(ctx),
					"failed to delete:"+key+" at version:"+version)
			}
		}

		wg := sync.WaitGroup{}
		for i := 0; i < numOfObjects; i++ {
			wg.Add(1)
			path := constStringWithIndex(i)
			delete(path, &wg)
		}
		wg.Wait()

		// Delete any keys created outside of setup at the end of test.
		var keys []string
		keys, _ = gcsV.Keys(ctx)
		for _, key := range keys {
			wg.Add(1)
			delete(key, &wg)
		}

		wg.Wait()
		t.Logf("Delete bucket %s ", bucket)
		err = client.Bucket(bucket).Delete(ctx)
		require.NoError(t, err, "Failed to delete bucket:"+bucket)
	}

	return gcsV, cleanup
}

func TestGcs_KeyVersions(t *testing.T) {
	ctx := context.Background()
	const (
		numOfObjects  = 4
		numOfVersions = 3
	)
	gcsV, cleanup := setupVersions(t, numOfObjects, numOfVersions)
	defer cleanup()

	ok, err := gcsV.IsVersioned(ctx)
	require.True(t, ok)
	require.NoError(t, err)

	keys, err := gcsV.Keys(ctx)
	require.NoError(t, err, "list keys")

	assert.Len(t, keys, numOfObjects)

	for _, key := range keys {
		versionSet := make(map[string]bool)
		versions, err := gcsV.KeyVersions(ctx, key)
		require.NoError(t, err, "list versions. key: "+key)
		assert.Len(t, versions, numOfVersions)

		for _, version := range versions {
			require.False(t, versionSet[version], "versions expected to be unique")
			versionSet[version] = true

			reader, erg := gcsV.GetVersion(ctx, key, version)
			require.NoError(t, erg)

			_, erd := ioutil.ReadAll(reader)
			require.NoError(t, erd)
		}
	}
}
