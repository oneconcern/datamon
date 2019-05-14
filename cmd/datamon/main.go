// Copyright © 2018 One Concern

package main

import (
	"log"
	"os"
	"runtime/pprof"

	"github.com/oneconcern/datamon/cmd/datamon/cmd"
)

func main() {
	// startCpuProf()
	// defer stopCpuProf()
	cmd.Execute()
}

func startCpuProf() {
	f, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal(err)
	}
	_ = pprof.StartCPUProfile(f)
}

func stopCpuProf() {
	pprof.StopCPUProfile()
}
