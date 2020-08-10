package cmd

import (
	"context"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/dlogger"
	"go.uber.org/zap"

	"github.com/spf13/cobra"
)

var repoDelete = &cobra.Command{
	Use:   "delete",
	Short: "Delete a named repo",
	Long: `Delete an existing datamon repository.

This command MUST NOT BE RUN concurrently.
`,
	Example: `% datamon repo delete --repo ritesh-datamon-test-repo`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		ctx := context.Background()

		logger, err := dlogger.GetLogger(params.root.logLevel)
		if err != nil {
			wrapFatalln("failed to set log level", err)
			return
		}

		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}

		logger.Info("deleting repo", zap.String("repo", params.repo.RepoName))
		err = core.DeleteRepo(params.repo.RepoName, remoteStores.meta)
		if err != nil {
			wrapFatalln("delete repo", err)
			return
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	err := repoDelete.MarkFlagRequired(addRepoNameOptionFlag(repoDelete))
	if err != nil {
		wrapFatalln("mark required flag", err)
		return
	}
	repoCmd.AddCommand(repoDelete)
}
