package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"

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

		ctx := context.Background()
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}

		var files []string
		if params.bundle.FileList != "" {
			files, err = fileList(params.bundle.FileList)
			if err != nil {
				wrapFatalln("reading file list", err)
				return
			}
		} else {
			if params.bundle.File != "" {
				files = []string{params.bundle.File}
			}
		}
		if len(files) == 0 {
			wrapFatalln("must specify at list a file or file list", nil)
			return
		}

		err = core.DeleteEntriesFromRepo(params.repo.RepoName, remoteStores.meta, files)
		if err != nil {
			wrapFatalln("delete repo", err)
			return
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	err := repoDeleteFiles.MarkFlagRequired(addRepoNameOptionFlag(repoDeleteFiles))
	if err != nil {
		wrapFatalln("mark required flag", err)
		return
	}
	addFileListFlag(repoDeleteFiles)
	addBundleFileFlag(repoDeleteFiles)

	repoDelete.AddCommand(repoDeleteFiles)
}

func fileList(index string) ([]string, error) {
	file, err := os.Open(index)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %s: %w", index, err)
	}
	lineScanner := bufio.NewScanner(file)
	files := make([]string, 0)
	for lineScanner.Scan() {
		files = append(files, lineScanner.Text())
	}
	return files, nil
}
