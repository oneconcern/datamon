// Copyright © 2018 One Concern

package cmd

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/dlogger"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var logger *zap.Logger

var rootCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Produce metrics for datamon functionaltiy",
	Long: `While the tests in the packages used to build the datamon binary are about correctness,
this executable exists to gather performance metrics, memory and cpu usage for example.
`,
	TraverseChildren: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if params.root.cpuProfPath != "" {
			f, err := os.Create(params.root.cpuProfPath)
			if err != nil {
				log.Fatal(err)
			}
			_ = pprof.StartCPUProfile(f)
		}
		if params.root.memProfPath != "" {
			if _, err := os.Stat(params.root.memProfPath); !os.IsNotExist(err) {
				if err := os.RemoveAll(params.root.memProfPath); err != nil {
					log.Fatal(err)
				}
			}
			if err := os.Mkdir(params.root.memProfPath, 0777); err != nil {
				log.Fatal(err)
			}
			core.MemProfDir = params.root.memProfPath
		}
	},
	// upstream api note:  *PostRun functions aren't called in case of a panic() in Run
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if params.root.cpuProfPath != "" {
			pprof.StopCPUProfile()
		}
	},
}

func Execute() {
	var err error
	if err = rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	var err error
	log.SetFlags(0)
	logLevel := "info"
	logger, err = dlogger.GetLogger(logLevel)
	if err != nil {
		log.Fatalln("Failed to set log level:" + err.Error())
	}

	addCpuProfPath(rootCmd)
	addMemProfPath(rootCmd)
}
