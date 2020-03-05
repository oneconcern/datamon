package cmd

import (
	"bytes"
	"context"
	"fmt"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/model"

	"github.com/spf13/cobra"
)

// ListSplitCmd lists all splits in a diamond
var ListSplitCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists splits in a diamond and in a repo",
	Long:  `Lists splits in a diamond and in a repo, ordered by their start time`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		datamonFlagsPtr := &datamonFlags
		optionInputs := newCliOptionInputs(config, datamonFlagsPtr)
		remoteStores, err := optionInputs.datamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}

		err = core.ListSplitsApply(datamonFlags.repo.RepoName, datamonFlags.diamond.diamondID, remoteStores, applySplitTemplate,
			core.ConcurrentList(datamonFlags.core.ConcurrencyFactor),
			core.BatchSize(datamonFlags.core.BatchSize))
		if err != nil {
			wrapFatalln("concurrent list splits", err)
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
	requireFlags(ListSplitCmd,
		addRepoNameOptionFlag(ListSplitCmd),
		addDiamondFlag(ListSplitCmd),
	)
	addCoreConcurrencyFactorFlag(ListSplitCmd, 500)
	addBatchSizeFlag(ListSplitCmd)

	SplitCmd.AddCommand(ListSplitCmd)
}

func applySplitTemplate(split model.SplitDescriptor) error {
	var buf bytes.Buffer
	if err := splitDescriptorTemplate(datamonFlags).Execute(&buf, split); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	log.Println(buf.String())
	return nil
}
