/*
 * Copyright © 2019 One Concern
 *
 */

package cmd

import (
	"bytes"
	"context"
	"time"

	context2 "github.com/oneconcern/datamon/pkg/context"
	status "github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/errors"
	"github.com/oneconcern/datamon/pkg/model"

	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

// ContextGetCommand is a command to retrieve metadata about a context
var ContextGetCommand = &cobra.Command{
	Use:   "get",
	Short: "Get a context info",
	Long:  "Get a Datamon context's info",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		defer func(t0 time.Time) {
			cliUsage(t0, "context get", err)
		}(time.Now())

		getContext()
	},
}

func getContext() {
	configStore := mustGetConfigStore()
	contextName := datamonFlags.context.Descriptor.Name
	has, err := configStore.Has(context.Background(),
		model.GetPathToContext(contextName))
	if err != nil {
		wrapFatalln("context does not exist", err)
		return
	}
	if !has {
		wrapFatalWithCodef(int(unix.ENOENT), "didn't find context %q", contextName)
		return
	}

	var rcvdContext model.Context
	datamonContext, err := context2.GetContext(context.Background(), configStore, contextName)
	if errors.Is(err, status.ErrNotFound) {
		wrapFatalWithCodef(int(unix.ENOENT), "didn't find repo %q", datamonFlags.repo.RepoName)
		return
	}
	if err != nil {
		wrapFatalln("failed to get context. ", err)
		return
	}
	rcvdContext = *datamonContext

	var buf bytes.Buffer
	err = contextTemplate(datamonFlags).Execute(&buf, rcvdContext)
	if err != nil {
		wrapFatalln("executing template", err)
		return
	}
	log.Println(buf.String())
}

func init() {
	requireFlags(ContextGetCommand,
		addContextFlag(ContextGetCommand),
	)

	ContextCmd.AddCommand(ContextGetCommand)
}
