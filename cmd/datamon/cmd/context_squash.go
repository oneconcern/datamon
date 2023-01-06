package cmd

import (
	"context"
	"time"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var contextSquash = &cobra.Command{
	Use:   "squash",
	Short: "Squash the history of all repos in a context",
	Long: `Squash all repos in a context so that only the latest bundle remains.

Optionally, the squashing may also retain past tagged bundles, or only past tagged bundles with a legit semver tag.

This command is equivalent to repeating "datamon repo squash" over all the repos of a context.
`,
	Example: `% datamon context squash  --retain-semver-tags --context dev`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		defer func(t0 time.Time) {
			cliUsage(t0, "context squash", err)
		}(time.Now())

		ctx := context.Background()
		optionInputs := newCliOptionInputs(config, &datamonFlags)
		remoteStores, err := optionInputs.datamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		logger, err := optionInputs.getLogger()
		if err != nil {
			wrapFatalln("create logger", err)
			return
		}

		// retrieve all repos in this context
		err = core.ListReposApply(remoteStores, applyRepoSquash(remoteStores, &datamonFlags, logger),
			core.ConcurrentList(datamonFlags.core.ConcurrencyFactor),
			core.BatchSize(datamonFlags.core.BatchSize),
			core.WithMetrics(datamonFlags.root.metrics.IsEnabled()),
		)
		if err != nil {
			wrapFatalln("download repo list", err)
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

func applyRepoSquash(remoteStores context2.Stores, datamonFlags *flagsT, logger *zap.Logger) func(model.RepoDescriptor) error {
	return func(repo model.RepoDescriptor) error {
		logger.Info("squashing repo",
			zap.String("repo", repo.Name),
			zap.Bool("retain all tags", datamonFlags.squash.RetainTags),
			zap.Bool("retain semver tags", datamonFlags.squash.RetainSemverTags),
		)

		return core.RepoSquash(remoteStores, repo.Name,
			core.WithRetainTags(datamonFlags.squash.RetainTags),
			core.WithRetainSemverTags(datamonFlags.squash.RetainSemverTags),
			core.WithRetainNLatest(datamonFlags.squash.RetainNLatest),
			core.ConcurrentList(datamonFlags.core.ConcurrencyFactor),
			core.BatchSize(datamonFlags.core.BatchSize),
		)
	}
}

func init() {
	requireFlags(contextSquash,
		addContextFlag(contextSquash),
	)
	addRetainTagsFlag(contextSquash)
	addRetainSemverTagsFlag(contextSquash)
	addRetainNLatestFlag(contextSquash)
	addCoreConcurrencyFactorFlag(contextSquash, 500)
	addBatchSizeFlag(contextSquash)

	ContextCmd.AddCommand(contextSquash)
}
