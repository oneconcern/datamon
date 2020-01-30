// +build fsintegration

package fuse

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/spf13/afero"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/core/mocks"
	fusemocks "github.com/oneconcern/datamon/pkg/fuse/mocks"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
)

const (
	testOnTempFS = true                 // for very large test cases, avoid tempFS
	throttleIOs  = 1 * time.Millisecond // on CI, we experiment hangs on RO mount when I/O parallel workload is too high
)

func testFsIntegEnv() (mocks.TestEnv, func(t testing.TB) func()) {
	var base string
	if !testOnTempFS {
		base = "."
	}
	tmp := stringorDie(ioutil.TempDir(base, "test-integration-"))

	// builds a temporary testing environment

	// mounting
	mountPath := stringorDie(ioutil.TempDir(tmp, "mount-")) // fuse mount
	stagingPath := stringorDie(ioutil.TempDir(tmp, "staging-"))

	// data source and download destination
	testRoot := stringorDie(ioutil.TempDir(tmp, "core-data-"))
	sourceDir := filepath.Join(testRoot, "bundle", "source")
	destinationDir := filepath.Join(testRoot, "bundle", "destination")
	originalDir := filepath.Join(testRoot, "internal")

	// context
	blobDir := filepath.Join(sourceDir, "blob")
	metaDir := filepath.Join(sourceDir, "meta")
	vmetaDir := filepath.Join(sourceDir, "vmeta")
	wal := filepath.Join(sourceDir, "wal")
	readLog := filepath.Join(sourceDir, "readLog")

	for _, dir := range []string{
		sourceDir, destinationDir, originalDir,
		blobDir, metaDir, vmetaDir, wal, readLog,
	} {
		err := os.MkdirAll(dir, 0700)
		if err != nil {
			panic(fmt.Errorf("could not create test environment dir %s: %v", dir, err))
		}
	}

	return mocks.TestEnv{
			LeafSize:               cafs.DefaultLeafSize,
			Repo:                   "bundle-mount-test-repo",
			BundleID:               "bundle456",
			TestRoot:               testRoot,
			SourceDir:              sourceDir,
			BlobDir:                blobDir,
			MetaDir:                metaDir,
			VmetaDir:               vmetaDir,
			Wal:                    wal,
			ReadLog:                readLog,
			DestinationDir:         destinationDir,
			ReBundleEntriesPerFile: 3,
			Original:               originalDir,
			DataDir:                "dir",
			PathToMount:            mountPath,
			PathToStaging:          stagingPath,
		}, func(t testing.TB) func() {
			return func() {
				t.Logf("unwinding integration test environment")
				_ = os.RemoveAll(tmp)
			}
		}
}

func TestStreamingRoMount(t *testing.T) {
	// mount a fuse fs from a bundle, with streaming enabled, verify the content of that mount
	// then randomly perform various syscalls on these files
	var (
		// 12 files in this bundle, stored in 1 index file
		bundleEntriesFileCount uint64 = 1
		dataFilesCount         uint64 = 12
	)

	ev, cleanTempDir := testFsIntegEnv()
	defer cleanTempDir(t)()

	t.Log("preparing fake bundle")
	_ = mocks.SetupFakeDataBundle(t, bundleEntriesFileCount, dataFilesCount, ev)

	t.Logf("preparing RO mount on %s", ev.PathToMount)
	bundle := fusemocks.FakeBundle(ev)
	fs, err := NewReadOnlyFS(bundle, Streaming(true))
	require.NoError(t, err)

	err = fs.MountReadOnly(ev.PathToMount)
	require.NoError(t, err)

	defer func() {
		fs.fsInternal.l.Info("unmounting RO mount")
		t.Log("unmounting RO mount")
		require.NoError(t, fs.Unmount(ev.PathToMount))
	}()

	fs.fsInternal.l.Info("verifying data files")
	t.Log("verifying data files")
	mocks.ValidateDataFiles(t, ev.Original, filepath.Join(ev.PathToMount, ev.DataDir))

	fs.fsInternal.l.Info("exercising fs syscalls")
	t.Log("exercising fs syscalls")
	testFSOperations(t, ev, fsROActions)
}

