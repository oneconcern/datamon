package main

import (
	"log"
	"os"
	"runtime/pprof"

	"github.com/oneconcern/datamon/cmd/backup2blobs/cmd"
)

func main() {
	f, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	cmd.Execute()
	defer pprof.StopCPUProfile()
}
