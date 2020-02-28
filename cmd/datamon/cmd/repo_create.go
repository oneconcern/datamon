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
		var err error

		defer func(t0 time.Time) {
			cliUsage(t0, "repo create", err)
		}(time.Now())

		ctx := context.Background()
		optionInputs := newCliOptionInputs(config, &datamonFlags)
		contributor, err := optionInputs.contributor()
		if err != nil {
			wrapFatalln("populate contributor struct", err)
			return
		}
		remoteStores, err := optionInputs.datamonContext(ctx)
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
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

func init() {
	requireFlags(repoCreate,
		// Metadata bucket
		addRepoNameOptionFlag(repoCreate),
		// Description
		addRepoDescription(repoCreate),
	)

	addContributorEmail(repoCreate)
	addContributorName(repoCreate)

	repoCmd.AddCommand(repoCreate)
}
