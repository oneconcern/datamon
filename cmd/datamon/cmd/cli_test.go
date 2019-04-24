package cmd

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/oneconcern/datamon/pkg/storage"

	"github.com/oneconcern/datamon/pkg/cafs"

	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"

	gcsStorage "cloud.google.com/go/storage"
	"github.com/oneconcern/datamon/internal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const (
	destinationDir = "../../../testdata/cli"
	sourceData     = destinationDir + "/data"
	consumedData   = destinationDir + "/downloads"
	repo1          = "test-repo1"
	repo2          = "test-repo2"
	timeForm       = "2006-01-02 15:04:05.999999999 -0700 MST"
)

type uploadTree struct {
	path string
	size int
	data []byte
}

var testUploadTrees = [][]uploadTree{{
	{
		path: "/small/1k",
		size: 1024,
	},
}, {
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
}, {
	{
		path: "/1/2/3/4/5/6/deep",
		size: 100,
	},
	{
		path: "/1/2/3/4/5/6/7/deeper",
		size: 200,
	},
},
}

type ExitMocks struct {
	mock.Mock
	fatalCalls int
}

func (m *ExitMocks) Fatalf(format string, v ...interface{}) {
	m.fatalCalls++
}

func (m *ExitMocks) Fatalln(v ...interface{}) {
	m.fatalCalls++
}

// https://github.com/stretchr/testify/issues/610
func MakeFatalfMock(m *ExitMocks) func(string, ...interface{}) {
	return func(format string, v ...interface{}) {
		m.Fatalf(format, v...)
	}
}

func MakeFatallnMock(m *ExitMocks) func(...interface{}) {
	return func(v ...interface{}) {
		m.Fatalln(v...)
	}
}

var exitMocks *ExitMocks

func setupTests(t *testing.T) func() {
	os.RemoveAll(destinationDir)
	ctx := context.Background()
	exitMocks = new(ExitMocks)
	log_Fatalf = MakeFatalfMock(exitMocks)
	log_Fatalln = MakeFatallnMock(exitMocks)
	btag := internal.RandStringBytesMaskImprSrc(15)
	bucketMeta := "datamontestmeta-" + btag
	bucketBlob := "datamontestblob-" + btag

	client, err := gcsStorage.NewClient(context.TODO(), option.WithScopes(gcsStorage.ScopeFullControl))
	err = client.Bucket(bucketMeta).Create(ctx, "onec-co", nil)
	require.NoError(t, err)
	err = client.Bucket(bucketBlob).Create(ctx, "onec-co", nil)
	require.NoError(t, err)
	repoParams.MetadataBucket = bucketMeta
	repoParams.BlobBucket = bucketBlob
	createTree()
	cleanup := func() {
		os.RemoveAll(destinationDir)
		deleteBucket(t, ctx, client, bucketMeta)
		deleteBucket(t, ctx, client, bucketBlob)
	}
	return cleanup
}

func TestCreateRepo(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	rootCmd.SetArgs([]string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	})
	require.NoError(t, rootCmd.Execute())
	rootCmd.SetArgs([]string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo2,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	})
	require.NoError(t, rootCmd.Execute())
	// negative test
	require.Equal(t, exitMocks.fatalCalls, 0)
	rootCmd.SetArgs([]string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	})
	require.NoError(t, rootCmd.Execute())
	require.Equal(t, exitMocks.fatalCalls, 1)
}

type repoListEntry struct {
	rawLine     string
	repo        string
	name        string
	description string
	email       string
	time        time.Time
}

