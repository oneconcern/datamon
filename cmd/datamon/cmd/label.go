package cmd

import (
	"github.com/spf13/cobra"
)

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "Commands to manage labels for a repo",
	Long: `Commands to manage labels for a repo.

A label is a key-value map from human-readable names to machine-readable
bundle ids.
`,
}

func init() {
	rootCmd.AddCommand(labelCmd)
}
