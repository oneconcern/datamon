// +build fsintegration

package core

import (
	"context"
	"crypto/md5" //#nosec
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/spf13/afero"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oneconcern/datamon/internal"
	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
)

const (
	testOnTempFS = true                 // for very large test cases, avoid tempFS
	throttleIOs  = 1 * time.Millisecond // on CI, we experiment hangs on RO mount when I/O parallel workload is too high
)

func testFsIntegEnv() (testEnv, func(t testing.TB) func()) {
	var tmp string
	if testOnTempFS {
		tmp = ""
	} else {
		tmp = "./work"
		_ = os.MkdirAll(tmp, 0700)
	}
	// builds a temporary testing environment
	mountPath := stringorDie(ioutil.TempDir(tmp, "mount-")) // fuse mount
	stagingPath := stringorDie(ioutil.TempDir(tmp, "staging-"))
	testRoot := stringorDie(ioutil.TempDir(tmp, "core-data-"))
	sourceDir := filepath.Join(testRoot, "bundle", "source")
	destinationDir := filepath.Join(testRoot, "bundle", "destination")

	blobDir := filepath.Join(sourceDir, "blob")
	metaDir := filepath.Join(sourceDir, "meta")
	vmetaDir := filepath.Join(sourceDir, "vmeta")
	wal := filepath.Join(sourceDir, "wal")
	readLog := filepath.Join(sourceDir, "readLog")

	for _, dir := range []string{blobDir, metaDir, vmetaDir, wal, readLog} {
		_ = os.MkdirAll(dir, 0700)
	}

	return testEnv{
			leafSize:               cafs.DefaultLeafSize,
			repo:                   "bundle-test-repo",
			bundleID:               "bundle456",
			testRoot:               testRoot,
			sourceDir:              sourceDir,
			blobDir:                blobDir,
			metaDir:                metaDir,
			vmetaDir:               vmetaDir,
			wal:                    wal,
			readLog:                readLog,
			destinationDir:         destinationDir,
			reBundleEntriesPerFile: 3,
			original:               filepath.Join(testRoot, "internal"),
			dataDir:                "dir/",
			pathToMount:            mountPath,
			pathToStaging:          stagingPath,
		}, func(t testing.TB) func() {
			return func() {
				t.Logf("unwinding integration test environment")
				tempDirs := []string{
					mountPath,
					stagingPath,
					sourceDir,
					destinationDir,
					testRoot,
				}
				if tmp != "" {
					tempDirs = append(tempDirs, tmp)
				}
				for _, toRemove := range tempDirs {
					_ = os.RemoveAll(toRemove)
				}
			}
		}
}

func TestRoMount(t *testing.T) {
	// mount a fuse fs from a bundle, verify the content of that mount
	// then randomly perform various syscalls on these files
	var (
		// 12 files in this bundle, stored in 1 index file
		bundleEntriesFileCount uint64 = 1
		dataFilesCount         uint64 = 12
	)

	ev, cleanTempDir := testFsIntegEnv()
	defer cleanTempDir(t)()

	t.Log("preparing fake bundle")
	cleanup := setupFakeDataBundle(t, bundleEntriesFileCount, dataFilesCount, ev)
	defer cleanup()

	t.Logf("preparing RO mount on %s", ev.pathToMount)
	bundle := fakeBundle(ev)
	fs, err := NewReadOnlyFS(bundle, testLogger())
	require.NoError(t, err)

	err = fs.MountReadOnly(ev.pathToMount)
	require.NoError(t, err)

	defer func() {
		fs.fsInternal.l.Info("unmounting RO mount")
		t.Log("unmounting RO mount")
		require.NoError(t, fs.Unmount(ev.pathToMount))
	}()

	fs.fsInternal.l.Info("verifying data files")
	t.Log("verifying data files")
	validateDataFiles(t, filepath.Join(ev.destinationDir, ev.dataDir), filepath.Join(ev.pathToMount, ev.dataDir))

	fs.fsInternal.l.Info("exercising fs syscalls")
	t.Log("exercising fs syscalls")
	testFSOperations(t, ev, fsROActions)
}

func fsROActions(pth string, info os.FileInfo, e chan<- error, wg *sync.WaitGroup) {
	l := testLogger()
	defer wg.Done()
	// randomized actions on FS
	actions := map[string]func(string, string, bool, chan<- error){
		"stat":               testStat,
		"bad-stat":           testBadStat,
		"readFile":           testReadFile,
		"bad-readFile":       testBadReadFile,
		"bad-overwriteFile":  testBadOverwriteFile,
		"bad-createFile":     testBadCreateFile,
		"bad-createFile2":    testBadCreateFile2,
		"bad-mkdir":          testBadMkdir,
		"bad-truncate":       testBadTruncate,
		"bad-chown":          testBadChown,
		"bad-chmod":          testBadChmod,
		"bad-remove":         testBadRemove,
		"bad-rename":         testBadRename,
		"bad-symlink":        testBadSymlink,
		"open-read-seek":     testOpenReadSeek,
		"bad-open-write":     testBadOpenWrite,
		"bad-open-overwrite": testBadOpenOverwrite,
		"bad-open-create":    testBadOpenCreate,
		"statfs":             testStatFS,
	}
	for action, fn := range actions {
		l.Info("fs-action", zap.String("action", action), zap.String("file", pth))
		fn(action, pth, info.IsDir(), e)
	}
}

