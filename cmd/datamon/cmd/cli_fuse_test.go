//go:build fuse_cli
// +build fuse_cli

package cmd

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/oneconcern/datamon/pkg/dlogger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	buildBinaryonce sync.Once
	testBinary      string
)

func testLogger(t testing.TB) *zap.Logger {
	// a logger for the test sequence itself
	return dlogger.MustGetLogger("debug").With(zap.String("test", t.Name()))
}

func testCommand(t testing.TB, withPipe, withEnv bool, target string, args ...string) (*exec.Cmd, io.ReadCloser) {
	// NOTE: this starts a process, not a goroutine like for other tests
	// That is primarily because we want to send a signal to the process
	// in order to unmount.
	cmd := exec.Command(target, args...)
	cmd.Env = []string{}
	if withEnv {
		cmd.Env = append(cmd.Env, "GOOGLE_APPLICATION_CREDENTIALS="+os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
		cmd.Env = append(cmd.Env, "DATAMON_CONTEXT="+os.Getenv("DATAMON_CONTEXT")) // TODO(fred): not sure this is used
		cmd.Env = append(cmd.Env, "DATAMON_GLOBAL_CONFIG="+os.Getenv("DATAMON_GLOBAL_CONFIG"))
	}
	cmd.Env = append(cmd.Env, "HOME="+os.Getenv("HOME"))
	cmd.Env = append(cmd.Env, "PATH="+os.Getenv("PATH"))

	if !withPipe {
		return cmd, nil
	}

	pipeOut, err := cmd.StdoutPipe()
	require.NoError(t, err)

	pipeErr, err := cmd.StderrPipe()
	require.NoError(t, err)

	// Combine stdout and stderr, tee this output to os.Stdout and return pipe reader for output scanning.
	// The copy is needed. Otherwise, no output is captured until the command ends.
	//
	// TODO(fred): output is combined	because we assert some output from stderr (CLI output using stdlib logger)
	// and some from stdout (pkg output using zap logger).
	// We should return both as separate pipes and assert their respective output separately.
	pipeR, pipeW := io.Pipe()
	c := io.MultiWriter(os.Stdout, pipeW)

	go func() {
		_, _ = io.Copy(c, pipeOut)
	}()

	go func() {
		_, _ = io.Copy(c, pipeErr)
	}()
	return cmd, pipeR
}

func testBundleMount(t *testing.T, testType string, waiter func(*zap.Logger, io.Reader)) {
	logger := testLogger(t)

	logger.Info("test initialization")
	repo := generateRepoName("fuse")
	cleanup := setupTests(t)
	defer cleanup()
	const testConcurrencyForUpload = "10"
	extraTestArgs := []string{"--loglevel", "info"} // set this to debug to get more tracing

	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo,
	}, "create repo", false)
	logger.Info("test repo created")

	runCmd(t, append([]string{"bundle",
		"upload",
		"--path", dirPathStr(t, testUploadTrees[1][0]),
		"--message", "read-only mount test bundle",
		"--repo", repo,
		"--concurrency-factor", testConcurrencyForUpload,
	}, extraTestArgs...), "upload bundle in order to test downloading individual files", false)
	logger.Info("test bundle uploaded")

	bundles, err := listBundles(t, repo)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, 1, bundles.Len(), "bundle count in test repo")

	pathBackingFs, _ := ioutil.TempDir("", "mmfs-")
	pathToMount, _ := ioutil.TempDir("", "mmp-")

	defer os.RemoveAll(pathBackingFs)
	defer os.RemoveAll(pathToMount)
	targetBinary := buildTestBinary(t)

	cmdParams := append(testMountCmdParams(t,
		testType,
		repo,
		pathToMount,
		pathBackingFs,
		bundles,
	), extraTestArgs...)
	cmdParams = append(cmdParams, "--context", testContext())
	logger.Info("datamon exec", zap.Strings("params", cmdParams))

	cmd, pipe := testCommand(t, true, true, targetBinary, cmdParams...)

	require.NoError(t, cmd.Start())
	var killed bool
	defer func() {
		if !killed {
			_ = cmd.Process.Kill()
		}
	}()

	logger.Info("waiting for mount to be ready")
	waiter(logger, pipe)

	logger.Info("starting with mount inspection")
	for _, file := range testUploadTrees[1] {
		expected := readTextFile(t, filePathStr(t, file))
		actual := readTextFile(t, filepath.Join(pathToMount, pathInBundle(file)))
		assert.Equal(t, len(expected), len(actual), "downloaded file '"+pathInBundle(file)+"' size")
		assert.Equal(t, expected, actual, "downloaded file '"+pathInBundle(file)+"' contents")
	}

	logger.Info("unmounting")
	require.NoError(t, cmd.Process.Kill())
	err = cmd.Wait()
	killed = true
	require.Equal(t, "signal: killed", err.Error(), "cmd exit with killed error")
}

func buildTestBinary(t testing.TB) string {
	buildBinaryonce.Do(func() {
		pathToBinary, _ := ioutil.TempDir("", "dtm-")
		testBinary = filepath.Join(pathToBinary, "datamon")
		build := exec.Command("go", "build", "-o", testBinary)
		build.Dir = ".."
		require.NoError(t, build.Run())
	})
	return testBinary
}

