// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
)

// branchDeleteCmd represents the delete command
var branchDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a branch from a repository",
	Long: `Deleting a branch from a repository erases the commits for this timeline.

Eventually this means that all the unreferenced objects will be removed from the data store.
`,
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo(initContext())
		if err != nil {
			log.Fatalln(err)
		}

		if err := repo.DeleteBranch(context.Background(), name); err != nil {
			log.Fatalln(err)
		}
		log.Printf("branch %q deleted", name)
	},
}

func init() {
	branchCmd.AddCommand(branchDeleteCmd)
	fls := branchDeleteCmd.Flags()
	fls.StringVar(&name, "name", "", "name for the branch to delete")
	branchDeleteCmd.MarkFlagRequired("name")
}
