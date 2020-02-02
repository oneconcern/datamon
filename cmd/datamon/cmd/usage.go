package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func filePrepender(_ string) string {
	return fmt.Sprintf("**Version: %s**\n\n", NewVersionInfo().Version)
}

// docCmd is a doc generation command powered by cobra
var docCmd = &cobra.Command{
	Use:   "usage",
	Short: "Generates documentation",
	Long:  `Command to generate usage documentation.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := doc.GenMarkdownTreeCustom(rootCmd, datamonFlags.doc.docTarget,
			filePrepender,
			func(s string) string { return s },
		)
		if err != nil {
			wrapFatalln("failed to generate doc", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(docCmd)
	addTargetFlag(docCmd)
}
