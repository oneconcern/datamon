package core_test

import (
	"bytes"
	"context"

	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/core"
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
	leafSize        = cafs.DefaultLeafSize
	entryFilesCount = 2
	dataFilesCount  = 4
	repo            = "bundle_test_repo"
	bundleID        = "bundle123"
	testRoot        = "../../testdata/core"
	sourceDir       = "../../testdata/core/bundle/source/"
	destinationDir  = "../../testdata/core/bundle/destination/"
	destinationDir2 = "../../testdata/core/bundle/destination2/"
	internalDir     = "../../testdata/core/internal/"
	dataDir         = "dir/"
)

var (
	timeStamp *time.Time
)

func TestBundle(t *testing.T) {
	cleanup()
	require.NoError(t, setup(t))
	destinationStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), destinationDir))
	sourceStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), sourceDir))
	archiveBundle := core.NewBundle(repo, bundleID, sourceStore)
	consumableBundle := core.NewBundle(repo, bundleID, destinationStore)
	require.NoError(t,
		core.Publish(context.Background(), archiveBundle, consumableBundle))
	require.NoError(t, validatePublish(t, destinationStore))
	destinationStore2 := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), destinationDir2))
	archiveBundle2 := core.NewBundle(repo, "", destinationStore2)
	require.NoError(t,
		core.Upload(context.Background(), *consumableBundle, archiveBundle2))
	require.True(t, validateUpload(t))
}

func validatePublish(t *testing.T, store storage.Store) error {
	// Check Bundle File
	reader, err := store.Get(context.Background(), model.GetConsumablePathToBundle(bundleID))
	require.NoError(t, err)

	bundleDescriptorBuffer, err := ioutil.ReadAll(reader)
	require.NoError(t, err)

	var bundleDescriptor model.Bundle
	err = yaml.Unmarshal(bundleDescriptorBuffer, &bundleDescriptor)
	require.NoError(t, err)

	require.True(t, validateBundleDescriptor(bundleDescriptor))

	// Check Files
	validateDataFiles(t, internalDir, destinationDir+dataDir)
	return nil
}

func validateUpload(t *testing.T) bool {
	// Check if the blobs are the same as download
	require.True(t, validateDataFiles(t, sourceDir+"/blobs/", destinationDir2+"/blobs/"))
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

func setup(t *testing.T) error {
	cleanup()

	sourceStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), sourceDir))
	blobStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), sourceDir+"blobs"))
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

func generateBundleDescriptor() model.Bundle {
	// Generate Bundle
	return model.Bundle{
		ID:                     bundleID,
		LeafSize:               leafSize,
		Message:                "test bundle",
		Timestamp:              *getTimeStamp(),
		Contributors:           []model.Contributor{{Name: "dev", Email: "dev@dev.com"}},
		BundleEntriesFileCount: entryFilesCount,
	}
}

func validateBundleDescriptor(descriptor model.Bundle) bool {
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
