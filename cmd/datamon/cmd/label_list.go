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

func applyLabelTemplate(label model.LabelDescriptor) error {
	var buf bytes.Buffer
	if err := labelDescriptorTemplate.Execute(&buf, label); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	log.Println(buf.String())
	return nil
}

// LabelListCommand lists the labels in a repo
var LabelListCommand = &cobra.Command{
	Use:   "list",
	Short: "List labels",
	Long:  "List the labels in a repo",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		remoteStores, err := paramsToDatamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		err = core.ListLabelsApply(datamonFlags.repo.RepoName, remoteStores, datamonFlags.label.Prefix, applyLabelTemplate,
			core.ConcurrentList(datamonFlags.core.ConcurrencyFactor),
			core.BatchSize(datamonFlags.core.BatchSize))

		if err != nil {
			wrapFatalln("download label list", err)
			return
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		populateRemoteConfig()
	},
}

func init() {

	requiredFlags := []string{addRepoNameOptionFlag(LabelListCommand)}
	addLabelPrefixFlag(LabelListCommand)
	addCoreConcurrencyFactorFlag(LabelListCommand, 500)
	addBatchSizeFlag(LabelListCommand)
	for _, flag := range requiredFlags {
		err := LabelListCommand.MarkFlagRequired(flag)
		if err != nil {
			wrapFatalln("mark required flag", err)
			return
		}
	}
	labelCmd.AddCommand(LabelListCommand)
}
