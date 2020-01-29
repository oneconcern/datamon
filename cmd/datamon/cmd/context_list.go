/*
 * Copyright © 2019 One Concern
 *
 */

package cmd

import (
	"github.com/oneconcern/datamon/pkg/core"
	"github.com/spf13/cobra"
)

// ContextListCommand is a command to list all available contexts
var ContextListCommand = &cobra.Command{
	Use:   "list",
	Short: "List available contexts",
	Long:  "List all available contexts in a remote configuration",
	Run: func(cmd *cobra.Command, args []string) {
		listContexts()
	},
}

func listContexts() {
	configStore := mustGetConfigStore()
	contexts, err := core.ListContexts(configStore)
	if err != nil {
		wrapFatalln("list contexts error", err)
		return
	}
	logStdOut("%v", contexts)
}

func init() {
	addConfigFlag(ContextListCommand)

	ContextCmd.AddCommand(ContextListCommand)
}
