package cmd

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/oneconcern/datamon/pkg/storage"

	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/oneconcern/datamon/pkg/storage/localfs"

	gcsStorage "cloud.google.com/go/storage"
	"github.com/oneconcern/datamon/internal/rand"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

const (
	destinationDir    = "../../../testdata/cli"
	sourceData        = destinationDir + "/data"
	consumedData      = destinationDir + "/downloads"
	repo1             = "test-repo1"
	repo2             = "test-repo2"
	timeForm          = "2006-01-02 15:04:05.999999999 -0700 MST"
	concurrencyFactor = "100"
	testLeafSize      = int(cafs.DefaultLeafSize)
)

type uploadTree struct {
	path string
	size int
}

func (u uploadTree) Root() string {
	return filepath.SplitList(filepath.FromSlash(u.path))[0]
}

var testUploadTrees = [][]uploadTree{{
	{
		path: "/small/1k",
		size: 1024,
	},
}, {
	{
		path: "/leafs/leafsize",
		size: testLeafSize,
	},
	{
		path: "/leafs/over-leafsize",
		size: testLeafSize + 1,
	},
	{
		path: "/leafs/under-leafsize",
		size: testLeafSize - 1,
	},
	{
		path: "/leafs/multiple-leafsize",
		size: testLeafSize * 3,
	},
	{
		path: "/leafs/root",
		size: 1,
	},
	{
		path: "/leafs/zero",
		size: 0,
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

func TestCreateRepo(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
	}, "create first test repo", false)
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo2,
	}, "create second test repo", false)
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
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
	for _, line := range getDataLogLines(t, string(lb), []string{}) {
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
	}, "create second test repo", false, AuthMock{name: "tests2", email: "datamon2@oneconcern.com"})
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

func TestGetRepo(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	repoName := rand.LetterString(8)
	runCmd(t, []string{"repo",
		"get",
		"--repo", repoName,
	}, "attempt get non-exist repo", true)
	require.Equal(t, int(unix.ENOENT), exitMocks.exitStatuses[len(exitMocks.exitStatuses)-1],
		"ENOENT on nonexistant label")
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repoName,
	}, "create second test repo", false)
	runCmd(t, []string{"repo",
		"get",
		"--repo", repoName,
	}, "get repo", false)
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
		//"--concurrency-factor", concurrencyFactor,
		"--concurrency-factor", "1",
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
	}, "create test repo", false)
	for _, tree := range testUploadTrees {
		testUploadBundle(t, tree[0])
	}
}

func TestUploadBundleFilePath(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
	}, "create test repo", false)
	file := testUploadTrees[0][0]
	runCmd(t, []string{"bundle",
		"upload",
		"--path", filePathStr(t, file),
		"--message", "The initial commit for the repo",
		"--repo", repo1,
		"--concurrency-factor", concurrencyFactor,
	}, "observe error on bundle upload path as file rather than directory", true)
}

func getTempFile(t *testing.T, filename string) (*os.File, func()) {
	if filename == "" {
		filename = "datamon_test_temp_file"
	}
	tempDirPath, err := ioutil.TempDir("", "datamon-TestUploadBundleFilelist")
	require.NoError(t, err, "create temp dir")
	tempPath := filepath.Join(tempDirPath, filename)
	tempFile, err := os.Create(tempPath)
	require.NoError(t, err, "create temp file")
	cleanup := func() {
		err := os.RemoveAll(tempDirPath)
		require.NoError(t, err, "rm temp dir")
	}
	return tempFile, cleanup
}

