// Copyright © 2018 One Concern
package param

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func createFUSEParams(t *testing.T) (fuseParams FUSEParams) {
	fuseParams, err := NewFUSEParams(
		FUSECoordPoint("/tmp/coord"),
		FUSEConfigBucketName("datamon-config-test-sdjfhga"),
		FUSEContextName("datamon-sidecar-test"),
	)
	require.NoError(t, err, "init postgres params")
	err = fuseParams.AddBundle(
		BDName("src"),
		BDSrcByLabel(
			"/tmp/mount",
			"ransom-datamon-test-repo",
			"testlabel",
		),
	)
	require.NoError(t, err, "add source bundle")
	err = fuseParams.AddBundle(
		BDName("dest"),
		BDDest("ransom-datamon-test-repo", "result of container coordination demo"),
		BDDestLabel("coordemo"),
		BDDestBundleIDFile("/tmp/bundleid.txt"),
	)
	require.NoError(t, err, "add destination bundle")
	return
}

func TestCreateFUSEParams(t *testing.T) {
	fuseParams := createFUSEParams(t)
	cerializedString, err := fuseParams.CerialString(CerialFmtYAM)
	require.NoError(t, err, "serialize string")

	expectedString := `globalOpts:
  sleepInsteadOfExit: false
  coordPoint: /tmp/coord
  configBucketName: datamon-config-test-sdjfhga
  contextName: datamon-sidecar-test
bundles:
- name: src
  srcPath: /tmp/mount
  srcRepo: ransom-datamon-test-repo
  srcLabel: testlabel
  srcBundle: ""
  destPath: ""
  destRepo: ""
  destMessage: ""
  destLabel: ""
  destBundleID: ""
- name: dest
  srcPath: ""
  srcRepo: ""
  srcLabel: ""
  srcBundle: ""
  destPath: ""
  destRepo: ransom-datamon-test-repo
  destMessage: result of container coordination demo
  destLabel: coordemo
  destBundleID: /tmp/bundleid.txt`

	require.Equal(t, expectedString, strings.Trim(cerializedString, "\n"))

	outDir, err := ioutil.TempDir("", "fuseserial")
	require.NoError(t, err, "create temp dir")

	/* this is a .yaml file according to major version details */
	outFile := filepath.Join(outDir, "fuse-sidecar-artifact")
	err = fuseParams.FirstCutSidecarFmt(outFile)
	require.NoError(t, err, "output artifact to filesystem function")
	v, err := ioutil.ReadFile(outFile)
	require.NoError(t, err, "read artifact from filesystem")
	require.Equal(t, expectedString, strings.Trim(string(v), "\n"))

}

func createPGParams(t *testing.T) (pgParams PGParams) {
	pgParams, err := NewPGParams(
		PGCoordPoint("/tmp/coord"),
		PGContributor(
			"contributor name",
			"contributor@oneconcern.com",
		),
	)
	require.NoError(t, err, "init postgres params")
	err = pgParams.AddDatabase(
		DBNameAndPort("db1", 5430),
		DBDest("ransom-datamon-test-repo", "postgres coordination example"),
		DBDestLabel("OUTPUT_LABEL"),
	)
	require.NoError(t, err, "add destination database")
	err = pgParams.AddDatabase(
		DBNameAndPort("db2", 5429),
		DBDest("ransom-datamon-test-repo", "postgres coordination example input"),
		DBSrcByLabel("ransom-datamon-test-repo", "pg-coord-example-input"),
	)
	require.NoError(t, err, "add source database")
	return
}

func TestCreatePGParams(t *testing.T) {
	pgParams := createPGParams(t)
	cerializedString, err := pgParams.CerialString(CerialFmtYAM)
	require.NoError(t, err, "serialize string")

	expectedString := `globalOpts:
  sleepInsteadOfExit: false
  ignorePGVersionMismatch: false
  coordPoint: /tmp/coord
  contributor:
    name: contributor name
    email: contributor@oneconcern.com
databases:
- name: db1
  pgPort: 5430
  destRepo: ransom-datamon-test-repo
  destMessage: postgres coordination example
  destLabel: OUTPUT_LABEL
  destBundleID: ""
  srcRepo: ""
  srcLabel: ""
  srcBundle: ""
- name: db2
  pgPort: 5429
  destRepo: ransom-datamon-test-repo
  destMessage: postgres coordination example input
  destLabel: ""
  destBundleID: ""
  srcRepo: ransom-datamon-test-repo
  srcLabel: pg-coord-example-input
  srcBundle: ""`

	require.Equal(t, expectedString, strings.Trim(cerializedString, "\n"))

	outDir, err := ioutil.TempDir("", "pgserial")
	require.NoError(t, err, "create temp dir")

	/* this is a .yaml file according to major version details */
	outFile := filepath.Join(outDir, "pg-sidecar-artifact")
	err = pgParams.FirstCutSidecarFmt(outFile)
	require.NoError(t, err, "output artifact to filesystem function")
	v, err := ioutil.ReadFile(outFile)
	require.NoError(t, err, "read artifact from filesystem")
	require.Equal(t, expectedString, strings.Trim(string(v), "\n"))

}
