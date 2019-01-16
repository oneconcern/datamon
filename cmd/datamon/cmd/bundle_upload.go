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

// uploadBundleCmd is the command to upload a bundle from Datamon and model it locally.
var uploadBundleCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a bundle",
	Long:  "Upload a bundle consisting of all files stored in a directory",
	Run: func(cmd *cobra.Command, args []string) {

		DieIfNotAccessible(bundleOptions.DataPath)

		destinationStore, err := gcs.New(repoParams.Bucket)
		if err != nil {
			log.Fatalln(err)
		}

		sourceStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), bundleOptions.DataPath))

		archiveBundle := core.NewBundle(repoParams.RepoName, bundleOptions.ID, sourceStore)
		consumableBundle := core.NewBundle(repoParams.RepoName, bundleOptions.ID, destinationStore)

		err = core.Publish(context.Background(), archiveBundle, consumableBundle)
		if err != nil {
			log.Fatalln(err)
		}
	},
}

func init() {

	requiredFlags := []string{addBucketNameFlag(uploadBundleCmd)}
	requiredFlags = append(requiredFlags, addRepoNameOptionFlag(uploadBundleCmd))
	requiredFlags = append(requiredFlags, addDataPathFlag(uploadBundleCmd))

	for _, flag := range requiredFlags {
		err := downloadBundleCmd.MarkFlagRequired(flag)
		if err != nil {
			log.Fatalln(err)
		}
	}

	bundleCmd.AddCommand(uploadBundleCmd)
}
