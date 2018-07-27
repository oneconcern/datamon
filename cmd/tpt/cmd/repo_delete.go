// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"log"

	"github.com/oneconcern/trumpet/pkg/engine"
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete a data repository",
	Long:    `Delete a data repository. This will only succeed when the repository is an orphan`,
	Aliases: []string{"del", "rm"},
	Run: func(cmd *cobra.Command, args []string) {
		tpt, err := engine.New("")
		if err != nil {
			log.Fatalln(err)
		}

		if err := tpt.DeleteRepo(context.Background(), repoOptions.Name); err != nil {
			log.Fatalln(err)
		}
		log.Printf("%s has been deleted", repoOptions.Name)
	},
}

func init() {
	repoCmd.AddCommand(deleteCmd)
	addRepoNameOption(deleteCmd)
}
