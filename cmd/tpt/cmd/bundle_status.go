// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

// bundleStatusCmd represents the status command
var bundleStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get the status of a bundle",
	Long: `Get the status of a bundle.

This command gives an overview of the uncommitted changes in a bundle.
So you see each file that's enlisted with its status (added, updated, removed).
`,
	Aliases: []string{"st"},
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo(initContext())
		if err != nil {
			log.Fatalln(err)
		}

		entries, err := repo.Stage().Status(context.Background())
		if err != nil {
			log.Fatalln(err)
		}

		for _, entry := range entries.Added {
			// TODO: do something less braindead than printing the path
			// stuff like A/M/D or +/x/- come to mind to indicate changes
			fmt.Println("  Added: ", entry.Path)
		}
		for _, entry := range entries.Deleted {
			// TODO: do something less braindead than printing the path
			// stuff like A/M/D or +/x/- come to mind to indicate changes
			fmt.Println("Removed: ", entry.Path)
		}

	},
}

func init() {
	bundleCmd.AddCommand(bundleStatusCmd)
	addRepoFlag(bundleStatusCmd)
}
