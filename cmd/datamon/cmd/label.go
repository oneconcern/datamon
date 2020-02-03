package cmd

import (
	"text/template"

	"github.com/spf13/cobra"
)

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "Commands to manage labels for a repo",
	Long: `Commands to manage labels for a repo.

A label is a name given to a bundle, analogous to a tag in git.

Labels are a mapping type from human-readable strings to commit hashes.

There's one such map per repo, so in particular, setting a label or uploading a bundle
with a label that already exists overwrites the commit hash previously associated with the
label:  There can be at most one commit hash associated with a label.  Conversely,
multiple labels can refer to the same bundle via its commit hash (bundle ID).`,
	Example: `Latest
production`,
	PreRun: func(cmd *cobra.Command, args []string) {
		config.populateRemoteConfig(&datamonFlags)
	},
}

var labelDescriptorTemplate func(flagsT) *template.Template

func init() {
	addTemplateFlag(labelCmd)
	rootCmd.AddCommand(labelCmd)

	labelDescriptorTemplate = func(opts flagsT) *template.Template {
		if opts.core.Template != "" {
			t, err := template.New("list line").Parse(datamonFlags.core.Template)
			if err != nil {
				wrapFatalln("invalid template", err)
			}
			return t
		}
		const listLineTemplateString = `{{.Name}} , {{.BundleID}} , {{.Timestamp}}`
		return template.Must(template.New("list line").Parse(listLineTemplateString))
	}
}
