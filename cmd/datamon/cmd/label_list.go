package cmd

import (
	"log"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/storage/gcs"

	"github.com/spf13/cobra"
)

var LabelListCommand = &cobra.Command{
	Use:   "list",
	Short: "List labels",
	Long:  "List the labels in a repo",
	Run: func(cmd *cobra.Command, args []string) {
		metaStore, err := gcs.New(repoParams.MetadataBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}
		keys, err := core.ListLabels(repoParams.RepoName, metaStore)
		if err != nil {
			logFatalln(err)
		}
		for _, key := range keys {
			log.Println(key)
		}
	},
}

func init() {

	requiredFlags := []string{addRepoNameOptionFlag(LabelListCommand)}

	for _, flag := range requiredFlags {
		err := BundleListCommand.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	labelCmd.AddCommand(LabelListCommand)
}
