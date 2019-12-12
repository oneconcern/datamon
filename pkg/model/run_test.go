/*
 * Copyright © 2019 One Concern
 *
 */

package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetPathToCategory(t *testing.T) {
	path := GetPathToCategory("category1")
	require.Equal(t, "categories/category1/category.yaml", path)
}

func TestGetPathToRun(t *testing.T) {
	path := GetPathToRun("category1", "run1", "id1")
	require.Equal(t, "runs/category1/run1/id1/run.yaml", path)
}

func TestGetPathToDataSetIn(t *testing.T) {
	path := GetPathToDataSetIn("category1", "run1", "id1", "firstStage", "stageid", "initContainer")
	require.Equal(t, "runs/category1/run1/id1/firstStage/stageid/initContainer/datain.yaml", path)
}

func TestGetPathToDataSetOut(t *testing.T) {
	path := GetPathToDataSetOut("category1", "runname", "runid", "firstStage", "stageid", "initContainer")
	require.Equal(t, "runs/category1/runname/runid/firstStage/stageid/initContainer/dataout.yaml", path)
}

func TestGetPathToContainer(t *testing.T) {
	path := GetPathToContainer("category1", "run1", "id1", "firstStage", "stageid", "initContainer")
	require.Equal(t, "runs/category1/run1/id1/firstStage/stageid/initContainer/container.yaml", path)
}

func TestGetPathToRunStatus(t *testing.T) {
	path := GetPathToRunStatus("category1", "run1", "id1")
	require.Equal(t, "runs/category1/run1/id1/runstatus.yaml", path)
}
