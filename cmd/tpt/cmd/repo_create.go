// Copyright © 2018 One Concern

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/oneconcern/trumpet/pkg/store/localfs"
	"log"
	"github.com/oneconcern/trumpet/pkg/store"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a data repository",
	Long: `Create a data repository.

The description field can use markdown formatting.`,

	Run: func(cmd *cobra.Command, args []string) {
		repodb := localfs.New()
		if err := repodb.Initialize(); err != nil {
			log.Fatalln(err)
		}

		var repo store.Repo
		repo.Name = repoOptions.Name
		repo.Description = repoOptions.Description

		if err := repodb.Create(&repo); err != nil && err != localfs.RepoAlreadyExists {
			log.Fatalln(err)
		}
		log.Printf("%s has been created", repoOptions.Name)
	},
}

func init() {
	repoCmd.AddCommand(createCmd)
	addRepoOptions(createCmd)
}
