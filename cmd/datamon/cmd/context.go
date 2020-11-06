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
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

var contextTemplate func(flagsT) *template.Template

func init() {
	addTemplateFlag(ContextCmd)
	addSkipAuthFlag(ContextCmd, true)
	rootCmd.AddCommand(ContextCmd)

	contextTemplate = func(opts flagsT) *template.Template {
		if opts.core.Template != "" {
			t, err := template.New("list line").Parse(datamonFlags.core.Template)
			if err != nil {
				wrapFatalln("invalid template", err)
			}
			return t
		}
		const listLineTemplateString = `Model Version: {{.Version}}, Name: {{.Name}}, WAL: {{.WAL}}, ReadLog: {{.ReadLog}}, Blob: {{.Blob}}, Metadata: {{.Metadata}}, Version Metadata: {{.VMetadata}}`
		return template.Must(template.New("list line").Parse(listLineTemplateString))
	}
}

func mustGetConfigStore() storage.Store {
	configStore, err := gcs.New(context.Background(), datamonFlags.core.Config, config.Credential)
	if err != nil {
		wrapFatalln("failed to create config store", err)
	}
	return configStore
}
