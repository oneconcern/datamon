package mocks

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/oneconcern/datamon/pkg/cafs"
	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/segmentio/ksuid"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// TestEnv describes a complete testing environment to mock
type TestEnv struct {
	// single bundle test cases
	BundleID               string
	LeafSize               uint32
	ReBundleEntriesPerFile int

	// test repo name
	Repo string

	TestRoot  string
	SourceDir string

	// context mocked locaions
	BlobDir  string
	MetaDir  string
	VmetaDir string
	Wal      string
	ReadLog  string

	DestinationDir   string
	Original         string
	DataDir          string
	ReArchiveMetaDir string
	ReArchiveBlobDir string

	// fs mount tests
	PathToMount   string
	PathToStaging string
}

var (
	onceTimestamp sync.Once
	timestamp     time.Time
)

// GetTestTimeStamp yields a unique timestamp for the duration of our tests
func GetTestTimeStamp() time.Time {
	onceTimestamp.Do(func() {
		timestamp = model.GetBundleTimeStamp()
	})
	t := timestamp
	return t
}

// TestLogger yields a zap logger for testing, essentially muting logs,
// in order to avoid too much output on CI. Activate DEBUG log when testing interactively
// by setting the DEBUG_TEST environment variable.
func TestLogger() *zap.Logger {
	if os.Getenv("DEBUG_TEST") != "" {
		l, _ := zap.NewDevelopment() // to get DEBUG  during test run
		return l
	}
	return zap.NewNop() // to limit test output
}

// FakeContext builds a minimal datamon context with blob and metastore.
func FakeContext(meta, blob string) context2.Stores {
	var metaStore, blobStore storage.Store

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

// FakeRepoDescriptor mocks a repo descriptor
func FakeRepoDescriptor(name string) model.RepoDescriptor {
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

// FakeBundleDescriptor mocks a BundleDescriptor
func FakeBundleDescriptor(bundleEntriesFileCount uint64, ev TestEnv) model.BundleDescriptor {
	return model.BundleDescriptor{
		ID:                     ev.BundleID,
		LeafSize:               ev.LeafSize,
		Message:                "test bundle",
		Timestamp:              GetTestTimeStamp(),
		Contributors:           []model.Contributor{{Name: "dev", Email: "dev@dev.com"}},
		BundleEntriesFileCount: bundleEntriesFileCount,
	}
}

// ValidatePublish validates a faked bundle published on a ConsumableStore
func ValidatePublish(t testing.TB, store storage.Store, bundleEntriesFileCount uint64, ev TestEnv) {
	// Check Bundle File
	bundleDescriptor := ReadBundleDescriptor(t, store, model.GetConsumablePathToBundle(ev.BundleID))
	require.Equal(t, bundleDescriptor, FakeBundleDescriptor(bundleEntriesFileCount, ev))

	// Check Files
	ValidateDataFiles(t, ev.Original, filepath.Join(ev.DestinationDir, ev.DataDir))
}

// ReadBundleDescriptor read a bundle descriptor from a metadata store.
//
// TODO: could be part of the core interface.
func ReadBundleDescriptor(t testing.TB, store storage.Store, pathToBundle string) model.BundleDescriptor {
	var bundleDescriptor model.BundleDescriptor

	reader, err := store.Get(context.Background(), pathToBundle)
	require.NoError(t, err)

	bundleDescriptorBuffer, err := ioutil.ReadAll(reader)
	require.NoError(t, err)

	err = yaml.Unmarshal(bundleDescriptorBuffer, &bundleDescriptor)
	require.NoError(t, err)

	return bundleDescriptor
}

// FakeDataFile generates some data files with faked data to compare post publish, write to internal folder
func FakeDataFile(t testing.TB, store storage.Store, ev TestEnv) model.BundleEntry {
	ksuid, err := ksuid.NewRandom()
	require.NoError(t, err)

	var size = 2 * int(ev.LeafSize)
	file := filepath.Join(ev.Original, ksuid.String())
	require.NoError(t, cafs.GenerateFile(file, size, ev.LeafSize))

	// Write individual blobs
	fs, err := cafs.New(
		cafs.LeafSize(ev.LeafSize),
		cafs.Backend(store),
		cafs.Logger(TestLogger()),
	)
	require.NoError(t, err)

	keys, err := cafs.GenerateCAFSChunks(file, fs)
	require.NoError(t, err)

	// return the Bundle Entry
	return model.BundleEntry{
		Hash:         keys.String(),
		NameWithPath: filepath.Join(ev.DataDir, ksuid.String()),
		FileMode:     0700,
		Size:         uint64(size),
	}
}

// ValidateDataFiles is a simplistic folder diff-er.
//
// TODO: make this a general purpose diff 2 folders or reuse a package
func ValidateDataFiles(t testing.TB, expectedDir, actualDir string) bool {
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

// SetupFakeDataBundle prepares some metadata for a fake bundle and returns a cleanup function
func SetupFakeDataBundle(t testing.TB,
	bundleEntriesFileCount uint64, // number of index files to track file entries
	dataFilesCount uint64, // number of files in one index file
	ev TestEnv,
) func() {
	t.Logf("creating mock data as datamon bundle %q: %d files (entries), count: %d",
		ev.BundleID, bundleEntriesFileCount, dataFilesCount)
	return SetupFakeDataBundleWithUnalignedFilelist(t, bundleEntriesFileCount, dataFilesCount, dataFilesCount, ev)
}

// SetupFakeDataBundleWithUnalignedFilelist ...
func SetupFakeDataBundleWithUnalignedFilelist(t testing.TB,
	bundleEntriesFileCount uint64,
	dataFilesCount uint64,
	lastDataFileCount uint64,
	ev TestEnv,
) func() {
	require.True(t, lastDataFileCount <= dataFilesCount, "last data file contains no more entries than other data files")

	require.True(t, 0 < lastDataFileCount, "last data file contains some entries")

	// mock storage on local FS
	blobStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.BlobDir))
	metaStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.MetaDir))

	require.NoError(t, os.MkdirAll(ev.Original, 0700))

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
			bundleEntry := FakeDataFile(t, blobStore, ev)
			t.Logf("generating data file entry: %s", bundleEntry.NameWithPath)
			bundleFileList.BundleEntries = append(bundleFileList.BundleEntries, bundleEntry)
		}

		buffer, err := yaml.Marshal(bundleFileList)
		require.NoError(t, err)

		destinationPath := model.GetArchivePathToBundleFileList(ev.Repo, ev.BundleID, i)
		t.Logf("put file list metadata on store: %s", destinationPath)
		require.NoError(t, metaStore.Put(context.Background(),
			destinationPath,
			bytes.NewReader(buffer),
			storage.NoOverWrite,
		))
	}

	bundleDescriptor := FakeBundleDescriptor(bundleEntriesFileCount, ev)
	buffer, err := yaml.Marshal(bundleDescriptor)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(ev.DestinationDir, 0700))

	t.Logf("put file bundle metadata on store: %s", model.GetArchivePathToBundle(ev.Repo, ev.BundleID))
	require.NoError(t, metaStore.Put(context.Background(),
		model.GetArchivePathToBundle(ev.Repo, ev.BundleID),
		bytes.NewReader(buffer),
		storage.NoOverWrite,
	))

	cleanup := func() {
		require.NoError(t, os.RemoveAll(ev.TestRoot))
	}
	return cleanup
}
