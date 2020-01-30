package core

import (
	"context"
	"math"
	"path/filepath"
	"strconv"

	context2 "github.com/oneconcern/datamon/pkg/context"

	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/core/mocks"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"gopkg.in/yaml.v2"

	"io/ioutil"
	"testing"
)

func testBundleEnv() mocks.TestEnv {
	// test parameters common to tests in bundle_test.go
	testRoot := filepath.FromSlash("../../testdata/core")
	sourceDir := filepath.Join(testRoot, "bundle", "source")
	return mocks.TestEnv{
		LeafSize:               cafs.DefaultLeafSize,
		Repo:                   "bundle-test-repo",
		BundleID:               "bundle123",
		TestRoot:               testRoot,
		SourceDir:              sourceDir,
		BlobDir:                filepath.Join(sourceDir, "blob"),
		MetaDir:                filepath.Join(sourceDir, "meta"),
		VmetaDir:               filepath.Join(sourceDir, "vmeta"),
		Wal:                    filepath.Join(sourceDir, "wal"),
		ReadLog:                filepath.Join(sourceDir, "readLog"),
		DestinationDir:         filepath.Join(testRoot, "bundle", "destination"),
		ReArchiveMetaDir:       filepath.Join(testRoot, "bundle", "destination2", "meta"),
		ReArchiveBlobDir:       filepath.Join(testRoot, "bundle", "destination2", "blob"),
		ReBundleEntriesPerFile: 3,
		Original:               filepath.Join(testRoot, "internal"),
		DataDir:                "dir",
	}
}

// fakeBundle builds a fake bundle for the testing environment, with fake
// context and consumable store located in DestinationDir.
func fakeBundle(ev mocks.TestEnv) *Bundle {
	bd := NewBDescriptor()
	return NewBundle(bd,
		Repo(ev.Repo),
		BundleID(ev.BundleID),
		ContextStores(mocks.FakeContext(ev.MetaDir, ev.BlobDir)),
		ConsumableStore(localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.DestinationDir))),
		Logger(mocks.TestLogger()),
	)
}

func TestBundle(t *testing.T) {
	var (
		bundleEntriesFileCount uint64 = 2
		dataFilesCount         uint64 = 4
	)

	ev := testBundleEnv()
	cleanup := mocks.SetupFakeDataBundle(t, bundleEntriesFileCount, dataFilesCount, ev)
	defer cleanup()

	// create repo in main context
	require.NoError(t, CreateRepo(mocks.FakeRepoDescriptor(ev.Repo), mocks.FakeContext(ev.MetaDir, "")))

	// create repo in archive context
	require.NoError(t, CreateRepo(mocks.FakeRepoDescriptor(ev.Repo), mocks.FakeContext(ev.ReArchiveMetaDir, "")))

	bundle := fakeBundle(ev)

	// Publish the bundle and compare with original
	require.NoError(t,
		implPublish(context.Background(),
			bundle,
			uint(dataFilesCount),
			func(s string) (bool, error) { return true, nil },
		))

	mocks.ValidatePublish(t, bundle.ConsumableStore, bundleEntriesFileCount, ev)

	// bundle id is set on upload
	bd := NewBDescriptor()
	consumableStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.DestinationDir))
	archiveBundle2 := NewBundle(bd,
		Repo(ev.Repo),
		ConsumableStore(consumableStore),
		ContextStores(mocks.FakeContext(ev.ReArchiveMetaDir, ev.ReArchiveBlobDir)),
	)
	require.NoError(t,
		implUpload(context.Background(), archiveBundle2, uint(ev.ReBundleEntriesPerFile), nil))

	require.True(t, validateUpload(t, bundle, archiveBundle2, bundleEntriesFileCount, dataFilesCount, ev))
}