func listRepos(t *testing.T) ([]repoListEntry, error) {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	log.SetOutput(w)
	//
	rootCmd.SetArgs([]string{"repo",
		"list",
	})
	require.NoError(t, rootCmd.Execute())
	//
	log.SetOutput(os.Stdout)
	w.Close()
	//
	lb, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	ls := string(lb)
	ll := strings.Split(strings.TrimSpace(ls), "\n")
	//
	m, err := regexp.MatchString(`Using config file`, ll[0])
	require.NoError(t, err)
	require.NotNil(t, m)
	//
	rles := make([]repoListEntry, 0)
	for _, line := range ll[1:] {
		sl := strings.Split(line, ",")
		t, err := time.Parse(timeForm, strings.TrimSpace(sl[4]))
		if err != nil {
			return nil, err
		}
		rle := repoListEntry{
			rawLine:     line,
			repo:        strings.TrimSpace(sl[0]),
			name:        strings.TrimSpace(sl[2]),
			description: strings.TrimSpace(sl[1]),
			email:       strings.TrimSpace(sl[3]),
			time:        t,
		}
		rles = append(rles, rle)
	}
	return rles, nil
}

func TestRepoList(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	ll, err := listRepos(t)
	require.NoError(t, err)
	require.Equal(t, len(ll), 0)
	testNow := time.Now()
	rootCmd.SetArgs([]string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	})
	require.NoError(t, rootCmd.Execute())
	ll, err = listRepos(t)
	require.NoError(t, err)
	require.Equal(t, len(ll), 1)
	require.Equal(t, ll[0].repo, repo1)
	require.Equal(t, ll[0].description, "testing")
	require.Equal(t, ll[0].name, "tests")
	require.Equal(t, ll[0].email, "datamon@oneconcern.com")
	require.True(t, testNow.Sub(ll[0].time).Seconds() < 3)
	testNow = time.Now()
	rootCmd.SetArgs([]string{"repo",
		"create",
		"--description", "testing too",
		"--repo", repo2,
		"--name", "tests2",
		"--email", "datamon2@oneconcern.com",
	})
	require.NoError(t, rootCmd.Execute())
	ll, err = listRepos(t)
	require.NoError(t, err)
	require.Equal(t, len(ll), 2)
	require.Equal(t, ll[0].repo, repo1)
	require.Equal(t, ll[0].description, "testing")
	require.Equal(t, ll[0].name, "tests")
	require.Equal(t, ll[0].email, "datamon@oneconcern.com")
	require.Equal(t, ll[1].repo, repo2)
	require.Equal(t, ll[1].description, "testing too")
	require.Equal(t, ll[1].name, "tests2")
	require.Equal(t, ll[1].email, "datamon2@oneconcern.com")
	require.True(t, testNow.Sub(ll[1].time).Seconds() < 3)
}

func testUploadBundle(t *testing.T, file uploadTree) {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	log.SetOutput(w)
	//
	cmd := []string{"bundle",
		"upload",
		"--path", dirPathStr(t, file),
		"--message", "The initial commit for the repo",
		"--repo", repo1,
	}
	rootCmd.SetArgs(cmd)
	require.NoError(t, rootCmd.Execute())
	//
	log.SetOutput(os.Stdout)
	w.Close()
	//
	lb, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	ls := string(lb)
	//
	m, err := regexp.MatchString(`Uploaded bundle`, ls)
	require.NoError(t, err)
	require.NotNil(t, m)
}

func TestUploadBundle(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	rootCmd.SetArgs([]string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	})
	require.NoError(t, rootCmd.Execute())
	for _, tree := range testUploadTrees {
		testUploadBundle(t, tree[0])
	}
}

func TestUploadBundle_filePath(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	rootCmd.SetArgs([]string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	})
	require.NoError(t, rootCmd.Execute())
	file := testUploadTrees[0][0]
	cmd := []string{"bundle",
		"upload",
		"--path", filePathStr(t, file),
		"--message", "The initial commit for the repo",
		"--repo", repo1,
	}

	require.Equal(t, exitMocks.fatalCalls, 0)
	rootCmd.SetArgs(cmd)
	require.NoError(t, rootCmd.Execute())
	require.Equal(t, exitMocks.fatalCalls, 1)
}

type bundleListEntry struct {
	rawLine string
	hash    string
	message string
	time    time.Time
}

