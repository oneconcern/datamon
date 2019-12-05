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
	Long: `Creates a new datamon repository.

Repo names must not contain special characters.
Allowed characters Unicode characters, digits and hyphen.

This is analogous to the "git init ..." command.`,
	Example: `% datamon repo create  --description "Ritesh's repo for testing" --repo ritesh-datamon-test-repo`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		contributor, err := paramsToContributor(datamonFlags)
		if err != nil {
			wrapFatalln("populate contributor struct", err)
			return
		}
		remoteStores, err := paramsToDatamonContext(ctx, datamonFlags)
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
		config.populateRemoteConfig(&datamonFlags)
	},
}

func init() {

	// Metadata bucket
	requiredFlags := []string{addRepoNameOptionFlag(repoCreate)}
	// Description
	requiredFlags = append(requiredFlags, addRepoDescription(repoCreate))

	addContributorEmail(repoCreate)
	addContributorName(repoCreate)

	addContextFlag(repoCreate)

	for _, flag := range requiredFlags {
		err := repoCreate.MarkFlagRequired(flag)
		if err != nil {
			wrapFatalln("mark required flag", err)
			return
		}
	}

	repoCmd.AddCommand(repoCreate)
}
