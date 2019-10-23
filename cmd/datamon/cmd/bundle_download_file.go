package cmd

import (
	"context"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/spf13/cobra"
)

var bundleDownloadFileCmd = &cobra.Command{
	Use:   "file",
	Short: "Download a file from bundle",
	Long:  "Download a readonly, non-interactive view of a single file from a bundle",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		destinationStore, err := paramsToDestStore(params, destTMaybeNonEmpty, "")
		if err != nil {
			wrapFatalln("create destination store", err)
			return
		}

		err = setLatestOrLabelledBundle(ctx, remoteStores.meta)
		if err != nil {
			wrapFatalln("determine bundle id", err)
			return
		}
		bd := core.NewBDescriptor()
		bundle := core.New(bd,
			core.Repo(params.repo.RepoName),
			core.MetaStore(remoteStores.meta),
			core.ConsumableStore(destinationStore),
			core.BlobStore(remoteStores.blob),
			core.BundleID(params.bundle.ID),
		)

		err = core.PublishFile(ctx, bundle, params.bundle.File)
		if err != nil {
			wrapFatalln("publish bundle", err)
			return
		}
	},
}

func init() {

	requiredFlags := []string{addRepoNameOptionFlag(bundleDownloadFileCmd)}
	addBundleFlag(bundleDownloadFileCmd)
	requiredFlags = append(requiredFlags, addDataPathFlag(bundleDownloadFileCmd))
	requiredFlags = append(requiredFlags, addBundleFileFlag(bundleDownloadFileCmd))

	addBlobBucket(bundleDownloadFileCmd)
	addBucketNameFlag(bundleDownloadFileCmd)
	addLabelNameFlag(bundleDownloadFileCmd)

	for _, flag := range requiredFlags {
		err := bundleDownloadFileCmd.MarkFlagRequired(flag)
		if err != nil {
			wrapFatalln("mark required flag", err)
			return
		}
	}

	BundleDownloadCmd.AddCommand(bundleDownloadFileCmd)
}