func TestUploadBundleFilelist(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
	}, "create test repo", false)
	tree := testUploadTrees[1]
	require.Greater(t, len(tree), 1)
	var filelistPath string
	fileNamesToUpload := make([]string, 0)
	{
		filelist, filelistCleanup := getTempFile(t, "to_upload.txt")
		defer filelistCleanup()
		filelistPath = filelist.Name()
		for idx, file := range tree {
			if idx > len(tree)/2 {
				break
			}
			fileNamesToUpload = append(fileNamesToUpload, pathInBundle(file))
		}
		_, err := filelist.WriteString(strings.Join(fileNamesToUpload, "\n"))
		require.NoError(t, err, "write names to filelist")
		require.NoError(t, filelist.Close(), "close filelist file")
	}
	bundles, err := listBundles(t, repo1)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, 0, bundles.Len(), "no bundles before upload")
	{
		file := tree[0]
		var (
			r, w *os.File
			lb   []byte
			m    bool
		)
		r, w, err = os.Pipe()
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
			"--concurrency-factor", concurrencyFactor,
			"--files", filelistPath,
		}, "upload bundle at "+dirPathStr(t, file), false)
		//
		log.SetOutput(os.Stdout)
		w.Close()
		//
		lb, err = ioutil.ReadAll(r)
		require.NoError(t, err, "i/o error reading patched log from pipe")
		ls := string(lb)
		//
		m, err = regexp.MatchString(`Uploaded bundle`, ls)
		require.NoError(t, err, "regexp match error.  likely a programming mistake in tests.")
		require.True(t, m, "expect confirmation message on upload")
	}
	bundles, err = listBundles(t, repo1)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, 1, bundles.Len(), "found bundle after upload")
	bundle := bundles.Last()
	bundleFilelistEntries := listBundleFiles(t, repo1, bundle.hash)
	require.Equalf(t, len(fileNamesToUpload), len(bundleFilelistEntries),
		"file count in bundle files list log [bundle: %v]", bundle)

	expectedFileNames := make(map[string]bool)
	for _, name := range fileNamesToUpload {
		expectedFileNames[name] = true
	}
	actualFileNames := make(map[string]bool)
	for _, bundleFilelistEntry := range bundleFilelistEntries {
		actualFileNames[bundleFilelistEntry.name] = true
	}
	require.Equalf(t, expectedFileNames, actualFileNames,
		"file names in bundle files list log [bundle: %v]", bundle)
}

func testUploadBundleLabel(t *testing.T, file uploadTree, label string) {
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
		"--label", label,
		"--concurrency-factor", concurrencyFactor,
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

func TestUploadBundleLabel(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
	}, "create test repo", false)
	file := testUploadTrees[0][0]
	label := "testlabel"
	testUploadBundleLabel(t, file, label)
}

type bundleListEntry struct {
	rawLine string
	hash    string
	message string
	time    time.Time
}

type bundleListEntries []bundleListEntry

func (b bundleListEntries) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
func (b bundleListEntries) Len() int {
	return len(b)
}
func (b bundleListEntries) Less(i, j int) bool {
	return b[i].hash < b[j].hash
}
func (b bundleListEntries) Last() bundleListEntry {
	return b[len(b)-1]
}

func listBundles(t *testing.T, repoName string) (bundleListEntries, error) {
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
	bles := make(bundleListEntries, 0)
	for _, line := range getDataLogLines(t, string(lb), []string{}) {
		sl := strings.Split(line, ",")
		t, err := time.Parse(timeForm, strings.TrimSpace(sl[1]))
		if err != nil {
			return nil, err
		}
		rle := bundleListEntry{
			rawLine: line,
			hash:    strings.TrimSpace(sl[0]), // bundle ID
			message: strings.TrimSpace(sl[2]),
			time:    t,
		}
		bles = append(bles, rle)
	}
	// bundles are ordered by lexicographic order of bundle IDs
	require.True(t, sort.IsSorted(bles))

	sort.Sort(bles) // sort test result by timestamp
	return bles, nil
}

func testListBundle(t *testing.T, file uploadTree, bcnt int) {
	msg := rand.LetterString(15)
	testNow := time.Now()
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, file),
		"--message", msg,
		"--repo", repo1,
		"--concurrency-factor", concurrencyFactor,
	}, "upload bundle at "+dirPathStr(t, file), false)
	bundles, err := listBundles(t, repo2)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, 0, bundles.Len(), "no bundles in secondary repo")

	bundles, err = listBundles(t, repo1)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, bcnt, bundles.Len(), "bundle count in test repo")

	found := false
	for _, b := range bundles {
		if msg == b.message {
			found = true
			break
		}
	}
	require.Truef(t, found, "bundle log message %q not found in returned bundles")
	require.True(t, testNow.Sub(bundles.Last().time).Seconds() < 3, "timestamp bounds after bundle create")
}

func TestListBundles(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
	}, "create second test repo", false)
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo2,
	}, "create second test repo", false)

	bundles, err := listBundles(t, repo1)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, bundles.Len(), 0, "no bundles created yet")

	bundles, err = listBundles(t, repo2)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, 0, bundles.Len(), "no bundles created yet")

	for i, tree := range testUploadTrees {
		testListBundle(t, tree[0], i+1)
	}
}

func TestGetBundle(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
	}, "create second test repo", false)
	label := rand.LetterString(8)
	runCmd(t, []string{"bundle",
		"get",
		"--repo", repo1,
		"--label", label,
	}, "attempt get non-exist bundle", true)
	require.Equal(t, int(unix.ENOENT), exitMocks.exitStatuses[len(exitMocks.exitStatuses)-1],
		"ENOENT on nonexistant bundle")
	files := testUploadTrees[0]
	file := files[0]
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, file),
		"--message", "label test bundle",
		"--repo", repo1,
		"--label", label,
		"--concurrency-factor", concurrencyFactor,
	}, "upload bundle at "+dirPathStr(t, file), false)
	runCmd(t, []string{"bundle",
		"get",
		"--repo", repo1,
		"--label", label,
	}, "get bundle", false)
}