func listBundles(t *testing.T, repoName string) ([]bundleListEntry, error) {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	log.SetOutput(w)
	//
	rootCmd.SetArgs([]string{"bundle",
		"list",
		"--repo", repoName,
	})
	require.NoError(t, rootCmd.Execute())
	//
	log.SetOutput(os.Stdout)
	w.Close()
	//
	lb, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	ls := string(lb)
	ll := strings.Split(strings.TrimSpace(ls), "\n")
	//
	m, err := regexp.MatchString(`Using config file`, ll[0])
	require.NoError(t, err)
	require.NotNil(t, m)
	//
	bles := make([]bundleListEntry, 0)
	for _, line := range ll[1:] {
		sl := strings.Split(line, ",")
		t, err := time.Parse(timeForm, strings.TrimSpace(sl[1]))
		if err != nil {
			return nil, err
		}
		rle := bundleListEntry{
			rawLine: line,
			hash:    strings.TrimSpace(sl[0]),
			message: strings.TrimSpace(sl[2]),
			time:    t,
		}
		bles = append(bles, rle)
	}
	return bles, nil
}

func testListBundle(t *testing.T, file uploadTree, bcnt int) {
	msg := internal.RandStringBytesMaskImprSrc(15)
	testNow := time.Now()
	rootCmd.SetArgs([]string{"bundle",
		"upload",
		"--path", dirPathStr(t, file),
		"--message", msg,
		"--repo", repo1,
	})
	require.NoError(t, rootCmd.Execute())
	ll, err := listBundles(t, repo2)
	require.NoError(t, err)
	require.Equal(t, len(ll), 0, "no bundles created yet")
	ll, err = listBundles(t, repo1)
	require.NoError(t, err)
	require.Equal(t, len(ll), bcnt)
	require.Equal(t, ll[len(ll)-1].message, msg)
	require.True(t, testNow.Sub(ll[len(ll)-1].time).Seconds() < 3)
}

func TestListBundles(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	rootCmd.SetArgs([]string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	})
	require.NoError(t, rootCmd.Execute())
	rootCmd.SetArgs([]string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo2,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	})
	require.NoError(t, rootCmd.Execute())
	ll, err := listBundles(t, repo1)
	require.NoError(t, err)
	require.Equal(t, len(ll), 0, "no bundles created yet")
	ll, err = listBundles(t, repo2)
	require.NoError(t, err)
	require.Equal(t, len(ll), 0, "no bundles created yet")

	for i, tree := range testUploadTrees {
		testListBundle(t, tree[0], i+1)
	}
}

func testDownloadBundle(t *testing.T, files []uploadTree, bcnt int) {
	msg := internal.RandStringBytesMaskImprSrc(15)
	rootCmd.SetArgs([]string{"bundle",
		"upload",
		"--path", dirPathStr(t, files[0]),
		"--message", msg,
		"--repo", repo1,
	})
	require.NoError(t, rootCmd.Execute())
	ll, err := listBundles(t, repo1)
	require.NoError(t, err)
	require.Equal(t, len(ll), bcnt)
	//
	destFS := afero.NewBasePathFs(afero.NewOsFs(), consumedData)
	dpc := "bundle-dl-" + ll[len(ll)-1].hash
	dp, err := filepath.Abs(filepath.Join(consumedData, dpc))
	if err != nil {
		t.Errorf("couldn't build file path: %v", err)
	}
	exists, err := afero.Exists(destFS, dpc)
	require.NoError(t, err)
	require.False(t, exists)
	rootCmd.SetArgs([]string{"bundle",
		"download",
		"--repo", repo1,
		"--destination", dp,
		"--bundle", ll[len(ll)-1].hash,
	})
	require.NoError(t, rootCmd.Execute())
	exists, err = afero.Exists(destFS, dpc)
	require.NoError(t, err)
	require.True(t, exists)
	//
	for _, file := range files {
		expected := readTextFile(t, filePathStr(t, file))
		actual := readTextFile(t, filepath.Join(dp, pathInBundle(t, file)))
		require.Equal(t, len(expected), len(actual))
		require.Equal(t, expected, actual)
	}
}

