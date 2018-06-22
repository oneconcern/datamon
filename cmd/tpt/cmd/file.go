// Copyright © 2018 One Concern

package cmd

import (
	"github.com/spf13/cobra"
)

// fileCmd represents the file command
var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "file related commands",
	Long:  `The file namespace contains the commands that have to do with files`,
}

func init() {
	rootCmd.AddCommand(fileCmd)
}
