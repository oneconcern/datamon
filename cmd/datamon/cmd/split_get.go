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

// GetSplitCmd retrieves the metadata for a single split
var GetSplitCmd = &cobra.Command{
	Use:   "get",
	Short: "Gets split info",
	Long: `Performs a direct lookup of a split.

Prints corresponding split metadata if the split exists,
exits with ENOENT status otherwise.`,
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

		split, err := core.GetSplit(datamonFlags.repo.RepoName, datamonFlags.diamond.diamondID, datamonFlags.split.splitID, remoteStores,
			core.SplitLogger(logger),
			core.SplitWithMetrics(datamonFlags.root.metrics.IsEnabled()))
		if err != nil {
			if errors.Is(err, status.ErrNotFound) {
				wrapFatalWithCodef(int(unix.ENOENT), "didn't find split %q", datamonFlags.split.splitID)
				return
			}
			wrapFatalln("error downloading diamond information", err)
			return
		}

		var buf bytes.Buffer
		err = splitDescriptorTemplate(datamonFlags).Execute(&buf, split)
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
	requireFlags(GetSplitCmd,
		addRepoNameOptionFlag(GetSplitCmd),
		addDiamondFlag(GetSplitCmd),
		addSplitFlag(GetSplitCmd),
	)

	SplitCmd.AddCommand(GetSplitCmd)
}
