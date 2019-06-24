package cmd

import (
	"fmt"
	"net/http"

	"github.com/oneconcern/datamon/pkg/web"

	"github.com/spf13/cobra"
)

const (
	webPort = "port"
)

var webSrv = &cobra.Command{
	Use:   "web",
	Short: "Webserver",
	Long:  "A webserver process to browse Datamon data",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("begin webserver")
		s, err := web.NewServer(web.ServerParams{
			MetadataBucket: params.repo.MetadataBucket,
			Credential:     config.Credential,
		})
		if err != nil {
			logFatalf("server init error %v", err)
		}
		r := web.InitRouter(s)
		fmt.Printf("serving on %d...\n", params.web.port)
		err = http.ListenAndServe(fmt.Sprintf(":%d", params.web.port), r)
		if err != nil {
			logFatalf("server listen error %v", err)
		}
	},
}

func init() {
	/* web params */
	addWebPortFlag(webSrv)

	/* core params */
	//	addBucketNameFlag(repoList)

	rootCmd.AddCommand(webSrv)
}
