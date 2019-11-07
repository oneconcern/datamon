/*
 * Copyright © 2019 One Concern
 *
 */

package cmd

import "github.com/spf13/cobra"

var ContextCmd = &cobra.Command{
	Use:        "context",
	Aliases:    nil,
	SuggestFor: nil,
	Short:      "Commands to manage contexts.",
	Long: "Commands to manage contexts. " +
		"A context is an instance of Datamon with set of repos, runs, labels etc.",
	PreRun: func(cmd *cobra.Command, args []string) {
		config.populateRemoteConfig(&datamonFlags)
	},
}

func init() {
	rootCmd.AddCommand(ContextCmd)
}
