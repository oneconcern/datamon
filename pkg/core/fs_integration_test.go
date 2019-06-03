// +build fsintegration

package core

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/spf13/afero"

	"github.com/stretchr/testify/require"

	"github.com/oneconcern/datamon/internal"
	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
)

var pathToMount = "/tmp/mount/"

func TestMount(t *testing.T) {
	require.NoError(t, Setup(t))
	consumableStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), destinationDir))
	metaStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), metaDir))
	blobStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), blobDir))
	bd := NewBDescriptor()
	bundle := New(bd,
		Repo(repo),
		BundleID(bundleID),
		MetaStore(metaStore),
		ConsumableStore(consumableStore),
		BlobStore(blobStore),
	)
	l, _ := zap.NewProduction()
	fs, err := NewReadOnlyFS(bundle, l)
	require.NoError(t, err)
	_ = os.Mkdir(pathToMount, 0777|os.ModeDir)
	err = fs.MountReadOnly(pathToMount)
	require.NoError(t, err)
	// uncomment to manually try out the FS
	// time.Sleep(time.Hour)
	resp, err := ioutil.ReadDir(pathToMount)
	require.NotNil(t, resp)
	require.NoError(t, err)
	validateDataFiles(t, destinationDir+dataDir, pathToMount+dataDir)
	require.NoError(t, fs.Unmount(pathToMount))
}

func TestMutableMount(t *testing.T) {
	require.NoError(t, Setup(t))
	consumableStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), destinationDir))
	metaStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), metaDir))
	blobStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), blobDir))
	bd := NewBDescriptor()
	bundle := New(bd,
		Repo(repo),
		BundleID(bundleID),
		MetaStore(metaStore),
		ConsumableStore(consumableStore),
		BlobStore(blobStore),
	)
	fs, _ := NewMutableFS(bundle, "/tmp/")
	_ = os.Mkdir(pathToMount, 0777|os.ModeDir)
	err := fs.MountMutable(pathToMount)
	require.NoError(t, err)
	// uncomment to manually try out the FS
	//	time.Sleep(time.Hour)
	resp, err := ioutil.ReadDir(pathToMount)
	require.NotNil(t, resp)
	require.NoError(t, err)
	//validateDataFiles(t, destinationDir+dataDir, pathToMount+dataDir)
	require.NoError(t, fs.Unmount(pathToMount))
}

type uploadFileTest struct {
	path string
	size int
	data []byte
}

type uploadTree []uploadFileTest

var testUploadTree = uploadTree{
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

// todo: setup and cleanup.  in particular, defer s'th to ensure fs unmounted

func TestMutableMountWrite(t *testing.T) {
	setupEmptyBundle(t)
	consumableStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), destinationDir))
	metaStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), metaDir))
	blobStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), blobDir))
	bd := NewBDescriptor()
	bundle := New(bd,
		Repo(repo),
		MetaStore(metaStore),
		ConsumableStore(consumableStore),
		BlobStore(blobStore),
	)
	fs, _ := NewMutableFS(bundle, "/tmp/")
	_ = os.Mkdir(pathToMount, 0777|os.ModeDir)
	err := fs.MountMutable(pathToMount)
	require.NoError(t, err)

	/* add files to filesystem */
	afs := afero.NewBasePathFs(afero.NewOsFs(), pathToMount)

	for idx := range testUploadTree {
		testUploadTree[idx].data = internal.RandBytesMaskImprSrc(testUploadTree[idx].size)
	}
	for _, uf := range testUploadTree {
		dirname, _ := filepath.Split(uf.path)
		require.NoError(t, aferoMkdirAll(afs, dirname, 0755))
		require.NoError(t, aferoWriteFile(afs, uf.path, uf.data, 0644))

	}

	require.Equal(t, bundle.BundleID, "")
	// todo: validate files on filesystem
	/* store files to bundle on unmount */
	require.NoError(t, fs.Unmount(pathToMount))
	require.NotEqual(t, bundle.BundleID, "")

	/* validate files stored to bundle */
	destFS := afero.NewBasePathFs(afero.NewOsFs(), destinationDir)
	consumableStore = localfs.New(destFS)
	metaStore = localfs.New(afero.NewBasePathFs(afero.NewOsFs(), metaDir))
	blobStore = localfs.New(afero.NewBasePathFs(afero.NewOsFs(), blobDir))
	bd = NewBDescriptor()
	bundle = New(bd,
		Repo(repo),
		BundleID(bundle.BundleID),
		MetaStore(metaStore),
		ConsumableStore(consumableStore),
		BlobStore(blobStore),
	)
	require.NoError(t, Publish(context.Background(), bundle))
	for _, uf := range testUploadTree {
		exists, err := afero.Exists(destFS, uf.path)
		require.NoError(t, err)
		require.True(t, exists)
		fbytes, err := afero.ReadFile(destFS, uf.path)
		require.NoError(t, err)
		require.Equal(t, fbytes, uf.data)
	}

}

