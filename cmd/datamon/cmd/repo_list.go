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
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		err = core.ListReposApply(remoteStores.meta, applyRepoTemplate,
			core.ConcurrentList(params.core.ConcurrencyFactor),
			core.BatchSize(params.core.BatchSize))
		if err != nil {
			wrapFatalln("download repo list", err)
			return
		}
	},
}

func init() {
	addBucketNameFlag(repoList)
	repoCmd.AddCommand(repoList)
}
