package cmd

import (
	"bytes"
	"context"
	"fmt"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/model"

	"github.com/spf13/cobra"
)

// ListDiamondCmd lists all diamonds
var ListDiamondCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists diamonds in a repo",
	Long:  `Lists diamonds in a repo, ordered by their start time`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		datamonFlagsPtr := &datamonFlags
		optionInputs := newCliOptionInputs(config, datamonFlagsPtr)
		remoteStores, err := optionInputs.datamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}

		err = core.ListDiamondsApply(datamonFlags.repo.RepoName, remoteStores, applyDiamondTemplate,
			core.ConcurrentList(datamonFlags.core.ConcurrencyFactor),
			core.BatchSize(datamonFlags.core.BatchSize))
		if err != nil {
			wrapFatalln("concurrent list diamonds", err)
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
	requireFlags(ListDiamondCmd,
		addRepoNameOptionFlag(ListDiamondCmd),
	)
	addCoreConcurrencyFactorFlag(ListDiamondCmd, 500)
	addBatchSizeFlag(ListDiamondCmd)

	DiamondCmd.AddCommand(ListDiamondCmd)
}

func applyDiamondTemplate(diamond model.DiamondDescriptor) error {
	var buf bytes.Buffer
	if err := diamondDescriptorTemplate(datamonFlags).Execute(&buf, diamond); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	log.Println(buf.String())
	return nil
}