/* the mutable filesystem writes a bundle via the Commit() function, and Commit() indirectly calls
 * calls cafs.Fs.Put().  this test currently simulates what happens when Put(), in particular, returns an error
 * by passing an alterate implementation of the Fs interface to the implementation of Commit().
 *
 * the intent of this sort of test is not specific to errors on Put(), and further tests could describe what
 * happens on various other io errors such as reading or writing from the backing filesystem as well
 * as the storage backing the cafs.
 */
func TestMutableMountCommitError(t *testing.T) {
	setupEmptyBundle(t)
	consumableStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), destinationDir))
	metaStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), metaDir))
	blobStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), blobDir))
	bd := NewBDescriptor()
	bundle := New(bd,
		Repo(repo),
		MetaStore(metaStore),
		ConsumableStore(consumableStore),
		BlobStore(blobStore),
	)
	fs, _ := NewMutableFS(bundle, "/tmp/")
	_ = os.Mkdir(pathToMount, 0777|os.ModeDir)
	err := fs.MountMutable(pathToMount)
	require.NoError(t, err)
	/* add files to filesystem */
	afs := afero.NewBasePathFs(afero.NewOsFs(), pathToMount)
	for idx := range testUploadTree {
		testUploadTree[idx].data = internal.RandBytesMaskImprSrc(testUploadTree[idx].size)
	}
	for _, uf := range testUploadTree {
		dirname, _ := filepath.Split(uf.path)
		require.NoError(t, aferoMkdirAll(afs, dirname, 0755))
		require.NoError(t, aferoWriteFile(afs, uf.path, uf.data, 0644))

	}
	// setup the mock
	caFsImpl, err := cafs.New(
		cafs.LeafSize(fs.fsInternal.bundle.BundleDescriptor.LeafSize),
		cafs.Backend(fs.fsInternal.bundle.BlobStore),
	)
	require.NoError(t, err)
	randErrData := internal.RandStringBytesMaskImprSrc(15)
	caFs := &testErrCaFs{fsImpl: caFsImpl, errMsg: randErrData}
	// ensure error data returned properly
	err = fs.fsInternal.commitImpl(caFs)
	require.NotNil(t, err)
	require.Equal(t, randErrData, err.Error())
	// cleanup
	require.NoError(t, fs.Unmount(pathToMount))
}

/* mock cafs.Fs used to simulate error */
// ??? moq?

type testErrCaFs struct {
	fsImpl cafs.Fs
	errMsg string
}

func (fs *testErrCaFs) Put(ctx context.Context, src io.Reader) (int64, cafs.Key, []byte, bool, error) {
	return 0, cafs.Key{}, make([]byte, 0), false, errors.New(fs.errMsg)
}

func (fs *testErrCaFs) Get(ctx context.Context, hash cafs.Key) (io.ReadCloser, error) {
	return fs.fsImpl.Get(ctx, hash)
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

/* os x fuse workarounds */
func aferoWriteFile(fs afero.Fs, filename string, data []byte, perm os.FileMode) (err error) {
	rc := 2
	for i := 0; i < rc; i++ {
		err = afero.WriteFile(fs, filename, data, perm)
		if err == nil {
			return
		}
	}
	return
}

func aferoMkdirAll(afs afero.Fs, name string, mode os.FileMode) (err error) {
	ndirs := len(strings.Split(name, string(os.PathSeparator)))
	rc := ndirs * 2
	for i := 0; i < rc; i++ {
		err = afs.MkdirAll(name, mode)
		if err == nil {
			return
		}
	}
	return
}

func TestMutableMountMkdir(t *testing.T) {
	setupEmptyBundle(t)
	consumableStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), destinationDir))
	metaStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), metaDir))
	blobStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), blobDir))
	bd := NewBDescriptor()
	bundle := New(bd,
		Repo(repo),
		BundleID(bundleID),
		MetaStore(metaStore),
		ConsumableStore(consumableStore),
		BlobStore(blobStore),
	)
	fs, _ := NewMutableFS(bundle, "/tmp/")
	_ = os.Mkdir(pathToMount, 0777|os.ModeDir)
	err := fs.MountMutable(pathToMount)
	require.NoError(t, err)
	afs := afero.NewBasePathFs(afero.NewOsFs(), pathToMount)
	fs.fsInternal.l.Info("creating directories")
	require.NoError(t, afs.Mkdir("i", 0755))
	fs.fsInternal.l.Info("created i/")
	for i := 0; i < 10; i++ {
		fs.fsInternal.l.Info("attempting to create i/j")
		err = afs.Mkdir("i/j", 0755)
		if err != nil {
			t.Logf("mkdir failed %v (%v)", err, i)
		} else {
			t.Logf("mkdir passed on attempt %v", i)
			break
		}
	}
}

func setupEmptyBundle(t *testing.T) {
	cleanup()
	require.NoError(t, os.MkdirAll(original, 0700))
	require.NoError(t, os.MkdirAll(blobDir, 0700))
	require.NoError(t, os.MkdirAll(metaDir, 0700))
	require.NoError(t, os.MkdirAll(destinationDir, 0700))
}