func TestGetLabel(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	label := rand.LetterString(8)
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
	}, "create first test repo", false)
	runCmd(t, []string{"label",
		"get",
		"--repo", repo1,
		"--label", label,
	}, "attempt get non-exist label", true)
	require.Equal(t, int(unix.ENOENT), exitMocks.exitStatuses[len(exitMocks.exitStatuses)-1],
		"ENOENT on nonexistant label")
	files := testUploadTrees[0]
	file := files[0]
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, file),
		"--message", "label test bundle",
		"--repo", repo1,
		"--label", label,
		"--concurrency-factor", concurrencyFactor,
	}, "upload bundle at "+dirPathStr(t, file), false)
	runCmd(t, []string{"label",
		"get",
		"--repo", repo1,
		"--label", label,
	}, "get label", false)
}

type labelListEntry struct {
	rawLine string
	name    string
	hash    string
	time    time.Time
}

func listLabels(t *testing.T, repoName string, prefix string) []labelListEntry {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	log.SetOutput(w)
	runCmd(t, []string{"label",
		"list",
		"--repo", repoName,
		"--prefix", prefix,
	}, "list labels", false)
	log.SetOutput(os.Stdout)
	w.Close()
	//
	lb, err := ioutil.ReadAll(r)
	require.NoError(t, err, "i/o error reading patched log from pipe")
	lles := make([]labelListEntry, 0)
	for _, line := range getDataLogLines(t, string(lb), []string{}) {
		sl := strings.Split(line, ",")
		time, err := time.Parse(timeForm, strings.TrimSpace(sl[2]))
		require.NoError(t, err, "couldn't parse label list time")
		lle := labelListEntry{
			rawLine: line,
			name:    strings.TrimSpace(sl[0]),
			hash:    strings.TrimSpace(sl[1]),
			time:    time,
		}
		lles = append(lles, lle)
	}
	return lles
}

func TestListLabels(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
	}, "create first test repo", false)
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo2,
	}, "create second test repo", false)
	ll := listLabels(t, repo1, "")
	require.Equal(t, len(ll), 0, "no labels created yet")
	ll = listLabels(t, repo2, "")
	require.Equal(t, 0, len(ll), "no labels created yet")
	file := testUploadTrees[0][0]
	label := rand.LetterString(8)
	testNow := time.Now()
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, file),
		"--message", "label test bundle",
		"--repo", repo1,
		"--label", label,
		"--concurrency-factor", concurrencyFactor,
	}, "upload bundle at "+dirPathStr(t, file), false)

	ll = listLabels(t, repo2, "")
	require.Equal(t, 0, len(ll), "no labels in secondary repo")

	ll = listLabels(t, repo1, "")
	require.Equal(t, 1, len(ll), "label count in test repo")

	labelEnt := ll[0]
	require.Equal(t, label, labelEnt.name, "found expected name in label entry")
	require.True(t, testNow.Sub(labelEnt.time).Seconds() < 3, "timestamp bounds after bundle create")

	ll = listLabels(t, repo1, label[:2])
	require.Equal(t, 1, len(ll), "label count in test repo")

	labelEnt = ll[0]
	require.Equal(t, label, labelEnt.name, "found expected name in label entry")
	require.True(t, testNow.Sub(labelEnt.time).Seconds() < 3, "timestamp bounds after bundle create")

	bundles, err := listBundles(t, repo1)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, 1, bundles.Len(), "bundle count in test repo")
	bundleEnt := bundles.Last()
	require.Equal(t, labelEnt.hash, bundleEnt.hash, "found unexpected hash in label entry")
}

