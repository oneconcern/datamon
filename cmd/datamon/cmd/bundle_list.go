package cmd

import (
	"bytes"
	"context"
	"log"
	"text/template"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/model"

	"github.com/spf13/cobra"
)

// BundleListCommand describes the CLI command for listing bundles
var BundleListCommand = &cobra.Command{
	Use:   "list",
	Short: "List bundles",
	Long:  "List the bundles in a repo, ordered by their key",
	Run: func(cmd *cobra.Command, args []string) {
		const listLineTemplateString = `{{.ID}} , {{.Timestamp}} , {{.Message}}`
		ctx := context.Background()
		listLineTemplate := template.Must(template.New("list line").Parse(listLineTemplateString))
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			logFatalln(err)
		}
		err = core.ListBundlesApply(params.repo.RepoName, remoteStores.meta, func(bundle model.BundleDescriptor) error {
			var buf bytes.Buffer
			e := listLineTemplate.Execute(&buf, bundle)
			if e != nil {
				log.Println("executing template:", e)
				return e
			}
			log.Println(buf.String())
			return nil
		}, core.ConcurrentBundleList(params.core.ConcurrencyFactor), core.BundleBatchSize(params.core.BatchSize))
		if err != nil {
			logFatalln(err)
		}
	},
}

func init() {

	requiredFlags := []string{addRepoNameOptionFlag(BundleListCommand)}

	addBucketNameFlag(BundleListCommand)
	addBlobBucket(BundleListCommand)
	addCoreConcurrencyFactorFlag(BundleListCommand)
	addBatchSizeFlag(BundleListCommand)

	for _, flag := range requiredFlags {
		err := BundleListCommand.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	bundleCmd.AddCommand(BundleListCommand)
}
