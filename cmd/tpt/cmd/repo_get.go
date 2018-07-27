// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"log"

	"github.com/oneconcern/trumpet/pkg/engine"
	"github.com/oneconcern/trumpet/pkg/store"
	"github.com/spf13/cobra"
)

type DataRepo struct {
	store.Repo    `json:",inline" yaml:",inline"`
	CurrentBranch string
}

// repoGetCmd represents the get command
var repoGetCmd = &cobra.Command{
	Use:   "get",
	Short: "get the details for a repository",
	Long:  `get the details for a repository as json`,
	Run: func(cmd *cobra.Command, args []string) {
		tpt, err := engine.New("")
		if err != nil {
			log.Fatalln(err)
		}

		repo, err := tpt.GetRepo(context.Background(), repoOptions.Name)
		if err != nil {
			log.Fatalln(err)
		}

		if err := print(repo); err != nil {
			log.Fatalln(err)
		}
	},
}

func init() {
	repoCmd.AddCommand(repoGetCmd)
	addRepoNameOption(repoGetCmd)
	addFormatFlag(repoGetCmd, "")
}
