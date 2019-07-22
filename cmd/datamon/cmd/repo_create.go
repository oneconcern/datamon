package cmd

import (
	"context"
	"time"

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
		ctx := context.Background()
		contributor, err := paramsToContributor(params)
		if err != nil {
			logFatalln(err)
		}
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			logFatalln(err)
		}
		repo := model.RepoDescriptor{
			Name:        params.repo.RepoName,
			Description: params.repo.Description,
			Timestamp:   time.Now(),
			Contributor: contributor,
		}
		err = core.CreateRepo(repo, remoteStores.meta)
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
