// +build fuse_cli

package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	gcsStorage "cloud.google.com/go/storage"
	"github.com/PuerkitoBio/goquery"
	"github.com/oneconcern/datamon/internal"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type testFile struct {
	data                        []byte
	size                        int
	listed, downloaded, mounted bool
}

// TestWorkshop goes through all the steps in our workshop.
//
// Test is run with spawned process on freshly built datamon binary, not test goroutine: no mocks, no tricks
func TestWorkshop(t *testing.T) {
	logger := testLogger(t)
	logger.Info("test initialization")
	repo := generateRepoName("workshop")
	targetBinary := buildTestBinary(t)

	// create config
	cmd, _ := testCommand(t, false, false, targetBinary,
		"config",
		"set",
		"--config", "workshop-config",
		"--credential", os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
	)
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "could not create config: %s", string(output))
	// TODO(fred): should separate assertion on stdout and stderr here
	assert.Contains(t, string(output), "config file created in")

	// obtain available contexts
	logger.Info("listing contexts")
	cmd, _ = testCommand(t, false, false, targetBinary,
		"context",
		"list",
	)

	output, err = cmd.CombinedOutput()
	require.NoErrorf(t, err, "could not list contexts, got: %s", string(output))
	// TODO(fred): should separate assertion on stdout and stderr here
	assert.Contains(t, string(output), "[dev]")

	// create repo
	cmd, _ = testCommand(t, false, false, targetBinary,
		"repo",
		"create",
		"--repo", repo,
		"--description", "workshop CI test repo",
	)
	output, err = cmd.CombinedOutput()
	require.NoErrorf(t, err, "got: %s", string(output))

	defer deleteRepoMeta(t, repo)

	// list repos
	cmd, _ = testCommand(t, false, false, targetBinary,
		"repo",
		"list",
	)
	output, err = cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(output), repo)

	// repo detail
	cmd, _ = testCommand(t, false, false, targetBinary,
		"repo",
		"get",
		"--repo", repo,
	)
	output, err = cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(output), repo)
	assert.Contains(t, string(output), "workshop CI test repo")

	// upload a bundle
	pathToData, _ := ioutil.TempDir("", "bdle-workshop-")
	defer os.RemoveAll(pathToData)
	msg := internal.RandStringBytesMaskImprSrc(15)

	files := makeTestWorkshopBundle(t, pathToData)

	cmd, _ = testCommand(t, false, false, targetBinary,
		"bundle",
		"upload",
		"--repo", repo,
		"--message", msg,
		"--path", pathToData,
	)
	output, err = cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(output), "Uploaded bundle id:")

	rex := regexp.MustCompile(`Uploaded bundle id:\s?([^\s]+)`)
	match := rex.FindSubmatch(output)
	require.Truef(t, len(match) > 1, "couldn't find bundle ID in output, got this instead: %s", string(output))
	bundleID := string(match[1])

	// list bundles
	cmd, _ = testCommand(t, false, false, targetBinary,
		"bundle",
		"list",
		"--repo", repo,
	)
	output, err = cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(output), bundleID)
	assert.Contains(t, string(output), msg)

	// set label
	label := internal.RandStringBytesMaskImprSrc(10)
	cmd, _ = testCommand(t, false, false, targetBinary,
		"label",
		"set",
		"--repo", repo,
		"--bundle", bundleID,
		"--label", label,
	)
	err = cmd.Run()
	require.NoError(t, err)

	// list labels
	cmd, _ = testCommand(t, false, false, targetBinary,
		"label",
		"list",
		"--repo", repo,
	)
	output, err = cmd.Output()
	require.NoError(t, err)
	rex1 := regexp.MustCompile(`(?m)` + regexp.QuoteMeta(label) + `\s*,\s*` + regexp.QuoteMeta(bundleID))
	require.Truef(t, rex1.Match(output), "output not matching: %s", string(output))

	// additional label
	extraLabel := internal.RandStringBytesMaskImprSrc(10)
	cmd, _ = testCommand(t, false, false, targetBinary,
		"label",
		"set",
		"--repo", repo,
		"--bundle", bundleID,
		"--label", extraLabel,
	)
	err = cmd.Run()
	require.NoError(t, err)

	cmd, _ = testCommand(t, false, false, targetBinary,
		"label",
		"list",
		"--repo", repo,
	)
	output, err = cmd.Output()
	require.NoError(t, err)
	require.True(t, rex1.Match(output))

	rex2 := regexp.MustCompile(`(?m)` + regexp.QuoteMeta(extraLabel) + `\s*,\s*` + regexp.QuoteMeta(bundleID))
	require.Truef(t, rex2.Match(output), "output not matching: %s", string(output))

	// prefix based label search
	cmd, _ = testCommand(t, false, false, targetBinary,
		"label",
		"list",
		"--repo", repo,
		"--prefix", extraLabel[:5],
	)
	output, err = cmd.Output()
	require.NoError(t, err)
	require.False(t, rex1.Match(output))
	require.Truef(t, rex2.Match(output), "output not matching: %s", string(output))

	// query files
	cmd, _ = testCommand(t, false, false, targetBinary,
		"bundle",
		"list",
		"files",
		"--repo", repo,
		"--bundle", bundleID,
	)
	output, err = cmd.Output()
	require.NoError(t, err)

	rex3 := regexp.MustCompile(`name:\s*([^\s,]+)\s*,\s*size:\s*([^\s,]+)`)
	rexB := regexp.MustCompile("Using bundle")
	scanner := bufio.NewScanner(bytes.NewBuffer(output))
	for scanner.Scan() {
		line := scanner.Text()
		if rexB.MatchString(line) {
			continue
		}
		match := rex3.FindStringSubmatch(line)
		require.True(t, len(match) > 2)
		name := match[1]
		sizeAsStr := match[2]
		size, eri := strconv.Atoi(sizeAsStr)
		require.NoError(t, eri)

		require.Contains(t, files, name)
		files[name].listed = true
		assert.Equal(t, files[name].size, size)
	}
	for name, file := range files {
		assert.Truef(t, file.listed, "expected file %s to be listed", name)
	}

	// mounting the bundle RO
	pathBackingFs, _ := ioutil.TempDir("", "mmfs-")
	pathToMount, _ := ioutil.TempDir("", "mmp-")
	defer os.RemoveAll(pathBackingFs)
	defer os.RemoveAll(pathToMount)

	cmd, pipe := testCommand(t, true, false, targetBinary,
		"bundle",
		"mount",
		"--repo", repo,
		"--label", label,
		"--mount", pathToMount,
		"--destination", pathBackingFs,
	)
	require.NoError(t, cmd.Start())
	var killed bool
	defer func() {
		if !killed {
			_ = cmd.Process.Kill()
		}
	}()

	logger.Info("waiting for mount to be ready")
	testWaitForReader(t, `"mounting"`, 5*time.Second)(logger, pipe)

	err = filepath.Walk(pathToMount, func(pth string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}
		name := filepath.Base(pth)
		require.Contains(t, files, name)
		files[name].mounted = true
		assert.Equal(t, files[name].size, int(info.Size()))
		b, erf := ioutil.ReadFile(pth)
		require.NoError(t, erf)
		assert.EqualValues(t, files[name].data, b)
		return nil
	})
	require.NoError(t, err)
	for name, file := range files {
		assert.Truef(t, file.mounted, "expected file %s to be mounted", name)
	}

	logger.Info("unmounting")
	require.NoError(t, cmd.Process.Kill())
	err = cmd.Wait()
	killed = true
	require.Equal(t, "signal: killed", err.Error(), "cmd exit with killed error")

	// downloading the bundle
	pathToDownload, _ := ioutil.TempDir("", "mmd-")
	defer os.RemoveAll(pathToDownload)

	cmd, _ = testCommand(t, false, false, targetBinary,
		"bundle",
		"download",
		"--repo", repo,
		"--label", label,
		"--destination", pathToDownload,
	)
	output, err = cmd.CombinedOutput()
	require.NoErrorf(t, err, "unexpected error, got: %s", string(output))

	err = filepath.Walk(pathToDownload, func(pth string, info os.FileInfo, _ error) error {
		rel, _ := filepath.Rel(pathToDownload, pth)
		if info.IsDir() || model.IsGeneratedFile(rel) {
			return nil
		}
		name := filepath.Base(pth)
		require.Contains(t, files, name)
		files[name].downloaded = true
		assert.Equal(t, files[name].size, int(info.Size()))
		b, erf := ioutil.ReadFile(pth)
		require.NoError(t, erf)
		assert.EqualValues(t, files[name].data, b)
		return nil
	})
	require.NoError(t, err)
	for name, file := range files {
		assert.Truef(t, file.downloaded, "expected file %s to be downloaded", name)
	}

	// downloading some files
	pathToSomeDownload, _ := ioutil.TempDir("", "mmd-")
	defer os.RemoveAll(pathToSomeDownload)

	var picked string
	for n := range files {
		picked = n
		break
	}

	cmd, _ = testCommand(t, false, false, targetBinary,
		"bundle",
		"download",
		"--repo", repo,
		"--label", label,
		"--destination", pathToSomeDownload,
		"--name-filter", picked[:8]+".+",
	)
	output, err = cmd.CombinedOutput()
	require.NoErrorf(t, err, "unexpected error, got: %s", string(output))

	err = filepath.Walk(pathToSomeDownload, func(pth string, info os.FileInfo, _ error) error {
		rel, _ := filepath.Rel(pathToSomeDownload, pth)
		if info.IsDir() || model.IsGeneratedFile(rel) {
			return nil
		}
		name := filepath.Base(pth)
		require.Equal(t, picked, name)
		assert.Equal(t, files[name].size, int(info.Size()))
		b, erf := ioutil.ReadFile(pth)
		require.NoError(t, erf)
		assert.EqualValues(t, files[name].data, b)
		return nil
	})
	require.NoError(t, err)

	// no test here for upgrade (see self_upgrade_test.go)

	logger.Info("test web")
	freePort, err := testGetFreePort()
	require.NoError(t, err, "find free port")
	port := strconv.Itoa(freePort)

	cmd, pipe = testCommand(t, true, false, targetBinary,
		"web",
		"--port", port,
		"--no-browser",
	)

	require.NoError(t, cmd.Start())
	var killedWeb bool
	defer func() {
		if !killedWeb {
			_ = cmd.Process.Kill()
		}
	}()

	waiter := testWaitForReader(
		t, "serving datamon UI", 5*time.Second,
	)

	logger.Info("waiting for web process to be ready")
	waiter(logger, pipe)
	logger.Info("web process ready")

	resp, err := http.Get("http://localhost:" + port + "/")
	require.NoError(t, err, "curl web service")
	// notdo: replete test of entire http api
	//   (contained in `pkg/web` tests)

	require.NotNil(t, resp)

	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	require.NoError(t, err, "parse body html")

	require.NotNil(t, doc)

	logger.Info(fmt.Sprintf("home page:\n%v", doc.Text()))

	// smoke test only: no content validation
}

