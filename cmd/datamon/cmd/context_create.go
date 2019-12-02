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

var ContextCreateCommand = &cobra.Command{
	Use:   "create",
	Short: "Create a context",
	Long:  "Create a context for Datamon",
	Run: func(cmd *cobra.Command, args []string) {
		createContext()
	},
}

func createContext() {
	configStore, err := gcs.New(context2.Background(), datamonFlags.core.Config, config.Credential)
	if err != nil {
		wrapFatalln("failed to create config store. ", err)
	}
	if datamonFlags.context.Descriptor.Name == "" {
		datamonFlags.context.Descriptor.Name = datamonFlags.context.Name
	}
	err = context.CreateContext(context2.Background(), configStore, datamonFlags.context.Descriptor)
	if err != nil {
		wrapFatalln("failed to create context: "+datamonFlags.context.Descriptor.Name, err)
	}
}

func init() {
	addConfigFlag(ContextCreateCommand)
	var requiredFlags []string
	requiredFlags = append(requiredFlags, addMetadataBucket(ContextCreateCommand))
	requiredFlags = append(requiredFlags, addVMetadataBucket(ContextCreateCommand))
	requiredFlags = append(requiredFlags, addBlobBucket(ContextCreateCommand))
	requiredFlags = append(requiredFlags, addWALBucket(ContextCreateCommand))
	requiredFlags = append(requiredFlags, addReadLogBucket(ContextCreateCommand))
	requiredFlags = append(requiredFlags, addContextFlag(ContextCreateCommand))

	for _, flag := range requiredFlags {
		err := ContextCreateCommand.MarkFlagRequired(flag)
		if err != nil {
			wrapFatalln("mark required flag", err)
			return
		}
	}
	ContextCmd.AddCommand(ContextCreateCommand)
}
