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
		contributor, err := paramsToContributor(datamonFlags)
		if err != nil {
			wrapFatalln("populate contributor struct", err)
			return
		}
		remoteStores, err := paramsToDatamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		repo := model.RepoDescriptor{
			Name:        datamonFlags.repo.RepoName,
			Description: datamonFlags.repo.Description,
			Timestamp:   time.Now(),
			Contributor: contributor,
		}
		err = core.CreateRepo(repo, remoteStores)
		if err != nil {
			wrapFatalln("create repo", err)
			return
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		populateRemoteConfig()
	},
}

func init() {

	// Metadata bucket
	requiredFlags := []string{addRepoNameOptionFlag(repoCreate)}
	// Description
	requiredFlags = append(requiredFlags, addRepoDescription(repoCreate))

	addContributorEmail(repoCreate)
	addContributorName(repoCreate)

	for _, flag := range requiredFlags {
		err := repoCreate.MarkFlagRequired(flag)
		if err != nil {
			wrapFatalln("mark required flag", err)
			return
		}
	}

	repoCmd.AddCommand(repoCreate)
}