func TestDownloadBundles(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	rootCmd.SetArgs([]string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	})
	require.NoError(t, rootCmd.Execute())
	for i, tree := range testUploadTrees {
		testDownloadBundle(t, tree, i+1)
	}
}

type bundleFileListEntry struct {
	rawLine string
	hash    string
	name    string
	size    int
}

func listBundleFiles(t *testing.T, repoName string, bid string) ([]bundleFileListEntry, error) {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	//
	rootCmd.SetArgs([]string{"bundle",
		"list",
		"files",
		"--repo", repoName,
		"--bundle", bid,
	})
	require.NoError(t, rootCmd.Execute())
	//
	os.Stdout = stdout
	w.Close()
	//
	lb, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	ls := string(lb)
	ll := strings.Split(strings.TrimSpace(ls), "\n")
	//
	m, err := regexp.MatchString(`Using config file`, ll[0])
	require.NoError(t, err)
	require.NotNil(t, m)
	//
	lms := make([]map[string]string, 0)
	for _, line := range ll[1:] {
		lm := make(map[string]string)
		sl := strings.Split(line, ",")
		for _, kvstr := range sl {
			kvslice := strings.Split(strings.TrimSpace(kvstr), ":")
			require.Equal(t, len(kvslice), 2)
			lm[kvslice[0]] = kvslice[1]
		}
		lm["_line"] = line
		lms = append(lms, lm)
	}
	bfles := make([]bundleFileListEntry, 0)
	for _, lm := range lms {
		name, has := lm["name"]
		require.True(t, has)
		hash, has := lm["hash"]
		require.True(t, has)
		sizeStr, has := lm["size"]
		require.True(t, has)
		size, err := strconv.Atoi(sizeStr)
		require.NoError(t, err)
		bfle := bundleFileListEntry{
			rawLine: lm["_line"],
			hash:    hash,
			name:    name,
			size:    size,
		}
		bfles = append(bfles, bfle)
	}
	return bfles, nil
}

func testListBundleFiles(t *testing.T, files []uploadTree, bcnt int) {
	msg := internal.RandStringBytesMaskImprSrc(15)
	rootCmd.SetArgs([]string{"bundle",
		"upload",
		"--path", dirPathStr(t, files[0]),
		"--message", msg,
		"--repo", repo1,
	})
	require.NoError(t, rootCmd.Execute())
	rll, err := listBundles(t, repo1)
	require.NoError(t, err)
	require.Equal(t, len(rll), bcnt)
	//
	bfles, err := listBundleFiles(t, repo1, rll[len(rll)-1].hash)
	require.NoError(t, err)
	require.Equal(t, len(bfles), len(files))
	/* test set equality of names while setting up maps to test data by name */
	bnsAc := make(map[string]bool)
	bflesM := make(map[string]bundleFileListEntry)
	for _, bfle := range bfles {
		bnsAc[bfle.name] = true
		bflesM[bfle.name] = bfle
	}
	bEx := make(map[string]bool)
	filesM := make(map[string]uploadTree)
	for _, file := range files {
		bEx[pathInBundle(t, file)] = true
		filesM[pathInBundle(t, file)] = file
	}
	require.Equal(t, bnsAc, bEx)
	for name, bfle := range bflesM {
		require.Equal(t, bfle.size, filesM[name].size)
	}
}

func TestListBundlesFiles(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	rootCmd.SetArgs([]string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	})
	require.NoError(t, rootCmd.Execute())

	for i, tree := range testUploadTrees {
		testListBundleFiles(t, tree, i+1)
	}
}

