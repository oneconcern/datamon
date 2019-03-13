// +build fsintegration

package core

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"

	"github.com/stretchr/testify/require"
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
	fs, err := NewReadOnlyFS(bundle)
	require.NoError(t, err)
	_ = os.Mkdir(pathToMount, 0777|os.ModeDir)
	err = fs.MountReadOnly(pathToMount)
	require.NoError(t, err)
	time.Sleep(time.Hour)
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
	time.Sleep(time.Hour)
	resp, err := ioutil.ReadDir(pathToMount)
	require.NotNil(t, resp)
	require.NoError(t, err)
	validateDataFiles(t, destinationDir+dataDir, pathToMount+dataDir)
	require.NoError(t, fs.Unmount(pathToMount))
}
