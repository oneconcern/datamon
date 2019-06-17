package cmd

import (
	"bytes"
	"log"
	"text/template"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/storage/gcs"

	"github.com/spf13/cobra"
)

var BundleListCommand = &cobra.Command{
	Use:   "list",
	Short: "List bundles",
	Long:  "List the bundles in a repo",
	Run: func(cmd *cobra.Command, args []string) {
		const listLineTemplateString = `{{.ID}} , {{.Timestamp}} , {{.Message}}`
		listLineTemplate := template.Must(template.New("list line").Parse(listLineTemplateString))
		store, err := gcs.New(repoParams.MetadataBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}
		bundleDescriptors, err := core.ListBundles(repoParams.RepoName, store)
		if err != nil {
			logFatalln(err)
		}
		for _, bd := range bundleDescriptors {
			var buf bytes.Buffer
			err := listLineTemplate.Execute(&buf, bd)
			if err != nil {
				log.Println("executing template:", err)
			}
			log.Println(buf.String())
		}
	},
}

func init() {

	requiredFlags := []string{addRepoNameOptionFlag(BundleListCommand)}

	addBucketNameFlag(BundleListCommand)
	addBlobBucket(BundleListCommand)

	for _, flag := range requiredFlags {
		err := BundleListCommand.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	bundleCmd.AddCommand(BundleListCommand)
}
