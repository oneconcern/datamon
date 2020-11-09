package cmd

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/model"

	"github.com/spf13/cobra"
)

func applyLabelTemplate(label model.LabelDescriptor) error {
	var buf bytes.Buffer
	if err := labelDescriptorTemplate(datamonFlags).Execute(&buf, label); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	log.Println(buf.String())
	return nil
}

// LabelListCommand lists the labels in a repo
var LabelListCommand = &cobra.Command{
	Use:   "list",
	Short: "List labels",
	Long: `List the labels in a repo.

This is analogous to the "git tag --list" command.`,
	Example: `% datamon label list --repo ritesh-test-repo
init , 1INzQ5TV4vAAfU2PbRFgPfnzEwR , 2019-03-12 22:10:24.159704 -0700 PDT`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		defer func(t0 time.Time) {
			cliUsage(t0, "label list", err)
		}(time.Now())

		ctx := context.Background()

		datamonFlagsPtr := &datamonFlags
		optionInputs := newCliOptionInputs(config, datamonFlagsPtr)
		remoteStores, err := optionInputs.datamonContext(ctx, ReadOnlyContext())
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		err = core.ListLabelsApply(datamonFlags.repo.RepoName, remoteStores, applyLabelTemplate,
			core.ConcurrentList(datamonFlags.core.ConcurrencyFactor),
			core.BatchSize(datamonFlags.core.BatchSize),
			core.WithMetrics(datamonFlags.root.metrics.IsEnabled()),
			core.WithLabelPrefix(datamonFlags.label.Prefix),
			core.WithLabelVersions(datamonFlags.core.WithLabelVersions),
		)

		if err != nil {
			wrapFatalln("download label list", err)
			return
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

func init() {
	requireFlags(LabelListCommand,
		addRepoNameOptionFlag(LabelListCommand),
	)

	addLabelPrefixFlag(LabelListCommand)
	addCoreConcurrencyFactorFlag(LabelListCommand, 500)
	addBatchSizeFlag(LabelListCommand)
	addLabelVersionsFlag(LabelListCommand)

	labelCmd.AddCommand(LabelListCommand)
}
