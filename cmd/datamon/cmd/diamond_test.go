package cmd

import (
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiamondCommit(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()

	repo := generateRepoName("diamond")

	runOK(t, "repo", "create", "--description", "testing", "--repo", repo)

	r, w := startCapture(t)
	runOK(t, "diamond", "initialize", "--repo", repo)

	diamondID := wantsKSUID(t, r, w)

	root := filePathStr(t, uploadTree{path: "/"})
	splits := make([]string, 0, len(testUploadTrees))
	for _, tree := range testUploadTrees {
		pth := tree[0].Root()
		r, w = startCapture(t)
		runOK(t, "diamond", "split", "add", "--repo", repo, "--diamond", diamondID, "--path", root, "--name-filter", pth)

		splitID := wantsKSUID(t, r, w)
		splits = append(splits, splitID)
	}

	r, w = startCapture(t)
	runOK(t, "diamond", "split", "list", "--repo", repo, "--diamond", diamondID)

	lines := endCapture(t, r, w, []string{})
	require.Len(t, lines, len(testUploadTrees))
	for i, line := range lines {
		split := firstField(line)
		assert.Equal(t, splits[i], split)

		r, w = startCapture(t)
		runOK(t, "diamond", "split", "get", "--repo", repo, "--diamond", diamondID, "--split", split)

		splitID := wantsOneLineFirstField(t, r, w)
		assert.Equal(t, split, splitID)
	}

	r, w = startCapture(t)
	runOK(t, "diamond", "commit", "--repo", repo, "--diamond", diamondID, "--message", "commit test message")

	_ = wantsBundleUploaded(t, r, w)
}

func TestDiamondList(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()

	repo := generateRepoName("diamond")

	const numDiamonds = 3
	diamonds := make([]string, 0, numDiamonds)

	runOK(t, "repo", "create", "--description", "testing", "--repo", repo)

	for i := 0; i < numDiamonds; i++ {
		r, w := startCapture(t)
		runOK(t, "diamond", "initialize", "--repo", repo)

		diamondID := wantsKSUID(t, r, w)
		diamonds = append(diamonds, diamondID)
	}

	r, w := startCapture(t)
	runOK(t, "diamond", "list", "--repo", repo)

	lines := endCapture(t, r, w, []string{})
	require.Len(t, lines, numDiamonds)

	for i, line := range lines {
		diamondID := firstField(line)
		assert.Equal(t, diamonds[i], diamondID)
	}

	for _, diamondID := range diamonds {
		runOK(t, "diamond", "cancel", "--repo", repo, "--diamond", diamondID)

		r, w := startCapture(t)
		runOK(t, "diamond", "get", "--repo", repo, "--diamond", diamondID)

		ID := wantsOneLineFirstField(t, r, w)
		assert.Equal(t, diamondID, ID)
	}
}

func TestDiamondUserSupplied(t *testing.T) {
	// test user suppled splitID
	cleanup := setupTests(t)
	defer cleanup()

	repo := generateRepoName("diamond")

	runOK(t, "repo", "create", "--description", "testing", "--repo", repo)

	r, w := startCapture(t)
	runOK(t, "diamond", "initialize", "--repo", repo)

	diamondID := wantsKSUID(t, r, w)

	root := filePathStr(t, uploadTree{path: "/"})
	rexSplit := regexp.MustCompile(`SPLIT-\d+`)
	splits := make([]string, 0, len(testUploadTrees))
	for i, tree := range testUploadTrees {
		pth := tree[0].Root()
		r, w = startCapture(t)
		runOK(t, "diamond", "split", "add", "--repo", repo, "--diamond", diamondID, "--split", "SPLIT-"+strconv.Itoa(i), "--path", root, "--name-filter", pth)

		// returns user supplied ID
		lines := endCapture(t, r, w, []string{})
		require.Len(t, lines, 1)
		splitID := lines[0]
		require.Regexp(t, rexSplit, splitID)
		splits = append(splits, splitID)
	}

	r, w = startCapture(t)
	runOK(t, "diamond", "split", "list", "--repo", repo, "--diamond", diamondID)

	lines := endCapture(t, r, w, []string{})
	require.Len(t, lines, len(testUploadTrees))
	for i, line := range lines {
		assert.Equal(t, firstField(line), "SPLIT-"+strconv.Itoa(i))
	}

	for i, line := range lines {
		split := firstField(line)
		assert.Equal(t, splits[i], split)

		r, w = startCapture(t)
		runOK(t, "diamond", "split", "get", "--repo", repo, "--diamond", diamondID, "--split", split)

		splitID := wantsOneLineFirstField(t, r, w)
		assert.Equal(t, split, splitID)
	}

	r, w = startCapture(t)
	runOK(t, "diamond", "commit", "--repo", repo, "--diamond", diamondID, "--message", "commit test message")

	_ = wantsBundleUploaded(t, r, w)
}

func startCapture(t testing.TB) (io.Reader, io.Closer) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	log.SetOutput(w)
	return r, w
}

func endCapture(t testing.TB, r io.Reader, w io.Closer, excludedPatterns []string) []string {
	log.SetOutput(os.Stdout)
	w.Close()
	lb, err := ioutil.ReadAll(r)
	require.NoError(t, err, "i/o error reading patched log from pipe")
	return getDataLogLines(t, string(lb), excludedPatterns)
}

func runOK(t *testing.T, args ...string) {
	runCmd(t, args, strings.Join(args, " "), false)
}

func wantsKSUID(t testing.TB, r io.Reader, w io.Closer) string {
	lines := endCapture(t, r, w, []string{})
	require.Len(t, lines, 1)
	_, err := ksuid.Parse(lines[0])
	require.NoError(t, err)
	return lines[0]
}

func wantsBundleUploaded(t testing.TB, r io.Reader, w io.Closer) string {
	lines := endCapture(t, r, w, []string{})
	require.Len(t, lines, 1)

	bundleUploadRex := regexp.MustCompile(`Uploaded bundle id:(\w+)`)
	matches := bundleUploadRex.FindStringSubmatch(lines[0])
	require.Len(t, matches, 2)
	bundleID := matches[1]
	_, err := ksuid.Parse(bundleID)
	require.NoError(t, err)
	return bundleID
}

func firstField(line string) string {
	return strings.Split(line, ",")[0] // to generalize this, one should trim the result
}

func wantsOneLineFirstField(t testing.TB, r io.Reader, w io.Closer) string {
	lines := endCapture(t, r, w, []string{})
	require.Len(t, lines, 1)
	return firstField(lines[0])
}
