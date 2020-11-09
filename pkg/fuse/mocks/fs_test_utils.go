package mocks

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
	"testing"

	"github.com/oneconcern/datamon/internal/rand"
	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/core/mocks"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FakeBundle produce some fake bundle structure for this test environment
func FakeBundle(ev mocks.TestEnv) *core.Bundle {
	return core.NewBundle(
		core.Repo(ev.Repo),
		core.BundleID(ev.BundleID),
		core.ContextStores(mocks.FakeContext(ev.MetaDir, ev.BlobDir)),
		core.ConsumableStore(localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.DestinationDir))),
		core.Logger(mocks.TestLogger()),
	)
}

// EmptyBundle produces an initialized bundle with proper staging and context, but no bundle ID yet
func EmptyBundle(ev mocks.TestEnv) *core.Bundle {
	bundle := core.NewBundle(
		core.BundleDescriptor(
			model.NewBundleDescriptor(model.Message("test bundle")),
		),
		core.Repo(ev.Repo),
		core.ConsumableStore(localfs.New(afero.NewBasePathFs(afero.NewOsFs(), ev.PathToStaging))),
		core.ContextStores(mocks.FakeContext(ev.MetaDir, ev.BlobDir)),
		core.Logger(mocks.TestLogger()),
	)
	return bundle
}

// TestInspectDir is a debug helper to inspect the various temp mounts before removal
func TestInspectDir(t testing.TB, pth string) {
	out, err := exec.Command("find", pth, "-ls").CombinedOutput()
	require.NoError(t, err)
	t.Log(string(out))
}

// UploadFileTest describes a mocked up uploaded file fixture
type UploadFileTest struct {
	path   string
	size   int
	data   []byte
	target string
	pthsum [md5.Size]byte
	cksum  [md5.Size]byte
	found  bool
	isDir  bool
}

// UploadTree define a mocked up file hierarchy
type UploadTree []*UploadFileTest

const defaultLeafSize = int(cafs.DefaultLeafSize)

// MakeTestUploadTree defines a default fixture to test a hierarchy of files, with different sizes
func MakeTestUploadTree() UploadTree {
	return UploadTree{
		{
			path: "/small/1k",
			size: 1024,
		},
		{
			path: "/leafs/leafsize",
			size: defaultLeafSize,
		},
		{
			path: "/leafs/over-leafsize",
			size: defaultLeafSize + 1,
		},
		{
			path: "/leafs/under-leafsize",
			size: defaultLeafSize - 1,
		},
		{
			path: "/leafs/multiple-leafsize",
			size: defaultLeafSize * 3,
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

// PopulateFS populates the file tree fixture, with some random data in mocked up files.
// It prepares the expected locations relative to mountPath.
//
// * write files to the mounted filesystem, with random data.
// * the returned structure keeps data and checksums to make further assertions
func PopulateFS(t testing.TB, mountPath string, fixtureBuilders ...func() UploadTree) UploadTree {
	testUploadTree := make(UploadTree, 0, len(fixtureBuilders))
	if len(fixtureBuilders) == 0 {
		testUploadTree = MakeTestUploadTree()
	}
	for _, builder := range fixtureBuilders {
		testUploadTree = append(testUploadTree, builder()...)
	}

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
		data := rand.Bytes(uf.size)
		require.NoError(t, ioutil.WriteFile(target, data, 0600))

		testUploadTree[idx].data = data
		//#nosec
		testUploadTree[idx].cksum = md5.Sum(data)

	}
	return testUploadTree
}

// PopulateFSWithDirs is the same as TestPopulateFS, but ignores size and just create
// directories instead of files.
func PopulateFSWithDirs(t testing.TB, mountPath string, withFile bool, fixtureBuilders ...func() UploadTree) UploadTree {
	testUploadTree := make(UploadTree, 0, len(fixtureBuilders))
	if len(fixtureBuilders) == 0 {
		testUploadTree = MakeTestUploadTree()
	}
	for _, builder := range fixtureBuilders {
		testUploadTree = append(testUploadTree, builder()...)
	}

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
	extraFiles := make([]*UploadFileTest, 0, len(testUploadTree))
	for _, uf := range testUploadTree {
		pth := path.Join(uf.path, ".datamonkeep")
		target := filepath.Join(mountPath, filepath.FromSlash(pth))
		// BUG(fred): spotted in rw mount. If we put all files with same content, the deduplication ends up with an error on commit...
		// => have to randomize for now
		// data := []byte(`not empty`)
		data := rand.Bytes(10)
		require.NoError(t, ioutil.WriteFile(target, data, 0600))

		normalizedPath, _ := filepath.Rel(mountPath, target)
		extraFiles = append(extraFiles, &UploadFileTest{
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

// AssertTree checks an actual FS structure against some reference in-memory fixture folder description
func AssertTree(t testing.TB, tree UploadTree, root string) {
	treeMap := make(map[[md5.Size]byte]*UploadFileTest, len(tree))
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
		treeMap[pthsum] = &UploadFileTest{
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

// NewErrPutCaFs produces a mocked up cafs.Fs used to simulate a Put error
func NewErrPutCaFs(t testing.TB, blob storage.Store, leafSize uint32) (cafs.Fs, error) {
	caFsImpl, err := cafs.New(
		cafs.LeafSize(leafSize),
		cafs.Backend(blob),
	)
	require.NoError(t, err)
	randErrData := rand.String(15)
	err = errors.New(randErrData)
	return &testErrCaFs{
		Fs:  caFsImpl,
		err: err,
	}, err
}

type testErrCaFs struct {
	cafs.Fs
	err error
}

func (fs *testErrCaFs) Put(ctx context.Context, src io.Reader) (cafs.PutRes, error) {
	return cafs.PutRes{}, fs.err
}