func TestNoStreamingRoMount(t *testing.T) {
	// mount a fuse fs from a bundle, with streaming enabled, verify the content of that mount
	// then randomly perform various syscalls on these files
	var (
		// 12 files in this bundle, stored in 1 index file
		bundleEntriesFileCount uint64 = 1
		dataFilesCount         uint64 = 12
	)

	ev, cleanTempDir := testFsIntegEnv()
	defer cleanTempDir(t)()

	t.Log("preparing fake bundle")
	_ = mocks.SetupFakeDataBundle(t, bundleEntriesFileCount, dataFilesCount, ev)

	t.Logf("preparing RO mount on %s", ev.PathToMount)
	bundle := fusemocks.FakeBundle(ev)
	fs, err := NewReadOnlyFS(bundle, Streaming(false))
	require.NoError(t, err)

	err = fs.MountReadOnly(ev.PathToMount)
	require.NoError(t, err)

	defer func() {
		fs.fsInternal.l.Info("unmounting RO mount")
		t.Log("unmounting RO mount")
		require.NoError(t, fs.Unmount(ev.PathToMount))
	}()

	fs.fsInternal.l.Info("verifying data files")
	t.Log("verifying data files")
	mocks.ValidateDataFiles(t, ev.Original, filepath.Join(ev.PathToMount, ev.DataDir))
	mocks.ValidateDataFiles(t, filepath.Join(ev.DestinationDir, ev.DataDir), filepath.Join(ev.PathToMount, ev.DataDir))

	fs.fsInternal.l.Info("exercising fs syscalls")
	t.Log("exercising fs syscalls")
	testFSOperations(t, ev, fsROActions)
}

func TestMutableMount(t *testing.T) {
	// smoke test on mutable mount: just fuse mount an initial empty bundle, then write a file

	ev, cleanTempDir := testFsIntegEnv()
	defer cleanTempDir(t)()

	bundle := fusemocks.EmptyBundle(ev)

	fs, err := NewMutableFS(bundle)
	require.NoError(t, err)

	err = fs.MountMutable(ev.PathToMount)
	require.NoError(t, err)

	defer func() {
		t.Log("unmounting mutable mount")
		require.NoError(t, fs.Unmount(ev.PathToMount))
	}()

	err = ioutil.WriteFile(filepath.Join(ev.PathToMount, "test"), []byte(`test data`), 0644)
	require.NoError(t, err)

	// uncomment to manually try out the FS
	//	time.Sleep(time.Hour)

	dirInfo, err := ioutil.ReadDir(ev.PathToMount)

	require.NotNil(t, dirInfo)
	require.NoError(t, err)
	assert.Len(t, dirInfo, 1)
	// TODO:
	// 1. after umount we should not commit when no write op has been performed
	// 2. we should not be able to overwrite an existing bundle
	// 3. commit has a bug on empty files
}

func TestMutableMountWrite(t *testing.T) {
	// write some data to a mutable mount, then commit the bundle upon unmounting
	// download the new bundle then compare the content
	ev, cleanTempDir := testFsIntegEnv()
	defer cleanTempDir(t)()

	bundle := fusemocks.EmptyBundle(ev)

	fs, err := NewMutableFS(bundle)
	require.NoError(t, err)

	err = fs.MountMutable(ev.PathToMount)
	require.NoError(t, err)

	var tree fusemocks.UploadTree
	defer func() {
		testAfterCommit(t, bundle, tree, ev, false)
	}()

	defer func() {
		t.Log("unmounting: this uploads the bundle")
		require.NoError(t, fs.Unmount(ev.PathToMount))
	}()

	t.Log("populating the new mount with data")
	tree = fusemocks.PopulateFS(t, ev.PathToMount)
	// DEBUG: fusemocks.TestInspectDir(t, ev.pathToMount)
}

/* The mutable filesystem writes a bundle via the Commit() function, and Commit() indirectly calls
 * calls cafs.Fs.Put().
 *
 * This test currently simulates what happens when Put(), in particular, returns an error
 * by passing an alterate implementation of the Fs interface to the implementation of Commit().
 *
 * The intent of this sort of test is not specific to errors on Put(), and further tests could describe what
 * happens on various other io errors such as reading or writing from the backing filesystem as well
 * as the storage backing the cafs.
 */
