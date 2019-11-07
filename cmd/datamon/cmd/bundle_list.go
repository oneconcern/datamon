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

func applyBundleTemplate(bundle model.BundleDescriptor) error {
	var buf bytes.Buffer
	err := bundleDescriptorTemplate.Execute(&buf, bundle)
	if err != nil {
		// NOTE(frederic): to be discussed - PR#267 introduced a change here
		// by stopping upon errors while it was previously non-blocking
		return fmt.Errorf("executing template: %w", err)
	}
	log.Println(buf.String())
	return nil
}

// BundleListCommand describes the CLI command for listing bundles
var BundleListCommand = &cobra.Command{
	Use:   "list",
	Short: "List bundles",
	Long:  "List the bundles in a repo, ordered by their bundle ID",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		remoteStores, err := paramsToDatamonContext(ctx, datamonFlags)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		err = core.ListBundlesApply(datamonFlags.repo.RepoName, remoteStores, applyBundleTemplate,
			core.ConcurrentBundleList(datamonFlags.core.ConcurrencyFactor),
			core.BundleBatchSize(datamonFlags.core.BatchSize))
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

	requiredFlags := []string{addRepoNameOptionFlag(BundleListCommand)}

	addCoreConcurrencyFactorFlag(BundleListCommand)
	addBatchSizeFlag(BundleListCommand)

	for _, flag := range requiredFlags {
		err := BundleListCommand.MarkFlagRequired(flag)
		if err != nil {
			wrapFatalln("mark required flag", err)
			return
		}
	}

	bundleCmd.AddCommand(BundleListCommand)
}
