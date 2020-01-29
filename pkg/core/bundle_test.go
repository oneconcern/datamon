package core

import (
	"bytes"
	"context"
	"math"
	"path/filepath"
	"strconv"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"go.uber.org/zap"

	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/segmentio/ksuid"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"gopkg.in/yaml.v2"

	"io/ioutil"
	"os"
	"testing"
	"time"
)

type testEnv struct {
	leafSize               uint32
	repo                   string
	bundleID               string
	testRoot               string
	sourceDir              string
	blobDir                string
	metaDir                string
	vmetaDir               string
	wal                    string
	readLog                string
	destinationDir         string
	reArchiveMetaDir       string
	reArchiveBlobDir       string
	reBundleEntriesPerFile int
	original               string
	dataDir                string
	// nolint: unused,structcheck
	pathToMount string
	// nolint: unused,structcheck
	pathToStaging string
}

func testBundleEnv() testEnv {
	// test parameters common to tests in bundle_test.go
	testRoot := filepath.FromSlash("../../testdata/core")
	sourceDir := filepath.Join(testRoot, "bundle", "source")
	return testEnv{
		leafSize:               cafs.DefaultLeafSize,
		repo:                   "bundle-test-repo",
		bundleID:               "bundle123",
		testRoot:               testRoot,
		sourceDir:              sourceDir,
		blobDir:                filepath.Join(sourceDir, "blob"),
		metaDir:                filepath.Join(sourceDir, "meta"),
		vmetaDir:               filepath.Join(sourceDir, "vmeta"),
		wal:                    filepath.Join(sourceDir, "wal"),
		readLog:                filepath.Join(sourceDir, "readLog"),
		destinationDir:         filepath.Join(testRoot, "bundle", "destination"),
		reArchiveMetaDir:       filepath.Join(testRoot, "bundle", "destination2", "meta"),
		reArchiveBlobDir:       filepath.Join(testRoot, "bundle", "destination2", "blob"),
		reBundleEntriesPerFile: 3,
		original:               filepath.Join(testRoot, "internal"),
		dataDir:                "dir",
	}
}

var (
	timeStamp *time.Time
)

func fakeContext(meta, blob string) context2.Stores {
	var (
		metaStore, blobStore storage.Store
	)
	if meta != "" {
		metaStore = localfs.New(afero.NewBasePathFs(afero.NewOsFs(), meta))
	}
	if blob != "" {
		blobStore = localfs.New(afero.NewBasePathFs(afero.NewOsFs(), blob))
	}
	ctx := context2.New()
	ctx.SetMetadata(metaStore)
	ctx.SetBlob(blobStore)
	return ctx
}

func fakeBundle(ev testEnv) *Bundle {
	bd := NewBDescriptor()
	return NewBundle(bd,
		Repo(ev.repo),
		BundleID(ev.bundleID),
		ContextStores(fakeContext(ev.metaDir, ev.blobDir)),
		ConsumableStore(localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.destinationDir))),
		Logger(testLogger()),
	)
}

func fakeRepoDescriptor(name string) model.RepoDescriptor {
	return model.RepoDescriptor{
		Name:        name,
		Description: "test",
		Timestamp:   time.Time{},
		Contributor: model.Contributor{
			Name:  "test",
			Email: "t@test.com",
		},
	}
}

func TestBundle(t *testing.T) {
	var (
		bundleEntriesFileCount uint64 = 2
		dataFilesCount         uint64 = 4
	)

	ev := testBundleEnv()
	cleanup := setupFakeDataBundle(t, bundleEntriesFileCount, dataFilesCount, ev)
	defer cleanup()

	// create repo in main context
	require.NoError(t, CreateRepo(fakeRepoDescriptor(ev.repo), fakeContext(ev.metaDir, "")))

	// create repo in archive context
	require.NoError(t, CreateRepo(fakeRepoDescriptor(ev.repo), fakeContext(ev.reArchiveMetaDir, "")))

	bundle := fakeBundle(ev)

	// Publish the bundle and compare with original
	require.NoError(t,
		implPublish(context.Background(), bundle, uint(dataFilesCount),
			func(s string) (bool, error) { return true, nil }))

	validatePublish(t, bundle.ConsumableStore, bundleEntriesFileCount, ev)

	// bundle id is set on upload
	bd := NewBDescriptor()
	consumableStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.destinationDir))
	archiveBundle2 := NewBundle(bd,
		Repo(ev.repo),
		ConsumableStore(consumableStore),
		ContextStores(fakeContext(ev.reArchiveMetaDir, ev.reArchiveBlobDir)),
	)
	require.NoError(t,
		implUpload(context.Background(), archiveBundle2, uint(ev.reBundleEntriesPerFile), nil))

	require.True(t, validateUpload(t, bundle, archiveBundle2, bundleEntriesFileCount, dataFilesCount, ev))
}