func TestSetLabel(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
	}, "create first test repo", false)
	ll := listLabels(t, repo1, "")
	require.Equal(t, len(ll), 0, "no labels created yet")
	file := testUploadTrees[0][0]
	msg := rand.LetterString(15)
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, file),
		"--message", msg,
		"--repo", repo1,
		"--concurrency-factor", concurrencyFactor,
	}, "upload bundle at "+dirPathStr(t, file), false)

	ll = listLabels(t, repo1, "")
	require.Equal(t, len(ll), 0, "no labels created yet")

	bundles, err := listBundles(t, repo1)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, 1, bundles.Len(), "bundle count in test repo")

	bundleEnt := bundles.Last()
	label := rand.LetterString(8)
	testNow := time.Now()
	runCmd(t, []string{"label",
		"set",
		"--label", label,
		"--bundle", bundleEnt.hash,
		"--repo", repo1,
	}, "set bundle label", false)
	ll = listLabels(t, repo1, "")
	require.NoError(t, err, "error out of listLabels() test helper")
	require.Equal(t, 1, len(ll), "label count in test repo")
	labelEnt := ll[0]
	require.True(t, testNow.Sub(labelEnt.time).Seconds() < 3, "timestamp bounds after label set")
	require.True(t, bundleEnt.time.Sub(labelEnt.time) < 0, "label set timestamp later than bundle upload timestamp")
	require.Equal(t, label, labelEnt.name, "label entry name")
	require.Equal(t, bundleEnt.hash, labelEnt.hash, "label entry hash")
}

func testDownloadBundle(t *testing.T, files []uploadTree, bcnt int) {
	msg := rand.LetterString(15)
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, files[0]),
		"--message", msg,
		"--repo", repo1,
		"--concurrency-factor", concurrencyFactor,
	}, "upload bundle at "+dirPathStr(t, files[0]), false)
	bundles, err := listBundles(t, repo1)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, bcnt, bundles.Len(), "bundle count in test repo")
	t.Logf("bundles listed: %v", bundles)
	//
	destFS := afero.NewBasePathFs(afero.NewOsFs(), consumedData)
	bundle := bundles.Last()
	dpc := "bundle-dl-" + bundle.hash
	dp, err := filepath.Abs(filepath.Join(consumedData, dpc))
	require.NoError(t, err, "couldn't build file path")
	exists, err := afero.Exists(destFS, dpc)
	require.NoError(t, err, "error out of afero upstream library.  possibly programming error in test.")
	require.False(t, exists, "no filesystem entry at destination path '"+dpc+"' before bundle upload")
	runCmd(t, []string{"bundle",
		"download",
		"--repo", repo1,
		"--destination", dp,
		"--bundle", bundle.hash,
		"--concurrency-factor", concurrencyFactor,
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
	}, "create second test repo", false)
	for i, tree := range testUploadTrees {
		testDownloadBundle(t, tree, i+1)
	}
}

func TestDownloadBundleNameFilter(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
	}, "create test repo", false)

	files := testUploadTrees[1]
	msg := rand.LetterString(15)
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, files[0]),
		"--message", msg,
		"--repo", repo1,
		"--concurrency-factor", concurrencyFactor,
	}, "upload bundle at "+dirPathStr(t, files[0]), false)
	bundles, err := listBundles(t, repo1)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, 1, bundles.Len(), "bundle count in test repo")
	//
	destFS := afero.NewBasePathFs(afero.NewOsFs(), consumedData)
	bundle := bundles.Last()
	dpc := "bundle-dl-" + bundle.hash
	dp, err := filepath.Abs(filepath.Join(consumedData, dpc))
	require.NoError(t, err, "couldn't build file path")
	exists, err := afero.Exists(destFS, dpc)
	require.NoError(t, err, "error out of afero upstream library.  possibly programming error in test.")
	require.False(t, exists, "no filesystem entry at destination path '"+dpc+"' before bundle upload")
	runCmd(t, []string{"bundle",
		"download",
		"--repo", repo1,
		"--destination", dp,
		"--bundle", bundle.hash,
		"--name-filter", "zer",
		"--concurrency-factor", concurrencyFactor,
	}, "download bundle uploaded from "+dirPathStr(t, files[0]), false)
	exists, err = afero.Exists(destFS, dpc)
	require.NoError(t, err, "error out of afero upstream library.  possibly programming error in test.")
	require.True(t, exists, "filesystem entry at at destination path '"+dpc+"' after bundle upload")
	//
	fileInfos := make(map[string]os.FileInfo)
	err = afero.Walk(destFS, dpc, func(path string, info os.FileInfo, err error) error {
		splitPath := strings.Split(path, string(os.PathSeparator))
		var isMetaFile bool
		for _, pathComponent := range splitPath {
			if pathComponent == ".datamon" {
				isMetaFile = true
				break
			}
		}
		t.Logf("file name %q has %d path components (%v) meta", path, len(splitPath), isMetaFile)
		if len(splitPath) > 1 && !isMetaFile {
			fileInfos[strings.Join(splitPath[1:], string(os.PathSeparator))] = info
		}
		return nil
	})
	require.NoError(t, err, "walk destination fs for download files list")

	t.Logf("after download, dest has file infos %v", fileInfos)

	require.Equal(t, len(fileInfos), 1,
		"name filter regexp removed entries")
	_, hasName := fileInfos["zero"]
	require.True(t, hasName, "found expected entry given regexp")

}

