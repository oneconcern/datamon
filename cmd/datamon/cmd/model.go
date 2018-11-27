package cmd

import "github.com/spf13/cobra"

var modelCmd = &cobra.Command{
	Use:   "model",
	Short: "Commands to manage models ",
	Long: `Commands to manage models.

`,
}

func init() {
	rootCmd.AddCommand(modelCmd)
}