func paramedTestPublishMetadata(t *testing.T, publish bool, ev mocks.TestEnv) {
	var (
		bundleEntriesFileCount uint64 = 3
		dataFilesCount         uint64 = 4
	)

	cleanup := mocks.SetupFakeDataBundleWithUnalignedFilelist(t,
		bundleEntriesFileCount, dataFilesCount, dataFilesCount/2, ev)
	defer cleanup()

	require.NoError(t, CreateRepo(mocks.FakeRepoDescriptor(ev.Repo), mocks.FakeContext(ev.MetaDir, "")))

	consumableStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.DestinationDir))
	bd := NewBDescriptor()
	bundle := NewBundle(bd,
		Repo(ev.Repo),
		BundleID(ev.BundleID),
		ConsumableStore(consumableStore),
		ContextStores(mocks.FakeContext(ev.MetaDir, ev.BlobDir)),
	)

	// Publish the bundle and compare with original
	require.NoError(t,
		implPublishMetadata(context.Background(), bundle, publish, uint(dataFilesCount)))

	validatePublishMetadata(t, bundle, publish, ev)
}

func TestPublishMetadata(t *testing.T) {
	paramedTestPublishMetadata(t, true, testBundleEnv())
}

func TestDownloadMetadata(t *testing.T) {
	paramedTestPublishMetadata(t, false, testBundleEnv())
}

func validatePublishMetadata(t *testing.T, bundle *Bundle, publish bool, ev mocks.TestEnv) {
	var i uint64

	consumableStore := bundle.ConsumableStore
	metaStore := bundle.MetaStore()

	readBundleEntries := func(store storage.Store,
		idx2Filelist func(uint64) string,
		pathToBundleDescriptor string,
	) []model.BundleEntry {
		bundleDescriptor := mocks.ReadBundleDescriptor(t, store, pathToBundleDescriptor)
		allBundleEntries := make([]model.BundleEntry, 0)
		for i = 0; i < bundleDescriptor.BundleEntriesFileCount; i++ {
			var currBundleEntries model.BundleEntries
			rdr, err := store.Get(context.Background(), idx2Filelist(i))
			require.NoError(t, err)
			buf, err := ioutil.ReadAll(rdr)
			require.NoError(t, err)
			require.NoError(t, yaml.Unmarshal(buf, &currBundleEntries))
			allBundleEntries = append(allBundleEntries, currBundleEntries.BundleEntries...)
		}
		return allBundleEntries
	}

	bundleEntriesListToMap := func(bundleEntries []model.BundleEntry) map[string]model.BundleEntry {
		bundleEntriesMap := make(map[string]model.BundleEntry)
		for _, bundleEntry := range bundleEntries {
			bundleEntriesMap[bundleEntry.NameWithPath] = bundleEntry
		}
		return bundleEntriesMap
	}

	compareBundleEntriesLists := func(
		bundleEntriesExpected []model.BundleEntry,
		bundleEntriesActual []model.BundleEntry,
	) {
		bundleEntriesExpectedMap := bundleEntriesListToMap(bundleEntriesExpected)
		bundleEntriesActualMap := bundleEntriesListToMap(bundleEntriesActual)
		require.Equal(t, len(bundleEntriesExpected), len(bundleEntriesActual),
			"found expected number of filelist entries")
		require.Equal(t, len(bundleEntriesExpectedMap), len(bundleEntriesActualMap),
			"found expected number of filelist entries up to name differences")
		for actualName, actualBundleEntry := range bundleEntriesActualMap {
			expectedBundleEntry, ok := bundleEntriesExpectedMap[actualName]
			require.True(t, ok, "actual name '"+actualName+"' exists in expected bundle entries")
			require.Equal(t, actualBundleEntry, expectedBundleEntry,
				"actual data entry matches expected entry")
		}
		require.Equal(t, bundleEntriesExpected, bundleEntriesActual,
			"found filelist entries in expected order")
	}

	metaBundleEntries := readBundleEntries(metaStore,
		func(i uint64) string { return model.GetArchivePathToBundleFileList(ev.Repo, ev.BundleID, i) },
		model.GetArchivePathToBundle(ev.Repo, ev.BundleID))
	if publish {
		consumableBundleEntries := readBundleEntries(consumableStore,
			func(i uint64) string {
				return filepath.Join(".datamon", ev.BundleID+"-bundle-files-"+strconv.Itoa(int(i))+".yaml")
			},
			model.GetConsumablePathToBundle(ev.BundleID))
		compareBundleEntriesLists(consumableBundleEntries, metaBundleEntries)
		compareBundleEntriesLists(metaBundleEntries, bundle.BundleEntries)
	} else {
		compareBundleEntriesLists(metaBundleEntries, bundle.BundleEntries)
	}

	t.Logf("tot bundle entries in memory %v", len(bundle.BundleEntries))
}

