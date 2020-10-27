package cmd

import (
	"context"
	"time"

	"github.com/oneconcern/datamon/pkg/core"
	"go.uber.org/zap"

	"github.com/spf13/cobra"
)

var repoRename = &cobra.Command{
	Use:   "rename {new repo name}",
	Short: "Rename a repo",
	Long: `Rename an existing datamon repository.

This command MUST NOT BE RUN concurrently.
`,
	Example: `% datamon repo rename --repo ritesh-datamon-test-repo ritesh-datamon-new-repo`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		defer func(t0 time.Time) {
			cliUsage(t0, "repo rename", err)
		}(time.Now())

		newName := args[0]

		ctx := context.Background()
		optionInputs := newCliOptionInputs(config, &datamonFlags)
		remoteStores, err := optionInputs.datamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		logger, err := optionInputs.getLogger()
		logger.Info("renaming repo", zap.String("repo", datamonFlags.repo.RepoName), zap.String("new repo", newName))
		err = core.RenameRepo(datamonFlags.repo.RepoName, newName, remoteStores)
		if err != nil {
			wrapFatalln("rename repo", err)
			return
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
	Args: cobra.MinimumNArgs(1),
}

func init() {
	requireFlags(repoRename,
		addRepoNameOptionFlag(repoRename),
	)
	repoCmd.AddCommand(repoRename)
}
