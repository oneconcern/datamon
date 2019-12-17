package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gotest.tools/icmd"
)

func TestSelfUpgrade(t *testing.T) {
	err := doCheckVersion()
	if err != nil {
		if strings.Contains(err.Error(), "no matching release from github repo") ||
			strings.Contains(err.Error(), "could not fetch release from github repo") {
			t.Logf("upgrade test disabled: repo artifacts not ready yet: %v", err)
			t.SkipNow()
		}
	}
	require.NoError(t, err)

	opts := upgradeFlags{verbose: true}
	require.Error(t, doSelfUpgrade(opts))

	opts.forceUgrade = true
	opts.selfBinary = "fake"
	require.Error(t, doSelfUpgrade(opts))

	dummyDir, err := ioutil.TempDir("", "dummy-exec")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(dummyDir) }()

	opts.selfBinary = filepath.Join(dummyDir, "datamon")

	err = ioutil.WriteFile(opts.selfBinary, []byte(`dummy`), 0700)
	require.NoError(t, err)

	require.NoError(t, doSelfUpgrade(opts))

	res := icmd.RunCommand(opts.selfBinary, "version")
	require.EqualValues(t, 0, res.ExitCode)

	rexp := regexp.MustCompile(`(?m)Version:\s*(.*?)\nBuild date:\s*(.*?)\nCommit:\s*(.*)`)
	assert.Truef(t, rexp.MatchString(res.Stdout()), "unexpected datamon version result on updated binary: %q", res.Stdout())
}