func testBundleDownloadFile(t *testing.T, file uploadTree, bid string) {
	dpc := "file-dl"
	dp, err := filepath.Abs(filepath.Join(consumedData, dpc))
	destFS := afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(consumedData, dpc))
	if err != nil {
		t.Errorf("couldn't build file path: %v", err)
	}
	cmd := []string{"bundle",
		"download",
		"file",
		"--file", pathInBundle(t, file),
		"--repo", repo1,
		"--bundle", bid,
		"--destination", dp,
	}
	rootCmd.SetArgs(cmd)
	require.NoError(t, rootCmd.Execute())
	// see iss #111 re. pathInBundle() use here and per-file cleanup below
	exists, err := afero.Exists(destFS, pathInBundle(t, file))
	require.NoError(t, err)
	require.True(t, exists)
	//
	expected := readTextFile(t, filePathStr(t, file))
	actual := readTextFile(t, filepath.Join(dp, pathInBundle(t, file)))
	require.Equal(t, len(expected), len(actual))
	require.Equal(t, expected, actual)
	/* per-file cleanup */
	err = destFS.RemoveAll(".datamon")
	require.NoError(t, err)
}

func testBundleDownloadFiles(t *testing.T, files []uploadTree, bcnt int) {
	msg := internal.RandStringBytesMaskImprSrc(15)
	rootCmd.SetArgs([]string{"bundle",
		"upload",
		"--path", dirPathStr(t, files[0]),
		"--message", msg,
		"--repo", repo1,
	})
	require.NoError(t, rootCmd.Execute())
	rll, err := listBundles(t, repo1)
	require.NoError(t, err)
	require.Equal(t, len(rll), bcnt)
	//
	for _, file := range files {
		testBundleDownloadFile(t, file, rll[len(rll)-1].hash)
	}
}

func TestBundlesDownloadFiles(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	rootCmd.SetArgs([]string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	})
	require.NoError(t, rootCmd.Execute())
	testBundleDownloadFiles(t, testUploadTrees[0], 1)
	testBundleDownloadFiles(t, testUploadTrees[1], 2)
	testBundleDownloadFiles(t, testUploadTrees[2], 3)
}

/** untested:
 * - bundle_mount.go
 * - config_generate.go
 */

func createTree() {
	sourceFS := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), sourceData))
	for _, tree := range testUploadTrees {
		for _, file := range tree {
			err := sourceFS.Put(context.Background(),
				file.path,
				bytes.NewReader(internal.RandBytesMaskImprSrc(file.size)),
				storage.IfNotPresent)
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
}

/** util */
/* absolute uploaded (to test file contents) */
func filePathStr(t *testing.T, file uploadTree) (path string) {
	path, err := filepath.Abs(filepath.Join(sourceData, file.path))
	if err != nil {
		t.Errorf("couldn't build file path: %v", err)
	}
	return
}

/* absolute path to root directory (to upload bundle) */
func dirPathStr(t *testing.T, file uploadTree) (path string) {
	/* the strings.Split gets the root directory name.
	 * would be cleaner to iterate on filepath.Split,
	 * although even in this case `os.PathSeparator` appears necessary.
	 */
	path, err := filepath.Abs(filepath.Join(sourceData, strings.Split(file.path, string(os.PathSeparator))[1]))
	if err != nil {
		t.Errorf("couldn't build file path: %v", err)
	}
	return
}

func pathInBundle(t *testing.T, file uploadTree) string {
	pathComp := strings.Split(file.path, string(os.PathSeparator))
	return filepath.Join(pathComp[2:]...)
}

// dupe: cafs/reader_test.go
// comparing large files could be faster by reading chunks and failing on the first chunk that differs
func readTextFile(t testing.TB, pth string) string {
	v, err := ioutil.ReadFile(pth)
	if err != nil {
		require.NoError(t, err)
	}
	return string(v)
}

/* objects can be deleted recursively.  non-empty buckets cannot be deleted. */
func deleteBucket(t *testing.T, ctx context.Context, client *gcsStorage.Client, bucketName string) {
	mb := client.Bucket(bucketName)
	oi := mb.Objects(ctx, &gcsStorage.Query{})
	for {
		objAttrs, err := oi.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Errorf("error iterating: %v", err)
		}
		obj := mb.Object(objAttrs.Name)
		if err := obj.Delete(ctx); err != nil {
			t.Errorf("error deleting object: %v", err)
		}
	}
	if err := mb.Delete(ctx); err != nil {
		t.Errorf("error deleting bucket %v", err)
	}
}
