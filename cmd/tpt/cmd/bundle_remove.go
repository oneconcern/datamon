// Copyright © 2018 One Concern

package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

// bundleRemoveCmd represents the remove command
var bundleRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a file from a bundle",
	Long: `Remove a file from a bundle.

When this file was created in a previous commit this will remove the file
from the repository when the repostitory gets pushed.

When this file was newly added, it will be removed from the staging area.
`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo()
		if err != nil {
			log.Fatalln(err)
		}

		if err := repo.Stage().Remove(args[0]); err != nil {
			log.Fatalln(err)
		}

		log.Println("removed", args[0])
	},
}

func init() {
	bundleCmd.AddCommand(bundleRemoveCmd)
	addRepoFlag(bundleRemoveCmd)

	for i := 1; i < 100; i++ {
		bundleRemoveCmd.MarkZshCompPositionalArgumentFile(i, "*")
	}

}
