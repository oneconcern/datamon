package gcs

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"

	gcsStorage "cloud.google.com/go/storage"
	"github.com/oneconcern/datamon/internal/rand"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

const (
	longPath = "this/is/a/long/path/to/an/object/the/object/is/under/this/path/list/with/prefix/please/"
)

func constStringWithIndex(i int) string {
	return longPath + fmt.Sprint(i)
}

func setup(t testing.TB, numOfObjects int, numOfVersions int) (storage.Store, func()) {
	ctx := context.Background()
	bucket := "deleteme-datamontest-" + rand.LetterString(15)

	client, err := gcsStorage.NewClient(context.TODO(), option.WithScopes(gcsStorage.ScopeFullControl))
	require.NoError(t, err)

	versioningEnabled := numOfVersions > 1

	require.NoError(t,
		client.Bucket(bucket).Create(ctx, "onec-co",
			&gcsStorage.BucketAttrs{
				VersioningEnabled: versioningEnabled,
			}),
		"Failed to create bucket:"+bucket)
	t.Logf("Created bucket %s ", bucket)

	newGcs := func(version storage.Version) gcs {
		var (
			gcsStore *gcs
			err      error
		)
		if version.GcsVersion() == storage.GcsSentinelVersion {
			gcsStore, err = implNew(context.TODO(), bucket,
				"", // Use GOOGLE_APPLICATION_CREDENTIALS env variable
			)
		} else {
			gcsStore, err = implNew(context.TODO(), bucket,
				"", // Use GOOGLE_APPLICATION_CREDENTIALS env variable
				WithVersion(version))
		}
		require.NoError(t, err, "failed to create gcs client")
		return *gcsStore
	}

	newGcsStore := func(version storage.Version) storage.Store {
		gcsInst := newGcs(version)
		return &gcsInst
	}

	// here, we demo the proper use of the sentinel value by _not_ passing it to google-cloud-go
	// and assigning semantics to the sentinel value using the sentinel value itself.,
	// i.e. -- the only property that the sentinel value need preserve is that it is not
	// a valid [in gcs lingo] generation number.
	gcsInst := newGcs(storage.NewVersionGcs(storage.GcsSentinelVersion))
	gcsInstStore := newGcsStore(storage.NewVersionGcs(storage.GcsSentinelVersion))
	require.NoError(t, err, "failed to create gcs client")
	wg := sync.WaitGroup{}
	makeGcsPut := func(gcs gcs) func(path string) error {
		return func(path string) error {
			buf := bytes.NewBufferString(path)
			if versioningEnabled {
				return gcs.Put(ctx, path, buf,
					storage.OverWrite)
			} else {
				return gcs.Put(ctx, path, buf,
					storage.NoOverWrite)
			}
		}
	}
	create := func(i int, wg *sync.WaitGroup) {
		defer wg.Done()
		path := constStringWithIndex(i)
		e := (makeGcsPut(gcsInst))(path)
		require.NoError(t, e, "Index at: "+fmt.Sprint(i))
	}
	for i := 0; i < numOfObjects; i++ {
		index := i
		for j := 0; j < numOfVersions; j++ {
			// Use path as payload
			wg.Add(1)
			go create(index, &wg)
		}
	}
	wg.Wait()

	cleanup := func() {
		delete := func(key string, wg *sync.WaitGroup) {
			defer wg.Done()

			if !versioningEnabled {
				err = gcsInst.Delete(ctx, key)
				require.NoError(t, err, "failed to delete:"+key)
			} else {
				versions, err := gcsInstStore.(storage.StoreVersioned).KeyVersions(ctx, key)
				require.NoError(t, err, "couldn't list versions:"+key)
				t.Logf("have versions %v\n", versions)
				for _, version := range versions {
					t.Logf("deleting version %q\n", version)
					gcsWithVersion := newGcs(storage.NewVersionGcs(version.GcsVersion()))
					require.NoError(t,
						gcsWithVersion.Delete(ctx, key),
						"failed to delete:"+key+" at version:"+version.String())
				}
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
		keys, _ = gcsInst.Keys(ctx)
		for _, key := range keys {
			wg.Add(1)
			delete(key, &wg)
		}

		wg.Wait()
		t.Logf("Delete bucket %s ", bucket)
		err = client.Bucket(bucket).Delete(ctx)
		require.NoError(t, err, "Failed to delete bucket:"+bucket)
	}

	return &gcsInst, cleanup
}

func TestGcs_Get(t *testing.T) {
	ctx := context.Background()
	count := 20
	gcs, cleanup := setup(t, count, 1)
	defer cleanup()
	for i := 0; i < count; i++ {
		rdr, err := gcs.Get(ctx, constStringWithIndex(i))
		require.NoError(t, err)

		b, err := ioutil.ReadAll(rdr)
		require.NoError(t, err)
		assert.Equal(t, constStringWithIndex(i), string(b))

		// ReadAt: buffer larger than length
		start := 1
		end := len(constStringWithIndex(i))
		rdrAt, err := gcs.GetAt(ctx, constStringWithIndex(i))
		require.NoError(t, err)
		p := make([]byte, 2*end)
		n, err := rdrAt.ReadAt(p, int64(start))
		require.NoError(t, err)
		assert.Equal(t, len(constStringWithIndex(i))-start, n)
		assert.Equal(t, constStringWithIndex(i)[start:], string(p[:n]))
		for m := n; m < len(p); m++ {
			assert.Equal(t, uint8(0x00), p[m])
		}

		// ReadAt: buffer smaller than length
		start = 2
		end = len(constStringWithIndex(i)) - 1
		rdrAt, err = gcs.GetAt(ctx, constStringWithIndex(i))
		require.NoError(t, err)
		p = make([]byte, end-start)
		n, err = rdrAt.ReadAt(p, int64(start))
		require.NoError(t, err)
		assert.Equal(t, end-start, n)
		assert.Equal(t, constStringWithIndex(i)[start:end], string(p[:n]))
		for m := n; m < len(p); m++ {
			assert.Equal(t, uint8(0x00), p[m])
		}

		// ReadAt: buffer zero
		start = 0
		rdrAt, err = gcs.GetAt(ctx, constStringWithIndex(i))
		require.NoError(t, err)
		p = make([]byte, 0)
		n, err = rdrAt.ReadAt(p, int64(start))
		require.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, 0, len(p))

		// ReadAt: offset beyond length
		start = len(constStringWithIndex(i)) + 1
		end = len(constStringWithIndex(i)) + 2
		rdrAt, err = gcs.GetAt(ctx, constStringWithIndex(i))
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

func TestGcs_Has(t *testing.T) {
	ctx := context.Background()
	count := 2
	gcs, cleanup := setup(t, count, 1)
	defer cleanup()

	for i := 0; i < count; i++ {

		has, err := gcs.Has(ctx, constStringWithIndex(i))
		require.NoError(t, err)
		require.True(t, has)

	}
	has, err := gcs.Has(ctx, constStringWithIndex(count+1))
	require.NoError(t, err)
	require.False(t, has)
}

func TestGcs_Put(t *testing.T) {
	ctx := context.Background()
	count := 3
	gcs, cleanup := setup(t, 0, 1)
	defer cleanup()
	for i := 0; i < count; i++ {
		err := gcs.Put(ctx, constStringWithIndex(i), bytes.NewBufferString(constStringWithIndex(i)), storage.NoOverWrite)
		require.NoError(t, err)

		rdr, err := gcs.Get(ctx, constStringWithIndex(i))
		require.NoError(t, err)

		b, err := ioutil.ReadAll(rdr)
		require.NoError(t, err)
		assert.Equal(t, constStringWithIndex(i), string(b))

		err = gcs.Delete(ctx, constStringWithIndex(i))
		require.NoError(t, err)
	}
}

func TestGcs_Keys(t *testing.T) {
	ctx := context.Background()

	gcs, cleanup := setup(t, 2, 1)
	defer cleanup()

	key, err := gcs.Keys(ctx)
	require.NoError(t, err)

	assert.Len(t, key, 2)

}

func TestGcs_KeyVersions(t *testing.T) {
	ctx := context.Background()
	numOfObjects := 4
	numOfVersions := 3
	gcs, cleanup := setup(t, numOfObjects, numOfVersions)
	defer cleanup()

	gcsV := gcs.(storage.StoreVersioned)

	keys, err := gcs.Keys(ctx)
	require.NoError(t, err, "list keys")

	assert.Len(t, keys, numOfObjects)

	for _, key := range keys {
		versionSet := make(map[storage.Version]bool)
		versions, err := gcsV.KeyVersions(ctx, key)
		require.NoError(t, err, "list versions. key: "+key)
		assert.Len(t, versions, numOfVersions)
		for _, version := range versions {
			require.False(t, versionSet[version], "versions all unique")
			versionSet[version] = true
		}
	}
}

func TestGcs_Delete(t *testing.T) {
	ctx := context.Background()
	count := 10
	gcs, cleanup := setup(t, 0, 1)
	defer cleanup()

	for i := 0; i < count; i++ {
		err := gcs.Put(ctx, constStringWithIndex(i),
			bytes.NewBufferString(constStringWithIndex(i)), storage.NoOverWrite)
		require.NoError(t, err)
	}
	for i := 0; i < count-1; i++ {
		err := gcs.Delete(ctx, constStringWithIndex(i))
		require.NoError(t, err)
	}
	keys, err := gcs.Keys(ctx)
	assert.NoError(t, err)
	assert.Len(t, keys, 1)
	err = gcs.Delete(ctx, constStringWithIndex(count-1))
	require.NoError(t, err)
	keys, err = gcs.Keys(ctx)
	assert.NoError(t, err)
	assert.Len(t, keys, 0)
}

func TestGcs_CreateNew(t *testing.T) {
	ctx := context.Background()
	gcs, cleanup := setup(t, 0, 1)
	defer cleanup()

	err := gcs.Put(ctx, constStringWithIndex(1), bytes.NewBufferString(constStringWithIndex(1)), storage.NoOverWrite)
	require.NoError(t, err)

	// Expected to fail, trying to create an Object that already exists without overwrite flag
	err = gcs.Put(ctx, constStringWithIndex(1), bytes.NewBufferString(constStringWithIndex(1)), storage.NoOverWrite)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "Precondition Failed"))

	err = gcs.Put(ctx, constStringWithIndex(1), bytes.NewBufferString(constStringWithIndex(1)), storage.OverWrite)
	require.NoError(t, err)

	err = gcs.Put(ctx, constStringWithIndex(2), bytes.NewBufferString(constStringWithIndex(1)), storage.OverWrite)
	require.NoError(t, err)
}

func TestGcs_KeysPrefix(t *testing.T) {
	ctx := context.Background()
	count := 124
	gcs, cleanup := setup(t, count, 1)
	defer cleanup()

	fetch := count - 1
	keys, next, err := gcs.KeysPrefix(ctx, "", "", "", fetch)
	require.NoError(t, err)
	require.Equal(t, fetch, len(keys))

	keys, next, err = gcs.KeysPrefix(ctx, next, "", "", fetch)
	require.NoError(t, err)
	require.Equal(t, count-fetch, len(keys))
	require.Equal(t, "", next)

	keys = *new([]string)
	next = ""
	for i := 0; i < count; i++ {
		var k []string
		k, next, err = gcs.KeysPrefix(ctx, next, "", "", count/10)
		require.NoError(t, err)
		keys = append(keys, k...)
		if next == "" {
			break
		}
	}
	require.Equal(t, count, len(keys))
}

var TotalObjects = 2156

func listKeysBatch(gcs storage.Store, b *testing.B, count int, batch int) {
	for n := 0; n < b.N; n++ {
		var next string
		var keys []string
		for {
			var k []string
			var err error
			k, next, err = gcs.KeysPrefix(context.Background(), next, "", "", batch)
			if err != nil {
				b.Error(err.Error())
				panic("hit error:" + err.Error())
			}
			keys = append(keys, k...)
			if next == "" {
				break
			}
		}
		if len(keys) != count {
			b.Error("incorrect key count: " + fmt.Sprint(count) + " len:" + fmt.Sprint(len(keys)))
		}
	}
}

func keysPrefix100(b *testing.B, gcs storage.Store) {
	listKeysBatch(gcs, b, TotalObjects, 100)
}
func keysPrefix500(b *testing.B, gcs storage.Store) {
	listKeysBatch(gcs, b, TotalObjects, 500)
}
func keysPrefix1000(b *testing.B, gcs storage.Store) {
	listKeysBatch(gcs, b, TotalObjects, 1000)
}
func keysPrefix1500(b *testing.B, gcs storage.Store) {
	listKeysBatch(gcs, b, TotalObjects, 1500)
}
func keysPrefix2000(b *testing.B, gcs storage.Store) {
	listKeysBatch(gcs, b, TotalObjects, 2000)
}
func BenchmarkRun(b *testing.B) {
	gcs, _ := setup(b, TotalObjects, 1)
	//defer cleanup()
	run := func(fn func(b2 *testing.B, gcs storage.Store)) {
		fn(b, gcs)
	}
	fns := []func(*testing.B, storage.Store){
		keysPrefix100,
		keysPrefix500,
		keysPrefix1000,
		keysPrefix1500,
		keysPrefix2000,
	}
	for _, fn := range fns {
		testFn := fn
		b.Run(runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name(), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				run(testFn)
			}
		})
	}
}
