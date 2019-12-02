package cmd

import (
	"text/template"

	"github.com/spf13/cobra"
)

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "Commands to manage labels for a repo",
	Long: `Commands to manage labels for a repo.

A label is a key-value map from human-readable names to machine-readable
bundle ids.
`,
}

var labelDescriptorTemplate *template.Template

func init() {
	rootCmd.AddCommand(labelCmd)

	labelDescriptorTemplate = func() *template.Template {
		const listLineTemplateString = `{{.Name}} , {{.BundleID}} , {{.Timestamp}}`
		return template.Must(template.New("list line").Parse(listLineTemplateString))
	}()
}
