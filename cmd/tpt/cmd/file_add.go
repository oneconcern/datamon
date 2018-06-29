// Copyright © 2018 One Concern

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// fileAddCmd represents the add command
var fileAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a file to upload",
	Long: `Add a file to the stage of a bundle, this does not yet upload the file.

This operation adds the file to be uploaded. At this stage we create a fingerprint for this file.
The file will be copied to a deduplicated staging area with its hash as name
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("add called")
	},
}

func init() {
	fileCmd.AddCommand(fileAddCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// fileAddCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// fileAddCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
