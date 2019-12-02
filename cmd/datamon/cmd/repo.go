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

Repos are datasets that are versioned and managed via bundles.
`,
}

var repoDescriptorTemplate *template.Template

func init() {
	rootCmd.AddCommand(repoCmd)

	repoDescriptorTemplate = func() *template.Template {
		const listLineTemplateString = `{{.Name}} , {{.Description}} , {{with .Contributor}}{{.Name}} , {{.Email}}{{end}} , {{.Timestamp}}`
		return template.Must(template.New("list line").Parse(listLineTemplateString))
	}()
}
