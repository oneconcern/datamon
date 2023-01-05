package cmd

import (
	"context"
	"time"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/spf13/cobra"
)

var repoSquash = &cobra.Command{
	Use:   "squash",
	Short: "Squash the history of a repo",
	Long: `Squash a repo so that only the latest bundle remains.

Optionally, the squashing may also retain past tagged bundles, or only past tagged bundles with a legit semver tag.
`,
	Example: `% datamon repo squash  --retain-semver-tags --repo ritesh-datamon-test-repo`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		defer func(t0 time.Time) {
			cliUsage(t0, "repo squash", err)
		}(time.Now())

		ctx := context.Background()
		optionInputs := newCliOptionInputs(config, &datamonFlags)
		remoteStores, err := optionInputs.datamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}

		// squash history for this repo
		err = core.RepoSquash(remoteStores, datamonFlags.repo.RepoName,
			core.WithRetainTags(datamonFlags.squash.RetainTags),
			core.WithRetainSemverTags(datamonFlags.squash.RetainSemverTags),
			core.ConcurrentList(datamonFlags.core.ConcurrencyFactor),
			core.BatchSize(datamonFlags.core.BatchSize),
		)
		if err != nil {
			wrapFatalln("squash repo", err)
			return
		}

		// report bundles for this repo after squashing
		err = core.ListBundlesApply(datamonFlags.repo.RepoName, remoteStores, applyBundleTemplate,
			core.ConcurrentList(datamonFlags.core.ConcurrencyFactor),
			core.BatchSize(datamonFlags.core.BatchSize),
			core.WithMetrics(datamonFlags.root.metrics.IsEnabled()),
		)
		if err != nil {
			wrapFatalln("concurrent list bundles", err)
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
	requireFlags(repoSquash,
		addRepoNameOptionFlag(repoSquash),
	)
	addRetainTagsFlag(repoSquash)
	addRetainSemverTagsFlag(repoSquash)
	addCoreConcurrencyFactorFlag(repoSquash, 500)
	addBatchSizeFlag(repoSquash)

	repoCmd.AddCommand(repoSquash)
}
