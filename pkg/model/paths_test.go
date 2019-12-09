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
