// Copyright © 2018 One Concern

package cmd

import (
	"github.com/oneconcern/pipelines/pkg/log"
	"github.com/spf13/cobra"
)

var logger log.Factory

var repoParams struct {
	Bucket   string
	RepoName string
}

func addRepoNameOptionFlag(cmd *cobra.Command) error {
	flags := cmd.Flags()
	flags.StringVarP(&repoParams.RepoName, "name", "n", "", "The name of this repository")
	return cmd.MarkFlagRequired("name")
}

func addBucketNameFlag(cmd *cobra.Command) error {
	flags := cmd.Flags()
	flags.StringVarP(&repoParams.RepoName, "bucket", "b", "", "The name of the bucket used by datamon")
	return cmd.MarkFlagRequired("bucket")
}
