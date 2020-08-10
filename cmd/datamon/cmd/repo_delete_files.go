package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/oneconcern/datamon/pkg/core"

	"github.com/spf13/cobra"
)

var repoDeleteFiles = &cobra.Command{
	Use:   "files",
	Short: "Deletes files from a named repo, altering all bundles",
	Long: `Deletes files in a file list from all bundles in an existing datamon repository.

This command MUST NOT BE RUN concurrently.
`,
	Example: `
% datamon repo delete files --repo ritesh-datamon-test-repo --files file-list.txt

% datamon repo delete files --repo ritesh-datamon-test-repo --file path/file-to-delete
`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		defer func(t0 time.Time) {
			cliUsage(t0, "repo delete files", err)
		}(time.Now())

		ctx := context.Background()
		optionInputs := newCliOptionInputs(config, &datamonFlags)
		remoteStores, err := optionInputs.datamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		var files []string
		if datamonFlags.bundle.FileList != "" {
			files, err = fileList(datamonFlags.bundle.FileList)
			if err != nil {
				wrapFatalln("reading file list", err)
				return
			}
		} else {
			if datamonFlags.bundle.File != "" {
				files = []string{datamonFlags.bundle.File}
			}
		}
		if len(files) == 0 {
			wrapFatalln("must specify at list a file or file list", nil)
			return
		}

		err = core.DeleteEntriesFromRepo(datamonFlags.repo.RepoName, remoteStores, files)
		if err != nil {
			wrapFatalln("delete repo", err)
			return
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

func init() {
	requireFlags(repoDeleteFiles,
		addRepoNameOptionFlag(repoDeleteFiles),
	)
	addFileListFlag(repoDeleteFiles)
	addBundleFileFlag(repoDeleteFiles)

	repoDelete.AddCommand(repoDeleteFiles)
}

func fileList(index string) ([]string, error) {
	file, err := os.Open(index)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %s: %w", datamonFlags.bundle.FileList, err)
	}
	lineScanner := bufio.NewScanner(file)
	files := make([]string, 0)
	for lineScanner.Scan() {
		files = append(files, lineScanner.Text())
	}
	return files, nil
}
