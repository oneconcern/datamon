package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/oneconcern/datamon/pkg/core"
	"go.uber.org/zap"

	"github.com/spf13/cobra"
)

var repoDeleteFiles = &cobra.Command{
	Use:   "files",
	Short: "Deletes files from a named repo, altering all bundles",
	Long: `Deletes files in a file list from all bundles in an existing datamon repository.

You must authenticate to perform this operation (can't --skip-auth).
You must specify the context with --context.

This command MUST NOT BE RUN concurrently.
`,
	Example: `
% datamon repo delete files --repo ritesh-datamon-test-repo --files file-list.txt --context dev

% datamon repo delete files --repo ritesh-datamon-test-repo --file path/file-to-delete --context dev
`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		defer func(t0 time.Time) {
			cliUsage(t0, "repo delete files", err)
		}(time.Now())

		ctx := context.Background()
		optionInputs := newCliOptionInputs(config, &datamonFlags)
		logger, err := optionInputs.getLogger()

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
			wrapFatalln("must specify at least one file or file list", nil)
			return
		}

		if !datamonFlags.root.forceYes && !userConfirm("delete repo files") {
			wrapFatalln("user aborted", nil)
			return
		}

		logger.Info("deleting files from repo", zap.String("repo", datamonFlags.repo.RepoName))
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