func TestMutableMount(t *testing.T) {
	// smoke test on mutable mount: just fuse mount an initial empty bundle, then write a file

	ev, cleanTempDir := testFsIntegEnv()
	defer cleanTempDir(t)()

	bundle := emptyBundle(ev)

	fs, err := NewMutableFS(bundle, ev.pathToStaging, testLogger())
	require.NoError(t, err)

	err = fs.MountMutable(ev.pathToMount)
	require.NoError(t, err)

	defer func() {
		t.Log("unmounting mutable mount")
		require.NoError(t, fs.Unmount(ev.pathToMount))
	}()

	err = ioutil.WriteFile(filepath.Join(ev.pathToMount, "test"), []byte(`test data`), 0644)
	require.NoError(t, err)

	// uncomment to manually try out the FS
	//	time.Sleep(time.Hour)

	dirInfo, err := ioutil.ReadDir(ev.pathToMount)

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

	bundle := emptyBundle(ev)

	fs, err := NewMutableFS(bundle, ev.pathToStaging, testLogger())
	require.NoError(t, err)

	err = fs.MountMutable(ev.pathToMount)
	require.NoError(t, err)

	var tree uploadTree
	defer func() {
		testAfterCommit(t, bundle, tree, ev, false)
	}()

	defer func() {
		t.Log("unmounting: this uploads the bundle")
		require.NoError(t, fs.Unmount(ev.pathToMount))
	}()

	t.Log("populating the new mount with data")
	tree = testPopulateFS(t, ev.pathToMount)
	// DEBUG: testInspectDir(t, ev.pathToMount)
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

	bundle := emptyBundle(ev)

	fs, err := NewMutableFS(bundle, ev.pathToStaging, testLogger())
	require.NoError(t, err)

	err = fs.MountMutable(ev.pathToMount)
	require.NoError(t, err)

	var tree uploadTree
	defer func() {
		testAfterCommit(t, bundle, tree, ev, false)
	}()

	defer func() {
		t.Log("unmounting: this uploads the bundle")
		require.NoError(t, fs.Unmount(ev.pathToMount))
	}()

	t.Log("populating the new mount with data")
	tree = testPopulateFS(t, ev.pathToMount)

	// setup the mock
	caFsImpl, err := cafs.New(
		cafs.LeafSize(fs.fsInternal.bundle.BundleDescriptor.LeafSize),
		cafs.Backend(fs.fsInternal.bundle.BlobStore()),
	)
	require.NoError(t, err)

	randErrData := internal.RandStringBytesMaskImprSrc(15)
	caFs := &testErrCaFs{fsImpl: caFsImpl, errMsg: randErrData}

	// ensure error data returned properly
	err = fs.fsInternal.commitImpl(caFs)
	require.NotNil(t, err)
	require.Equal(t, randErrData, err.Error())
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

func testMutableMountMkdirWithFile(t *testing.T, withFile bool) {
	ev, cleanTempDir := testFsIntegEnv()
	defer cleanTempDir(t)()

	bundle := emptyBundle(ev)

	fs, err := NewMutableFS(bundle, ev.pathToStaging, testLogger())
	require.NoError(t, err)

	err = fs.MountMutable(ev.pathToMount)
	require.NoError(t, err)

	var tree uploadTree
	defer func() {
		testAfterCommit(t, bundle, tree, ev, !withFile)
	}()

	defer func() {
		t.Log("unmounting: this uploads the bundle")
		require.NoError(t, fs.Unmount(ev.pathToMount))
	}()

	t.Logf("populating the new mount with directories, empty: %t", !withFile)
	tree = testPopulateFSWithDirs(t, ev.pathToMount, withFile)
}

type uploadFileTest struct {
	path   string
	size   int
	data   []byte
	target string
	pthsum [md5.Size]byte
	cksum  [md5.Size]byte
	found  bool
	isDir  bool
}

type uploadTree []*uploadFileTest

func makeTestUploadTree() uploadTree {
	return uploadTree{
		{
			path: "/small/1k",
			size: 1024,
		},
		{
			path: "/leafs/leafsize",
			size: cafs.DefaultLeafSize,
		},
		{
			path: "/leafs/over-leafsize",
			size: cafs.DefaultLeafSize + 1,
		},
		{
			path: "/leafs/under-leafsize",
			size: cafs.DefaultLeafSize - 1,
		},
		{
			path: "/leafs/multiple-leafsize",
			size: cafs.DefaultLeafSize * 3,
		},
		{
			path: "/leafs/root",
			size: 1,
		},
		{
			path: "/1/2/3/4/5/6/deep",
			size: 100,
		},
		{
			path: "/1/2/3/4/5/6/7/deeper",
			size: 200,
		},
	}
}

func testPopulateFS(t testing.TB, mountPath string) uploadTree {
	// add files to the mounted filesystem, with random data.
	// The returned structure keeps data and checksums to make further assertions
	testUploadTree := makeTestUploadTree()

	t.Logf("populating test upload on mount: %s", mountPath)
	for idx, uf := range testUploadTree {
		target := filepath.Join(mountPath, filepath.FromSlash(uf.path))
		dirname := filepath.Dir(target)
		require.NoError(t, os.MkdirAll(dirname, 0755))

		normalizedPath, _ := filepath.Rel(mountPath, target)
		testUploadTree[idx].pthsum = md5.Sum([]byte(normalizedPath)) //#nosec
		testUploadTree[idx].target = target

		if uf.isDir {
			require.NoError(t, os.MkdirAll(target, 0755))
			continue
		}
		data := internal.RandBytesMaskImprSrc(uf.size)
		require.NoError(t, ioutil.WriteFile(target, data, 0644))

		testUploadTree[idx].data = data
		//#nosec
		testUploadTree[idx].cksum = md5.Sum(data)

	}
	return testUploadTree
}

func testPopulateFSWithDirs(t testing.TB, mountPath string, withFile bool) uploadTree {
	// same as testPopulateFS, but ignores size and just create
	// directories instead of files.
	testUploadTree := makeTestUploadTree()

	t.Logf("populating test upload on mount: %s", mountPath)
	for idx, uf := range testUploadTree {
		target := filepath.Join(mountPath, filepath.FromSlash(uf.path))
		require.NoError(t, os.MkdirAll(target, 0755))

		testUploadTree[idx].target = target
		testUploadTree[idx].isDir = true
		normalizedPath, _ := filepath.Rel(mountPath, target)
		testUploadTree[idx].pthsum = md5.Sum([]byte(normalizedPath)) //#nosec
	}
	if !withFile {
		return testUploadTree
	}
	// add some _non-empty_ .datamonkeep file under each directory
	extraFiles := make([]*uploadFileTest, 0, len(testUploadTree))
	for _, uf := range testUploadTree {
		pth := path.Join(uf.path, ".datamonkeep")
		target := filepath.Join(mountPath, filepath.FromSlash(pth))
		// TODO: if we put all files with same content, the deduplication ends up with an error on commit...
		// => have to randomize for now
		//data := []byte(`not empty`)
		data := internal.RandBytesMaskImprSrc(10)
		require.NoError(t, ioutil.WriteFile(target, data, 0644))

		normalizedPath, _ := filepath.Rel(mountPath, target)
		extraFiles = append(extraFiles, &uploadFileTest{
			path:   pth,
			target: target,
			//#nosec
			pthsum: md5.Sum([]byte(normalizedPath)),
			//#nosec
			cksum: md5.Sum(data),
		})
	}
	return append(testUploadTree, extraFiles...)
}

func emptyBundle(ev testEnv) *Bundle {
	// produce an initialized bundle with proper staging and context
	bd := NewBDescriptor(Message("test bundle"))
	bundle := NewBundle(bd,
		Repo(ev.repo),
		ConsumableStore(localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.pathToStaging))),
		ContextStores(fakeContext(ev.metaDir, ev.blobDir)),
	)
	return bundle
}

