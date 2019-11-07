package cmd

import (
	"bytes"
	"context"
	"log"

	"github.com/oneconcern/datamon/pkg/core"

	"github.com/spf13/cobra"
)

var LabelListCommand = &cobra.Command{
	Use:   "list",
	Short: "List labels",
	Long:  "List the labels in a repo",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		remoteStores, err := paramsToDatamonContext(ctx, datamonFlags)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}

		labelDescriptors, err := core.ListLabels(datamonFlags.repo.RepoName, remoteStores, datamonFlags.label.Prefix)
		if err != nil {
			wrapFatalln("download label list", err)
			return
		}
		for _, ld := range labelDescriptors {
			var buf bytes.Buffer
			err := labelDescriptorTemplate.Execute(&buf, ld)
			if err != nil {
				wrapFatalln("executing template", err)
				return
			}
			log.Println(buf.String())
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		config.populateRemoteConfig(&datamonFlags)
	},
}

func init() {

	requiredFlags := []string{addRepoNameOptionFlag(LabelListCommand)}
	addLabelPrefixFlag(LabelListCommand)
	for _, flag := range requiredFlags {
		err := LabelListCommand.MarkFlagRequired(flag)
		if err != nil {
			wrapFatalln("mark required flag", err)
			return
		}
	}
	labelCmd.AddCommand(LabelListCommand)
}
