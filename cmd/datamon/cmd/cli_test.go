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
	logFatalf = MakeFatalfMock(exitMocks)
	logFatalln = MakeFatallnMock(exitMocks)
	btag := internal.RandStringBytesMaskImprSrc(15)
	bucketMeta := "datamontestmeta-" + btag
	bucketBlob := "datamontestblob-" + btag

	client, err := gcsStorage.NewClient(context.TODO(), option.WithScopes(gcsStorage.ScopeFullControl))
	require.NoError(t, err, "couldn't create bucket client")
	err = client.Bucket(bucketMeta).Create(ctx, "onec-co", nil)
	require.NoError(t, err, "couldn't create metadata bucket")
	err = client.Bucket(bucketBlob).Create(ctx, "onec-co", nil)
	require.NoError(t, err, "couldn't create blob bucket")
	repoParams.MetadataBucket = bucketMeta
	repoParams.BlobBucket = bucketBlob
	createTree()
	cleanup := func() {
		os.RemoveAll(destinationDir)
		deleteBucket(ctx, t, client, bucketMeta)
		deleteBucket(ctx, t, client, bucketBlob)
	}
	return cleanup
}

func runCmd(t *testing.T, cmd []string, intentMsg string, expectError bool) {
	fatalCallsBefore := exitMocks.fatalCalls
	rootCmd.SetArgs(cmd)
	require.NoError(t, rootCmd.Execute(), "error executing '"+strings.Join(cmd, " ")+"' : "+intentMsg)
	if expectError {
		require.Equal(t, fatalCallsBefore+1, exitMocks.fatalCalls,
			"ran '"+strings.Join(cmd, " ")+"' expecting error and didn't see one in mocks : "+intentMsg)
	} else {
		require.Equal(t, fatalCallsBefore, exitMocks.fatalCalls,
			"unexpected error in mocks on '"+strings.Join(cmd, " ")+"' : "+intentMsg)
	}
}

func TestCreateRepo(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	}, "create first test repo", false)
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo2,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	}, "create second test repo", false)
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	}, "observe error on create repo with duplicate name", true)
}

type repoListEntry struct {
	rawLine     string
	repo        string
	name        string
	description string
	email       string
	time        time.Time
}

/* for tests that need to read stdout into data structures, this function converts
 * a string to a slice of lines, each of which can be parsed into a struct.
 */
func getDataLogLines(t *testing.T, ls string, ignorePatterns []string) []string {
	ll := strings.Split(strings.TrimSpace(ls), "\n")
	if len(ll) == 0 {
		return ll
	}
	var repoLinesStart int
	for repoLinesStart < len(ll) && ll[repoLinesStart] == "" {
		repoLinesStart++
	}
	for {
		if !(repoLinesStart < len(ll)) {
			break
		}
		var sawPattern bool
		for _, ip := range ignorePatterns {
			m, err := regexp.MatchString(ip, ll[repoLinesStart])
			require.NoError(t, err, "regexp match error.  likely a programming mistake in tests.")
			if m {
				repoLinesStart++
				sawPattern = true
				break
			}
		}
		if !sawPattern {
			break
		}
	}
	if repoLinesStart == len(ll) {
		return make([]string, 0)
	}
	return ll[repoLinesStart:]
}

func listRepos(t *testing.T) ([]repoListEntry, error) {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	log.SetOutput(w)
	//
	runCmd(t, []string{"repo",
		"list",
	}, "create second test repo", false)
	//
	log.SetOutput(os.Stdout)
	w.Close()
	//
	lb, err := ioutil.ReadAll(r)
	require.NoError(t, err, "i/o error reading patched log from pipe")
	//
	rles := make([]repoListEntry, 0)
	for _, line := range getDataLogLines(t, string(lb), []string{`Using config file`}) {
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
	require.NoError(t, err, "error out of listRepos() test helper")
	require.Equal(t, len(ll), 0, "expect empty repo list before creating repos")
	testNow := time.Now()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	}, "create first test repo", false)
	ll, err = listRepos(t)
	require.NoError(t, err, "error out of listRepos() test helper")
	require.Equal(t, 1, len(ll), "one repo in list after create")
	require.Equal(t, repo1, ll[0].repo, "repo name after first create")
	require.Equal(t, "testing", ll[0].description, "repo description after first create")
	require.Equal(t, "tests", ll[0].name, "contributor name after first create")
	require.Equal(t, "datamon@oneconcern.com", ll[0].email, "contributor email after first create")
	require.True(t, testNow.Sub(ll[0].time).Seconds() < 3, "timestamp bounds after first create")
	testNow = time.Now()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing too",
		"--repo", repo2,
		"--name", "tests2",
		"--email", "datamon2@oneconcern.com",
	}, "create second test repo", false)
	ll, err = listRepos(t)
	require.NoError(t, err, "error out of listRepos() test helper")
	require.Equal(t, 2, len(ll), "two repos in list after second create")
	require.Equal(t, repo1, ll[0].repo, "first repo name after second create")
	require.Equal(t, "testing", ll[0].description, "first repo description after second create")
	require.Equal(t, "tests", ll[0].name, "first contributor name after second create")
	require.Equal(t, "datamon@oneconcern.com", ll[0].email, "first contributor email after second create")
	require.Equal(t, repo2, ll[1].repo, "second repo name after second create")
	require.Equal(t, "testing too", ll[1].description, "second repo description after second create")
	require.Equal(t, "tests2", ll[1].name, "second contributor name after second create")
	require.Equal(t, "datamon2@oneconcern.com", ll[1].email, "second contributor email after second create")
	require.True(t, testNow.Sub(ll[1].time).Seconds() < 3, "second timestamp bounds after second create")
}