// nolint: unused,deadcode
func testInspectDir(t testing.TB, pth string) {
	// debug helper to inspect the various temp mounts before removal
	out, err := exec.Command("find", pth, "-ls").CombinedOutput()
	require.NoError(t, err)
	t.Log(string(out))
}

func testAfterCommit(t testing.TB, bundle *Bundle, tree uploadTree, ev testEnv, expectEmpty bool) {
	t.Log("post-commit verifications")
	// new bundle has been created
	require.NotEmpty(t, bundle.BundleID)

	t.Logf("downloading newly created bundle in: %s", ev.destinationDir)
	bundle.ConsumableStore = localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.destinationDir))
	require.NoError(t, Publish(context.Background(), bundle))

	t.Log("verify downloaded bundle")
	if !expectEmpty {
		assertTree(t, tree, ev.destinationDir)
		return
	}
	// expected empty bundle
	t.Log("only empty things where created: the new bundle should be empty")
	infoDir, err := ioutil.ReadDir(ev.destinationDir)
	require.NoError(t, err)
	require.Len(t, infoDir, 1)
	assert.Equal(t, ".datamon", infoDir[0].Name())
}

func assertTree(t testing.TB, tree uploadTree, root string) {
	// check an actual FS structure against a reference in-memory fixture folder description
	treeMap := make(map[[md5.Size]byte]*uploadFileTest, len(tree))
	dirMap := make(map[string]struct{}, len(tree))
	for _, uf := range tree {
		treeMap[uf.pthsum] = uf
		if uf.isDir {
			continue
		}
		// completing the tree map with all intermediate directories
		pth := strings.TrimPrefix(uf.path, "/")
		parts := strings.Split(pth, "/")
		if len(parts) < 2 {
			continue
		}
		for i := 1; i < len(parts); i++ {
			normalizedPath := filepath.Join(parts[0:i]...)
			dirMap[normalizedPath] = struct{}{}
		}
	}
	for dir := range dirMap {
		//#nosec
		pthsum := md5.Sum([]byte(dir))
		if _, ok := treeMap[pthsum]; ok {
			continue
		}
		treeMap[pthsum] = &uploadFileTest{
			path:  dir,
			isDir: true,
		}
	}
	err := filepath.Walk(root, func(target string, info os.FileInfo, walkErr error) error {
		pth, _ := filepath.Rel(root, target)
		if model.IsGeneratedFile(pth) || pth == "." {
			return nil
		}
		//#nosec
		pthsum := md5.Sum([]byte(pth))
		uf, ok := treeMap[pthsum]

		if info.IsDir() {
			if !assert.True(t, ok, "found directory in destination dir %s which was not in the reference tree", pth) {
				return nil
			}
			uf.found = true
			assert.True(t, uf.isDir, "found directory in destination dir %s, but was expected to be a file", pth)
			return nil
		}
		if !assert.True(t, ok, "found file in destination dir %s which was not in the reference tree", pth) {
			return nil
		}
		uf.found = true
		data, er := ioutil.ReadFile(target)
		require.NoError(t, er, "could not read file in destination  %s: %v", pth, er)
		// #nosec
		cksum := md5.Sum(data)
		assert.True(t, cksum == uf.cksum, "file %s found in destination, but content differ from reference tree", pth)
		return nil
	})
	require.NoError(t, err)
	for _, uf := range treeMap {
		assert.True(t, uf.found, "file %s in reference tree but not found in destination", uf.path)
	}
}

