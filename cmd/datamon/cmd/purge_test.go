package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gcsStorage "cloud.google.com/go/storage"
	"github.com/oneconcern/datamon/internal/rand"
	"github.com/oneconcern/datamon/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func TestPurgeRepo(t *testing.T) {
	cleanup := setupTests(t)
	defer cleanup()

	const (
		repo11 = "purge-test-repo11"
		repo12 = "purge-test-repo12"
		repo13 = "purge-test-repo13"
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
	defer func() {
		_ = os.RemoveAll(".datamon-index")
	}()

	for i, file := range []string{"file1", "file2", "file3", "file4"} {
		content := rand.Bytes(1024)

		switch i {
		case 1:
			require.NoError(t,
				os.WriteFile(filepath.Join(input1, file), content, 0600),
			)
		case 3:
			require.NoError(t,
				os.WriteFile(filepath.Join(input2, file), content, 0600),
			)
		default:
			require.NoError(t,
				os.WriteFile(filepath.Join(input1, file), content, 0600),
			)
			require.NoError(t,
				os.WriteFile(filepath.Join(input2, file), content, 0600),
			)
		}
	}

	content := rand.Bytes(1024)
	require.NoError(t,
		os.WriteFile(filepath.Join(input3, "file5"), content, 0600),
	)

	dcontext := testContext()
	const expectedBlobs = 8

	t.Run("create a repo", func(t *testing.T) {
		runCmd(t, []string{"repo",
			"create",
			"--description", "test-purge",
			"--repo", repo11,
			"--context", dcontext,
		}, "create first test repo", false)

		runCmd(t, []string{"repo",
			"create",
			"--description", "testing",
			"--repo", repo12,
			"--context", dcontext,
		}, "create second test repo", false)

		runCmd(t, []string{"repo",
			"create",
			"--description", "testing",
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

		blobKeys := getBlobKeys(t)
		require.Len(t, blobKeys, expectedBlobs)
	})

	t.Run("create index", func(t *testing.T) {
		r, w := startCapture(t)
		runCmd(t, []string{"purge",
			"build-reverse-lookup",
			"--context", dcontext,
		}, "create index", false)

		lines := endCapture(t, r, w, []string{})
		var found bool
		for _, line := range lines {
			if strings.Contains(line, "Num entries (blob keys): 8") {
				found = true
				break
			}
		}
		require.True(t, found)
	})

	t.Run("delete index", func(t *testing.T) {
		runCmd(t, []string{"purge",
			"build-reverse-lookup",
			"--context", dcontext,
		}, "delete index", false)
	})

	t.Run("create index again", func(t *testing.T) {
		r, w := startCapture(t)
		runCmd(t, []string{"purge",
			"build-reverse-lookup",
			"--context", dcontext,
		}, "create index again", false)

		lines := endCapture(t, r, w, []string{})
		var found bool
		for _, line := range lines {
			if strings.Contains(line, "Num entries (blob keys): 8") {
				found = true
				break
			}
		}
		require.True(t, found)
	})

	t.Run("delete repo11", func(t *testing.T) {
		runCmd(t, []string{"repo",
			"delete",
			"--repo", repo11,
			"--force-yes",
			"--context", dcontext,
		}, "delete repo #1", false)

		blobKeysAfterDelete := getBlobKeys(t)
		require.Len(t, blobKeysAfterDelete, expectedBlobs)
	})

	t.Run("refresh index", func(t *testing.T) {
		r, w := startCapture(t)
		runCmd(t, []string{"purge",
			"build-reverse-lookup",
			"--context", dcontext,
		}, "refresh index", false)

		lines := endCapture(t, r, w, []string{})
		var found bool
		for _, line := range lines {
			if strings.Contains(line, "Num entries (blob keys): 6") {
				found = true
				break
			}
		}
		require.True(t, found)
	})

	t.Run("purge #1", func(t *testing.T) {
		r, w := startCapture(t)
		runCmd(t, []string{"purge",
			"delete-unused",
			"--loglevel", "debug",
			"--context", dcontext,
		}, "delete-unused #1", false)

		blobKeysAfterPurge1 := getBlobKeys(t)
		require.Len(t, blobKeysAfterPurge1, 6) // deleted only storage for file1 (-2 blob keys)

		lines := endCapture(t, r, w, []string{})
		var found int
		for _, line := range lines {
			switch {
			case strings.Contains(line, "Num blob keys scanned: 8"):
				fallthrough
			case strings.Contains(line, "Num blob keys found in use: 6"):
				fallthrough
			case strings.Contains(line, "Num blob keys found more recent than index: 0"):
				fallthrough
			case strings.Contains(line, "Num blob keys deleted: 2"):
				found++
				continue
			}
		}
		require.Equal(t, 4, found)
	})

	t.Run("delete repo12", func(t *testing.T) {
		runCmd(t, []string{"repo",
			"delete",
			"--repo", repo12,
			"--force-yes",
			"--context", dcontext,
		}, "delete repo #2", false)
	})

	t.Run("refresh index #2", func(t *testing.T) {
		r, w := startCapture(t)
		runCmd(t, []string{"purge",
			"build-reverse-lookup",
			"--context", dcontext,
		}, "refresh index #2", false)

		lines := endCapture(t, r, w, []string{})
		var found bool
		for _, line := range lines {
			if strings.Contains(line, "Num entries (blob keys): 0") {
				found = true
				break
			}
		}
		require.True(t, found)
	})

	// TODO: Add new repo and files, more recent than index

	t.Run("purge #2 - dry run", func(t *testing.T) {
		runCmd(t, []string{"purge",
			"delete-unused",
			"--loglevel", "debug",
			"--context", dcontext,
			"--dry-run",
		}, "delete-unused #2 (dry-run)", false)

		blobKeysAfterPurgeDryRun := getBlobKeys(t)
		require.Len(t, blobKeysAfterPurgeDryRun, 6)
	})

	t.Run("purge #2", func(t *testing.T) {
		r, w := startCapture(t)
		runCmd(t, []string{"purge",
			"delete-unused",
			"--loglevel", "debug",
			"--context", dcontext,
		}, "delete-unused #2", false)

		blobKeysAfterPurge2 := getBlobKeys(t)
		require.Len(t, blobKeysAfterPurge2, 0) // all blobs are deleted

		lines := endCapture(t, r, w, []string{})
		var found int
		for _, line := range lines {
			switch {
			case strings.Contains(line, "Num blob keys scanned: 6"):
				fallthrough
			case strings.Contains(line, "Num blob keys found in use: 0"):
				fallthrough
			case strings.Contains(line, "Num blob keys found more recent than index: 0"):
				fallthrough
			case strings.Contains(line, "Num blob keys deleted: 6"):
				found++
				continue
			}
		}
		require.Equal(t, 4, found)
	})

	t.Run("create new data, with outdated index", func(t *testing.T) {
		runCmd(t, []string{"bundle",
			"upload",
			"--path", input3,
			"--message", "The initial commit for the repo",
			"--repo", repo13,
			"--context", dcontext,
		}, fmt.Sprintf("upload bundle at %q", input3), false)
	})

	t.Run("delete repo13", func(t *testing.T) {
		runCmd(t, []string{"repo",
			"delete",
			"--repo", repo13,
			"--force-yes",
			"--context", dcontext,
		}, "delete repo #3", false)
	})

	t.Run("purge #3", func(t *testing.T) {
		r, w := startCapture(t)
		runCmd(t, []string{"purge",
			"delete-unused",
			"--loglevel", "debug",
			"--context", dcontext,
		}, "delete-unused #3", false)

		blobKeysAfterPurge3 := getBlobKeys(t)
		require.Len(t, blobKeysAfterPurge3, 2) // newer blobs are not deleted

		lines := endCapture(t, r, w, []string{})
		var found int
		for _, line := range lines {
			switch {
			case strings.Contains(line, "Num blob keys scanned: 2"):
				fallthrough
			case strings.Contains(line, "Num blob keys found in use: 0"):
				fallthrough
			case strings.Contains(line, "Num blob keys found more recent than index: 2"):
				fallthrough
			case strings.Contains(line, "Num blob keys deleted: 0"):
				found++
				continue
			}
		}
		require.Equal(t, 4, found)
	})

	t.Run("delete index", func(t *testing.T) {
		runCmd(t, []string{"purge",
			"delete-reverse-lookup",
			"--loglevel", "debug",
			"--context", dcontext,
		}, "delete-index", false)
	})
}

func getBlobKeys(t testing.TB) []string {
	ctx := context.Background()
	client, err := gcsStorage.NewClient(ctx, option.WithScopes(gcsStorage.ScopeFullControl))
	require.NoError(t, err, "couldn't create gcs client")

	bucketBlob := datamonFlags.context.Descriptor.Blob
	it := client.Bucket(bucketBlob).Objects(context.Background(), &gcsStorage.Query{Prefix: ""})

	keys := make([]string, 0, 10)
	for {
		attrs, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}

			require.NoError(t, err)
		}

		keys = append(keys, attrs.Name)
	}

	return keys
}
