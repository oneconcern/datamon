package cmd

import (
	"context"
	"os"
	"path/filepath"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var bundleDownloadFileCmd = &cobra.Command{
	Use:   "file",
	Short: "Download a file from bundle",
	Long:  "Download a readonly, non-interactive view of a single file from a bundle",
	Run: func(cmd *cobra.Command, args []string) {

		metadataStore, err := gcs.New(repoParams.MetadataBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}
		blobStore, err := gcs.New(repoParams.BlobBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}
		path, err := filepath.Abs(filepath.Clean(bundleOptions.DataPath))
		if err != nil {
			logFatalf("Failed path validation: %s", err)
		}
		_ = os.MkdirAll(path, 0700)
		destinationStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), path))
		err = setLatestBundle(metadataStore)
		if err != nil {
			logFatalln(err)
		}
		bd := core.NewBDescriptor()
		bundle := core.New(bd,
			core.Repo(repoParams.RepoName),
			core.MetaStore(metadataStore),
			core.ConsumableStore(destinationStore),
			core.BlobStore(blobStore),
			core.BundleID(bundleOptions.ID),
		)

		err = core.PublishFile(context.Background(), bundle, bundleOptions.File)
		if err != nil {
			logFatalln(err)
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

	for _, flag := range requiredFlags {
		err := bundleDownloadFileCmd.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	BundleDownloadCmd.AddCommand(bundleDownloadFileCmd)
}
