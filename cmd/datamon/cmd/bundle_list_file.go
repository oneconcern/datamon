package cmd

import (
	"context"
	"fmt"

	"github.com/oneconcern/datamon/pkg/core"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/spf13/cobra"
)

var bundleFileList = &cobra.Command{
	Use:   "files",
	Short: "List files in a bundle",
	Long:  "List all the files in a bundle",
	Run: func(cmd *cobra.Command, args []string) {

		store, err := gcs.New(repoParams.MetadataBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}
		err = setLatestBundle(store)
		if err != nil {
			logFatalln(err)
		}
		bundle := core.Bundle{
			RepoID:           repoParams.RepoName,
			BundleID:         bundleOptions.ID,
			MetaStore:        store,
			ConsumableStore:  nil,
			BlobStore:        nil,
			BundleDescriptor: model.BundleDescriptor{},
			BundleEntries:    nil,
		}
		err = core.PopulateFiles(context.Background(), &bundle)
		if err != nil {
			logFatalln(err)
		}
		for _, e := range bundle.BundleEntries {
			fmt.Printf("name:%s, size:%d, hash:%s\n", e.NameWithPath, e.Size, e.Hash)
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

	for _, flag := range requiredFlags {
		err := BundleDownloadCmd.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}
	BundleListCommand.AddCommand(bundleFileList)
}
