package cmd

import (
	"context"
	"time"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/spf13/cobra"
)

var bundleDownloadFileCmd = &cobra.Command{
	Use:   "file",
	Short: "Download a file from bundle",
	Long: `Download a readonly, non-interactive view of a single file
from a bundle.

You may use the "--label" flag as an alternate way to specify a particular bundle.
`,
	Example: `% datamon bundle download file --file datamon/cmd/repo_list.go --repo ritesh-test-repo --bundle 1ISwIzeAR6m3aOVltAsj1kfQaml --destination /tmp`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		defer func(t0 time.Time) {
			cliUsage(t0, "bundle download file", err)
		}(time.Now())

		ctx := context.Background()

		optionInputs := newCliOptionInputs(config, &datamonFlags)
		remoteStores, err := optionInputs.datamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}

		destinationStore, err := optionInputs.destStore(destTMaybeNonEmpty, "")
		if err != nil {
			wrapFatalln("create destination store", err)
			return
		}

		err = setLatestOrLabelledBundle(ctx, remoteStores)
		if err != nil {
			wrapFatalln("determine bundle id", err)
			return
		}

		bundleOpts, err := optionInputs.bundleOpts(ctx)
		if err != nil {
			wrapFatalln("failed to initialize bundle options", err)
		}
		bundleOpts = append(bundleOpts, core.Repo(datamonFlags.repo.RepoName))
		bundleOpts = append(bundleOpts, core.ConsumableStore(destinationStore))
		bundleOpts = append(bundleOpts, core.BundleID(datamonFlags.bundle.ID))
		bundleOpts = append(bundleOpts, core.BundleWithMetrics(datamonFlags.root.metrics.IsEnabled()))

		bundle := core.NewBundle(
			bundleOpts...,
		)

		err = core.PublishFile(ctx, bundle, datamonFlags.bundle.File)
		if err != nil {
			wrapFatalln("publish bundle", err)
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
	requireFlags(bundleDownloadFileCmd,
		addRepoNameOptionFlag(bundleDownloadFileCmd),
		addDataPathFlag(bundleDownloadFileCmd),
		addBundleFileFlag(bundleDownloadFileCmd),
	)

	addLabelNameFlag(bundleDownloadFileCmd)
	addBundleFlag(bundleDownloadFileCmd)

	BundleDownloadCmd.AddCommand(bundleDownloadFileCmd)
}
