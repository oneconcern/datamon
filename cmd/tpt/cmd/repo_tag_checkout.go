// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
)

// tagCheckoutCmd represents the checkout command
var tagCheckoutCmd = &cobra.Command{
	Use:   "checkout",
	Short: "Checkout the files for the given branch",
	Long:  `Checkout the files for the specified branch and switch to the branch`,
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo(initContext())
		if err != nil {
			log.Fatalln(err)
		}

		sn, err := repo.Checkout(context.Background(), name, "")
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
	tagCmd.AddCommand(tagCheckoutCmd)
	fls := tagCheckoutCmd.Flags()
	fls.StringVar(&name, "name", "", "name for the tag to checkout")
	tagCheckoutCmd.MarkFlagRequired("name")
	addFormatFlag(tagCheckoutCmd, "")
}
