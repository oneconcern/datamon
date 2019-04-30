package cmd

import (
	"time"

	"github.com/oneconcern/datamon/pkg/storage/gcs"

	"github.com/oneconcern/datamon/pkg/core"

	"github.com/oneconcern/datamon/pkg/model"

	"github.com/spf13/cobra"
)

var repoCreate = &cobra.Command{
	Use:   "create",
	Short: "Create a named repo",
	Long: "Create a repo. Repo names must not contain special characters. " +
		"Allowed characters Unicode characters, digits and hyphen. Example: dm-test-repo-1",
	Run: func(cmd *cobra.Command, args []string) {
		store, err := gcs.New(repoParams.MetadataBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}

		repo := model.RepoDescriptor{
			Name:        repoParams.RepoName,
			Description: repoParams.Description,
			Timestamp:   time.Now(),
			Contributor: model.Contributor{
				Email: repoParams.ContributorEmail,
				Name:  repoParams.ContributorName,
			},
		}
		err = core.CreateRepo(repo, store)
		if err != nil {
			logFatalln(err)
		}
	},
}

func init() {

	// Metadata bucket
	requiredFlags := []string{addRepoNameOptionFlag(repoCreate)}
	// Description
	requiredFlags = append(requiredFlags, addRepoDescription(repoCreate))

	addContributorEmail(repoCreate)
	addContributorName(repoCreate)
	addBucketNameFlag(repoCreate)

	for _, flag := range requiredFlags {
		err := repoCreate.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	repoCmd.AddCommand(repoCreate)
}
