package core

import (
	"bytes"
	"context"

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
	"reflect"
	"testing"
	"time"
)

const (
	leafSize         = cafs.DefaultLeafSize
	entryFilesCount  = 2
	dataFilesCount   = 4
	repo             = "bundle_test_repo"
	bundleID         = "bundle123"
	testRoot         = "../../testdata/core"
	sourceDir        = "../../testdata/core/bundle/source/"
	blobDir          = "../../testdata/core/bundle/source/blob"
	destinationDir   = "../../testdata/core/bundle/destination/"
	reArchiveDir     = "../../testdata/core/bundle/destination2/"
	reArchiveBlobDir = "../../testdata/core/bundle/destination2/blob"
	internalDir      = "../../testdata/core/internal/"
	dataDir          = "dir/"
)

var (
	timeStamp *time.Time
)

func TestBundle(t *testing.T) {

	require.NoError(t, Setup(t))

	consumableStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), destinationDir))
	archiveStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), sourceDir))
	blobStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), blobDir))

	bundle := NewBundle(repo, bundleID, archiveStore, consumableStore, blobStore)

	require.NoError(t,
		Publish(context.Background(), bundle))

	require.NoError(t, validatePublish(t, consumableStore))

	reArchive := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), reArchiveDir))

	b := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), reArchiveBlobDir))
	archiveBundle2 := NewBundle(repo, "", reArchive, consumableStore, b)

	require.NoError(t,
		Upload(context.Background(), archiveBundle2))

	require.True(t, validateUpload(t))
}

func validatePublish(t *testing.T, store storage.Store) error {
	// Check Bundle File
	reader, err := store.Get(context.Background(), model.GetConsumablePathToBundle(bundleID))
	require.NoError(t, err)

	bundleDescriptorBuffer, err := ioutil.ReadAll(reader)
	require.NoError(t, err)

	var bundleDescriptor model.BundleDescriptor
	err = yaml.Unmarshal(bundleDescriptorBuffer, &bundleDescriptor)
	require.NoError(t, err)

	require.True(t, validateBundleDescriptor(bundleDescriptor))

	// Check Files
	validateDataFiles(t, internalDir, destinationDir+dataDir)
	return nil
}

func validateUpload(t *testing.T) bool {
	// Check if the blobs are the same as download
	require.True(t, validateDataFiles(t, blobDir, reArchiveBlobDir))
	// Check if the bundle json and file list are what is expected.
	return true
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
	require.NoError(test, cafs.GenerateFile(internalDir+ksuid.String(), size, leafSize))
	// Write individual blobs
	fs, err := cafs.New(
		cafs.LeafSize(leafSize),
		cafs.Backend(store),
	)
	require.NoError(test, err)
	keys, err := cafs.GenerateCAFSChunks(internalDir+ksuid.String(), fs)
	require.NoError(test, err)
	// return the Bundle Entry
	return model.BundleEntry{
		Hash:         keys.String(),
		NameWithPath: dataDir + ksuid.String(),
		FileMode:     0700,
		Size:         uint64(size),
	}
}

func Setup(t *testing.T) error {
	cleanup()

	sourceStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), sourceDir))
	blobStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), blobDir))
	require.NoError(t, os.MkdirAll(internalDir, 0700))
	var i, j uint64

	for i = 0; i < entryFilesCount; i++ {

		bundleEntry := generateDataFile(t, blobStore)

		bundleFileList := model.BundleEntries{BundleEntries: []model.BundleEntry{bundleEntry}}

		for j = 0; j < (dataFilesCount - 1); j++ {
			bundleEntry = generateDataFile(t, blobStore)
			bundleFileList.BundleEntries = append(bundleFileList.BundleEntries, bundleEntry)
		}

		buffer, err := yaml.Marshal(bundleFileList)
		require.NoError(t, err)
		destinationPath := model.GetArchivePathToBundleFileList(repo, bundleID, i)
		require.NoError(t,
			sourceStore.Put(context.Background(), destinationPath, bytes.NewReader(buffer)))
	}

	bundleDescriptor := generateBundleDescriptor()

	buffer, err := yaml.Marshal(bundleDescriptor)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(destinationDir, 0700))

	return sourceStore.Put(context.Background(), model.GetArchivePathToBundle(repo, bundleID), bytes.NewReader(buffer))
}

func generateBundleDescriptor() model.BundleDescriptor {
	// Generate Bundle
	return model.BundleDescriptor{
		ID:                     bundleID,
		LeafSize:               leafSize,
		Message:                "test bundle",
		Timestamp:              *getTimeStamp(),
		Contributors:           []model.Contributor{{Name: "dev", Email: "dev@dev.com"}},
		BundleEntriesFileCount: entryFilesCount,
	}
}

func validateBundleDescriptor(descriptor model.BundleDescriptor) bool {
	expectedBundle := generateBundleDescriptor()
	return reflect.DeepEqual(descriptor, expectedBundle)
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

func cleanup() {
	os.RemoveAll(testRoot)
}
