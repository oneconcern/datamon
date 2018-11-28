// Copyright © 2018 One Concern

package cmd

import (
	"fmt"
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
			// #nosec
			fmt.Fprintln(os.Stderr, "specify a shell to generate completions for bash or zsh")
			os.Exit(1)
		}
		shell := args[0]
		if shell != bash && shell != zsh {
			// #nosec
			fmt.Fprintln(os.Stderr, "the only supported shells are bash and zsh")
		}
		if shell == bash {
			err := rootCmd.GenBashCompletion(os.Stdout)
			fmt.Fprintln(os.Stderr, "failed to generate bash completion:", err)
		}

		if shell == zsh {
			err := rootCmd.GenZshCompletion(os.Stdout)
			fmt.Fprintln(os.Stderr, "failed to generate zsh completion:", err)
		}
	},
}

func init() {
	completionCmd.Hidden = true
	rootCmd.AddCommand(completionCmd)
}
