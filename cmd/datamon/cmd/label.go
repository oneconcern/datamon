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

type LabelOptions struct {
	Name string
}

var labelOptions = LabelOptions{}

func init() {
	rootCmd.AddCommand(labelCmd)
}

func addLabelNameFlag(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&labelOptions.Name, labelName, "", "The human-readable name of a label")
	return path
}
