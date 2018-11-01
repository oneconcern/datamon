// Copyright © 2018 One Concern

package cmd

import (
	"github.com/spf13/cobra"
)

// bundleCmd represents the bundle related commands
var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Commands to manage bundles for a repo",
	Long: `Commands to manage bundles for a repo.

A bundle is a group of files that were changed together.
Every bundle is an entry in the history of a repository at a point in time.
`,
}

var bundleOptions struct {
	Id        string
	DataPath  string
}

func init() {
	rootCmd.AddCommand(bundleCmd)
}

func addBundleFlag(cmd *cobra.Command) error {
	cmd.Flags().StringVarP(&bundleOptions.Id, "hash", "i", "", "The hash id for the bundle")
	return cmd.MarkFlagRequired("hash")
}

func addDataPathFlag(cmd *cobra.Command) error {
	cmd.Flags().StringVarP(&bundleOptions.DataPath, "destination", "d", "", "The path to the download folder")
	return cmd.MarkFlagRequired("destination")
}
