/*
 * Copyright © 2019 One Concern
 *
 */

package cmd

import (
	"context"
	"text/template"

	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/spf13/cobra"
)

// ContextCmd is a command to manage datamon contexts
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

var contextTemplate *template.Template

func init() {
	rootCmd.AddCommand(ContextCmd)

	contextTemplate = func() *template.Template {
		const listLineTemplateString = `Model Version: {{.Version}}, Name: {{.Name}}, WAL: {{.WAL}}, ReadLog: {{.ReadLog}}, Blob: {{.Blob}}, Metadata: {{.Metadata}}, Version Metadata: {{.VMetadata}}`
		return template.Must(template.New("list line").Parse(listLineTemplateString))
	}()
}

func mustGetConfigStore() storage.Store {
	configStore, err := gcs.New(context.Background(), datamonFlags.core.Config, config.Credential)
	if err != nil {
		wrapFatalln("failed to create config store", err)
	}
	return configStore
}
