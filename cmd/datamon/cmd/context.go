/*
 * Copyright © 2019 One Concern
 *
 */

package cmd

import (
	"text/template"

	"github.com/spf13/cobra"
)

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
		const listLineTemplateString = `{{.Version}} , {{.Name}} , {{.WAL}} , {{.ReadLog}} , {{.Blob}} , {{.Metadata}} , {{.VMetadata}}`
		return template.Must(template.New("list line").Parse(listLineTemplateString))
	}()
}