type diffEntrySide struct {
	hash string
	size string
}

type diffEntry struct {
	rawLine    string
	typeLetter string
	name       string
	remote     diffEntrySide
	local      diffEntrySide
}

func TestDiffBundle(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	files := testUploadTrees[1]
	label := rand.LetterString(8)
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
	}, "create second test repo", false)
	//
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, files[0]),
		"--message", "diff orig",
		"--label", label,
		"--repo", repo1,
		"--concurrency-factor", "20",
	}, "upload bundle at "+dirPathStr(t, files[0]), false)
	//
	destFS := afero.NewBasePathFs(afero.NewOsFs(), consumedData)
	dpc := "bundle-diff"
	dp, err := filepath.Abs(filepath.Join(consumedData, dpc))
	require.NoError(t, err, "couldn't build file path")
	exists, err := afero.Exists(destFS, dpc)
	require.NoError(t, err, "error out of afero upstream library.  possibly programming error in test.")
	require.False(t, exists, "no filesystem entry at destination path '"+dpc+"' before bundle upload")
	runCmd(t, []string{"bundle",
		"download",
		"--repo", repo1,
		"--destination", dp,
		"--label", label,
		"--concurrency-factor", concurrencyFactor,
	}, "download bundle uploaded from "+dirPathStr(t, files[0]), false)
	exists, err = afero.Exists(destFS, dpc)
	require.NoError(t, err, "error out of afero upstream library.  possibly programming error in test.")
	require.True(t, exists, "filesystem entry at at destination path '"+dpc+"' after bundle upload")
	//
	srcFS := afero.NewBasePathFs(afero.NewOsFs(), dirPathStr(t, files[0]))
	delFile := files[0]
	difFile := files[1]
	addFileName := "add"
	f, err := srcFS.OpenFile(addFileName, os.O_CREATE|os.O_WRONLY|os.O_SYNC|0600, 0600)
	require.NoError(t, err, "open file to add")
	_, err = f.WriteString("additional")
	require.NoError(t, err, "write to additional file")
	require.NoError(t, f.Close(), "close add file")
	require.NoError(t, srcFS.Remove(pathInBundle(delFile)), "remove file from downloaded bundle")
	require.NoError(t, srcFS.Remove(pathInBundle(difFile)), "remove file to change")
	f, err = srcFS.OpenFile(pathInBundle(difFile), os.O_CREATE|os.O_WRONLY|os.O_SYNC|0600, 0600)
	require.NoError(t, err, "open file to change")
	_, err = f.WriteString(rand.String(10))
	require.NoError(t, err, "write to changed file")
	require.NoError(t, f.Close(), "close changed file")
	//
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, files[0]),
		"--message", "diff update",
		"--label", label,
		"--repo", repo1,
		"--concurrency-factor", concurrencyFactor,
	}, "upload bundle at "+dirPathStr(t, files[0]), false)
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	log.SetOutput(w)
	runCmd(t, []string{"bundle",
		"diff",
		"--repo", repo1,
		"--destination", dp,
		"--label", label,
		"--concurrency-factor", concurrencyFactor,
	}, "download bundle uploaded from "+dirPathStr(t, files[0]), false)
	log.SetOutput(os.Stdout)
	w.Close()
	//
	lb, err := ioutil.ReadAll(r)
	require.NoError(t, err, "i/o error reading patched log from pipe")
	des := make([]diffEntry, 0)
	for _, line := range getDataLogLines(t, string(lb), []string{}) {
		if strings.HasPrefix(line, "Using bundle") {
			continue
		}
		sl := strings.Split(line, ",")
		require.Truef(t, len(sl) > 4, "expected at least 5 comma separated items in output, got: %s", line)
		de := diffEntry{
			rawLine:    line,
			typeLetter: strings.TrimSpace(sl[0]),
			name:       strings.TrimSpace(sl[1]),
			local: diffEntrySide{
				size: strings.TrimSpace(sl[2]),
				hash: strings.TrimSpace(sl[3]),
			},
			remote: diffEntrySide{
				size: strings.TrimSpace(sl[4]),
				hash: strings.TrimSpace(sl[5]),
			},
		}
		des = append(des, de)
	}
	require.Equal(t, len(des), 3, "found expected number of diff entries")
	desm := make(map[string]diffEntry)
	for _, de := range des {
		desm[de.name] = de
	}
	require.Equal(t, len(desm), 3, "diff entries have unique names")
	de, ok := desm[addFileName]
	require.True(t, ok, "found add file entry")
	require.Equal(t, de.typeLetter, "A", "found correct type on add file entry")
	require.Equal(t, de.remote.hash, "", "no remote hash on add file entry")
	require.NotEqual(t, de.local.hash, "", "local hash on add file entry")
	de, ok = desm[pathInBundle(difFile)]
	require.True(t, ok, "found update file entry")
	require.Equal(t, de.typeLetter, "U", "found correct type on update file entry")
	require.NotEqual(t, de.remote.hash, "", "remote hash on update file entry")
	require.NotEqual(t, de.local.hash, "", "local hash on update file entry")
	de, ok = desm[pathInBundle(delFile)]
	require.True(t, ok, "found delete file entry")
	require.Equal(t, de.typeLetter, "D", "found correct type on delete file entry")
	require.NotEqual(t, de.remote.hash, "", "remote hash on delete file entry")
	require.Equal(t, de.local.hash, "", "no local hash on delete file entry")
}

