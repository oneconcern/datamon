// Copyright © 2018 One Concern

package cmd

import (
	"fmt"
	"log"

	"github.com/oneconcern/trumpet"

	"github.com/spf13/cobra"
)

// repoListCmd represents the list command
var repoListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List the known data repositories",
	Long:    `List the known data repositories`,
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		tpt, err := trumpet.New("")
		if err != nil {
			log.Fatalln(err)
		}

		repos, err := tpt.ListRepo()
		if err != nil {
			log.Fatalln(err)
		}

		log.Println("found", len(repos), "repos")
		for _, repo := range repos {
			fmt.Println(repo.Name)
		}
	},
}

func init() {
	repoCmd.AddCommand(repoListCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// repoListCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// repoListCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