func makeTestWorkshopBundle(t testing.TB, pathToData string) map[string]*testFile {
	files := make(map[string]*testFile, 10)
	for i := 0; i < 9; i++ {
		name := internal.RandStringBytesMaskImprSrc(10)
		data := internal.RandBytesMaskImprSrc(100)
		err := ioutil.WriteFile(filepath.Join(pathToData, name), data, 0644)
		require.NoError(t, err)
		files[name] = &testFile{data: data, size: 100}
	}
	name := internal.RandStringBytesMaskImprSrc(10)
	data := []byte{} // empty file
	err := ioutil.WriteFile(filepath.Join(pathToData, name), data, 0644)
	require.NoError(t, err)
	files[name] = &testFile{data: data, size: 0}
	return files
}

func TestWorkshopBrewInstall(t *testing.T) {
	if runtime.GOOS == "linux" { // TODO: don't know where brew installs things on OSX
		home, _ := os.UserHomeDir()
		cmdPath := strings.Join([]string{
			filepath.Join("/", "home", "linuxbrew", ".linuxbrew", "bin"),
			filepath.Join(home, ".linuxbrew", "bin"),
			os.Getenv("PATH"),
		}, ":")
		os.Setenv("PATH", cmdPath)
	}

	// brew install (linux flavor)
	cmd := exec.Command("brew", "--version")
	if output, err := cmd.Output(); err != nil {
		t.Logf("brew not installed. Skipping test")
		t.SkipNow()
	} else {
		t.Logf("using brew: %s", string(output))
	}

	output, err := exec.Command("brew", "tap", "oneconcern/datamon").CombinedOutput()
	require.NoErrorf(t, err, "could not run brew tap: %v:\n%s", err, string(output))

	output, err = exec.Command("brew", "install", "datamon").CombinedOutput()
	require.NoErrorf(t, err, "could not run brew install: %v:\n%s", err, string(output))

	// version
	datamon, err := exec.LookPath("datamon2")
	require.NoErrorf(t, err, "datamon not found in path: %v", err)

	output, err = exec.Command(datamon, "version").CombinedOutput()
	require.NoErrorf(t, err, "could not run datamon: %s\n%s", err, string(output))
}

