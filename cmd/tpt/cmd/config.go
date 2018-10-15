// Copyright © 2018 One Concern

package cmd

import (
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Commands to manage the config of tpt",
	Long:  `The namespace for managing config settings of tpt`,
}

func init() {
	rootCmd.AddCommand(configCmd)
}
