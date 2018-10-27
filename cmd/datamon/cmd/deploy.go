package cmd

import "github.com/spf13/cobra"

// bundleCmd represents the bundle command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Commands to deploy functions ",
	Long: `Commands to deploy functions.

`,
}

func init() {
	rootCmd.AddCommand(deployCmd)
}

