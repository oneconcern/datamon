package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/oneconcern/datamon/internal/rand"
	"github.com/stretchr/testify/require"
)

func TestSquashRepo(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()

	const (
		repo11 = "squash-test-repo11"
		repo12 = "squash-test-repo12"
		repo13 = "squash-test-repo13"
	)

	input1, err := os.MkdirTemp(".", "")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(input1)
	}()
	input2, err := os.MkdirTemp(".", "")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(input2)
	}()
	input3, err := os.MkdirTemp(".", "")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(input3)
	}()

	doUpdate := func(t testing.TB, inputs ...string) {
		for _, input := range inputs {
			for _, file := range []string{"file1", "file2", "file3", "file4"} {
				content := rand.Bytes(1024)
				require.NoError(t,
					os.WriteFile(filepath.Join(input, file), content, 0600),
				)
			}
		}
	}

	doUpdate(t, input1, input2, input3)
	dcontext := testContext()

	t.Run("create repos populated with bundles", func(t *testing.T) {
		runCmd(t, []string{"repo",
			"create",
			"--description", "test-squash",
			"--repo", repo11,
			"--context", dcontext,
		}, "create first test repo", false)

		runCmd(t, []string{"repo",
			"create",
			"--description", "test-squash",
			"--repo", repo12,
			"--context", dcontext,
		}, "create second test repo", false)

		runCmd(t, []string{"repo",
			"create",
			"--description", "test-squash",
			"--repo", repo13,
			"--context", dcontext,
		}, "create third test repo", false)

		runCmd(t, []string{"bundle",
			"upload",
			"--path", input1,
			"--message", "The initial commit for the repo",
			"--repo", repo11,
			"--context", dcontext,
		}, fmt.Sprintf("upload bundle at %q", input1), false)

		runCmd(t, []string{"bundle",
			"upload",
			"--path", input2,
			"--message", "The initial commit for the repo",
			"--repo", repo12,
			"--context", dcontext,
		}, fmt.Sprintf("upload bundle at %q", input2), false)

		runCmd(t, []string{"bundle",
			"upload",
			"--path", input3,
			"--message", "The initial commit for the repo",
			"--repo", repo13,
			"--context", dcontext,
		}, fmt.Sprintf("upload bundle at %q", input3), false)

		// insert additional bundles
		doUpdate(t, input2)

		runCmd(t, []string{"bundle",
			"upload",
			"--path", input2,
			"--message", "second commit for the repo",
			"--repo", repo12,
			"--context", dcontext,
		}, fmt.Sprintf("upload bundle at %q", input2), false)

		doUpdate(t, input2)

		runCmd(t, []string{"bundle",
			"upload",
			"--path", input2,
			"--message", "last commit for the repo",
			"--repo", repo12,
			"--context", dcontext,
		}, fmt.Sprintf("upload bundle at %q", input2), false)

		doUpdate(t, input3)

		runCmd(t, []string{"bundle",
			"upload",
			"--path", input3,
			"--message", "second commit for the repo",
			"--repo", repo13,
			"--context", dcontext,
		}, fmt.Sprintf("upload bundle at %q", input3), false)

		doUpdate(t, input3)

		runCmd(t, []string{"bundle",
			"upload",
			"--path", input3,
			"--message", "third commit for the repo (with label)",
			"--repo", repo13,
			"--label", "zorg",
			"--context", dcontext,
		}, fmt.Sprintf("upload bundle at %q", input3), false)

		doUpdate(t, input3)

		runCmd(t, []string{"bundle",
			"upload",
			"--path", input3,
			"--message", "fourth commit for the repo (with semver label)",
			"--repo", repo13,
			"--label", "v1.2.3",
			"--context", dcontext,
		}, fmt.Sprintf("upload bundle at %q", input3), false)

		doUpdate(t, input3)

		runCmd(t, []string{"bundle",
			"upload",
			"--path", input3,
			"--message", "fifth commit for the repo (with semver label, without the v prefix)",
			"--repo", repo13,
			"--label", "1.2.3",
			"--context", dcontext,
		}, fmt.Sprintf("upload bundle at %q", input3), false)

		doUpdate(t, input3)

		runCmd(t, []string{"bundle",
			"upload",
			"--path", input3,
			"--message", "sixth and last commit for the repo",
			"--repo", repo13,
			"--context", dcontext,
		}, fmt.Sprintf("upload bundle at %q", input3), false)
	})

	t.Run("should squash repo11 and do nothing", func(t *testing.T) {
		r, w := startCapture(t)
		runCmd(t, []string{"repo",
			"squash",
			"--repo", repo11,
		}, fmt.Sprintf("squashed repo %q", repo11), false)
		lines := endCapture(t, r, w, []string{})
		require.Len(t, lines, 1)
		require.Contains(t, lines[0], "The initial commit for the repo")
	})

	t.Run("should squash repo12 and get only latest commit", func(t *testing.T) {
		r, w := startCapture(t)
		runCmd(t, []string{"repo",
			"squash",
			"--repo", repo12,
		}, fmt.Sprintf("squashed repo %q", repo12), false)
		lines := endCapture(t, r, w, []string{})
		require.Len(t, lines, 1)
		require.Contains(t, lines[0], "last commit for the repo")
	})

	t.Run("should squash repo13 with retain tags and get 3rd, 4th and last commits", func(t *testing.T) {
		r, w := startCapture(t)
		runCmd(t, []string{"repo",
			"squash",
			"--repo", repo13,
			"--retain-tags",
		}, fmt.Sprintf("squashed repo %q", repo13), false)
		lines := endCapture(t, r, w, []string{})
		require.Len(t, lines, 4)
		require.Contains(t, lines[0], "third commit for the repo (with label)")
		require.Contains(t, lines[1], "fourth commit for the repo (with semver label)")
		require.Contains(t, lines[2], "fifth commit for the repo (with semver label, without the v prefix)")
		require.Contains(t, lines[3], "sixth and last commit for the repo")
	})

	t.Run("should squash repo13 with retain semver tags and get 4th, 5th and last commits", func(t *testing.T) {
		r, w := startCapture(t)
		runCmd(t, []string{"repo",
			"squash",
			"--repo", repo13,
			"--retain-semver-tags",
		}, fmt.Sprintf("squashed repo %q", repo13), false)
		lines := endCapture(t, r, w, []string{})
		require.Len(t, lines, 3)
		require.Contains(t, lines[0], "fourth commit for the repo (with semver label)")
		require.Contains(t, lines[1], "fifth commit for the repo (with semver label, without the v prefix)")
		require.Contains(t, lines[2], "sixth and last commit for the repo")
	})

	t.Run("should squash repo13 with no tags retained and get only the last commit", func(t *testing.T) {
		r, w := startCapture(t)
		runCmd(t, []string{"repo",
			"squash",
			"--repo", repo13,
		}, fmt.Sprintf("squashed repo %q", repo13), false)
		lines := endCapture(t, r, w, []string{})
		require.Len(t, lines, 1)
		require.Contains(t, lines[0], "sixth and last commit for the repo")
	})

	t.Run("with squash context", func(t *testing.T) {
		t.Run("upload new bundles", func(t *testing.T) {
			runCmd(t, []string{"bundle",
				"upload",
				"--path", input1,
				"--message", "another commit",
				"--repo", repo11,
				"--context", dcontext,
			}, fmt.Sprintf("upload bundle at %q", input1), false)

			runCmd(t, []string{"bundle",
				"upload",
				"--path", input2,
				"--message", "commit to squash",
				"--repo", repo12,
				"--context", dcontext,
			}, fmt.Sprintf("upload bundle at %q", input2), false)

			runCmd(t, []string{"bundle",
				"upload",
				"--path", input2,
				"--message", "another commit",
				"--repo", repo12,
				"--context", dcontext,
			}, fmt.Sprintf("upload bundle at %q", input2), false)

			runCmd(t, []string{"bundle",
				"upload",
				"--path", input3,
				"--message", "another commit",
				"--repo", repo13,
				"--context", dcontext,
			}, fmt.Sprintf("upload bundle at %q", input3), false)

			runCmd(t, []string{"bundle",
				"upload",
				"--path", input3,
				"--message", "another retained commit",
				"--repo", repo13,
				"--label", "v4.5.6",
				"--context", dcontext,
			}, fmt.Sprintf("upload bundle at %q", input3), false)

			runCmd(t, []string{"bundle",
				"upload",
				"--path", input3,
				"--message", "last commit",
				"--repo", repo13,
				"--context", dcontext,
			}, fmt.Sprintf("upload bundle at %q", input3), false)
		})

		t.Run("should squash all repos in this context", func(t *testing.T) {
			runCmd(t, []string{"context",
				"squash",
				"--context", dcontext,
				"--retain-semver-tags",
			}, fmt.Sprintf("squashed context %q", dcontext), false)

			// squash context does not report: pull the bundles per repo
			r, w := startCapture(t)
			runCmd(t, []string{"bundle",
				"list",
				"--context", dcontext,
				"--repo", repo11,
			}, fmt.Sprintf("bundle list %q", repo11), false)
			lines := endCapture(t, r, w, []string{})
			require.Len(t, lines, 1)
			require.Contains(t, lines[0], "another commit")

			r, w = startCapture(t)
			runCmd(t, []string{"bundle",
				"list",
				"--context", dcontext,
				"--repo", repo12,
			}, fmt.Sprintf("bundle list %q", repo12), false)
			lines = endCapture(t, r, w, []string{})
			require.Len(t, lines, 1)
			require.Contains(t, lines[0], "another commit")

			r, w = startCapture(t)
			runCmd(t, []string{"bundle",
				"list",
				"--context", dcontext,
				"--repo", repo13,
			}, fmt.Sprintf("bundle list %q", repo13), false)
			lines = endCapture(t, r, w, []string{})
			require.Len(t, lines, 2)
			require.Contains(t, lines[0], "another retained commit")
			require.Contains(t, lines[1], "last commit")
		})
	})
}