func testUploadBundle(t *testing.T, file uploadTree) {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	log.SetOutput(w)
	//
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, file),
		"--message", "The initial commit for the repo",
		"--repo", repo1,
	}, "upload bundle at "+dirPathStr(t, file), false)
	//
	log.SetOutput(os.Stdout)
	w.Close()
	//
	lb, err := ioutil.ReadAll(r)
	require.NoError(t, err, "i/o error reading patched log from pipe")
	ls := string(lb)
	//
	m, err := regexp.MatchString(`Uploaded bundle`, ls)
	require.NoError(t, err, "regexp match error.  likely a programming mistake in tests.")
	require.True(t, m, "expect confirmation message on upload")
}

func TestUploadBundle(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	}, "create test repo", false)
	for _, tree := range testUploadTrees {
		testUploadBundle(t, tree[0])
	}
}

func TestUploadBundle_filePath(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	}, "create test repo", false)
	file := testUploadTrees[0][0]
	runCmd(t, []string{"bundle",
		"upload",
		"--path", filePathStr(t, file),
		"--message", "The initial commit for the repo",
		"--repo", repo1,
	}, "observe error on bundle upload path as file rather than directory", true)
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
	runCmd(t, []string{"bundle",
		"list",
		"--repo", repoName,
	}, "list bundles", false)
	log.SetOutput(os.Stdout)
	w.Close()
	//
	lb, err := ioutil.ReadAll(r)
	require.NoError(t, err, "i/o error reading patched log from pipe")
	bles := make([]bundleListEntry, 0)
	for _, line := range getDataLogLines(t, string(lb), []string{`Using config file`}) {
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
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, file),
		"--message", msg,
		"--repo", repo1,
	}, "upload bundle at "+dirPathStr(t, file), false)
	ll, err := listBundles(t, repo2)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, 0, len(ll), "no bundles in secondary repo")
	ll, err = listBundles(t, repo1)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, bcnt, len(ll), "bundle count in test repo")
	require.Equal(t, msg, ll[len(ll)-1].message, "bundle log message")
	require.True(t, testNow.Sub(ll[len(ll)-1].time).Seconds() < 3, "timestamp bounds after bundle create")
}

func TestListBundles(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	}, "create second test repo", false)
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo2,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	}, "create second test repo", false)
	ll, err := listBundles(t, repo1)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, len(ll), 0, "no bundles created yet")
	ll, err = listBundles(t, repo2)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, 0, len(ll), "no bundles created yet")
	for i, tree := range testUploadTrees {
		testListBundle(t, tree[0], i+1)
	}
}

func testDownloadBundle(t *testing.T, files []uploadTree, bcnt int) {
	msg := internal.RandStringBytesMaskImprSrc(15)
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, files[0]),
		"--message", msg,
		"--repo", repo1,
	}, "upload bundle at "+dirPathStr(t, files[0]), false)

	ll, err := listBundles(t, repo1)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, bcnt, len(ll), "bundle count in test repo")
	//
	destFS := afero.NewBasePathFs(afero.NewOsFs(), consumedData)
	dpc := "bundle-dl-" + ll[len(ll)-1].hash
	dp, err := filepath.Abs(filepath.Join(consumedData, dpc))
	if err != nil {
		t.Errorf("couldn't build file path: %v", err)
	}
	exists, err := afero.Exists(destFS, dpc)
	require.NoError(t, err, "error out of afero upstream library.  possibly programming error in test.")
	require.False(t, exists, "no filesystem entry at destination path '"+dpc+"' before bundle upload")
	runCmd(t, []string{"bundle",
		"download",
		"--repo", repo1,
		"--destination", dp,
		"--bundle", ll[len(ll)-1].hash,
	}, "download bundle uploaded from "+dirPathStr(t, files[0]), false)
	exists, err = afero.Exists(destFS, dpc)
	require.NoError(t, err, "error out of afero upstream library.  possibly programming error in test.")
	require.True(t, exists, "filesystem entry at at destination path '"+dpc+"' after bundle upload")
	//
	for _, file := range files {
		expected := readTextFile(t, filePathStr(t, file))
		actual := readTextFile(t, filepath.Join(dp, pathInBundle(file)))
		require.Equal(t, len(expected), len(actual), "downloaded file '"+pathInBundle(file)+"' size")
		require.Equal(t, expected, actual, "downloaded file '"+pathInBundle(file)+"' contents")
	}
}

func TestDownloadBundles(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	}, "create second test repo", false)
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

