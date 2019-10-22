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
		infoLogger.Println("begin webserver")
		// todo: pass storage.Store
		s, err := web.NewServer(web.ServerParams{
			MetadataBucket: params.repo.MetadataBucket,
			Credential:     config.Credential,
		})
		if err != nil {
			wrapFatalln("server init error", err)
			return
		}
		r := web.InitRouter(s)
		infoLogger.Printf("serving on %d...", params.web.port)
		err = http.ListenAndServe(fmt.Sprintf(":%d", params.web.port), r)
		if err != nil {
			wrapFatalln("server listen error", err)
			return
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
