package core

import (
	"bytes"
	"context"
	"math"
	"strconv"

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

const (
	leafSize               = cafs.DefaultLeafSize
	repo                   = "bundle-test-repo"
	bundleID               = "bundle123"
	testRoot               = "../../testdata/core"
	sourceDir              = "../../testdata/core/bundle/source/"
	blobDir                = sourceDir + "/blob"
	metaDir                = sourceDir + "/meta"
	destinationDir         = "../../testdata/core/bundle/destination/"
	reArchiveMetaDir       = "../../testdata/core/bundle/destination2/meta"
	reArchiveBlobDir       = "../../testdata/core/bundle/destination2/blob"
	reBundleEntriesPerFile = 3
	original               = "../../testdata/core/internal/"
	dataDir                = "dir/"
)

var (
	timeStamp *time.Time
)

func TestBundle(t *testing.T) {
	var bundleEntriesFileCount uint64 = 2
	var dataFilesCount uint64 = 4
	cleanup := setupFakeDataBundle(t, bundleEntriesFileCount, dataFilesCount)
	defer cleanup()
	consumableStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), destinationDir))
	metaStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), metaDir))
	blobStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), blobDir))
	reArchiveBlob := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), reArchiveBlobDir))
	reArchive := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), reArchiveMetaDir))
	require.NoError(t, CreateRepo(model.RepoDescriptor{
		Name:        repo,
		Description: "test",
		Timestamp:   time.Time{},
		Contributor: model.Contributor{
			Name:  "test",
			Email: "t@test.com",
		},
	}, metaStore))
	require.NoError(t, CreateRepo(model.RepoDescriptor{
		Name:        repo,
		Description: "test",
		Timestamp:   time.Time{},
		Contributor: model.Contributor{
			Name:  "test",
			Email: "t@test.com",
		},
	}, reArchive))

	bd := NewBDescriptor()
	bundle := New(bd,
		Repo(repo),
		BundleID(bundleID),
		MetaStore(metaStore),
		ConsumableStore(consumableStore),
		BlobStore(blobStore),
	)

	// Publish the bundle and compare with original
	require.NoError(t,
		implPublish(context.Background(), bundle, uint(dataFilesCount),
			func(s string) (bool, error) { return true, nil }))

	validatePublish(t, consumableStore, bundleEntriesFileCount)

	/* bundle id is set on upload */
	archiveBundle2 := New(bd,
		Repo(repo),
		MetaStore(reArchive),
		ConsumableStore(consumableStore),
		BlobStore(reArchiveBlob),
	)
	require.NoError(t,
		implUpload(context.Background(), archiveBundle2, reBundleEntriesPerFile, nil))

	require.True(t, validateUpload(t, bundle, archiveBundle2, bundleEntriesFileCount, dataFilesCount))
}

func paramedTestPublishMetadata(t *testing.T, publish bool) {
	var bundleEntriesFileCount uint64 = 3
	var dataFilesCount uint64 = 4
	cleanup := setupFakeDataBundleWithUnalignedFilelist(t,
		bundleEntriesFileCount, dataFilesCount, dataFilesCount/2)
	defer cleanup()
	consumableStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), destinationDir))
	metaStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), metaDir))
	blobStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), blobDir))
	require.NoError(t, CreateRepo(model.RepoDescriptor{
		Name:        repo,
		Description: "test",
		Timestamp:   time.Time{},
		Contributor: model.Contributor{
			Name:  "test",
			Email: "t@test.com",
		},
	}, metaStore))

	bd := NewBDescriptor()
	bundle := New(bd,
		Repo(repo),
		BundleID(bundleID),
		MetaStore(metaStore),
		ConsumableStore(consumableStore),
		BlobStore(blobStore),
	)

	// Publish the bundle and compare with original
	require.NoError(t,
		implPublishMetadata(context.Background(), bundle, publish, uint(dataFilesCount)))

	validatePublishMetadata(t, bundle, publish)
}

func TestPublishMetadata(t *testing.T) {
	paramedTestPublishMetadata(t, true)
}

func TestDownloadMetadata(t *testing.T) {
	paramedTestPublishMetadata(t, false)
}

func validatePublishMetadata(t *testing.T, bundle *Bundle, publish bool) {
	var i uint64

	consumableStore := bundle.ConsumableStore
	metaStore := bundle.MetaStore

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
		func(i uint64) string { return model.GetArchivePathToBundleFileList(repo, bundleID, i) },
		model.GetArchivePathToBundle(repo, bundleID))
	if publish {
		consumableBundleEntries := readBundleEntries(consumableStore,
			func(i uint64) string {
				return ".datamon/" + bundleID + "-bundle-files-" + strconv.Itoa(int(i)) + ".json"
			},
			model.GetConsumablePathToBundle(bundleID))
		compareBundleEntriesLists(consumableBundleEntries, metaBundleEntries)
		compareBundleEntriesLists(metaBundleEntries, bundle.BundleEntries)
	} else {
		compareBundleEntriesLists(metaBundleEntries, bundle.BundleEntries)
	}

	t.Logf("tot bundle entries in memory %v", len(bundle.BundleEntries))
}

func validatePublish(t *testing.T, store storage.Store,
	bundleEntriesFileCount uint64,
) {
	// Check Bundle File
	bundleDescriptor := readBundleDescriptor(t, store, model.GetConsumablePathToBundle(bundleID))
	require.Equal(t, bundleDescriptor, generateBundleDescriptor(bundleEntriesFileCount))
	// Check Files
	validateDataFiles(t, original, destinationDir+dataDir)
}