func TestUpdateBundle(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	files := testUploadTrees[1]
	label := rand.LetterString(8)
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
	}, "create second test repo", false)
	//
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, files[0]),
		"--message", "diff orig",
		"--label", label,
		"--repo", repo1,
		"--concurrency-factor", "20",
	}, "upload bundle at "+dirPathStr(t, files[0]), false)
	//
	destFS := afero.NewBasePathFs(afero.NewOsFs(), consumedData)
	dpc := "bundle-update"
	dp, err := filepath.Abs(filepath.Join(consumedData, dpc))
	require.NoError(t, err, "couldn't build file path")
	exists, err := afero.Exists(destFS, dpc)
	require.NoError(t, err, "error out of afero upstream library.  possibly programming error in test.")
	require.False(t, exists, "no filesystem entry at destination path '"+dpc+"' before bundle upload")
	runCmd(t, []string{"bundle",
		"download",
		"--repo", repo1,
		"--destination", dp,
		"--label", label,
		"--concurrency-factor", "20",
	}, "download bundle uploaded from "+dirPathStr(t, files[0]), false)
	exists, err = afero.Exists(destFS, dpc)
	require.NoError(t, err, "error out of afero upstream library.  possibly programming error in test.")
	require.True(t, exists, "filesystem entry at at destination path '"+dpc+"' after bundle upload")
	//
	srcFS := afero.NewBasePathFs(afero.NewOsFs(), dirPathStr(t, files[0]))
	delFile := files[0]
	difFile := files[1]
	addFileName := "additional"
	f, err := srcFS.OpenFile(addFileName, os.O_CREATE|os.O_WRONLY|os.O_SYNC|0600, 0600)
	require.NoError(t, err, "open file to add")
	_, err = f.WriteString("additional")
	require.NoError(t, err, "write to additional file")
	require.NoError(t, f.Close(), "close add file")
	require.NoError(t, srcFS.Remove(pathInBundle(delFile)), "remove file from downloaded bundle")
	require.NoError(t, srcFS.Remove(pathInBundle(difFile)), "remove file to change")
	f, err = srcFS.OpenFile(pathInBundle(difFile), os.O_CREATE|os.O_WRONLY|os.O_SYNC|0600, 0600)
	require.NoError(t, err, "open file to change")
	_, err = f.WriteString(rand.String(10))
	require.NoError(t, err, "write to changed file")
	require.NoError(t, f.Close(), "close changed file")
	//
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, files[0]),
		"--message", "diff update",
		"--label", label,
		"--repo", repo1,
		"--concurrency-factor", "20",
	}, "upload bundle at "+dirPathStr(t, files[0]), false)
	runCmd(t, []string{"bundle",
		"update",
		"--repo", repo1,
		"--destination", dp,
		"--label", label,
		"--concurrency-factor", "20",
	}, "download bundle uploaded from "+dirPathStr(t, files[0]), false)

	ddpc := "bundle-download"
	ddp, err := filepath.Abs(filepath.Join(consumedData, ddpc))
	require.NoError(t, err, "couldn't build file path")
	datamonFlags.bundle.ID = ""
	runCmd(t, []string{"bundle",
		"download",
		"--repo", repo1,
		"--destination", ddp,
		"--label", label,
		"--concurrency-factor", "20",
	}, "download bundle uploaded from "+dirPathStr(t, files[0]), false)

	destUpdateFS := afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(consumedData, dpc))
	destDownloadFS := afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(consumedData, ddpc))

	udPaths := make(map[string]bool)
	err = afero.Walk(destUpdateFS, "", func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err, "error walking upload destination path")
		if path != "" && !info.IsDir() {
			udPaths[path] = true

			exists, err = afero.Exists(destDownloadFS, path)
			require.NoError(t, err, "error out of afero upstream library.  possibly programming error in test.")
			require.True(t, exists, "found path in download dir")
		}

		return nil
	})
	require.NoError(t, err, "walking update destination")

	err = afero.Walk(destDownloadFS, "", func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err, "error walking upload destination path")
		if path != "" && !info.IsDir() {
			require.True(t, udPaths[path], "download path exists in update")
		}
		return nil
	})
	require.NoError(t, err, "walking download destination")

	for path := range udPaths {

		df, err := destDownloadFS.Open(path)
		require.NoError(t, err, "open downloaded file")
		dv, err := ioutil.ReadAll(df)
		require.NoError(t, err, "read downloaded file")

		uf, err := destUpdateFS.Open(path)
		require.NoError(t, err, "open updated file")
		uv, err := ioutil.ReadAll(uf)
		require.NoError(t, err, "read downloaded file")

		t.Logf("comparing update and destination files at path: '%v'", path)

		require.Equal(t, string(dv), string(uv), "downloaded file matches updated file '"+path+"'")
	}

}

