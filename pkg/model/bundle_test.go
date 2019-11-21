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

func TestGetArchivePathComponents(t *testing.T) {
	var apc ArchivePathComponents
	var err error
	// core/bundle_list.go
	bundleDescriptorPath1 := "bundles/test-repo/1Jbb3SicFGoKB7JQJZdCCwdBQwE/bundle.yaml"
	apc, err = GetArchivePathComponents(bundleDescriptorPath1)
	require.NoError(t, err)
	require.Equal(t, apc.Repo, "test-repo")
	require.Equal(t, apc.BundleID, "1Jbb3SicFGoKB7JQJZdCCwdBQwE")
	require.Equal(t, apc.ArchiveFileName, "bundle.yaml")
	require.Equal(t, apc.LabelName, "")
	bundleFilelistPath1 := "bundles/test-repo/1Jbb3SicFGoKB7JQJZdCCwdBQwE/bundle-files-0.yaml"
	apc, err = GetArchivePathComponents(bundleFilelistPath1)
	require.NoError(t, err)
	require.Equal(t, apc.Repo, "test-repo")
	require.Equal(t, apc.BundleID, "1Jbb3SicFGoKB7JQJZdCCwdBQwE")
	require.Equal(t, apc.ArchiveFileName, "bundle-files-0.yaml")
	require.Equal(t, apc.LabelName, "")
	// core/repo_list.go
	repoPath1 := "repos/test-repo/repo.yaml"
	apc, err = GetArchivePathComponents(repoPath1)
	require.NoError(t, err)
	require.Equal(t, apc.Repo, "test-repo")
	require.Equal(t, apc.BundleID, "")
	require.Equal(t, apc.ArchiveFileName, "")
	require.Equal(t, apc.LabelName, "")
	// core/label_list.go
	labelPath1 := "labels/test-repo/test-label/label.yaml"
	apc, err = GetArchivePathComponents(labelPath1)
	require.NoError(t, err)
	require.Equal(t, apc.Repo, "test-repo")
	require.Equal(t, apc.BundleID, "")
	require.Equal(t, apc.ArchiveFileName, "")
	require.Equal(t, "test-label", apc.LabelName)
	labelPath2 := "labels/test-repo/test.label/label.yaml"
	apc, err = GetArchivePathComponents(labelPath2)
	require.NoError(t, err)
	require.Equal(t, apc.Repo, "test-repo")
	require.Equal(t, apc.BundleID, "")
	require.Equal(t, apc.ArchiveFileName, "")
	require.Equal(t, "test.label", apc.LabelName)
}
