// Copyright © 2018 One Concern

package cmd

import (
	"github.com/spf13/cobra"
)

// repoCmd represents the repo related commands
var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Commands to manage repos",
	Long: `Commands to manage repos.

Repos are datasets that are versioned and managed via bundles.
`,
}

func init() {
	rootCmd.AddCommand(repoCmd)
}
