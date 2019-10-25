// Copyright © 2018 One Concern
package param

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateFUSEParams(t *testing.T) {
	fuseParams, err := NewFUSEParams(
		FUSECoordPoint("/tmp/coord"),
		FUSEContributor(
			"contributor name",
			"contributor@oneconcern.com",
		),
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
}

func TestCreatePGParams(t *testing.T) {
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
}
