package cmd

import (
	"bytes"
	"context"

	"github.com/oneconcern/datamon/pkg/core"
	status "github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/errors"

	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

// GetDiamondCmd retrieves the metadata for a diamond
var GetDiamondCmd = &cobra.Command{
	Use:   "get",
	Short: "Gets diamond info",
	Long: `Performs a direct lookup of a diamond.

Prints corresponding diamond metadata if the diamond exists,
exits with ENOENT status otherwise.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		datamonFlagsPtr := &datamonFlags
		optionInputs := newCliOptionInputs(config, datamonFlagsPtr)
		remoteStores, err := optionInputs.datamonContext(ctx, ReadOnlyContext())
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		logger, err := optionInputs.getLogger()
		if err != nil {
			wrapFatalln("get logger", err)
			return
		}

		diamond, err := core.GetDiamond(datamonFlags.repo.RepoName, datamonFlags.diamond.diamondID, remoteStores,
			core.DiamondLogger(logger),
			core.DiamondWithMetrics(datamonFlags.root.metrics.IsEnabled()))
		if err != nil {
			if errors.Is(err, status.ErrNotFound) {
				wrapFatalWithCodef(int(unix.ENOENT), "didn't find diamond %q", datamonFlags.diamond.diamondID)
				return
			}
			wrapFatalln("error downloading diamond information", err)
			return
		}

		var buf bytes.Buffer
		err = diamondDescriptorTemplate(datamonFlags).Execute(&buf, diamond)
		if err != nil {
			wrapFatalln("executing template", err)
		}
		log.Println(buf.String())
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

func init() {
	requireFlags(GetDiamondCmd,
		addRepoNameOptionFlag(GetDiamondCmd),
		addDiamondFlag(GetDiamondCmd),
	)

	DiamondCmd.AddCommand(GetDiamondCmd)
}
