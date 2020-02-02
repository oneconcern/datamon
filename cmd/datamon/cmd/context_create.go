/*
 * Copyright © 2019 One Concern
 *
 */

package cmd

import (
	context2 "context"

	"github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/spf13/cobra"
)

// ContextCreateCommand is a command to create a datamon context
var ContextCreateCommand = &cobra.Command{
	Use:   "create",
	Short: "Create a context",
	Long:  "Create a context for Datamon",
	Run: func(cmd *cobra.Command, args []string) {
		createContext()
	},
}

func createContext() {
	configStore, err := gcs.New(context2.Background(), datamonFlags.core.Config, config.Credential,
		gcs.Logger(config.mustGetLogger(datamonFlags)),
	)
	if err != nil {
		wrapFatalln("failed to create config store", err)
	}
	err = context.CreateContext(context2.Background(), configStore, datamonFlags.context.Descriptor)
	if err != nil {
		wrapFatalln("failed to create context: "+datamonFlags.context.Descriptor.Name, err)
	}
}

func init() {
	requireFlags(ContextCreateCommand,
		addMetadataBucket(ContextCreateCommand),
		addVMetadataBucket(ContextCreateCommand),
		addBlobBucket(ContextCreateCommand),
		addWALBucket(ContextCreateCommand),
		addReadLogBucket(ContextCreateCommand),
		addContextFlag(ContextCreateCommand),
	)

	ContextCmd.AddCommand(ContextCreateCommand)
}
