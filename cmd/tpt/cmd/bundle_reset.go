// Copyright © 2018 One Concern

package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

// bundleResetCmd represents the reset command
var bundleResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the bundle to the last commit",
	Long: `Reset the bundle to the last commit.

This command will clear the stage for a bundle
`,
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo()
		if err != nil {
			log.Fatalln(err)
		}

		if err := repo.Stage().Clear(); err != nil {
			log.Fatalln(err)
		}

		log.Println("bundle was reset")
	},
}

func init() {
	bundleCmd.AddCommand(bundleResetCmd)
	addRepoFlag(bundleResetCmd)
}
