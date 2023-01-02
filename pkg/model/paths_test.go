package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type archivePathFixture struct {
	name       string
	path       string
	wantsError bool
	expected   ArchivePathComponents
}

func archivePathTestCases() []archivePathFixture {
	return []archivePathFixture{
		// happy path
		{
			name: "bundle descriptor",
			path: "bundles/test-repo/1Jbb3SicFGoKB7JQJZdCCwdBQwE/bundle.yaml",
			expected: ArchivePathComponents{
				Repo:            "test-repo",
				BundleID:        "1Jbb3SicFGoKB7JQJZdCCwdBQwE",
				ArchiveFileName: "bundle.yaml",
				LabelName:       "",
			},
		},
		{
			name: "bundle files index",
			path: "bundles/test-repo/1Jbb3SicFGoKB7JQJZdCCwdBQwE/bundle-files-0.yaml",
			expected: ArchivePathComponents{
				Repo:            "test-repo",
				BundleID:        "1Jbb3SicFGoKB7JQJZdCCwdBQwE",
				ArchiveFileName: "bundle-files-0.yaml",
				LabelName:       "",
			},
		},
		{
			name: "repo descriptor",
			path: "repos/test-repo/repo.yaml",
			expected: ArchivePathComponents{
				Repo:            "test-repo",
				BundleID:        "",
				ArchiveFileName: "repo.yaml",
				LabelName:       "",
			},
		},
		{
			name: "label descriptor",
			path: "labels/test-repo/test-label/label.yaml",
			expected: ArchivePathComponents{
				Repo:            "test-repo",
				BundleID:        "",
				ArchiveFileName: "label.yaml",
				LabelName:       "test-label",
			},
		},
		{
			name: "another label descriptor",
			path: "labels/test-repo/test.label/label.yaml",
			expected: ArchivePathComponents{
				Repo:            "test-repo",
				BundleID:        "",
				ArchiveFileName: "label.yaml",
				LabelName:       "test.label",
			},
		},
		{
			name: "another context descriptor",
			path: "contexts/prod/context.yaml",
			expected: ArchivePathComponents{
				ArchiveFileName: "context.yaml",
				Context:         "prod",
			},
		},
		{
			name:       "wrong context path",
			path:       "contexts/context.yaml",
			wantsError: true,
		},
		// error cases
		{
			name:       "invalid path (no parts)",
			path:       "",
			wantsError: true,
		},
		{
			name:       "invalid path (no leading part)",
			path:       "/labels/test-repo/test-label/label.yaml",
			wantsError: true,
		},
		{
			name:       "invalid labels (way too short)",
			path:       "labels/x",
			wantsError: true,
		},
		{
			name:       "invalid labels (too short)",
			path:       "labels/test-repo/test-label",
			wantsError: true,
		},
		{
			name:       "invalid labels (wrong file)",
			path:       "labels/test-repo/test-label/wrong.yaml",
			wantsError: true,
		},
		{
			name:       "invalid repos (way too short)",
			path:       "repos/",
			wantsError: true,
		},
		{
			name:       "invalid repos (too short)",
			path:       "repos/test-repo",
			wantsError: true,
		},
		{
			name:       "invalid repos (wrong file)",
			path:       "repos/test-repo/wrong.yaml",
			wantsError: true,
		},
		{
			name:       "invalid bundles (way too short)",
			path:       "bundles",
			wantsError: true,
		},
		{
			name:       "invalid bundles (too short)",
			path:       "bundles/test-repo/",
			wantsError: true,
		},
		{
			name:       "invalid bundles (wrong file)",
			path:       "bundles/test-repo/{ID}/wrong.yaml",
			wantsError: true,
		},
		{
			name:       "invalid bundles (wrong index file)",
			path:       "bundles/test-repo/{ID}/bundle-files-abc.yaml",
			wantsError: true,
		},
		{
			name:       "invalid bundles (wrong index file, bis)",
			path:       "bundles/test-repo/{ID}/bundle-files-abc.yml",
			wantsError: true,
		},
		{
			name: "diamond descriptor",
			path: "diamonds/test-repo/1Jbb3SicFGoKB7JQJZdCCwdBQwE/diamond-done.yaml",
			expected: ArchivePathComponents{
				Repo:            "test-repo",
				DiamondID:       "1Jbb3SicFGoKB7JQJZdCCwdBQwE",
				ArchiveFileName: "diamond-done.yaml",
				IsFinalState:    true,
			},
		},
		{
			name: "diamond descriptor",
			path: "diamonds/test-repo/1Jbb3SicFGoKB7JQJZdCCwdBQwE/diamond-running.yaml",
			expected: ArchivePathComponents{
				Repo:            "test-repo",
				DiamondID:       "1Jbb3SicFGoKB7JQJZdCCwdBQwE",
				ArchiveFileName: "diamond-running.yaml",
			},
		},
		{
			name: "split descriptor",
			path: "diamonds/test-repo/1Jbb3SicFGoKB7JQJZdCCwdBQwE/splits/1Jbb3SicFGoKB7JQJZdCCwdBQwE/split-done.yaml",
			expected: ArchivePathComponents{
				Repo:            "test-repo",
				DiamondID:       "1Jbb3SicFGoKB7JQJZdCCwdBQwE",
				SplitID:         "1Jbb3SicFGoKB7JQJZdCCwdBQwE",
				ArchiveFileName: "split-done.yaml",
				IsFinalState:    true,
			},
		},
		{
			name: "split file index",
			path: "diamonds/test-repo/1Jbb3SicFGoKB7JQJZdCCwdBQwE/splits/1Jbb3SicFGoKB7JQJZdCCwdBQwE/1Jbb3SicFGoKB7JQJZdCCwdBQwE/bundle-files-001.yaml",
			expected: ArchivePathComponents{
				Repo:            "test-repo",
				DiamondID:       "1Jbb3SicFGoKB7JQJZdCCwdBQwE",
				SplitID:         "1Jbb3SicFGoKB7JQJZdCCwdBQwE",
				GenerationID:    "1Jbb3SicFGoKB7JQJZdCCwdBQwE",
				ArchiveFileName: "bundle-files-001.yaml",
			},
		},
		{
			name:       "diamond ID is not a ksuid",
			path:       "diamonds/test-repo/notAKSUID/diamond-done.yaml",
			wantsError: true,
		},
		{
			name:       "invalid diamond descriptor",
			path:       "diamonds/test-repo/1Jbb3SicFGoKB7JQJZdCCwdBQwE/diamond-gone.yaml",
			wantsError: true,
		},
		{
			name:       "invalid split descriptor",
			path:       "diamonds/test-repo/1Jbb3SicFGoKB7JQJZdCCwdBQwE/splits/split-gone.yaml",
			wantsError: true,
		},
		{
			name:       "invalid path with some split descriptor",
			path:       "diamonds/test-repo/1Jbb3SicFGoKB7JQJZdCCwdBQwE/splits/split-done.yaml/somemore.yaml",
			wantsError: true,
		},
		{
			name:       "invalid path to split index file descriptor",
			path:       "diamonds/test-repo/1Jbb3SicFGoKB7JQJZdCCwdBQwE/splits/1Jbb3SicFGoKB7JQJZdCCwdBQwE/notAKSUID/bundle-files-001.yaml",
			wantsError: true,
		},
		{
			name:       "invalid split index file descriptor",
			path:       "diamonds/test-repo/1Jbb3SicFGoKB7JQJZdCCwdBQwE/splits/1Jbb3SicFGoKB7JQJZdCCwdBQwE/1Jbb3SicFGoKB7JQJZdCCwdBQwE/some-files-001.yaml",
			wantsError: true,
		},
		{
			name:       "invalid diamond path",
			path:       "diamonds/test-repo/",
			wantsError: true,
		},
		{
			name: "empty diamond path",
			path: "diamonds/test-repo/1Jbb3SicFGoKB7JQJZdCCwdBQwE/",
			expected: ArchivePathComponents{
				Repo:            "test-repo",
				DiamondID:       "1Jbb3SicFGoKB7JQJZdCCwdBQwE",
				ArchiveFileName: "",
			},
		},
		{
			name: "empty split path",
			path: "diamonds/test-repo/1Jbb3SicFGoKB7JQJZdCCwdBQwE/splits/",
			expected: ArchivePathComponents{
				Repo:            "test-repo",
				DiamondID:       "1Jbb3SicFGoKB7JQJZdCCwdBQwE",
				ArchiveFileName: "",
			},
		},
		{
			name: "empty split contents path",
			path: "diamonds/test-repo/1Jbb3SicFGoKB7JQJZdCCwdBQwE/splits/1Jbb3SicFGoKB7JQJZdCCwdBQwE/",
			expected: ArchivePathComponents{
				Repo:            "test-repo",
				DiamondID:       "1Jbb3SicFGoKB7JQJZdCCwdBQwE",
				SplitID:         "1Jbb3SicFGoKB7JQJZdCCwdBQwE",
				ArchiveFileName: "",
			},
		},
	}
}
func TestGetArchivePathComponents(t *testing.T) {
	for _, toPin := range archivePathTestCases() {
		testcase := toPin
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			apc, err := GetArchivePathComponents(testcase.path)
			if testcase.wantsError {
				require.Error(t, err)
				assert.Empty(t, apc)
			} else {
				require.NoError(t, err)
				assert.EqualValues(t, testcase.expected, apc)
			}
		})
	}
}

func TestReverseIndexChunk(t *testing.T) {
	res, err := ReverseIndexChunk("chunk-593.yaml")
	require.NoError(t, err)
	require.Equal(t, uint64(593), res)

	res, err = ReverseIndexChunk("dir/dir/chunk-593.yaml")
	require.NoError(t, err)
	require.Equal(t, uint64(593), res)

	res, err = ReverseIndexChunk("reverse-index/chunk-1.yaml")
	require.NoError(t, err)
	require.Equal(t, uint64(1), res)

	_, err = ReverseIndexChunk("chunk-zork.yaml")
	require.Error(t, err)
}
