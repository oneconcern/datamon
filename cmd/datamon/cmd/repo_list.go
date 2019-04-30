package cmd

import (
	"log"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/spf13/cobra"
)

var repoList = &cobra.Command{
	Use:   "list",
	Short: "List repos",
	Long:  "List repos that have been created",
	Run: func(cmd *cobra.Command, args []string) {
		store, err := gcs.New(repoParams.MetadataBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}
		keys, err := core.ListRepos(store)
		if err != nil {
			logFatalln(err)
		}
		for _, key := range keys {
			log.Println(key)
		}
	},
}

func init() {
	addBucketNameFlag(repoList)
	repoCmd.AddCommand(repoList)
}
