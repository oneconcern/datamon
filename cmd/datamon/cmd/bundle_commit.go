// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/oneconcern/datamon/pkg/engine"
	"github.com/spf13/cobra"
)

var (
	message string
	branch  string
)

// bundleCommitCmd represents the commit command
var bundleCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Create a commit with the currently staged files",
	Long: `Create a commit with the currently staged files.

This won't yet make changes in the'`,
	Aliases: []string{"ci"},
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo(initContext())
		if err != nil {
			log.Fatalln(err)
		}

		bundle, err := repo.CreateCommit(context.Background(), message, "")
		if err != nil {
			log.Fatalln(err)
		}
		if bundle.IsEmpty {
			log.Println("commit empty, skipping")
		}
		fmt.Println(bundle.ID)
	},
}

func init() {
	bundleCmd.AddCommand(bundleCommitCmd)
	addRepoFlag(bundleCommitCmd)
	fls := bundleCommitCmd.Flags()

	fls.StringVarP(&message, "message", "m", "", "the commit message")
	bundleCommitCmd.MarkFlagRequired("message")
	fls.StringVarP(&branch, "branch", "b", engine.DefaultBranch, "the branch to use")
}
