// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
)

// bundleCheckoutCmd represents the checkout command
var bundleCheckoutCmd = &cobra.Command{
	Use:   "checkout",
	Short: "Checkout the files that belong to a certain bundleman",
	Long:  `Updates files in the working tree to match the version in the index`,
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo(initContext())
		if err != nil {
			log.Fatalln(err)
		}
		snapshot, err := repo.Checkout(context.Background(), repo.CurrentBranch, "")
		if err != nil {
			log.Fatalln(err)
		}
		print(snapshot)
	},
}

func init() {
	bundleCmd.AddCommand(bundleCheckoutCmd)
	addRepoFlag(bundleCheckoutCmd)
	addFormatFlag(bundleCheckoutCmd, "")
}
