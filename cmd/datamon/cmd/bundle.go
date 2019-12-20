// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"
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
		config.populateRemoteConfig(&datamonFlags)
	},
}

func init() {
	rootCmd.AddCommand(bundleCmd)
}

func bundleDescriptorTemplate(withLabels bool) *template.Template {
	// bundle rendering comes in 2 flavors: one without the labels field, the other with the
	// comma separated list of labels set on the bundle. When the label field is wanted,
	// but empty, it takes the <no label> value.
	var listLineTemplateString string
	if !withLabels {
		listLineTemplateString = `{{.ID}} , {{.Timestamp}} , {{.Message}}`
	} else {
		// template with labels
		listLineTemplateString = `{{.ID}} ,{{.Labels}}, {{.Timestamp}} , {{.Message}}`
	}
	return template.Must(template.New("list line").Parse(listLineTemplateString))
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

// displayBundleLabels constructs a string representation of a list of labels associated to a bundle
func displayBundleLabels(bundleID string, labels []model.LabelDescriptor) string {
	bundleLabels := make([]string, 0, len(labels))
	for _, label := range labels {
		if label.BundleID == bundleID {
			bundleLabels = append(bundleLabels, label.Name)
		}
	}
	if len(bundleLabels) > 0 {
		// using ";" to keep simple split rule on main fields
		return "(" + strings.Join(bundleLabels, ";") + ")"
	}
	return "<no label>"
}
