// Copyright © 2018 One Concern


package cmd

import (
	"github.com/spf13/cobra"
	"github.com/oneconcern/trumpet/pkg/store/localfs"
	"log"
	"fmt"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the known data repositories",
	Long: `List the known data repositories`,
	Run: func(cmd *cobra.Command, args []string) {
		repo := localfs.New()
		if err := repo.Initialize(); err != nil {
			log.Fatalln(err)
		}

		names, err := repo.List()
		if err != nil {
			log.Fatalln(err)
		}

		log.Println("found", len(names), "repos")
		for _, nm := range names {
			fmt.Println(nm)
		}
	},
}

func init() {
	repoCmd.AddCommand(listCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
