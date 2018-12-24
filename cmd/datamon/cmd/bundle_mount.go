// Copyright © 2018 One Concern

package cmd

import (
	"log"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"

	"github.com/spf13/cobra"
)

// Mount a read only view of a bundle
var mountBundleCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount a bundle",
	Long:  "Mount a readonly, non-interactive view of the entire data that is part of a bundle",
	Run: func(cmd *cobra.Command, args []string) {

		DieIfNotAccessible(bundleOptions.DataPath)

		sourceStore := gcs.New(repoParams.Bucket)
		destinationStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), bundleOptions.DataPath))
		archiveBundle := core.NewBundle(repoParams.RepoName, bundleOptions.ID, sourceStore, destinationStore)
		fs, err := core.NewDatamonFS(archiveBundle)
		if err != nil {
			log.Fatalln(err)
		}
		err = fs.Mount(bundleOptions.DataPath)
		if err != nil {
			log.Fatalln(err)
		}
	},
}

func init() {

	// Source
	requiredFlags := []string{addBucketNameFlag(mountBundleCmd)}
	requiredFlags = append(requiredFlags, addRepoNameOptionFlag(mountBundleCmd))

	// Bundle to mount
	requiredFlags = append(requiredFlags, addBundleFlag(mountBundleCmd))

	// Destination
	requiredFlags = append(requiredFlags, addDataPathFlag(mountBundleCmd))

	for _, flag := range requiredFlags {
		err := mountBundleCmd.MarkFlagRequired(flag)
		if err != nil {
			log.Fatalln(err)
		}
	}

	bundleCmd.AddCommand(mountBundleCmd)
}
