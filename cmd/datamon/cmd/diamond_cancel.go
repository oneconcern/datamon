package cmd

import (
	"context"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/model"

	"github.com/spf13/cobra"
)

// CancelDiamondCmd cancels a diamond
var CancelDiamondCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancels a diamond",
	Long:  `Explicitly cancels a diamond: no commit operation will be accepted`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		datamonFlagsPtr := &datamonFlags
		optionInputs := newCliOptionInputs(config, datamonFlagsPtr)
		remoteStores, err := optionInputs.datamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		logger, err := optionInputs.getLogger()
		if err != nil {
			wrapFatalln("get logger", err)
			return
		}

		diamond, err := core.GetDiamond(datamonFlags.repo.RepoName, datamonFlags.diamond.diamondID, remoteStores)
		if err != nil {
			wrapFatalln("error retrieving diamond", err)
			return
		}

		d := core.NewDiamond(datamonFlags.repo.RepoName, remoteStores,
			core.DiamondDescriptor(model.NewDiamondDescriptor(
				model.DiamondClone(diamond),
			)),
			core.DiamondLogger(logger),
			core.DiamondWithMetrics(datamonFlags.root.metrics.IsEnabled()),
		)

		err = d.Cancel()
		if err != nil {
			wrapFatalln("diamond cancel", err)
		}
		infoLogger.Printf("diamond %s canceled", diamond.DiamondID)
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

func init() {
	requireFlags(CancelDiamondCmd,
		addRepoNameOptionFlag(CancelDiamondCmd),
		addDiamondFlag(CancelDiamondCmd),
	)
	addDiamondTagFlag(CancelDiamondCmd)

	DiamondCmd.AddCommand(CancelDiamondCmd)
}
