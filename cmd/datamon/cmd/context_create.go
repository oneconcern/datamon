/*
 * Copyright © 2019 One Concern
 *
 */

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var ContextCreateCommand = &cobra.Command{
	Use:   "create",
	Short: "Create a context",
	Long:  "Create a context for Datamon",
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println("top ContextCreateCommand Run")
		fmt.Println(createContext == nil)

		createContext()
	},
	PreRun: func(cmd *cobra.Command, args []string) {

		fmt.Println("top ContextCreateCommand PreRun")

		populateRemoteConfig()
	},
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
