/*
 * Copyright © 2019 One Concern
 *
 */

package cmd

import (
	"bytes"
	"context"
	"log"

	context2 "github.com/oneconcern/datamon/pkg/context"
	status "github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/model"

	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

var ContextGetCommand = &cobra.Command{
	Use:   "get",
	Short: "Get a context info",
	Long:  "Get a Datamon context's info",
	Run: func(cmd *cobra.Command, args []string) {
		getContext()
	},
}

func getContext() {
	configStore, err := gcs.New(context.Background(), datamonFlags.core.Config, config.Credential)
	if err != nil {
		wrapFatalln("failed to create config store. ", err)
	}
	contextName := datamonFlags.context.Descriptor.Name
	has, err := configStore.Has(context.Background(),
		model.GetPathToContext(contextName))
	if err != nil {
		wrapFatalln("failed to test if context exists. ", err)
		return
	}
	if !has {
		wrapFatalWithCode(int(unix.ENOENT), "didn't find context %q", contextName)
		return
	}

	var rcvdContext model.Context
	datamonContext, err := context2.GetContext(context.Background(), configStore, contextName)
	if err == status.ErrNotFound {
		wrapFatalWithCode(int(unix.ENOENT), "didn't find repo %q", datamonFlags.repo.RepoName)
		return
	}
	if err != nil {
		wrapFatalln("failed to get context. ", err)
		return
	}
	rcvdContext = *datamonContext

	var buf bytes.Buffer
	err = contextTemplate.Execute(&buf, rcvdContext)
	if err != nil {
		wrapFatalln("executing template", err)
		return
	}
	log.Println(buf.String())
}

func init() {
	addConfigFlag(ContextGetCommand)
	var requiredFlags []string
	requiredFlags = append(requiredFlags, addContextFlag(ContextGetCommand))

	for _, flag := range requiredFlags {
		err := ContextGetCommand.MarkFlagRequired(flag)
		if err != nil {
			wrapFatalln("mark required flag", err)
			return
		}
	}
	ContextCmd.AddCommand(ContextGetCommand)
}