func testMountCmdParams(t testing.TB, testType, repo, pathToMount, pathBackingFs string, bundles bundleListEntries) []string {
	switch testType {
	case "nostream-dest":
		return []string{
			"bundle", "mount",
			"--repo", repo,
			"--bundle", bundles[0].hash,
			"--destination", pathBackingFs,
			"--mount", pathToMount,
			"--stream=false",
		}
	case "stream-dest":
		return []string{
			"bundle", "mount",
			"--repo", repo,
			"--bundle", bundles[0].hash,
			"--destination", pathBackingFs,
			"--mount", pathToMount,
		}
	case "nostream-nodest":
		return []string{
			"bundle", "mount",
			"--repo", repo,
			"--bundle", bundles[0].hash,
			"--mount", pathToMount,
		}
	case "mutable":
		return []string{
			"bundle", "mount", "new",
			"--repo", repo,
			"--message", "mutabletest",
			"--destination", pathBackingFs,
			"--mount", pathToMount,
		}
	default:
		require.True(t, false, "unexpected test type '"+testType+"'")
		return nil
	}
}

func testWaitForReader(t testing.TB, token string, defaultWait time.Duration) func(*zap.Logger, io.Reader) {
	return func(l *zap.Logger, output io.Reader) {
		// awaits expected ready message
		// TODO(fred): we may use that to time out on waiting if needed
		ticker := time.NewTicker(10 * time.Second)
		done := make(chan bool)
		defer func() {
			done <- true
		}()

		go func() {
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					l.Info("Waiting for mount readiness...")
				}
			}
		}()
		s := bufio.NewScanner(output)
		started := false
		for s.Scan() {
			buf := s.Text()
			if len(buf) > 0 {
				started = true
			}
			if strings.Contains(buf, token) {
				l.Info("OK ready")
				return
			}
		}
		if !started {
			time.Sleep(defaultWait) // fallback if pipe closes unexpectedly
		}
	}
}

func TestBundleMount(t *testing.T) {
	testBundleMount(t, "stream-dest", testWaitForReader(t, `"mounting"`, 5*time.Second))
}

func TestBundleMountNoStream(t *testing.T) {
	testBundleMount(t, "nostream-dest", testWaitForReader(t, `"mounting"`, 5*time.Second))
}

func TestBundleMountNoStreamNoDest(t *testing.T) {
	testBundleMount(t, "nostream-nodest", testWaitForReader(t, `"mounting"`, 5*time.Second))
}

func captureOutputProgress(p io.Reader) (*bytes.Buffer, *sync.WaitGroup) {
	var (
		wg sync.WaitGroup
		b  bytes.Buffer
	)
	wg.Add(1)
	latch := make(chan struct{})
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		latch <- struct{}{}
		_, _ = b.ReadFrom(p)
	}(&wg)
	<-latch
	return &b, &wg
}

func TestBundleMutableMount(t *testing.T) {
	logger := testLogger(t)
	repo := generateRepoName("fuse-mutable")
	extraTestArgs := []string{"--loglevel", "info"} // set this to debug to get more tracing

	logger.Info("test initialization")
	cleanup := setupTests(t)
	defer cleanup()

	runCmd(t, []string{"repo",
		"create",
		"--description", "testing",
		"--repo", repo,
	},
		"create repo", false)
	logger.Info("test repo created")

	pathBackingFs, _ := ioutil.TempDir("", "mmfs-")
	pathToMount, _ := ioutil.TempDir("", "mmp-")

	defer os.RemoveAll(pathBackingFs)
	defer os.RemoveAll(pathToMount)

	targetBinary := buildTestBinary(t)

	bundles, err := listBundles(t, repo)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, 0, bundles.Len(), "bundle count in test repo")

	cmdParams := append(testMountCmdParams(t,
		"mutable",
		repo,
		pathToMount,
		pathBackingFs,
		nil,
	), extraTestArgs...)
	cmdParams = append(cmdParams, "--context", testContext())
	logger.Info("datamon exec", zap.Strings("params", cmdParams))

	cmd, pipe := testCommand(t, true, true, targetBinary, cmdParams...)

	require.NoError(t, cmd.Start())
	var killed bool
	defer func() {
		if !killed {
			_ = cmd.Process.Kill()
		}
	}()

	testWaitForReader(t, `"mounting"`, 5*time.Second)(logger, pipe)

	logger.Info("copying files to the mount")
	// copy files to mount
	createTestUploadTree(t, pathToMount, testUploadTrees[1])

	// checking the backing storage
	backingFileInfos, err := ioutil.ReadDir(pathBackingFs)
	require.NoError(t, err)
	assert.Equal(t, len(testUploadTrees[1]), len(backingFileInfos),
		"found expected count of files stored by inode")

	logger.Info("unmounting with commit")
	require.NoError(t, cmd.Process.Signal(os.Interrupt))

	logger.Info("capturing commit output")
	b, wg := captureOutputProgress(pipe)

	logger.Debug("waiting for cmd to terminate")
	require.NoError(t, cmd.Wait())
	killed = true
	_ = pipe.Close()
	logger.Debug("waiting for pipe to end")
	wg.Wait()

	logger.Info("asserting created bundle")
	rex := regexp.MustCompile(`(?m)^bundle:\s*([^\s]+)$`)
	match := rex.FindSubmatch(b.Bytes())
	require.Truef(t, len(match) > 1, "couldn't find bundle ID in output, got this instead: %s", b.String())
	bundleID := string(match[1])

	bundles, err = listBundles(t, repo)
	require.NoError(t, err, "error out of listBundles() test helper")
	require.Equal(t, 1, bundles.Len(), "bundle count in test repo")

	bundle := bundles.Last()
	logger.Info("bundles list output", zap.String("bundle", bundle.hash))
	require.Equal(t, bundleID, bundle.hash)
}
