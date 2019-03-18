// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"log"
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
	Long:  "Download a readonly, non-interactive view of the entire data that is part of a bundle",
	Run: func(cmd *cobra.Command, args []string) {

		sourceStore, err := gcs.New(repoParams.MetadataBucket, config.Credential)
		if err != nil {
			log.Fatalln(err)
		}
		blobStore, err := gcs.New(repoParams.BlobBucket, config.Credential)
		if err != nil {
			log.Fatalln(err)
		}
		path, err := filepath.Abs(filepath.Clean(bundleOptions.DataPath))
		if err != nil {
			log.Fatalf("Failed path validation: %s", err)
		}
		// Ignore error
		_ = os.MkdirAll(path, 0700)
		fs := afero.NewBasePathFs(afero.NewOsFs(), path)
		empty, err := afero.IsEmpty(fs, "/")
		if err != nil {
			log.Fatalf("Failed path validation: %s", err)
		}
		if !empty {
			log.Fatalf("%s should be empty", path)
		}
		destinationStore := localfs.New(fs)

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
			log.Fatalln(err)
		}
	},
}

func init() {

	// Source
	requiredFlags := []string{addRepoNameOptionFlag(BundleDownloadCmd)}

	// Bundle to download
	requiredFlags = append(requiredFlags, addBundleFlag(BundleDownloadCmd))

	// Destination
	requiredFlags = append(requiredFlags, addDataPathFlag(BundleDownloadCmd))

	// Blob bucket
	addBlobBucket(BundleDownloadCmd)
	addBucketNameFlag(BundleDownloadCmd)

	for _, flag := range requiredFlags {
		err := BundleDownloadCmd.MarkFlagRequired(flag)
		if err != nil {
			log.Fatalln(err)
		}
	}

	bundleCmd.AddCommand(BundleDownloadCmd)
}
