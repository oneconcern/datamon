package cmd

import (
	"bytes"
	"context"

	"github.com/oneconcern/datamon/pkg/core"

	"github.com/spf13/cobra"
)

// InitDiamondCmd starts a new diamond
var InitDiamondCmd = &cobra.Command{
	Use:   "initialize",
	Short: "Starts a new diamond",
	Long: `A new diamond is started and its unique ID returned. Use the diamond ID to start splits within that diamond.

Example:
datamon diamond initialize --repo my-repo
304102BC687E087CC3A811F21D113CCF
`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		datamonFlagsPtr := &datamonFlags
		optionInputs := newCliOptionInputs(config, datamonFlagsPtr)
		remoteStores, err := optionInputs.datamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}

		diamond, err := core.CreateDiamond(datamonFlags.repo.RepoName, remoteStores)
		if err != nil {
			wrapFatalln("error creating diamond", err)
			return
		}

		var buf bytes.Buffer
		err = useDiamondTemplate(datamonFlags).Execute(&buf, diamond)
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
	requireFlags(InitDiamondCmd,
		addRepoNameOptionFlag(InitDiamondCmd),
	)

	DiamondCmd.AddCommand(InitDiamondCmd)
}
