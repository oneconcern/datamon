/*
 * Copyright © 2019 One Concern
 *
 */

package cmd

import (
	"time"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/spf13/cobra"
)

// ContextListCommand is a command to list all available contexts
var ContextListCommand = &cobra.Command{
	Use:   "list",
	Short: "List available contexts",
	Long:  "List all available contexts in a remote configuration",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		defer func(t0 time.Time) {
			cliUsage(t0, "context list", err)
		}(time.Now())

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
	log.Printf("%v", contexts)
}

func init() {
	addConfigFlag(ContextListCommand)

	ContextCmd.AddCommand(ContextListCommand)
}
