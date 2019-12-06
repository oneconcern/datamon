package cmd

import (
	"github.com/spf13/cobra"
)

var configParse = &cobra.Command{
	Use:   "parse",
	Short: "Parse and output config",
	Long: `see subcommands for sidecar-specific parsing
`,
}

func init() {
	requiredFlags := []string{}

	for _, flag := range requiredFlags {
		err := configParse.MarkFlagRequired(flag)
		if err != nil {
			terminate(err)
		}
	}

	rootCmd.AddCommand(configParse)

}
