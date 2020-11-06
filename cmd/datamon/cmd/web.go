package cmd

import (
	"context"
	"net"
	"net/http"
	"strconv"

	"github.com/oneconcern/datamon/pkg/metrics"
	"github.com/oneconcern/datamon/pkg/web"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var webSrv = &cobra.Command{
	Use:   "web",
	Short: "Webserver",
	Long:  "A webserver process to browse datamon data",
	Run: func(cmd *cobra.Command, args []string) {
		if datamonFlags.root.metrics.IsEnabled() {
			// do not record timings or failures for long running or daemonized commands, do not wait for completion to report
			datamonFlags.root.metrics.m.Usage.Inc("web")
			metrics.Flush()
		}

		infoLogger.Println("begin webserver")
		datamonFlagsPtr := &datamonFlags
		optionInputs := newCliOptionInputs(config, datamonFlagsPtr)
		stores, err := optionInputs.datamonContext(context.Background())
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

		listener, err := net.Listen("tcp4", net.JoinHostPort("", strconv.Itoa(datamonFlags.web.port)))
		if err != nil {
			wrapFatalln("listener init error", err)
			return
		}

		r := web.InitRouter(s)

		latch := make(chan struct{})
		errServe := make(chan error)
		go func() {
			webServer := new(http.Server)
			webServer.SetKeepAlivesEnabled(true)
			webServer.Handler = r
			latch <- struct{}{}
			errServe <- webServer.Serve(listener)
		}()

		<-latch
		infoLogger.Printf("serving datamon UI at %s...", listener.Addr().String())

		if !datamonFlags.web.noBrowser {
			err = browser.OpenURL("http://" + listener.Addr().String())
			if err != nil {
				wrapFatalln("cannot launch browser", err)
				return
			}
		}

		err = <-errServe
		if err != nil {
			wrapFatalln("server error", err)
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

func init() {
	addWebPortFlag(webSrv)
	addWebNoBrowserFlag(webSrv)
	addSkipAuthFlag(webSrv)

	rootCmd.AddCommand(webSrv)
}
