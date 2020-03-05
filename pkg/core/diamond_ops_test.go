package core

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"testing"

	"github.com/oneconcern/datamon/pkg/core/mocks"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	diamondRepo = "repo-with-diamonds"
)

func buildDiamondTest() (func(), string, string, string) {
	meta, _ := ioutil.TempDir("", "meta")
	vmeta, _ := ioutil.TempDir("", "vmeta")
	blob, _ := ioutil.TempDir("", "blob")
	return func() {
		_ = os.RemoveAll(meta)
		_ = os.RemoveAll(vmeta)
		_ = os.RemoveAll(blob)
	}, meta, vmeta, blob
}

func TestDiamondOpsGetDiamondStore(t *testing.T) {
	cleanup, meta, vmeta, blob := buildDiamondTest()
	defer cleanup()

	store := GetDiamondStore(mocks.FakeContext2(meta, vmeta, blob))
	assert.Equal(t, "localfs@"+vmeta, store.String())
}

func TestDiamondOps(t *testing.T) {
	cleanup, meta, vmeta, blob := buildDiamondTest()
	defer cleanup()

	ctx := mocks.FakeContext2(meta, vmeta, blob)

	require.NoError(t, CreateRepo(model.RepoDescriptor{Name: diamondRepo, Description: "test"}, ctx))

	metad, err := CreateDiamond(diamondRepo, ctx)
	require.NoError(t, err)

	store := GetDiamondStore(ctx)
	keys, err := store.Keys(context.Background())
	require.NoError(t, err)
	require.Len(t, keys, 1)

	diamondKey := path.Base(path.Dir(keys[0]))
	require.NoError(t, DiamondExists(diamondRepo, diamondKey, ctx))
	assert.Equal(t, metad.DiamondID, diamondKey)

	diamond, err := GetDiamond(diamondRepo, diamondKey, ctx)
	require.NoError(t, err)

	assert.Equal(t, diamondKey, diamond.DiamondID)
	assert.Equal(t, model.DiamondInitialized, diamond.State)
	assert.True(t, diamond.EndTime.IsZero())

	require.NoError(t, diamondReady(diamondRepo, diamondKey, ctx))
}

func TestDiamondOpsFailures(t *testing.T) {
	cleanup, meta, vmeta, blob := buildDiamondTest()
	defer cleanup()

	ctx := mocks.FakeContext2(meta, vmeta, blob)

	_, err := CreateDiamond(diamondRepo, ctx)
	require.Error(t, err) // repo is absent

	require.Error(t, DiamondExists(diamondRepo, "fancy-diamond", ctx)) // diamond is absent

	require.NoError(t, CreateRepo(model.RepoDescriptor{Name: diamondRepo, Description: "test"}, ctx))

	_, err = CreateDiamond(diamondRepo, ctx)
	require.NoError(t, err)

	store := GetDiamondStore(ctx)
	keys, err := store.Keys(context.Background())
	require.NoError(t, err)
	require.Len(t, keys, 1)

	diamondKey := path.Base(path.Dir(keys[0]))

	require.NoError(t, DiamondExists(diamondRepo, diamondKey, ctx))
	diamond, err := GetDiamond(diamondRepo, diamondKey, ctx)
	require.NoError(t, err)

	coreDiamond := NewDiamond(diamondRepo, ctx, DiamondDescriptor(&diamond))
	err = coreDiamond.WithState(model.DiamondInitialized).uploadDescriptor() // recreate existing diamond
	require.Error(t, err)

	require.NoError(t, diamondReady(diamondRepo, diamond.DiamondID, ctx)) // ok: diamond is ready

	err = coreDiamond.WithState(model.DiamondCanceled).uploadDescriptor() // ok: terminate diamond
	require.NoErrorf(t, err, "storage returned unexpected error: %#v", err)

	require.Error(t, diamondReady(diamondRepo, diamond.DiamondID, ctx)) // diamond is no more ready

	err = coreDiamond.WithState(model.DiamondDone).uploadDescriptor() // diamond already terminated
	require.Error(t, err)

	err = coreDiamond.Cancel() // diamond already terminated
	require.Error(t, err)

	_, err = CreateDiamond(diamondRepo, ctx,
		DiamondDescriptor(model.NewDiamondDescriptor(model.DiamondID(""))),
	)
	require.NoError(t, err)

	descriptor := model.NewDiamondDescriptor()
	descriptor.DiamondID = ""

	_, err = CreateDiamond(diamondRepo, ctx, DiamondDescriptor(descriptor))
	require.Error(t, err)
}

