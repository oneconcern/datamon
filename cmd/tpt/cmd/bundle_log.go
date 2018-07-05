// Copyright © 2018 One Concern

package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// logCmd represents the log command
var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Get commit history",
	Long:  `Displays a list of commits with their messages`,
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo()
		if err != nil {
			log.Fatalln(err)
		}

		commits, err := repo.ListCommits()
		if err != nil {
			log.Fatalln(err)
		}

		for _, c := range commits {
			fmt.Print("     ID:  ")
			color.Set(color.FgMagenta)
			fmt.Println(c.ID)
			color.Unset()
			fmt.Print("Authors: ")
			for i, v := range c.Committers {
				if i > 0 {
					fmt.Print(", ")
				}
				color.Set(color.FgYellow)
				fmt.Print(v.String())
				color.Unset()
			}
			fmt.Println()
			fmt.Print("   Date: ")
			color.Set(color.FgYellow)
			fmt.Println(c.Timestamp.Format(time.RFC3339))
			color.Unset()
			fmt.Println()
			fmt.Println(c.Message)
		}
	},
}

func init() {
	bundleCmd.AddCommand(logCmd)
	addRepoFlag(logCmd)
}
