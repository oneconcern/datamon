// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"fmt"
	"text/template"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/spf13/cobra"
)

// bundleCmd represents the bundle related commands
var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Commands to manage bundles for a repo",
	Long: `Commands to manage bundles for a repo.

A bundle is a group of files that are tracked and changed together.
Every bundle is an entry in the history of a repository at a point in time.
`,
}

var bundleDescriptorTemplate *template.Template

func init() {
	rootCmd.AddCommand(bundleCmd)
	addBucketNameFlag(bundleCmd)
	addBlobBucket(bundleCmd)

	bundleDescriptorTemplate = func() *template.Template {
		const listLineTemplateString = `{{.ID}} , {{.Timestamp}} , {{.Message}}`
		return template.Must(template.New("list line").Parse(listLineTemplateString))
	}()
}

func setLatestOrLabelledBundle(ctx context.Context, store storage.Store) error {
	switch {
	case params.bundle.ID != "" && params.label.Name != "":
		return fmt.Errorf("--" + addBundleFlag(nil) + " and --" + addLabelNameFlag(nil) + " flags are mutually exclusive")
	case params.bundle.ID == "" && params.label.Name == "":
		key, err := core.GetLatestBundle(params.repo.RepoName, store)
		if err != nil {
			return err
		}
		params.bundle.ID = key
	case params.bundle.ID == "" && params.label.Name != "":
		label := core.NewLabel(nil,
			core.LabelName(params.label.Name),
		)
		bundle := core.New(core.NewBDescriptor(),
			core.Repo(params.repo.RepoName),
			core.MetaStore(store),
		)
		if err := label.DownloadDescriptor(ctx, bundle, true); err != nil {
			return err
		}
		params.bundle.ID = label.Descriptor.BundleID
	}
	fmt.Printf("Using bundle: %s\n", params.bundle.ID)
	return nil
}
