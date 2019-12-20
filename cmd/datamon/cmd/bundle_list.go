package cmd

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/model"

	"github.com/spf13/cobra"
)

func applyBundleTemplate(labels []model.LabelDescriptor) func(model.BundleDescriptor) error {
	return func(bundle model.BundleDescriptor) error {
		var (
			buf bytes.Buffer
			err error
		)

		if labels != nil {
			data := struct {
				model.BundleDescriptor
				Labels string
			}{
				BundleDescriptor: bundle,
			}
			data.Labels = displayBundleLabels(bundle.ID, labels)
			err = bundleDescriptorTemplate(true).Execute(&buf, data)
		} else {
			err = bundleDescriptorTemplate(false).Execute(&buf, bundle)
		}
		if err != nil {
			return fmt.Errorf("executing template: %w", err)
		}
		log.Println(buf.String())
		return nil
	}
}

// BundleListCommand describes the CLI command for listing bundles
var BundleListCommand = &cobra.Command{
	Use:   "list",
	Short: "List bundles",
	Long: `List the bundles in a repo, ordered by their bundle ID.

This is analogous to the "git log" command. The bundle ID works like a git commit hash.`,
	Example: `% datamon bundle list --repo ritesh-test-repo
1INzQ5TV4vAAfU2PbRFgPfnzEwR , 2019-03-12 22:10:24.159704 -0700 PDT , Updating test bundle`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		remoteStores, err := paramsToDatamonContext(ctx, datamonFlags)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}

		var labels []model.LabelDescriptor
		if datamonFlags.bundle.WithLabels {
			// optionally starts by retrieving labels on this repo
			labels = getLabels(remoteStores)
		}

		err = core.ListBundlesApply(datamonFlags.repo.RepoName, remoteStores,
			applyBundleTemplate(labels),
			core.ConcurrentList(datamonFlags.core.ConcurrencyFactor),
			core.BatchSize(datamonFlags.core.BatchSize))
		if err != nil {
			wrapFatalln("concurrent list bundles", err)
			return
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		config.populateRemoteConfig(&datamonFlags)
	},
}

func init() {
	requireFlags(BundleListCommand,
		addRepoNameOptionFlag(BundleListCommand),
	)

	addCoreConcurrencyFactorFlag(BundleListCommand, 500)
	addBatchSizeFlag(BundleListCommand)
	addWithLabelFlag(BundleListCommand)

	bundleCmd.AddCommand(BundleListCommand)
}
