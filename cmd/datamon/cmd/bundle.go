// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"fmt"
	"text/template"

	context2 "github.com/oneconcern/datamon/pkg/context"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/spf13/cobra"
)

// bundleCmd represents the bundle related commands
var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Commands to manage bundles for a repo",
	Long: `Commands to manage bundles for a repo.

A bundle is a point in time read-only view of a repo,
analogous to a git commit.

A bundle is composed of individual files that are tracked and changed
together.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		config.populateRemoteConfig(&datamonFlags)
	},
}

var bundleDescriptorTemplate *template.Template

func init() {
	rootCmd.AddCommand(bundleCmd)

	bundleDescriptorTemplate = func() *template.Template {
		const listLineTemplateString = `{{.ID}} , {{.Timestamp}} , {{.Message}}`
		return template.Must(template.New("list line").Parse(listLineTemplateString))
	}()
}

func setLatestOrLabelledBundle(ctx context.Context, remote context2.Stores) error {
	switch {
	case datamonFlags.bundle.ID != "" && datamonFlags.label.Name != "":
		return fmt.Errorf("--%s and --%s datamonFlags are mutually exclusive",
			addBundleFlag(nil),
			addLabelNameFlag(nil))
	case datamonFlags.bundle.ID == "" && datamonFlags.label.Name == "":
		key, err := core.GetLatestBundle(datamonFlags.repo.RepoName, remote)
		if err != nil {
			return err
		}
		datamonFlags.bundle.ID = key
	case datamonFlags.bundle.ID == "" && datamonFlags.label.Name != "":
		label := core.NewLabel(nil,
			core.LabelName(datamonFlags.label.Name),
		)
		bundle := core.NewBundle(core.NewBDescriptor(),
			core.Repo(datamonFlags.repo.RepoName),
			core.ContextStores(remote),
		)
		if err := label.DownloadDescriptor(ctx, bundle, true); err != nil {
			return err
		}
		datamonFlags.bundle.ID = label.Descriptor.BundleID
	}
	log.Printf("Using bundle: %s", datamonFlags.bundle.ID)
	return nil
}

func getConcurrencyFactor(batchSize int) int {
	// concurrency factor calculation for download, upload & mount
	concurrency := datamonFlags.bundle.ConcurrencyFactor / batchSize
	if concurrency == 0 {
		return 1
	}
	return concurrency
}
