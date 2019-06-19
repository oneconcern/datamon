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

		metadataStore, err := gcs.New(params.repo.MetadataBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}
		blobStore, err := gcs.New(params.repo.BlobBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}
		path, err := filepath.Abs(filepath.Clean(params.bundle.DataPath))
		if err != nil {
			logFatalf("Failed path validation: %s", err)
		}
		_ = os.MkdirAll(path, 0700)
		destinationStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), path))
		err = setLatestOrLabelledBundle(metadataStore)
		if err != nil {
			logFatalln(err)
		}
		bd := core.NewBDescriptor()
		bundle := core.New(bd,
			core.Repo(params.repo.RepoName),
			core.MetaStore(metadataStore),
			core.ConsumableStore(destinationStore),
			core.BlobStore(blobStore),
			core.BundleID(params.bundle.ID),
		)

		err = core.PublishFile(context.Background(), bundle, params.bundle.File)
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
	addLabelNameFlag(bundleDownloadFileCmd)

	for _, flag := range requiredFlags {
		err := bundleDownloadFileCmd.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	BundleDownloadCmd.AddCommand(bundleDownloadFileCmd)
}
