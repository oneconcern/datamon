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

func applyRepoTemplate(repo model.RepoDescriptor) error {
	var buf bytes.Buffer
	if err := repoDescriptorTemplate.Execute(&buf, repo); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	log.Println(buf.String())
	return nil
}

var repoList = &cobra.Command{
	Use:   "list",
	Short: "List repos",
	Long:  "List repos that have been created",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		remoteStores, err := paramsToDatamonContext(ctx, datamonFlags)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		err = core.ListReposApply(remoteStores, applyRepoTemplate,
			core.ConcurrentList(datamonFlags.core.ConcurrencyFactor),
			core.BatchSize(datamonFlags.core.BatchSize))
		if err != nil {
			wrapFatalln("download repo list", err)
			return
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		config.populateRemoteConfig(&datamonFlags)
	}, // https://github.com/spf13/cobra/issues/458
}

func init() {
	addCoreConcurrencyFactorFlag(repoList, 500)
	addBatchSizeFlag(repoList)
	repoCmd.AddCommand(repoList)
}
