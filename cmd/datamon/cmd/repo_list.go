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

func applyRepoTemplate(repo model.RepoDescriptor) error {
	var buf bytes.Buffer
	if err := repoDescriptorTemplate(datamonFlags).Execute(&buf, repo); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	log.Println(buf.String())
	return nil
}

var repoList = &cobra.Command{
	Use:   "list",
	Short: "List repos",
	Long:  `List repos that have been created`,
	Example: `% datamon repo list --context ctx2
fred , test fred , Frédéric Bidon , frederic@oneconcern.com , 2019-12-05 14:01:18.181535 +0100 CET`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		defer func(t0 time.Time) {
			cliUsage(t0, "repo list", err)
		}(time.Now())

		ctx := context.Background()
		datamonFlagsPtr := &datamonFlags
		optionInputs := newCliOptionInputs(config, datamonFlagsPtr)
		remoteStores, err := optionInputs.datamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		err = core.ListReposApply(remoteStores, applyRepoTemplate,
			core.ConcurrentList(datamonFlags.core.ConcurrencyFactor),
			core.BatchSize(datamonFlags.core.BatchSize),
			core.WithMetrics(datamonFlags.root.metrics.IsEnabled()),
		)
		if err != nil {
			wrapFatalln("download repo list", err)
			return
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	}, // https://github.com/spf13/cobra/issues/458
}

func init() {
	addCoreConcurrencyFactorFlag(repoList, 500)
	addBatchSizeFlag(repoList)
	addSkipAuthFlag(repoList)
	repoCmd.AddCommand(repoList)
}
