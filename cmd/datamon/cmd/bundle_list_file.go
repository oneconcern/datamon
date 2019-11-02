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
		remoteStores, err := paramsToDatamonContext(ctx, datamonFlags)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		err = setLatestOrLabelledBundle(ctx, remoteStores)
		if err != nil {
			wrapFatalln("determine bundle id", err)
			return
		}
		bundleOpts := paramsToBundleOpts(remoteStores)
		bundleOpts = append(bundleOpts, core.Repo(datamonFlags.repo.RepoName))
		bundleOpts = append(bundleOpts, core.BundleID(datamonFlags.bundle.ID))
		bundle := core.NewBundle(core.NewBDescriptor(),
			bundleOpts...,
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
	PreRun: func(cmd *cobra.Command, args []string) {
		config.populateRemoteConfig(&datamonFlags)
	},
}

func init() {

	// Source
	requiredFlags := []string{addRepoNameOptionFlag(bundleFileList)}

	// Bundle to download
	addBundleFlag(bundleFileList)

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
