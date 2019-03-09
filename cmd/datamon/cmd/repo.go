// Copyright © 2018 One Concern

package cmd

import (
	"github.com/spf13/cobra"
)

// repoCmd represents the repo related commands
var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Commands to manage repos",
	Long: `Commands to manage repos.

Repos are datasets that are versioned and managed via bundles.
`,
}

func init() {
	rootCmd.AddCommand(repoCmd)
}

type RepoParams struct {
	MetadataBucket   string
	RepoName         string
	BlobBucket       string
	Description      string
	ContributorEmail string
	ContributorName  string
}

var repoParams = RepoParams{}

func addRepoNameOptionFlag(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&repoParams.RepoName, repo, "", "The name of this repository")
	return repo
}

func addBucketNameFlag(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&repoParams.MetadataBucket, meta, "datamon-meta-data", "The name of the bucket used by datamon metadata")
	return meta
}

func addRepoDescription(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&repoParams.Description, description, "", "The description for the repo")
	return description
}

func addBlobBucket(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&repoParams.BlobBucket, blob, "datamon-blob-data", "The name of the bucket hosting the datamon blobs")
	return blob
}

func addContributorEmail(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&repoParams.ContributorEmail, contributorEmail, "", "The email of the contributor")
	return contributorEmail
}
func addContributorName(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&repoParams.ContributorName, contributorName, "", "The name of the contributor")
	return contributorName
}