func paramedTestPublishMetadata(t *testing.T, publish bool, ev testEnv) {
	var (
		bundleEntriesFileCount uint64 = 3
		dataFilesCount         uint64 = 4
	)

	cleanup := setupFakeDataBundleWithUnalignedFilelist(t,
		bundleEntriesFileCount, dataFilesCount, dataFilesCount/2, ev)
	defer cleanup()

	require.NoError(t, CreateRepo(fakeRepoDescriptor(ev.repo), fakeContext(ev.metaDir, "")))

	consumableStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.destinationDir))
	bd := NewBDescriptor()
	bundle := NewBundle(bd,
		Repo(ev.repo),
		BundleID(ev.bundleID),
		ConsumableStore(consumableStore),
		ContextStores(fakeContext(ev.metaDir, ev.blobDir)),
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

func validatePublishMetadata(t *testing.T, bundle *Bundle, publish bool, ev testEnv) {
	var i uint64

	consumableStore := bundle.ConsumableStore
	metaStore := bundle.MetaStore()

	readBundleEntries := func(store storage.Store,
		idx2Filelist func(uint64) string,
		pathToBundleDescriptor string,
	) []model.BundleEntry {
		bundleDescriptor := readBundleDescriptor(t, store, pathToBundleDescriptor)
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
		func(i uint64) string { return model.GetArchivePathToBundleFileList(ev.repo, ev.bundleID, i) },
		model.GetArchivePathToBundle(ev.repo, ev.bundleID))
	if publish {
		consumableBundleEntries := readBundleEntries(consumableStore,
			func(i uint64) string {
				return filepath.Join(".datamon", ev.bundleID+"-bundle-files-"+strconv.Itoa(int(i))+".yaml")
			},
			model.GetConsumablePathToBundle(ev.bundleID))
		compareBundleEntriesLists(consumableBundleEntries, metaBundleEntries)
		compareBundleEntriesLists(metaBundleEntries, bundle.BundleEntries)
	} else {
		compareBundleEntriesLists(metaBundleEntries, bundle.BundleEntries)
	}

	t.Logf("tot bundle entries in memory %v", len(bundle.BundleEntries))
}

func validatePublish(t *testing.T, store storage.Store,
	bundleEntriesFileCount uint64,
	ev testEnv,
) {
	// Check Bundle File
	bundleDescriptor := readBundleDescriptor(t, store, model.GetConsumablePathToBundle(ev.bundleID))
	require.Equal(t, bundleDescriptor, generateBundleDescriptor(bundleEntriesFileCount, ev))
	// Check Files
	validateDataFiles(t, ev.original, filepath.Join(ev.destinationDir, ev.dataDir))
}

func validateUpload(t *testing.T,
	origBundle *Bundle, uploadedBundle *Bundle,
	bundleEntriesFileCount uint64,
	dataFilesCount uint64,
	ev testEnv,
) bool {
	// Check if the blobs are the same as download
	require.True(t, validateDataFiles(t, ev.blobDir, ev.reArchiveBlobDir))
	// Check the file list contents
	origFileList := readBundleFilelist(t, origBundle, bundleEntriesFileCount, ev)
	reEntryFilesCount := uint64(math.Ceil(float64(bundleEntriesFileCount*dataFilesCount) / float64(ev.reBundleEntriesPerFile)))
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
	bundleDescriptor := readBundleDescriptor(t, uploadedBundle.MetaStore(), pathToBundleDescriptor)
	expectedBundleDescriptor := generateBundleDescriptor(bundleEntriesFileCount, ev)
	require.Equal(t, bundleDescriptor.Parents, expectedBundleDescriptor.Parents)
	require.Equal(t, bundleDescriptor.LeafSize, expectedBundleDescriptor.LeafSize)
	// ??? unchecked values on on BundleDescriptor.  what values to test at this level of abstraction?
	return true
}

func readBundleFilelist(t *testing.T,
	bundle *Bundle,
	bundleEntriesFileCount uint64,
	ev testEnv,
) []model.BundleEntry {
	var fileList []model.BundleEntry
	for i := 0; i < int(bundleEntriesFileCount); i++ {
		bundleEntriesReader, err := bundle.MetaStore().Get(context.Background(),
			model.GetArchivePathToBundleFileList(ev.repo, bundle.BundleID, uint64(i)))
		require.NoError(t, err)
		var bundleEntries model.BundleEntries
		bundleEntriesBuffer, err := ioutil.ReadAll(bundleEntriesReader)
		require.NoError(t, err)
		require.NoError(t, yaml.Unmarshal(bundleEntriesBuffer, &bundleEntries))
		fileList = append(fileList, bundleEntries.BundleEntries...)
	}
	// verify that the file count is correct
	_, err := bundle.MetaStore().Get(context.Background(),
		model.GetArchivePathToBundleFileList(ev.repo, bundle.BundleID, bundleEntriesFileCount))
	require.NotNil(t, err)
	return fileList
}

func readBundleDescriptor(t *testing.T,
	store storage.Store,
	pathToBundle string) model.BundleDescriptor {
	var bundleDescriptor model.BundleDescriptor
	reader, err := store.Get(context.Background(), pathToBundle)
	require.NoError(t, err)
	bundleDescriptorBuffer, err := ioutil.ReadAll(reader)
	require.NoError(t, err)
	err = yaml.Unmarshal(bundleDescriptorBuffer, &bundleDescriptor)
	require.NoError(t, err)
	return bundleDescriptor
}

func getTimeStamp() *time.Time {
	// TODO: should be done with once
	if timeStamp == nil {
		t := model.GetBundleTimeStamp()
		timeStamp = &t
		return timeStamp
	}
	return timeStamp
}

func generateDataFile(test *testing.T, store storage.Store, ev testEnv) model.BundleEntry {
	// Generate data files with faked data to compare post publish, write to internal folder
	ksuid, err := ksuid.NewRandom()
	require.NoError(test, err)
	var size = 2 * int(ev.leafSize)
	file := filepath.Join(ev.original, ksuid.String())
	require.NoError(test, cafs.GenerateFile(file, size, ev.leafSize))
	// Write individual blobs
	fs, err := cafs.New(
		cafs.LeafSize(ev.leafSize),
		cafs.Backend(store),
		cafs.Logger(testLogger()),
	)
	require.NoError(test, err)
	keys, err := cafs.GenerateCAFSChunks(file, fs)
	require.NoError(test, err)
	// return the Bundle Entry
	return model.BundleEntry{
		Hash:         keys.String(),
		NameWithPath: filepath.Join(ev.dataDir, ksuid.String()),
		FileMode:     0700,
		Size:         uint64(size),
	}
}

func setupFakeDataBundle(t *testing.T,
	bundleEntriesFileCount uint64, // number of index files to track file entries
	dataFilesCount uint64, // number of files in one index file
	ev testEnv,
) func() {
	t.Logf("creating mock data as datamon bundle %q: %d files (entries), count: %d",
		ev.bundleID, bundleEntriesFileCount, dataFilesCount)
	return setupFakeDataBundleWithUnalignedFilelist(t,
		bundleEntriesFileCount, dataFilesCount, dataFilesCount, ev)
}

func setupFakeDataBundleWithUnalignedFilelist(t *testing.T,
	bundleEntriesFileCount uint64,
	dataFilesCount uint64,
	lastDataFileCount uint64,
	ev testEnv,
) func() {
	require.True(t, lastDataFileCount <= dataFilesCount,
		"last data file contains no more entries than other data files")
	require.True(t, 0 < lastDataFileCount, "last data file contains some entries")
	cleanup := func() {
		require.NoError(t, os.RemoveAll(ev.testRoot))
	}
	cleanup()

	// mock storage on local FS
	blobStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.blobDir))
	metaStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.metaDir))
	require.NoError(t, os.MkdirAll(ev.original, 0700))
	var i, j uint64

	for i = 0; i < bundleEntriesFileCount; i++ {
		bundleFileList := model.BundleEntries{BundleEntries: make([]model.BundleEntry, 0)}
		var currDataFilesCount uint64
		if i == bundleEntriesFileCount-1 {
			currDataFilesCount = lastDataFileCount
		} else {
			currDataFilesCount = dataFilesCount
		}
		for j = 0; j < currDataFilesCount; j++ {
			bundleEntry := generateDataFile(t, blobStore, ev)
			t.Logf("generating data file entry: %s", bundleEntry.NameWithPath)
			bundleFileList.BundleEntries = append(bundleFileList.BundleEntries, bundleEntry)
		}
		buffer, err := yaml.Marshal(bundleFileList)
		require.NoError(t, err)
		destinationPath := model.GetArchivePathToBundleFileList(ev.repo, ev.bundleID, i)
		t.Logf("put file list metadata on store: %s", destinationPath)
		require.NoError(t, metaStore.Put(context.Background(),
			destinationPath,
			bytes.NewReader(buffer),
			storage.NoOverWrite,
		))
	}
	bundleDescriptor := generateBundleDescriptor(bundleEntriesFileCount, ev)
	buffer, err := yaml.Marshal(bundleDescriptor)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(ev.destinationDir, 0700))
	t.Logf("put file bundle metadata on store: %s", model.GetArchivePathToBundle(ev.repo, ev.bundleID))
	require.NoError(t, metaStore.Put(context.Background(),
		model.GetArchivePathToBundle(ev.repo, ev.bundleID),
		bytes.NewReader(buffer),
		storage.NoOverWrite,
	))
	return cleanup
}

