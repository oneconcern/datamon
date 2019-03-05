// Copyright © 2018 One Concern

package cmd

import (
	"github.com/spf13/cobra"
)

var repoParams struct {
	MetadataBucket string
	RepoName       string
	BlobBucket     string
}

var name = "name"
var bucket = "meta"
var blob = "blob"

func addRepoNameOptionFlag(cmd *cobra.Command) string {
	cmd.Flags().StringVarP(&repoParams.RepoName, name, "n", "", "The name of this repository")
	return name
}

func addBucketNameFlag(cmd *cobra.Command) string {
	cmd.Flags().StringVarP(&repoParams.MetadataBucket, bucket, "m", "", "The name of the bucket used by datamon metadata")
	return bucket
}

func addBlobBucket(cmd *cobra.Command) string {
	cmd.Flags().StringVarP(&repoParams.BlobBucket, blob, "b", "", "The name of the bucket hosting the datamon blobs")
	return name
}
