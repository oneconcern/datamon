package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetArchivePathToBundle(t *testing.T) {
	pathToBundleDescriptor := "bundles/myrepo/123/bundle.yaml"
	require.Equal(t, pathToBundleDescriptor,
		GetArchivePathToBundle("myrepo", "123"))
}

func TestGetConsumablePathToBundleFileList(t *testing.T) {
	pathToBundleFileList := "bundles/myrepo/123/bundle-files-10.yaml"
	require.Equal(t, pathToBundleFileList,
		GetArchivePathToBundleFileList("myrepo", "123", 10))
}

func TestGetArchivePathPrefixToBundles(t *testing.T) {
	prefix := "bundles/myrepo/"
	require.Equal(t, prefix,
		GetArchivePathPrefixToBundles("myrepo"))
}