func deleteRepoMeta(t testing.TB, repo string) {
	// delete the test repo on gs://datamon-workshop
	var err error
	defer func() {
		if err != nil {
			t.Logf("WARNING:could not access metadata bucket to clean up test with repo %s...: %v", repo, err)
		}
	}()
	err = deleteMeta(t, "workshop-dev-meta", repo)
	if err != nil {
		t.Logf("WARNING: error encountered when cleaning up test with repo %s: %v", repo, err)
	}
	err = deleteVMeta(t, "workshop-dev-vmeta", repo)
}

func deleteMeta(t testing.TB, bucket, repo string) (err error) {
	ctx := context.Background()
	client, erc := gcsStorage.NewClient(ctx, option.WithScopes(gcsStorage.ScopeFullControl))
	if erc != nil {
		return erc
	}

	metaBucket := client.Bucket(bucket)

	metaRepo := metaBucket.Object(model.GetArchivePathToRepoDescriptor(repo))
	if metaRepo == nil {
		err = fmt.Errorf("could not find path to repo in metadata")
		return
	}

	name := metaRepo.ObjectName()
	t.Logf("about to delete %s", name)
	err = metaRepo.Delete(ctx)
	if err != nil {
		return
	}

	bundles := metaBucket.Objects(ctx, &gcsStorage.Query{Prefix: model.GetArchivePathPrefixToBundles(repo)})
	if bundles == nil {
		err = fmt.Errorf("could not find path to bundles for repo in metadata")
		return
	}

	for {
		bundleAttrs, eri := bundles.Next()
		if eri == iterator.Done {
			break
		}
		if eri != nil {
			return eri
		}
		bundle := metaBucket.Object(bundleAttrs.Name)
		name = bundle.ObjectName()
		t.Logf("about to delete %s", name)
		err = bundle.Delete(ctx)
		if err != nil {
			return
		}
	}
	return err
}

// lifted from `github.com/phayes/freeport`
// testGetFreePort asks the kernel for a free open port that is ready to use.
func testGetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	// this is the stdlib function that chooses a port,
	// where port 0 has semantic meaning.
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func deleteVMeta(t testing.TB, bucket, repo string) (err error) {
	ctx := context.Background()
	client, erc := gcsStorage.NewClient(ctx, option.WithScopes(gcsStorage.ScopeFullControl))
	if err != nil {
		return erc
	}

	vmetaBucket := client.Bucket(bucket)
	labels := vmetaBucket.Objects(ctx, &gcsStorage.Query{Prefix: model.GetArchivePathPrefixToLabels(repo)})
	if labels == nil {
		t.Logf("no label to clean in repo %s", repo)
		return
	}

	for {
		labelAttrs, eri := labels.Next()
		if eri == iterator.Done {
			break
		}
		if eri != nil {
			return eri
		}
		label := vmetaBucket.Object(labelAttrs.Name)
		name := label.ObjectName()
		t.Logf("about to delete %s", name)
		err = label.Delete(ctx)
		if err != nil {
			return
		}
	}
	return err
}
