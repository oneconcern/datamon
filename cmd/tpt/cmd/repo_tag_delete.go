// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
)

// tagDeleteCmd represents the delete command
var tagDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a tag from a repository",
	Long: `Delete a tag from a repository.

Eventually this means that all the orphaned objects will be removed from the data store.
`,
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo(initContext())
		if err != nil {
			log.Fatalln(err)
		}

		if err := repo.DeleteTag(context.Background(), name); err != nil {
			log.Fatalln(err)
		}
		log.Printf("tag %q deleted", name)
	},
}

func init() {
	tagCmd.AddCommand(tagDeleteCmd)
	fls := tagDeleteCmd.Flags()
	fls.StringVar(&name, "name", "", "name for the tag to delete")
	tagDeleteCmd.MarkFlagRequired("name")
}
