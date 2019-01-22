// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"log"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// downloadBundleCmd is the command to download a specific bundle from Datamon and model it locally. The primary purpose
// is to get a readonly view for the data that is part of a bundle.
var downloadBundleCmd = &cobra.Command{
	Use:   "download",
	Short: "Download a bundle",
	Long:  "Download a readonly, non-interactive view of the entire data that is part of a bundle",
	Run: func(cmd *cobra.Command, args []string) {

		DieIfNotAccessible(bundleOptions.DataPath)

		sourceStore, err := gcs.New(repoParams.Bucket)
		if err != nil {
			log.Fatalln(err)
		}
		destinationStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), bundleOptions.DataPath))
		archiveBundle := core.NewBundle(repoParams.RepoName, bundleOptions.ID, sourceStore)
		consumableBundle := core.NewBundle(repoParams.RepoName, bundleOptions.ID, destinationStore)
		err = core.Publish(context.Background(), archiveBundle, consumableBundle)
		if err != nil {
			log.Fatalln(err)
		}
	},
}

func init() {

	// Source
	requiredFlags := []string{addBucketNameFlag(downloadBundleCmd)}
	requiredFlags = append(requiredFlags, addRepoNameOptionFlag(downloadBundleCmd))

	// Bundle to download
	requiredFlags = append(requiredFlags, addBundleFlag(downloadBundleCmd))

	// Destination
	requiredFlags = append(requiredFlags, addDataPathFlag(downloadBundleCmd))

	for _, flag := range requiredFlags {
		err := downloadBundleCmd.MarkFlagRequired(flag)
		if err != nil {
			log.Fatalln(err)
		}
	}

	bundleCmd.AddCommand(downloadBundleCmd)
}