func TestDownloadBundleByLabel(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
	}, "create test repo", false)
	files := testUploadTrees[0]
	file := files[0]
	label := rand.LetterString(8)
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, file),
		"--message", "label test bundle",
		"--repo", repo1,
		"--label", label,
		"--concurrency-factor", concurrencyFactor,
	}, "upload bundle at "+dirPathStr(t, file), false)
	// dupe: testDownloadBundle()
	destFS := afero.NewBasePathFs(afero.NewOsFs(), consumedData)
	dpc := "bundle-dl-" + label
	dp, err := filepath.Abs(filepath.Join(consumedData, dpc))
	if err != nil {
		t.Errorf("couldn't build file path: %v", err)
	}
	exists, err := afero.Exists(destFS, dpc)
	require.NoError(t, err, "error out of afero upstream library.  possibly programming error in test.")
	require.False(t, exists, "no filesystem entry at destination path '"+dpc+"' before bundle download")
	runCmd(t, []string{"bundle",
		"download",
		"--repo", repo1,
		"--destination", dp,
		"--label", label,
		"--concurrency-factor", concurrencyFactor,
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
	log.SetOutput(w)
	runCmd(t, []string{"bundle",
		"list",
		"files",
		"--repo", repoName,
		"--bundle", bid,
	}, "get bundle files list", false)
	//
	log.SetOutput(os.Stdout)
	w.Close()
	//
	lb, err := ioutil.ReadAll(r)
	require.NoError(t, err, "i/o error reading patched stdout from pipe")
	t.Logf("output collected: %s", string(lb))
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
	msg := rand.LetterString(15)
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, files[0]),
		"--message", msg,
		"--repo", repo1,
		"--concurrency-factor", concurrencyFactor,
	}, "create second test repo", false)

	bundles, err := listBundles(t, repo1)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, bcnt, bundles.Len(), "bundle count in test repo")
	bundle := bundles.Last()
	bfles := listBundleFiles(t, repo1, bundle.hash)
	require.Equalf(t, len(files), len(bfles), "file count in bundle files list log [bundle: %v]", bundle)
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
	msg := rand.LetterString(15)
	runCmd(t, []string{"bundle",
		"upload",
		"--path", dirPathStr(t, files[0]),
		"--message", msg,
		"--repo", repo1,
		"--concurrency-factor", concurrencyFactor,
	}, "upload bundle in order to test downloading individual files", false)
	bundles, err := listBundles(t, repo1)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, bcnt, bundles.Len(), "bundle count in test repo")
	//
	bundle := bundles.Last()
	for _, file := range files {
		testBundleDownloadFile(t, file, bundle.hash)
	}
}

func TestBundlesDownloadFiles(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo1,
	}, "create repo", false)

	testBundleDownloadFiles(t, testUploadTrees[0], 1)
	testBundleDownloadFiles(t, testUploadTrees[1], 2)
	testBundleDownloadFiles(t, testUploadTrees[2], 3)
}

/** untested:
 * - config_generate.go
 */

func createAllTestUploadTrees(t *testing.T) {
	sourceFS := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), sourceData))
	for _, tree := range testUploadTrees {
		createTestUploadTreeHelper(t, sourceFS, tree, 1)
	}
}

func createTestUploadTree(t *testing.T, pathRoot string, tree []uploadTree) {
	sourceFS := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), pathRoot))
	createTestUploadTreeHelper(t, sourceFS, tree, 2)
}

