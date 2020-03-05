package core

import (
	"context"
	"path"
	"sort"
	"sync"
	"testing"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/core/mocks"
	"github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/errors"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitOpsGetSplitStore(t *testing.T) {
	cleanup, meta, vmeta, blob := buildDiamondTest()
	defer cleanup()

	store := GetSplitStore(mocks.FakeContext2(meta, vmeta, blob))
	assert.Equal(t, "localfs@"+vmeta, store.String())
}

func TestSplitOps(t *testing.T) {
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

	const numSplits = 3
	metas := make(model.SplitDescriptors, 0, numSplits)
	for i := 0; i < numSplits; i++ {
		meta, ers := CreateSplit(diamondRepo, diamondKey, ctx)
		require.NoError(t, ers)
		metas = append(metas, meta)
	}
	sort.Sort(metas)

	splits, err := ListSplits(diamondRepo, diamondKey, ctx)
	require.NoError(t, err)
	require.Len(t, splits, numSplits)

	sort.Sort(metas)
	for i := 0; i < numSplits; i++ {
		s := splits[i]
		split, err := GetSplit(diamondRepo, diamondKey, s.SplitID, ctx)
		require.NoError(t, err)
		assert.EqualValues(t, s, split)
		assert.EqualValues(t, metas[i], split)
	}
}

func spinTestSplit(t testing.TB, ctx context2.Stores, diamondID string, state model.SplitState, opts ...SplitOption) *Split {
	// spin new split
	split := NewSplit(diamondRepo, diamondID, ctx, opts...)
	// complete split
	err := split.WithState(state).uploadDescriptor()
	require.NoError(t, err)

	err = split.downloadDescriptor()
	require.NoError(t, err)

	return split
}

func TestSplitOpsFailures(t *testing.T) {
	cleanup, meta, vmeta, blob := buildDiamondTest()
	defer cleanup()

	ctx := mocks.FakeContext2(meta, vmeta, blob)

	_, err := CreateSplit("unknown-repo)", "unknown-diamond", ctx)
	require.Error(t, err) // non existent repo

	require.NoError(t, CreateRepo(model.RepoDescriptor{Name: diamondRepo, Description: "test"}, ctx))

	metad, err := CreateDiamond(diamondRepo, ctx)
	require.NoError(t, err)

	// failure: diamond is required
	_, err = CreateSplit(diamondRepo, "", ctx)
	require.Error(t, err)

	diamonds, err := ListDiamonds(diamondRepo, ctx)
	require.NoError(t, err)
	require.Len(t, diamonds, 1)

	diamond := diamonds[0]
	diamondID := diamond.DiamondID
	assert.Equal(t, metad.DiamondID, diamondID)

	metas, err := CreateSplit(diamondRepo, diamondID, ctx)
	require.NoError(t, err)

	splits, err := ListSplits(diamondRepo, diamondID, ctx)
	require.NoError(t, err)
	require.Len(t, splits, 1)
	split := splits[0]
	assert.EqualValues(t, metas, split)

	coreDiamond := NewDiamond(diamondRepo, ctx, DiamondDescriptor(&diamond))
	err = coreDiamond.WithState(model.DiamondDone).uploadDescriptor()
	require.NoError(t, err)

	// failure: diamond is done
	_, err = CreateSplit(diamondRepo, diamondID, ctx)
	require.Error(t, err)

	err = coreDiamond.WithState(model.DiamondInitialized).uploadDescriptor() // diamond can't be initialized again
	require.Error(t, err)

	// spin up new diamond
	coreDiamond = NewDiamond(diamondRepo, ctx)
	err = coreDiamond.WithState(model.DiamondInitialized).uploadDescriptor()
	require.NoError(t, err)
	diamondID = coreDiamond.DiamondDescriptor.DiamondID

	// spin new split, same ID
	coreSplit := spinTestSplit(t, ctx, diamondID, model.SplitDone, SplitDescriptor(&split))

	// failure: starting split with same ID when it is done
	_, err = CreateSplit(diamondRepo, diamondID, ctx,
		SplitDescriptor(&coreSplit.SplitDescriptor),
		SplitLogger(mocks.TestLogger()),
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, status.ErrSplitAlreadyDone))

	// failure: trying to complete a split which is already done
	err = coreSplit.WithState(model.SplitDone).uploadDescriptor()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file exist") // NOTE: atm, the error return is storage dependent.

	// failure: trying to restart a split which is already done
	_, err = CreateSplit(diamondRepo, diamondID, ctx, SplitDescriptor(&coreSplit.SplitDescriptor))
	require.Error(t, err)
	assert.True(t, errors.Is(err, status.ErrSplitAlreadyDone))

	// spin new split
	coreSplit = spinTestSplit(t, ctx, diamondID, model.SplitRunning)

	// success: starting split with same ID when it has failed (that is precisely the point of restarting it)
	_, err = CreateSplit(diamondRepo, diamondID, ctx, SplitDescriptor(&coreSplit.SplitDescriptor))
	require.NoError(t, err)

	// spin new running split
	coreSplit = spinTestSplit(t, ctx, diamondID, model.SplitRunning)

	// success: starting split with same ID when it is in running state (but should actually not be running)
	_, err = CreateSplit(diamondRepo, diamondID, ctx, SplitDescriptor(&coreSplit.SplitDescriptor))
	require.NoError(t, err)

	// success: starting split with same ID when it is in running state (but should actually not be running)
	_, err = CreateSplit(diamondRepo, diamondID, ctx, SplitDescriptor(&coreSplit.SplitDescriptor))
	require.NoError(t, err)

	// failure: must exist option enabled
	_, err = CreateSplit(diamondRepo, diamondID, ctx, SplitMustExist(true))
	require.Error(t, err)

	// no option, with some arbitrary user-defined split ID: ok
	_, err = CreateSplit(diamondRepo, diamondID, ctx, SplitDescriptor(model.NewSplitDescriptor(model.SplitID("fred-pod-1"))))
	require.NoError(t, err)
}

func TestSplitOpsListApply(t *testing.T) {
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
	assert.Equal(t, metad.DiamondID, diamondKey)

	require.NoError(t, DiamondExists(diamondRepo, diamondKey, ctx))

	const numSplits = 5
	metas := make(model.SplitDescriptors, numSplits)
	var wg sync.WaitGroup
	for i := 0; i < numSplits; i++ {
		wg.Add(1)
		go func(t *testing.T, i int, wg *sync.WaitGroup) {
			defer wg.Done()
			var ers error
			meta, ers := CreateSplit(diamondRepo, diamondKey, ctx)
			require.NoError(t, ers)
			metas[i] = meta
		}(t, i, &wg)
	}
	wg.Wait()
	sort.Sort(metas)

	listed := make([]model.SplitDescriptor, 0, numSplits)
	err = ListSplitsApply(diamondRepo, diamondKey, ctx,
		func(sd model.SplitDescriptor) error {
			listed = append(listed, sd)
			return nil
		})
	require.NoError(t, err)
	require.Len(t, listed, numSplits)

	for i := range listed {
		assert.Equal(t, metas[i], listed[i])
	}
}
