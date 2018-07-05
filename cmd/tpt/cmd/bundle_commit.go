// Copyright © 2018 One Concern

package cmd

import (
	"fmt"
	"log"

	"github.com/oneconcern/trumpet"

	"github.com/spf13/cobra"
)

var (
	message string
	branch  string
)

// commitCmd represents the commit command
var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Create a commit with the currently staged files",
	Long: `Create a commit with the currently staged files.

This won't yet make changes in the'`,
	Aliases: []string{"ci"},
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo()
		if err != nil {
			log.Fatalln(err)
		}

		hash, empty, err := repo.CreateCommit(message, "")
		if err != nil {
			log.Fatalln(err)
		}
		if empty {
			log.Println("commit empty, skipping")
		}
		fmt.Println(hash)
	},
}

func init() {
	bundleCmd.AddCommand(commitCmd)
	addRepoFlag(commitCmd)
	fls := commitCmd.Flags()
	fls.StringVarP(&message, "message", "m", "", "the commit message")
	commitCmd.MarkFlagRequired("message")
	fls.StringVarP(&branch, "branch", "b", trumpet.DefaultBranch, "the branch to use")
}
