// Copyright © 2018 One Concern

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/model"

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
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

var (
	useBundleTemplate        func(flagsT) *template.Template
	bundleDescriptorTemplate func(flagsT) *template.Template
)

func init() {
	addTemplateFlag(bundleCmd)
	addSkipAuthFlag(bundleCmd, true)
	rootCmd.AddCommand(bundleCmd)

	bundleDescriptorTemplate = func(opts flagsT) *template.Template {
		if opts.core.Template != "" {
			t, err := template.New("list line").Parse(datamonFlags.core.Template)
			if err != nil {
				wrapFatalln("invalid template", err)
			}
			return t
		}
		const listLineTemplateString = `{{.ID}} , {{.Timestamp}} , {{.Message}}`
		return template.Must(template.New("list line").Parse(listLineTemplateString))
	}

	useBundleTemplate = func(_ flagsT) *template.Template {
		const useBundleTemplateString = `Using bundle: {{.ID}}`
		return template.Must(template.New("use bundle").Parse(useBundleTemplateString))
	}
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
		label := core.NewLabel(
			core.LabelWithMetrics(datamonFlags.root.metrics.IsEnabled()),
			core.LabelDescriptor(
				model.NewLabelDescriptor(
					model.LabelName(datamonFlags.label.Name),
				),
			))
		bundle := core.NewBundle(
			core.Repo(datamonFlags.repo.RepoName),
			core.ContextStores(remote),
			core.BundleWithMetrics(datamonFlags.root.metrics.IsEnabled()),
		)
		if err := label.DownloadDescriptor(ctx, bundle, true); err != nil {
			return err
		}
		datamonFlags.bundle.ID = label.Descriptor.BundleID
	}

	// when descriptor template is overridden, skip this heading: means that the user is expecting some specific fields
	if datamonFlags.core.Template == "" {
		var buf bytes.Buffer
		if err := useBundleTemplate(datamonFlags).Execute(&buf, struct{ ID string }{ID: datamonFlags.bundle.ID}); err != nil {
			wrapFatalln("executing template", err)
		}
		log.Println(buf.String())
	}
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
