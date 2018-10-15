// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
)

var (
	name     string
	topLevel bool
	checkout bool
)

// branchCreateCmd represents the create command
var branchCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a branch based of the current branch",
	Long: `Create a branch based of the current branch.

A branch represents an alternative timeline for data to evolve.
`,
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo(initContext())
		if err != nil {
			log.Fatalln(err)
		}

		if err := repo.CreateBranch(context.Background(), name, topLevel); err != nil {
			log.Fatalln(err)
		}

		log.Printf("branch %q created", name)
		if checkout {
			sn, err := repo.Checkout(context.Background(), name, "")
			if err != nil {
				log.Fatalln(err)
			}
			print(sn)
		}
	},
}

func init() {
	branchCmd.AddCommand(branchCreateCmd)

	addFormatFlag(branchCreateCmd, "")

	fls := branchCreateCmd.Flags()
	fls.StringVar(&name, "name", "", "name for the branch to create")
	branchCreateCmd.MarkFlagRequired("name")
	fls.BoolVar(&topLevel, "top-level", false, "when present the branch will start a completely new line of history")
	fls.BoolVar(&checkout, "checkout", false, "when present the current branch will be switched to the new one")
}
