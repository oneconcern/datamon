// Copyright © 2018 One Concern

package cmd

import (
	"log"

	"github.com/oneconcern/trumpet/pkg/store"
	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a data repository",
	Long: `Create a data repository.

The description field can use markdown formatting.`,

	Run: func(cmd *cobra.Command, args []string) {
		ctx := initContext()
		tpt, err := initEngine()
		if err != nil {
			log.Fatalln(err)
		}

		repo, err := tpt.CreateRepo(ctx, repoOptions.Name, repoOptions.Description)
		if err != nil && err != store.RepoAlreadyExists {
			log.Fatalln(err)
		}

		if err == store.RepoAlreadyExists {
			log.Printf("%s already existed, no action taken", repo.Name)
		} else {
			log.Printf("%s has been created", repo.Name)
		}
	},
}

func init() {
	repoCmd.AddCommand(createCmd)
	addRepoOptions(createCmd)
}
