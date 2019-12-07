package cmd

import (
	"context"

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

		ctx := context.Background()

		remoteStores, err := paramsToDatamonContext(ctx, datamonFlags)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}

		destinationStore, err := paramsToDestStore(datamonFlags, destTMaybeNonEmpty, "")
		if err != nil {
			wrapFatalln("create destination store", err)
			return
		}

		err = setLatestOrLabelledBundle(ctx, remoteStores)
		if err != nil {
			wrapFatalln("determine bundle id", err)
			return
		}

		bundleOpts := paramsToBundleOpts(remoteStores)
		bundleOpts = append(bundleOpts, core.Repo(datamonFlags.repo.RepoName))
		bundleOpts = append(bundleOpts, core.ConsumableStore(destinationStore))
		bundleOpts = append(bundleOpts, core.BundleID(datamonFlags.bundle.ID))

		bundle := core.NewBundle(core.NewBDescriptor(),
			bundleOpts...,
		)

		err = core.PublishFile(ctx, bundle, datamonFlags.bundle.File)
		if err != nil {
			wrapFatalln("publish bundle", err)
			return
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		config.populateRemoteConfig(&datamonFlags)
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
