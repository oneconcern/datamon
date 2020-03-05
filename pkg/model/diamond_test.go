package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathsForDiamonds(t *testing.T) {
	expected := "diamonds/myrepo/"
	assert.Equal(t, expected, GetArchivePathPrefixToDiamonds("myrepo"))

	expected = "diamonds/myrepo/123/diamond-done.yaml"
	assert.Equal(t, expected, GetArchivePathToFinalDiamond("myrepo", "123"))

	expected = "diamonds/myrepo/123/diamond-running.yaml"
	assert.Equal(t, expected, GetArchivePathToDiamond("myrepo", "123", DiamondInitialized))

	assert.Equal(t, expected, GetArchivePathToInitialDiamond("myrepo", "123"))

	expected = "diamonds/myrepo/123/splits/"
	assert.Equal(t, expected, GetArchivePathPrefixToSplits("myrepo", "123"))

	expected = "diamonds/myrepo/123/splits/456/split-done.yaml"
	assert.Equal(t, expected, GetArchivePathToSplit("myrepo", "123", "456", SplitDone))

	assert.Equal(t, expected, GetArchivePathToFinalSplit("myrepo", "123", "456"))

	expected = "diamonds/myrepo/123/splits/456/split-running.yaml"
	assert.Equal(t, expected, GetArchivePathToInitialSplit("myrepo", "123", "456"))

	expected = "diamonds/myrepo/123/splits/456/789/bundle-files-1.yaml"
	assert.Equal(t, expected, GetArchivePathToSplitFileList("myrepo", "123", "456", "789", 1))

	expected = ".conflicts/123/a/abc.go"
	assert.Equal(t, expected, GenerateConflictPath("123", "a/abc.go"))

	assert.Equal(t, expected, GenerateConflictPath("123", "/a/abc.go"))

	expected = ".checkpoints/123/a/abc.go"
	assert.Equal(t, expected, GenerateCheckpointPath("123", "a/abc.go"))

	assert.Equal(t, expected, GenerateCheckpointPath("123", "/a/abc.go"))
}