func createTestUploadTreeHelper(t *testing.T, sourceFS storage.Store, tree []uploadTree, rc int) {
	for _, file := range tree {
		var err error
		for i := 0; i < rc; i++ {
			err = sourceFS.Put(context.Background(),
				file.path,
				bytes.NewReader(rand.Bytes(file.size)),
				storage.NoOverWrite)
			if err == nil {
				break
			}
		}
		require.NoError(t, err)
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
	 *
	 * TODO(fred): filepath.Abs(filepath.FromSlash(path.Join(sourceData strings.Split(file.path)[1])))
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
	gcs, err := gcs.New(context.TODO(), bucketName, "") // Use GOOGLE_APPLICATION_CREDENTIALS env variable
	require.NoError(t, err, "failed to create gcs client")
	var keys []string
	keys, err = gcs.Keys(ctx)
	require.NoError(t, err, "get object names created during test")
	for _, key := range keys {
		err = gcs.Delete(ctx, key)
		require.NoError(t, err, "failed to delete:"+key)
	}
	if err := mb.Delete(ctx); err != nil {
		t.Errorf("error deleting bucket %v", err)
	}
}

/* for tests that need to read stdout into data structures, this function converts
 * a string to a slice of lines, each of which can be parsed into a struct.
 * TODO: Refactor using bufio and bufio.Scanner. See examples in cli_fuse_test.go
 */
func getDataLogLines(t testing.TB, ls string, ignorePatterns []string) []string {
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

func TestDeleteRepo(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()
	const (
		repo  = "repoToDelete"
		label = "labelToDelete"
	)

	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo,
	}, "create test repo to delete", false)

	for i, tree := range testUploadTrees {
		runCmd(t, []string{"bundle",
			"upload",
			"--path", dirPathStr(t, tree[0]),
			"--message", "The initial commit for the repo",
			"--repo", repo,
			"--label", label + strconv.Itoa(i),
			"--concurrency-factor", "1",
		}, "upload bundle at "+dirPathStr(t, tree[0]), false)
	}

	runCmd(t, []string{"repo",
		"delete",
		"files",
		"--file",
		"leafsize",
		"--repo", repo,
	}, "delete file from test repo", false)

	for i := range testUploadTrees {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		runCmd(t, []string{"bundle",
			"list",
			"files",
			"--repo", repo,
			"--label", label + strconv.Itoa(i),
		}, "list files after single file delete", false)

		log.SetOutput(w)
		log.SetOutput(os.Stdout)
		w.Close()

		lb, err := ioutil.ReadAll(r)
		require.NoError(t, err, "i/o error reading patched log from pipe")
		for _, line := range getDataLogLines(t, string(lb), []string{}) {
			assert.NotContains(t, line, "name:leafsize")
		}
	}

	filelist, filelistCleanup := getTempFile(t, "to_delete.txt")
	defer filelistCleanup()
	_, err := filelist.WriteString("/1/2/3/4/5/6/7/deeper\n")
	require.NoError(t, err)
	_, err = filelist.WriteString("/1/2/3/4/5/6/deep\n")
	require.NoError(t, err)

	runCmd(t, []string{"repo",
		"delete",
		"files",
		"--files",
		filelist.Name(),
		"--repo", repo,
	}, "delete files from test repo", false)

	for i := range testUploadTrees {
		r, w, e := os.Pipe()
		require.NoError(t, e)
		runCmd(t, []string{"bundle",
			"list",
			"files",
			"--repo", repo,
			"--label", label + strconv.Itoa(i),
		}, "list files after single file delete", false)

		log.SetOutput(w)
		log.SetOutput(os.Stdout)
		w.Close()

		lb, e := ioutil.ReadAll(r)
		require.NoError(t, e, "i/o error reading patched log from pipe")
		for _, line := range getDataLogLines(t, string(lb), []string{}) {
			assert.NotContains(t, line, "name:/1/2/3/4/5/6/7/deeper")
			assert.NotContains(t, line, "name:/1/2/3/4/5/6/deep")
		}
	}

	runCmd(t, []string{"repo",
		"delete",
		"--repo", repo,
	}, "delete test repo", false)

	r, w, err := os.Pipe()
	require.NoError(t, err)

	runCmd(t, []string{"repo",
		"list",
	}, "verify repo is deleted", false)

	log.SetOutput(w)
	log.SetOutput(os.Stdout)
	w.Close()
	lb, err := ioutil.ReadAll(r)
	require.NoError(t, err, "i/o error reading patched log from pipe")
	for _, line := range getDataLogLines(t, string(lb), []string{}) {
		assert.NotContains(t, line, repo)
	}

	runCmd(t, []string{"label",
		"list",
		"--repo", repo,
	}, "verify labels are deleted", true)
}
