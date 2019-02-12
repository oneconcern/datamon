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

		destinationStore, err := gcs.New(repoParams.MetadataBucket)
		if err != nil {
			log.Fatalln(err)
		}
		blobStore, err := gcs.New(repoParams.BlobBucket)
		if err != nil {
			log.Fatalln(err)
		}

		sourceStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), bundleOptions.DataPath))

		bundle := core.NewBundle(repoParams.RepoName, bundleOptions.ID, sourceStore, destinationStore, blobStore)

		err = core.Publish(context.Background(), bundle)
		if err != nil {
			log.Fatalln(err)
		}
	},
}

func init() {

	requiredFlags := []string{addBucketNameFlag(uploadBundleCmd)}
	requiredFlags = append(requiredFlags, addBlobBucket(uploadBundleCmd))
	requiredFlags = append(requiredFlags, addRepoNameOptionFlag(uploadBundleCmd))
	requiredFlags = append(requiredFlags, addFolderPathFlag(uploadBundleCmd))

	for _, flag := range requiredFlags {
		err := downloadBundleCmd.MarkFlagRequired(flag)
		if err != nil {
			log.Fatalln(err)
		}
	}

	bundleCmd.AddCommand(uploadBundleCmd)
}
