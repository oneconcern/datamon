// Copyright © 2018 One Concern

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// tagCmd represents the tag command
var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Commands related to tags of a data repository",
	Long: `Tags are meant to be fairly static, once assigned to a commit they are unlikely to change in the future.

You can look at tags as being a named point in time for a repository.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("tags called")
	},
}

func init() {
	repoCmd.AddCommand(tagCmd)
}
