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

type WebParams struct {
	port int
}

var webParams = WebParams{}

var webSrv = &cobra.Command{
	Use:   "web",
	Short: "Webserver",
	Long:  "A webserver process to browse Datamon data",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("begin webserver")
		s, err := web.NewServer(web.ServerParams{
			MetadataBucket: repoParams.MetadataBucket,
			Credential:     config.Credential,
		})
		if err != nil {
			logFatalf("server init error %v", err)
		}
		r := web.InitRouter(s)
		fmt.Printf("serving on %d...\n", webParams.port)
		err = http.ListenAndServe(fmt.Sprintf(":%d", webParams.port), r)
		if err != nil {
			logFatalf("server listen error %v", err)
		}
	},
}

func addWebPortFlag(cmd *cobra.Command) string {
	cmd.Flags().IntVar(&webParams.port, webPort, 3003, "Port number for the web server")
	return repo
}

func init() {
	/* web params */
	addWebPortFlag(webSrv)

	/* core params */
	//	addBucketNameFlag(repoList)

	rootCmd.AddCommand(webSrv)
}
