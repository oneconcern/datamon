package cmd

import (
	"context"
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
		stores, err := paramsToDatamonContext(context.Background())
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		s, err := web.NewServer(web.ServerParams{
			Stores:     stores,
			Credential: config.Credential,
		})
		if err != nil {
			wrapFatalln("server init error", err)
			return
		}
		r := web.InitRouter(s)
		infoLogger.Printf("serving on %d...", datamonFlags.web.port)
		err = http.ListenAndServe(fmt.Sprintf(":%d", datamonFlags.web.port), r)
		if err != nil {
			wrapFatalln("server listen error", err)
			return
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		populateRemoteConfig()
	},
}

func init() {
	/* web datamonFlags */
	addWebPortFlag(webSrv)

	/* core datamonFlags */
	//	addMetadataBucket(repoList)

	rootCmd.AddCommand(webSrv)
}