func generateBundleDescriptor(bundleEntriesFileCount uint64, ev testEnv) model.BundleDescriptor {
	// Generate Bundle
	return model.BundleDescriptor{
		ID:                     ev.bundleID,
		LeafSize:               ev.leafSize,
		Message:                "test bundle",
		Timestamp:              *getTimeStamp(),
		Contributors:           []model.Contributor{{Name: "dev", Email: "dev@dev.com"}},
		BundleEntriesFileCount: bundleEntriesFileCount,
	}
}

func validateDataFiles(t *testing.T, expectedDir string, actualDir string) bool {
	// TODO: make this a general purpose diff 2 folders or reuse a package
	fileListExpected, err := ioutil.ReadDir(expectedDir)
	require.NoError(t, err)
	fileListActual, err := ioutil.ReadDir(actualDir)
	require.NoError(t, err)
	require.Equal(t, len(fileListExpected), len(fileListActual))
	for _, fileExpected := range fileListExpected {
		found := false
		for _, fileActual := range fileListActual {
			if fileExpected.Name() == fileActual.Name() {
				found = true
				require.Equal(t, fileExpected.Size(), fileActual.Size())
				// TODO: Issue #35
				// require.Equal(t, fileExpected.Mode(), fileActual.Mode())
			}
		}
		require.True(t, found)
	}
	return true
}

func TestBundle_StoreGet(t *testing.T) {
	ev := testBundleEnv()
	metaStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.metaDir))
	vmetaStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.vmetaDir))
	blobStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.blobDir))
	walStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.wal))
	readLog := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.readLog))
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

func testLogger() *zap.Logger {
	if os.Getenv("DEBUG_TEST") != "" {
		l, _ := zap.NewDevelopment() // to get DEBUG  during test run
		return l
	}
	return zap.NewNop() // to limit test output
}
