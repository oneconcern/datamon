package cmd

import "github.com/spf13/cobra"

var functionCmd = &cobra.Command{
	Use:   "function",
	Short: "Commands to manage functions ",
	Long: `Commands to manage functions.

`,
}

func init() {
	rootCmd.AddCommand(functionCmd)
}


