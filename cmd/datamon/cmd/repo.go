// Copyright © 2018 One Concern

package cmd

import (
	"github.com/spf13/cobra"
)

var repoParams struct {
	Bucket   string
	RepoName string
}

var name = "name"
var bucket = "bucket"

func addRepoNameOptionFlag(cmd *cobra.Command) string {
	cmd.Flags().StringVarP(&repoParams.RepoName, name, "n", "", "The name of this repository")
	return name
}

func addBucketNameFlag(cmd *cobra.Command) string {
	cmd.Flags().StringVarP(&repoParams.RepoName, bucket, "b", "", "The name of the bucket used by datamon")
	return bucket
}
