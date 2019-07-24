package cmd

import (
	"bytes"
	"context"
	"log"
	"text/template"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/spf13/cobra"
)

var repoList = &cobra.Command{
	Use:   "list",
	Short: "List repos",
	Long:  "List repos that have been created",
	Run: func(cmd *cobra.Command, args []string) {
		const listLineTemplateString = `{{.Name}} , {{.Description}} , {{with .Contributor}}{{.Name}} , {{.Email}}{{end}} , {{.Timestamp}}`
		ctx := context.Background()
		listLineTemplate := template.Must(template.New("list line").Parse(listLineTemplateString))
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			logFatalln(err)
		}
		repos, err := core.ListRepos(remoteStores.meta)
		if err != nil {
			logFatalln(err)
		}
		for _, rd := range repos {
			var buf bytes.Buffer
			err := listLineTemplate.Execute(&buf, rd)
			if err != nil {
				log.Println("executing template:", err)
			}
			log.Println(buf.String())
		}
	},
}

func init() {
	addBucketNameFlag(repoList)
	repoCmd.AddCommand(repoList)
}
