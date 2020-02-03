// Copyright © 2018 One Concern

package cmd

import (
	"text/template"

	"github.com/spf13/cobra"
)

// repoCmd represents the repo related commands
var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Commands to manage repos",
	Long: `Commands to manage repos.

A datamon repository is analogous to a git repository.

Repos are datasets with a unified lifecycle.
They are versioned and managed via bundles.
`,
	PreRun: func(cmd *cobra.Command, args []string) {
		config.populateRemoteConfig(&datamonFlags)
	},
}

var repoDescriptorTemplate func(flagsT) *template.Template

func init() {
	addTemplateFlag(repoCmd)
	rootCmd.AddCommand(repoCmd)

	repoDescriptorTemplate = func(opts flagsT) *template.Template {
		if opts.core.Template != "" {
			t, err := template.New("list line").Parse(datamonFlags.core.Template)
			if err != nil {
				wrapFatalln("invalid template", err)
			}
			return t
		}
		const listLineTemplateString = `{{.Name}} , {{.Description}} , {{with .Contributor}}{{.Name}} , {{.Email}}{{end}} , {{.Timestamp}}`
		return template.Must(template.New("list line").Parse(listLineTemplateString))
	}
}