func TestMutableMountCommitError(t *testing.T) {
	ev, cleanTempDir := testFsIntegEnv()
	defer cleanTempDir(t)()

	bundle := fusemocks.EmptyBundle(ev)

	fs, err := NewMutableFS(bundle)
	require.NoError(t, err)

	err = fs.MountMutable(ev.PathToMount)
	require.NoError(t, err)

	var tree fusemocks.UploadTree
	defer func() {
		testAfterCommit(t, bundle, tree, ev, false)
	}()

	defer func() {
		t.Log("unmounting: this uploads the bundle")
		require.NoError(t, fs.Unmount(ev.PathToMount))
	}()

	t.Log("populating the new mount with data")
	tree = fusemocks.PopulateFS(t, ev.PathToMount)

	caFs, expectedError := fusemocks.NewErrPutCaFs(t,
		fs.fsInternal.bundle.BlobStore(),
		fs.fsInternal.bundle.BundleDescriptor.LeafSize,
	)

	// ensure error data returned properly
	err = fs.fsInternal.commitImpl(caFs)
	require.NotNil(t, err)
	assert.Equal(t, expectedError, err)
}

func TestMutableMountMkdirEmpty(t *testing.T) {
	// write some empty directories to a mutable mount, then commit the bundle upon unmounting
	// download the new bundle then compare the content
	//
	// part 1: all directories are empty: the bundle is empty
	testMutableMountMkdirWithFile(t, false)
}

func TestMutableMountMkdir(t *testing.T) {
	// part 2: all directories contain a non-empty file: the bundle is valid
	testMutableMountMkdirWithFile(t, true)
}

func testMutableMountMkdirWithFile(t testing.TB, withFile bool) {
	ev, cleanTempDir := testFsIntegEnv()
	defer cleanTempDir(t)()

	bundle := fusemocks.EmptyBundle(ev)

	fs, err := NewMutableFS(bundle)
	require.NoError(t, err)

	err = fs.MountMutable(ev.PathToMount)
	require.NoError(t, err)

	var tree fusemocks.UploadTree
	defer func() {
		testAfterCommit(t, bundle, tree, ev, !withFile)
	}()

	defer func() {
		t.Log("unmounting: this uploads the bundle")
		require.NoError(t, fs.Unmount(ev.PathToMount))
	}()

	t.Logf("populating the new mount with directories, empty: %t", !withFile)
	tree = fusemocks.PopulateFSWithDirs(t, ev.PathToMount, withFile)
}

func testAfterCommit(t testing.TB, bundle *core.Bundle, tree fusemocks.UploadTree, ev mocks.TestEnv, expectEmpty bool) {
	t.Log("post-commit verifications")
	// a new bundle has been created
	require.NotEmpty(t, bundle.BundleID)

	t.Logf("downloading newly created bundle in: %s", ev.DestinationDir)
	bundle.ConsumableStore = localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.DestinationDir))
	require.NoError(t, core.Publish(context.Background(), bundle))

	t.Log("verify downloaded bundle")
	if !expectEmpty {
		fusemocks.AssertTree(t, tree, ev.DestinationDir)
		return
	}
	// expected empty bundle
	t.Log("only empty things where created: the new bundle should be empty")
	infoDir, err := ioutil.ReadDir(ev.DestinationDir)
	require.NoError(t, err)
	require.Len(t, infoDir, 1)
	assert.Equal(t, ".datamon", infoDir[0].Name())
}

func testFSOperations(t testing.TB, ev mocks.TestEnv, opsFunc func(string, os.FileInfo, chan<- error, *sync.WaitGroup)) {
	// an exerciser for operations or scenarios of operations on a fuse mount populated with some files
	errC := make(chan error)
	var (
		errFS error
		wg1   sync.WaitGroup
		wg2   sync.WaitGroup
	)

	wg1.Add(1)
	go func(e <-chan error, wg *sync.WaitGroup) {
		defer wg.Done()
		for r := range e {
			if errFS == nil {
				errFS = r
			}
		}
	}(errC, &wg1)
	l := mocks.TestLogger()
	l.Info("start walking")

	err := filepath.Walk(ev.PathToMount, func(pth string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			t.Logf("walk error on %q: %v", pth, walkErr)
			return walkErr
		}
		if pth == ev.PathToMount {
			return nil
		}
		l.Info("walking", zap.String("file", pth))
		wg2.Add(1)
		go opsFunc(pth, info, errC, &wg2)
		if throttleIOs > 0 {
			time.Sleep(throttleIOs) // throttling test (experimented hangs when run on CI)
		}

		return nil
	})
	require.NoError(t, err)

	l.Info("waiting for ops to terminate")
	wg2.Wait()
	close(errC)
	l.Info("waiting for error events")
	wg1.Wait()
	assert.NoError(t, errFS)
	l.Info("done with ops")
}

func stringorDie(arg string, err error) string {
	if err != nil {
		panic(err)
	}
	return arg
}