func validateUpload(t *testing.T,
	origBundle *Bundle, uploadedBundle *Bundle,
	bundleEntriesFileCount uint64,
	dataFilesCount uint64,
	ev mocks.TestEnv,
) bool {
	// Check if the blobs are the same as download
	require.True(t, mocks.ValidateDataFiles(t, ev.BlobDir, ev.ReArchiveBlobDir))

	// Check the file list contents
	origFileList := readBundleFilelist(t, origBundle, bundleEntriesFileCount, ev)
	reEntryFilesCount := uint64(math.Ceil(float64(bundleEntriesFileCount*dataFilesCount) / float64(ev.ReBundleEntriesPerFile)))
	reFileList := readBundleFilelist(t, uploadedBundle, reEntryFilesCount, ev)
	require.Equal(t, len(origFileList), len(reFileList))

	// in particular, note that the file lists might not list the files in the same order.
	origBundleEntries := make(map[string]model.BundleEntry)

	for _, ent := range origFileList {
		origBundleEntries[ent.Hash] = ent
	}

	for _, reEnt := range reFileList {
		origEnt, exists := origBundleEntries[reEnt.Hash]
		require.True(t, exists)
		require.Equal(t, reEnt.NameWithPath, origEnt.NameWithPath)
		// ??? unchecked values on on BundleEntry.  what values to test at this level of abstraction?
	}

	// Check Bundle File
	pathToBundleDescriptor := model.GetArchivePathToBundle(uploadedBundle.RepoID, uploadedBundle.BundleID)
	bundleDescriptor := mocks.ReadBundleDescriptor(t, uploadedBundle.MetaStore(), pathToBundleDescriptor)
	expectedBundleDescriptor := mocks.FakeBundleDescriptor(bundleEntriesFileCount, ev)
	require.Equal(t, bundleDescriptor.Parents, expectedBundleDescriptor.Parents)
	require.Equal(t, bundleDescriptor.LeafSize, expectedBundleDescriptor.LeafSize)
	// ??? unchecked values on on BundleDescriptor.  what values to test at this level of abstraction?
	return true
}

func readBundleFilelist(t *testing.T,
	bundle *Bundle,
	bundleEntriesFileCount uint64,
	ev mocks.TestEnv,
) []model.BundleEntry {
	var fileList []model.BundleEntry

	for i := 0; i < int(bundleEntriesFileCount); i++ {
		bundleEntriesReader, err := bundle.MetaStore().Get(context.Background(),
			model.GetArchivePathToBundleFileList(ev.Repo, bundle.BundleID, uint64(i)))
		require.NoError(t, err)
		var bundleEntries model.BundleEntries

		bundleEntriesBuffer, err := ioutil.ReadAll(bundleEntriesReader)
		require.NoError(t, err)
		require.NoError(t, yaml.Unmarshal(bundleEntriesBuffer, &bundleEntries))
		fileList = append(fileList, bundleEntries.BundleEntries...)
	}

	// verify that the file count is correct
	_, err := bundle.MetaStore().Get(context.Background(),
		model.GetArchivePathToBundleFileList(ev.Repo, bundle.BundleID, bundleEntriesFileCount))
	require.NotNil(t, err)
	return fileList
}

func TestBundle_StoreGet(t *testing.T) {
	ev := testBundleEnv()

	metaStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.MetaDir))
	vmetaStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.VmetaDir))
	blobStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.BlobDir))
	walStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.Wal))
	readLog := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.ReadLog))

	dmc := context2.NewStores(walStore, readLog, blobStore, metaStore, vmetaStore)
	b := Bundle{
		contextStores: dmc,
	}
	require.Equal(t, metaStore, b.MetaStore())
	require.Equal(t, vmetaStore, b.VMetaStore())
	require.Equal(t, blobStore, b.BlobStore())
	require.Equal(t, walStore, b.WALStore())
	require.Equal(t, readLog, b.ReadLogStore())
}