func listBundleFiles(t *testing.T, repoName string, bid string) []bundleFileListEntry {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	//
	runCmd(t, []string{"bundle",
		"list",
		"files",
		"--repo", repoName,
		"--bundle", bid,
	}, "get bundle files list", false)

	//
	os.Stdout = stdout
	w.Close()
	//
	lb, err := ioutil.ReadAll(r)
	require.NoError(t, err, "i/o error reading patched stdout from pipe")
	lms := make([]map[string]string, 0)
	for _, line := range getDataLogLines(t, string(lb), []string{`Using bundle`}) {
		lm := make(map[string]string)
		sl := strings.Split(line, ",")
		for _, kvstr := range sl {
			kvslice := strings.Split(strings.TrimSpace(kvstr), ":")
			require.Equal(t, 2, len(kvslice), "key-val parse error of bundle files list log lines")
			lm[kvslice[0]] = kvslice[1]
		}
		lm["_line"] = line
		lms = append(lms, lm)
	}
	bfles := make([]bundleFileListEntry, 0)
	for _, lm := range lms {
		name, has := lm["name"]
		require.True(t, has, "didn't find 'name' in parsed key-val bundle files list log line entry")
		hash, has := lm["hash"]
		require.True(t, has, "didn't find 'hash' in parsed key-val bundle files list log line entry")
		sizeStr, has := lm["size"]
		require.True(t, has, "didn't find 'size' in parsed key-val bundle files list log line entry")
		size, err := strconv.Atoi(sizeStr)
		require.NoError(t, err, "parse error of size string from  bundle files list")
		bfle := bundleFileListEntry{
			rawLine: lm["_line"],
			hash:    hash,
			name:    name,
			size:    size,
		}
		bfles = append(bfles, bfle)
	}
	return bfles
}

func testListBundleFiles(t *testing.T, files []uploadTree, bcnt int) {
	msg := internal.RandStringBytesMaskImprSrc(15)
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, files[0]),
		"--message", msg,
		"--repo", repo1,
	}, "create second test repo", false)

	rll, err := listBundles(t, repo1)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, bcnt, len(rll), "bundle count in test repo")
	//
	bfles := listBundleFiles(t, repo1, rll[len(rll)-1].hash)
	require.Equal(t, len(files), len(bfles), "file count in bundle files list log")
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
		bEx[pathInBundle(file)] = true
		filesM[pathInBundle(file)] = file
	}
	require.Equal(t, bEx, bnsAc, "bundle files list log compared to fixture data: list's name set")
	for name, bfle := range bflesM {
		require.Equal(t, filesM[name].size, bfle.size, "bundle files list log compared to fixture data: "+
			"entry's size '"+name+"'")
	}
}

func TestListBundlesFiles(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	}, "create second test repo", false)
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
	runCmd(t, []string{"bundle",
		"download",
		"file",
		"--file", pathInBundle(file),
		"--repo", repo1,
		"--bundle", bid,
		"--destination", dp,
	}, "download bundle file "+pathInBundle(file), false)
	// see iss #111 re. pathInBundle() use here and per-file cleanup below
	exists, err := afero.Exists(destFS, pathInBundle(file))
	require.NoError(t, err, "error out of afero upstream library.  possibly programming error in test.")
	require.True(t, exists, "filesystem entry exists in specified file location individual file download")
	//
	expected := readTextFile(t, filePathStr(t, file))
	actual := readTextFile(t, filepath.Join(dp, pathInBundle(file)))
	require.Equal(t, len(expected), len(actual), "downloaded file '"+pathInBundle(file)+"' size")
	require.Equal(t, actual, expected, "downloaded file '"+pathInBundle(file)+"' contents")
	/* per-file cleanup */
	require.NoError(t, destFS.RemoveAll(".datamon"),
		"error removing per-file download metadata (in order to allow downloading more indiv files)")
}

func testBundleDownloadFiles(t *testing.T, files []uploadTree, bcnt int) {
	msg := internal.RandStringBytesMaskImprSrc(15)
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, files[0]),
		"--message", msg,
		"--repo", repo1,
	}, "upload bundle in order to test downloading individual files", false)
	rll, err := listBundles(t, repo1)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, bcnt, len(rll), "bundle count in test repo")
	//
	for _, file := range files {
		testBundleDownloadFile(t, file, rll[len(rll)-1].hash)
	}
}

func TestBundlesDownloadFiles(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
		"--name", "tests",
		"--email", "datamon@oneconcern.com",
	}, "create repo", false)

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

func pathInBundle(file uploadTree) string {
	pathComp := strings.Split(file.path, string(os.PathSeparator))
	return filepath.Join(pathComp[2:]...)
}

// dupe: cafs/reader_test.go
// comparing large files could be faster by reading chunks and failing on the first chunk that differs
func readTextFile(t testing.TB, pth string) string {
	v, err := ioutil.ReadFile(pth)
	if err != nil {
		require.NoError(t, err, "error reading file at '"+pth+"'")
	}
	return string(v)
}

/* objects can be deleted recursively.  non-empty buckets cannot be deleted. */
func deleteBucket(ctx context.Context, t *testing.T, client *gcsStorage.Client, bucketName string) {
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
