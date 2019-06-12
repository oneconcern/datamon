package cmd

import (
	"bytes"
	"log"
	"text/template"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/storage/gcs"

	"github.com/spf13/cobra"
)

var LabelListCommand = &cobra.Command{
	Use:   "list",
	Short: "List labels",
	Long:  "List the labels in a repo",
	Run: func(cmd *cobra.Command, args []string) {
		const listLineTemplateString = `{{.Name}} , {{.BundleID}} , {{.Timestamp}}`
		listLineTemplate := template.Must(template.New("list line").Parse(listLineTemplateString))
		metaStore, err := gcs.New(repoParams.MetadataBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}
		labelDescriptors, err := core.ListLabels(repoParams.RepoName, metaStore)
		if err != nil {
			logFatalln(err)
		}
		for _, ld := range labelDescriptors {
			var buf bytes.Buffer
			err := listLineTemplate.Execute(&buf, ld)
			if err != nil {
				log.Println("executing template:", err)
			}
			log.Println(buf.String())
		}
	},
}

func init() {

	requiredFlags := []string{addRepoNameOptionFlag(LabelListCommand)}

	for _, flag := range requiredFlags {
		err := LabelListCommand.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	labelCmd.AddCommand(LabelListCommand)
}