func TestDiamondListApply(t *testing.T) {
	cleanup, meta, vmeta, blob := buildDiamondTest()
	defer cleanup()

	// Use localfs WithLock option to emulate proper concurrency on the mocked fs.
	// This should work correctly with gcs.
	ctx := mocks.FakeContext2(meta, vmeta, blob)

	require.NoError(t, CreateRepo(model.RepoDescriptor{Name: diamondRepo, Description: "test"}, ctx))

	const numDiamonds = 10
	for i := 0; i < numDiamonds; i++ {
		_, err := CreateDiamond(diamondRepo, ctx)
		require.NoError(t, err)
	}

	store := GetDiamondStore(ctx)
	keys, err := store.Keys(context.Background())
	require.NoError(t, err)
	require.Len(t, keys, numDiamonds)

	listed := make(model.DiamondDescriptors, 0, numDiamonds)
	err = ListDiamondsApply(diamondRepo, ctx,
		func(dd model.DiamondDescriptor) error {
			listed = append(listed, dd)
			return nil
		})
	require.NoError(t, err)
	require.Len(t, listed, numDiamonds)
	for _, diamond := range listed {
		assert.Equal(t, model.DiamondInitialized, diamond.State)
		assert.Len(t, diamond.Splits, 0)
	}

	const numSplits = 5
	var wg sync.WaitGroup
	for _, key := range keys {
		diamondID := path.Base(path.Dir(key))

		for i := 0; i < numSplits; i++ {
			wg.Add(1)
			go func(t *testing.T, wg *sync.WaitGroup) {
				defer wg.Done()
				_, ers := CreateSplit(diamondRepo, diamondID, ctx)
				require.NoError(t, ers)
			}(t, &wg)
		}
	}
	wg.Wait()

	listed = make(model.DiamondDescriptors, 0, numDiamonds)
	err = ListDiamondsApply(diamondRepo, ctx,
		func(dd model.DiamondDescriptor) error {
			listed = append(listed, dd)
			return nil
		})
	require.NoError(t, err)
	require.Len(t, listed, numDiamonds)
	for _, diamond := range listed {
		assert.Equal(t, model.DiamondInitialized, diamond.State)
		assert.Len(t, diamond.Splits, 0) // splits are added when done, not now
	}
}

func TestDiamondOpsCancel(t *testing.T) {
	cleanup, meta, vmeta, blob := buildDiamondTest()
	defer cleanup()

	ctx := mocks.FakeContext2(meta, vmeta, blob)

	_, err := CreateDiamond(diamondRepo, ctx)
	require.Error(t, err)

	require.Error(t, DiamondExists(diamondRepo, "fancy-diamond", ctx))

	require.NoError(t, CreateRepo(model.RepoDescriptor{Name: diamondRepo, Description: "test"}, ctx))

	diamond, err := CreateDiamond(diamondRepo, ctx)
	require.NoError(t, err)

	d := NewDiamond(diamondRepo, ctx, DiamondDescriptor(&diamond))
	require.NoError(t, d.Cancel())

	require.NoError(t, d.downloadDescriptor())
	assert.Equal(t, model.DiamondCanceled, d.DiamondDescriptor.State)

	_, err = CreateSplit(diamondRepo, diamond.DiamondID, ctx)
	require.Error(t, err)
}
