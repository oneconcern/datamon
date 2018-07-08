// Copyright © 2018 One Concern

package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

// branchCheckoutCmd represents the checkout command
var branchCheckoutCmd = &cobra.Command{
	Use:   "checkout",
	Short: "Checkout the files for the given branch",
	Long:  `Checkout the files for the specified branch and switch to the branch`,
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo()
		if err != nil {
			log.Fatalln(err)
		}

		sn, err := repo.Checkout(name, "")
		if err != nil {
			log.Fatalln(err)
		}
		if sn.ID == "" {
			log.Println("checked out empty branch, workspace is cleared")
		}
		print(sn)
	},
}

func init() {
	branchCmd.AddCommand(branchCheckoutCmd)
	fls := branchCheckoutCmd.Flags()
	fls.StringVar(&name, "name", "", "name for the branch to checkout")
	branchCheckoutCmd.MarkFlagRequired("name")
	addFormatFlag(branchCheckoutCmd, "")
}
