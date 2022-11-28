package cmd

import (
	"fmt"
	"testing"
)

func TestPurgeRepo(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()

	const (
		repo11 = "purge-test-repo11"
		repo12 = "purge-test-repo12"
		input  = "temp"
		// TODO: prepare input with 1 common file
	)

	t.Run("create a repo", func(t *testing.T) {
		runCmd(t, []string{"repo",
			"create",
			"--description", "test-purge",
			"--repo", repo11,
			"--context", testContext(),
		}, "create first test repo", false)
		runCmd(t, []string{"repo",
			"create",
			"--description", "testing",
			"--repo", repo12,
			"--context", testContext(),
		}, "create second test repo", false)
		runCmd(t, []string{"bundle",
			"upload",
			"--path", input,
			"--message", "The initial commit for the repo",
			"--repo", repo11,
			"--context", testContext(),
		}, "upload bundle at .", false)
		// TODO: scan blobs for further asserting presence
		runCmd(t, []string{"bundle",
			"upload",
			"--path", input,
			"--message", "The initial commit for the repo",
			"--repo", repo12,
			"--context", testContext(),
		}, fmt.Sprintf("upload bundle at %q", input), false)
		// TODO: scan blobs for further asserting presence
	})

	t.Run("create index", func(t *testing.T) {
		runCmd(t, []string{"purge",
			"build-reverse-lookup",
			"--context", testContext(),
		}, "create index", false)
	})

	t.Run("delete index", func(t *testing.T) {
		runCmd(t, []string{"purge",
			"build-reverse-lookup",
			"--context", testContext(),
		}, "delete index", false)
	})

	t.Run("create index again", func(t *testing.T) {
		runCmd(t, []string{"purge",
			"build-reverse-lookup",
			"--context", testContext(),
		}, "create index again", false)
	})

	t.Run("delete repo11", func(t *testing.T) {
		runCmd(t, []string{"repo",
			"delete",
			"--repo", repo11,
			"--force-yes",
			"--context", testContext(),
		}, "delete repo #1", false)
	})
	// TODO: blobs are all there

	t.Run("refresh index", func(t *testing.T) {
		runCmd(t, []string{"purge",
			"build-reverse-lookup",
			"--context", testContext(),
		}, "refresh index", false)
	})
	// TODO: purge dry run

	t.Run("purge #1", func(t *testing.T) {
		runCmd(t, []string{"purge",
			"delete-unused",
			"--loglevel", "debug",
			"--context", testContext(),
		}, "delete-unused #1", false)
	})
	// TODO: only blobs that pertain to #1 are deleted, common blob is there

	t.Run("delete repo12", func(t *testing.T) {
		runCmd(t, []string{"repo",
			"delete",
			"--repo", repo12,
			"--force-yes",
			"--context", testContext(),
		}, "delete repo #2", false)
	})

	t.Run("refresh index #2", func(t *testing.T) {
		runCmd(t, []string{"purge",
			"build-reverse-lookup",
			"--context", testContext(),
		}, "refresh index #2", false)
	})

	t.Run("purge #2", func(t *testing.T) {
		runCmd(t, []string{"purge",
			"delete-unused",
			"--loglevel", "debug",
			"--context", testContext(),
		}, "delete-unused #2", false)
	})
	// TODO: all blobs are deleted
}
