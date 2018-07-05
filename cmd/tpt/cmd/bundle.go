// Copyright © 2018 One Concern

package cmd

import (
	"github.com/spf13/cobra"
)

// bundleCmd represents the bundle command
var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Commands to manage bundles for a repo",
	Long: `Commands to manage bundles for a repo.

A bundle is a group of files that were changed together.
Every bundle is an entry in the history of a repository.
`,
}

func init() {
	rootCmd.AddCommand(bundleCmd)
}
