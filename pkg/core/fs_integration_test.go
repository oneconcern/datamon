// +build fsintegration

package core

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"

	"github.com/stretchr/testify/require"
)

func TestMount(t *testing.T) {
	require.NoError(t, Setup(t))
	destinationStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), destinationDir))
	sourceStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), sourceDir))
	bundle := NewBundle(repo, bundleID, sourceStore, destinationStore)
	fs, err := NewDatamonFS(bundle)
	require.NoError(t, err)
	_ = os.Mkdir("/tmp/mount2", 0777|os.ModeDir)
	err = fs.Mount("/tmp/mount2")
	require.NoError(t, err)
	//time.Sleep(time.Hour)
	resp, err := ioutil.ReadDir("/tmp/mount2")
	require.NotNil(t, resp)
	require.NoError(t, err)
	require.NoError(t, fs.Unmount("/tmp/mount2"))
}