func validateUpload(t *testing.T,
	origBundle *Bundle, uploadedBundle *Bundle,
	bundleEntriesFileCount uint64,
	dataFilesCount uint64,
) bool {
	// Check if the blobs are the same as download
	require.True(t, validateDataFiles(t, blobDir, reArchiveBlobDir))
	// Check the file list contents
	origFileList := readBundleFilelist(t, origBundle, bundleEntriesFileCount)
	reEntryFilesCount := uint64(math.Ceil(float64(bundleEntriesFileCount*dataFilesCount) / float64(reBundleEntriesPerFile)))
	reFileList := readBundleFilelist(t, uploadedBundle, reEntryFilesCount)
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
	bundleDescriptor := readBundleDescriptor(t, uploadedBundle.MetaStore, pathToBundleDescriptor)
	expectedBundleDescriptor := generateBundleDescriptor(bundleEntriesFileCount)
	require.Equal(t, bundleDescriptor.Parents, expectedBundleDescriptor.Parents)
	require.Equal(t, bundleDescriptor.LeafSize, expectedBundleDescriptor.LeafSize)
	// ??? unchecked values on on BundleDescriptor.  what values to test at this level of abstraction?
	return true
}

func readBundleFilelist(t *testing.T,
	bundle *Bundle,
	bundleEntriesFileCount uint64,
) []model.BundleEntry {
	var fileList []model.BundleEntry
	for i := 0; i < int(bundleEntriesFileCount); i++ {
		bundleEntriesReader, err := bundle.MetaStore.Get(context.Background(),
			model.GetArchivePathToBundleFileList(repo, bundle.BundleID, uint64(i)))
		require.NoError(t, err)
		var bundleEntries model.BundleEntries
		bundleEntriesBuffer, err := ioutil.ReadAll(bundleEntriesReader)
		require.NoError(t, err)
		require.NoError(t, yaml.Unmarshal(bundleEntriesBuffer, &bundleEntries))
		fileList = append(fileList, bundleEntries.BundleEntries...)
	}
	// verify that the file count is correct
	_, err := bundle.MetaStore.Get(context.Background(),
		model.GetArchivePathToBundleFileList(repo, bundle.BundleID, bundleEntriesFileCount))
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
	if timeStamp == nil {
		t := model.GetBundleTimeStamp()
		timeStamp = &t
		return timeStamp
	}
	return timeStamp
}

func generateDataFile(test *testing.T, store storage.Store) model.BundleEntry {
	// Generate data files to compare post publish, write to internal folder
	ksuid, err := ksuid.NewRandom()
	require.NoError(test, err)
	var size int = 2 * leafSize
	file := original + ksuid.String()
	require.NoError(test, cafs.GenerateFile(file, size, leafSize))
	// Write individual blobs
	fs, err := cafs.New(
		cafs.LeafSize(leafSize),
		cafs.Backend(store),
	)
	require.NoError(test, err)
	keys, err := cafs.GenerateCAFSChunks(file, fs)
	require.NoError(test, err)
	// return the Bundle Entry
	return model.BundleEntry{
		Hash:         keys.String(),
		NameWithPath: dataDir + ksuid.String(),
		FileMode:     0700,
		Size:         uint64(size),
	}
}

func setupFakeDataBundle(t *testing.T,
	bundleEntriesFileCount uint64,
	dataFilesCount uint64,
) func() {
	return setupFakeDataBundleWithUnalignedFilelist(t,
		bundleEntriesFileCount, dataFilesCount, dataFilesCount)
}

func setupFakeDataBundleWithUnalignedFilelist(t *testing.T,
	bundleEntriesFileCount uint64,
	dataFilesCount uint64,
	lastDataFileCount uint64,
) func() {
	require.True(t, lastDataFileCount <= dataFilesCount,
		"last data file contains no more entries than other data files")
	require.True(t, 0 < lastDataFileCount, "last data file contains some entries")
	cleanup := func() {
		require.NoError(t, os.RemoveAll(testRoot))
	}
	cleanup()
	blobStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), blobDir))
	metaStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), metaDir))
	require.NoError(t, os.MkdirAll(original, 0700))
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
			bundleEntry := generateDataFile(t, blobStore)
			bundleFileList.BundleEntries = append(bundleFileList.BundleEntries, bundleEntry)
		}
		buffer, err := yaml.Marshal(bundleFileList)
		require.NoError(t, err)
		destinationPath := model.GetArchivePathToBundleFileList(repo, bundleID, i)
		require.NoError(t, metaStore.Put(context.Background(),
			destinationPath,
			bytes.NewReader(buffer),
			storage.IfNotPresent,
		))
	}
	bundleDescriptor := generateBundleDescriptor(bundleEntriesFileCount)
	buffer, err := yaml.Marshal(bundleDescriptor)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(destinationDir, 0700))
	require.NoError(t, metaStore.Put(context.Background(),
		model.GetArchivePathToBundle(repo, bundleID),
		bytes.NewReader(buffer),
		storage.IfNotPresent,
	))
	return cleanup
}

func generateBundleDescriptor(bundleEntriesFileCount uint64) model.BundleDescriptor {
	// Generate Bundle
	return model.BundleDescriptor{
		ID:                     bundleID,
		LeafSize:               leafSize,
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
