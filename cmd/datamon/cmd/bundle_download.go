// Copyright © 2018 One Concern

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

var BundleDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download a bundle",
	Long: "Download a readonly, non-interactive view of the entire data that is part of a bundle. If --bundle is not specified" +
		" the latest bundle will be downloaded",
	Run: func(cmd *cobra.Command, args []string) {

		sourceStore, err := gcs.New(repoParams.MetadataBucket, config.Credential)
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
		// Ignore error
		_ = os.MkdirAll(path, 0700)
		fs := afero.NewBasePathFs(afero.NewOsFs(), path)
		empty, err := afero.IsEmpty(fs, "/")
		if err != nil {
			logFatalf("Failed path validation: %s", err)
		}
		if !empty {
			logFatalf("%s should be empty", path)
		}
		destinationStore := localfs.New(fs)

		err = setLatestBundle(sourceStore)
		if err != nil {
			logFatalln(err)
		}
		bd := core.NewBDescriptor()
		bundle := core.New(bd,
			core.Repo(repoParams.RepoName),
			core.MetaStore(sourceStore),
			core.ConsumableStore(destinationStore),
			core.BlobStore(blobStore),
			core.BundleID(bundleOptions.ID),
		)

		err = core.Publish(context.Background(), bundle)
		if err != nil {
			logFatalln(err)
		}
	},
}

func init() {

	// Source
	requiredFlags := []string{addRepoNameOptionFlag(BundleDownloadCmd)}

	// Destination
	requiredFlags = append(requiredFlags, addDataPathFlag(BundleDownloadCmd))

	// Bundle to download
	addBundleFlag(BundleDownloadCmd)
	// Blob bucket
	addBlobBucket(BundleDownloadCmd)
	addBucketNameFlag(BundleDownloadCmd)

	for _, flag := range requiredFlags {
		err := BundleDownloadCmd.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	bundleCmd.AddCommand(BundleDownloadCmd)
}
