package cmd

import (
	"context"
	"log"

	"github.com/oneconcern/datamon/pkg/core"

	"github.com/spf13/cobra"
)

var bundleFileList = &cobra.Command{
	Use:   "files",
	Short: "List files in a bundle",
	Long:  "List all the files in a bundle",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		err = setLatestOrLabelledBundle(ctx, remoteStores.meta)
		if err != nil {
			wrapFatalln("determine bundle id", err)
			return
		}
		bundle := core.New(core.NewBDescriptor(),
			core.Repo(params.repo.RepoName),
			core.MetaStore(remoteStores.meta),
			core.BundleID(params.bundle.ID),
		)
		err = core.PopulateFiles(context.Background(), bundle)
		if err != nil {
			wrapFatalln("download filelist", err)
			return
		}
		for _, e := range bundle.BundleEntries {
			log.Printf("name:%s, size:%d, hash:%s", e.NameWithPath, e.Size, e.Hash)
		}
	},
}

func init() {

	// Source
	requiredFlags := []string{addRepoNameOptionFlag(bundleFileList)}

	// Bundle to download
	addBundleFlag(bundleFileList)

	addBlobBucket(bundleFileList)
	addBucketNameFlag(bundleFileList)
	addLabelNameFlag(bundleFileList)

	for _, flag := range requiredFlags {
		err := BundleDownloadCmd.MarkFlagRequired(flag)
		if err != nil {
			wrapFatalln("mark required flag", err)
			return
		}
	}
	BundleListCommand.AddCommand(bundleFileList)
}
