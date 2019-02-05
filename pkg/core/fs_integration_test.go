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
	destinationStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), destinationDir))
	sourceStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), sourceDir))
	bundle := NewBundle(repo, bundleID, sourceStore, destinationStore)
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
	destinationStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), destinationDir))
	sourceStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), sourceDir))
	bundle := NewBundle(repo, bundleID, sourceStore, destinationStore)
	fs := NewMutableFS(bundle, "/tmp/")
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
