// Copyright © 2018 One Concern


package cmd

import (
	"github.com/spf13/cobra"
	"github.com/oneconcern/trumpet/pkg/store/localfs"
	"log"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a data repository",
	Long: `Delete a data repository. This will only succeed when the repository is an orphan`,
	Run: func(cmd *cobra.Command, args []string) {
		repodb := localfs.New()
		if err := repodb.Initialize(); err != nil {
			log.Fatalln(err)
		}

		if err := repodb.Delete(repoOptions.Name); err != nil {
			log.Fatalln(err)
		}
		log.Printf("%s has been deleted", repoOptions.Name)
	},
}

func init() {
	repoCmd.AddCommand(deleteCmd)
	addRepoNameOption(deleteCmd)
}
