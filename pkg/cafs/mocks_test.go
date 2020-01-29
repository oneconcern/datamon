package cafs

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/oneconcern/datamon/pkg/dlogger"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"
)

const (
	destDir = "../../testdata"

	leafSize uint32 = 1.5 * 1024 * 1024
)

func mustBytes(res []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return res
}

type testFile struct {
	Original string
	RootHash string
	Parts    int
	Size     int
}

func testFiles(destDir string) []testFile {
	return []testFile{
		{
			Original: filepath.Join(destDir, "original", "small"),
			RootHash: filepath.Join(destDir, "roots", "small"),
			Parts:    1,
			Size:     int(leafSize - 512),
		},
		{
			Original: filepath.Join(destDir, "original", "twoparts"),
			RootHash: filepath.Join(destDir, "roots", "twoparts"),
			Parts:    2,
			Size:     int(leafSize + leafSize/2),
		},
		{
			Original: filepath.Join(destDir, "original", "fourparts"),
			RootHash: filepath.Join(destDir, "roots", "fourparts"),
			Parts:    4,
			Size:     int(3*leafSize + leafSize/2),
		},
		{
			Original: filepath.Join(destDir, "original", "tenparts"),
			RootHash: filepath.Join(destDir, "roots", "tenparts"),
			Parts:    10,
			Size:     int(10*leafSize - 512),
		},
		{
			Original: filepath.Join(destDir, "original", "exact10"),
			RootHash: filepath.Join(destDir, "roots", "exact10"),
			Parts:    10,
			Size:     int(10 * leafSize),
		},
		{
			Original: filepath.Join(destDir, "original", "under10"),
			RootHash: filepath.Join(destDir, "roots", "under10"),
			Parts:    10,
			Size:     int(10*leafSize - 1),
		},
		{
			Original: filepath.Join(destDir, "original", "over10"),
			RootHash: filepath.Join(destDir, "roots", "over10"),
			Parts:    11,
			Size:     int(10*leafSize + 1),
		},
		{
			Original: filepath.Join(destDir, "original", "onetwoeigth-not-tree-root"),
			RootHash: filepath.Join(destDir, "roots", "onetwoeigth-not-tree-root"),
			Parts:    1,
			Size:     128,
		},
		{
			Original: filepath.Join(destDir, "original", "fivetwelve-not-tree-root"),
			RootHash: filepath.Join(destDir, "roots", "fivetwelve-not-tree-root"),
			Parts:    1,
			Size:     512,
		},
		{
			Original: filepath.Join(destDir, "original", "tiny"),
			RootHash: filepath.Join(destDir, "roots", "tiny"),
			Parts:    1,
			Size:     15,
		},
	}
}

func TestMain(m *testing.M) {
	if _, _, _, err := setupTestData(destDir, testFiles(destDir)); err != nil {
		log.Fatalln(err)
	}
	os.Exit(m.Run())
}

func setupTestData(dir string, files []testFile) (*testDataGenerator, storage.Store, Fs, error) {
	os.RemoveAll(dir)
	blobs := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(dir, "cafs")))
	fs, err := New(
		LeafSize(leafSize),
		Backend(blobs),
		LeafTruncation(false),
		Logger(dlogger.MustGetLogger("warn")),
	)
	if err != nil {
		return nil, nil, nil, err
	}

	g := &testDataGenerator{destDir: dir, leafSize: leafSize, fs: fs}
	if err = g.Initialize(); err != nil {
		return nil, nil, nil, err
	}
	return g, blobs, fs, g.Generate(files)
}

type testDataGenerator struct {
	destDir  string
	leafSize uint32
	fs       Fs
}

func (t *testDataGenerator) Initialize() error {
	if err := os.MkdirAll(filepath.Join(t.destDir, "original"), 0700); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(t.destDir, "cafs"), 0700); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(t.destDir, "roots"), 0700); err != nil {
		return err
	}
	return nil
}

func (t *testDataGenerator) Generate(files []testFile) error {
	for _, ps := range files {
		if err := GenerateFile(ps.Original, ps.Size, t.leafSize); err != nil {
			return err
		}

		if err := GenerateCAFSFile(ps.Original, t.fs, t.destDir); err != nil {
			return err
		}
	}
	return nil
}