/* mock cafs.Fs used to simulate error */
// ??? moq?

type testErrCaFs struct {
	fsImpl cafs.Fs
	errMsg string
}

func (fs *testErrCaFs) GetAddressingScheme() string {
	return "blake"
}

func (fs *testErrCaFs) Put(ctx context.Context, src io.Reader) (cafs.PutRes, error) {
	return cafs.PutRes{}, errors.New(fs.errMsg)
}

func (fs *testErrCaFs) Get(ctx context.Context, hash cafs.Key) (io.ReadCloser, error) {
	return fs.fsImpl.Get(ctx, hash)
}

func (fs *testErrCaFs) GetAt(ctx context.Context, hash cafs.Key) (io.ReaderAt, error) {
	return fs.fsImpl.GetAt(ctx, hash)
}

func (fs *testErrCaFs) Delete(ctx context.Context, hash cafs.Key) error {
	return fs.fsImpl.Delete(ctx, hash)
}

func (fs *testErrCaFs) Clear(ctx context.Context) error {
	return fs.fsImpl.Clear(ctx)
}

func (fs *testErrCaFs) Keys(ctx context.Context) ([]cafs.Key, error) {
	return fs.fsImpl.Keys(ctx)
}

func (fs *testErrCaFs) RootKeys(ctx context.Context) ([]cafs.Key, error) {
	return fs.fsImpl.RootKeys(ctx)
}

func (fs *testErrCaFs) Has(ctx context.Context, key cafs.Key, cfgs ...cafs.HasOption) (bool, []cafs.Key, error) {
	return fs.fsImpl.Has(ctx, key, cfgs...)
}

func testFSOperations(t testing.TB, ev testEnv, opsFunc func(string, os.FileInfo, chan<- error, *sync.WaitGroup)) {
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
	l := testLogger()
	l.Info("start walking")

	err := filepath.Walk(ev.pathToMount, func(pth string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			t.Logf("walk error on %q: %v", pth, walkErr)
			return walkErr
		}
		if pth == ev.pathToMount {
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

func testLogger() *zap.Logger {
	//return zap.NewNop() // to limit test output
	l, _ := zap.NewDevelopment() // to get DEBUG  during test run
	return l
}
