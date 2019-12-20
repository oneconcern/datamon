package cmd

import (
	"text/template"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/model"
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

var labelDescriptorTemplate *template.Template

func init() {
	rootCmd.AddCommand(labelCmd)

	labelDescriptorTemplate = func() *template.Template {
		const listLineTemplateString = `{{.Name}} , {{.BundleID}} , {{.Timestamp}}`
		return template.Must(template.New("list line").Parse(listLineTemplateString))
	}()
}

// getLabels is a helper to synchronously retrieve labels and enrich the result from other commands
func getLabels(remoteStores context2.Stores) []model.LabelDescriptor {
	labels, err := core.ListLabels(datamonFlags.repo.RepoName, remoteStores, "",
		core.ConcurrentList(datamonFlags.core.ConcurrencyFactor),
		core.BatchSize(datamonFlags.core.BatchSize))
	if err != nil {
		wrapFatalln("could not download label list: %w", err)
	}
	return labels
}
