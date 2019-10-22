// Copyright © 2018 One Concern

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

const bash = "bash"
const zsh = "zsh"

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion SHELL",
	Short: "generate completions for the datamon command",
	Long: `Generate completions for your shell

	For bash add the following line to your ~/.bashrc

		eval "$(datamon completion bash)"

	For zsh add generate a file:

		datamon completion zsh > /usr/local/share/zsh/site-functions/_datamon

	`,
	ValidArgs: []string{bash, zsh},
	Args:      cobra.OnlyValidArgs,

	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			wrapFatalln("specify a shell to generate completions for bash or zsh", nil)
			return
		}
		shell := args[0]
		if shell != bash && shell != zsh {
			wrapFatalln("the only supported shells are bash and zsh", nil)
			return
		}
		if shell == bash {
			if err := rootCmd.GenBashCompletion(os.Stdout); err != nil {
				wrapFatalln("failed to generate bash completion", err)
				return
			}
		}

		if shell == zsh {
			if err := rootCmd.GenZshCompletion(os.Stdout); err != nil {
				wrapFatalln("failed to generate zsh completion", err)
				return
			}
		}
	},
}

func init() {
	completionCmd.Hidden = true
	rootCmd.AddCommand(completionCmd)
}
