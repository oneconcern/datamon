package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oneconcern/datamon/pkg/core"
	"go.uber.org/zap"

	"github.com/spf13/cobra"
)

var repoDelete = &cobra.Command{
	Use:   "delete",
	Short: "Delete a named repo",
	Long: `Delete an existing datamon repository.

You must authenticate to perform this operation (can't --skip-auth).
You must specify the context with --context.

This command MUST NOT BE RUN concurrently.
`,
	Example: `% datamon repo delete --repo ritesh-datamon-test-repo --context dev`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		defer func(t0 time.Time) {
			cliUsage(t0, "repo delete", err)
		}(time.Now())

		ctx := context.Background()
		optionInputs := newCliOptionInputs(config, &datamonFlags)
		remoteStores, err := optionInputs.datamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		logger, err := optionInputs.getLogger()

		if !datamonFlags.root.forceYes && !userConfirm("delete") {
			wrapFatalln("user aborted", nil)
			return
		}

		logger.Info("deleting repo", zap.String("repo", datamonFlags.repo.RepoName))
		err = core.DeleteRepo(datamonFlags.repo.RepoName, remoteStores)
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

func userConfirm(action string) bool {
	log.Printf("Are you sure you want to %s from repo medata for %q [y|n]", action, datamonFlags.repo.RepoName)
	var answer string
	fmt.Scanln(&answer)
	yesno := strings.ToLower(answer)
	return yesno == "y" || yesno == "yes"
}
