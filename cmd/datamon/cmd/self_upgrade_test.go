package cmd

import (
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gotest.tools/icmd"
)

func TestSelfUpgrade(t *testing.T) {
	err := doCheckVersion()
	if strings.Contains(err.Error(), "no matching release from github repo") ||
		strings.Contains(err.Error(), "could not fetch release from github repo") {
		t.Logf("upgrade test disabled: repo artifacts not ready yet: %v", err)
		t.SkipNow()
	}
	require.NoError(t, err)

	opts := upgradeFlags{verbose: true}
	require.Error(t, doSelfUpgrade(opts))

	opts.forceUgrade = true
	opts.selfBinary = "fake"
	require.Error(t, doSelfUpgrade(opts))

	dummyExec, err := ioutil.TempFile("", "dummy-exec")
	require.NoError(t, err)
	defer func() { _ = os.Remove(dummyExec.Name()) }()

	_, err = dummyExec.Write([]byte(`dummy`))
	require.NoError(t, err)
	require.NoError(t, dummyExec.Close())

	opts.selfBinary = dummyExec.Name()
	require.NoError(t, doSelfUpgrade(opts))

	err = os.Chmod(dummyExec.Name(), 0700)
	require.NoError(t, err)

	_ = os.Setenv("DATAMON_GLOBAL_CONFIG", "x") // TODO: we should remove this requirement on some commands
	res := icmd.RunCommand(dummyExec.Name(), "version")
	require.EqualValues(t, 0, res.ExitCode)

	rexp := regexp.MustCompile(`(?m)Version:\s*(.*?)\nBuild date:\s*(.*?)\nCommit:\s*(.*)`)
	assert.Truef(t, rexp.MatchString(res.Stdout()), "unexpected datamon version result on updated binary: %q", res.Stdout())
}
