// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
)

// tagCreateCmd represents the create command
var tagCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a tag based of the current branch",
	Long: `Create a tag based of the current branch.

A tag represents an immutable anchor as a named commit.
`,
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo(initContext())
		if err != nil {
			log.Fatalln(err)
		}

		if err := repo.CreateTag(context.Background(), name); err != nil {
			log.Fatalln(err)
		}

		log.Printf("tag %q created", name)
	},
}

func init() {
	tagCmd.AddCommand(tagCreateCmd)
	fls := tagCreateCmd.Flags()
	fls.StringVar(&name, "name", "", "name for the tag to create")
	tagCreateCmd.MarkFlagRequired("name")
}
