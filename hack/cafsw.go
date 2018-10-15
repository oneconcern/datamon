// +build ignore

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/oneconcern/datamon/pkg/cafs"
)

func main() {
	start := time.Now()
	defer func() {
		log.Printf("took: %s", time.Now().Sub(start))
	}()

	if err := os.MkdirAll("/tmp/scratchdata", os.ModePerm); err != nil {
		log.Fatalln(err)
	}
	fs, err := cafs.New(
		cafs.Directory("/tmp/scratchdata"),
	)
	if err != nil {
		log.Fatalln(err)
	}
	const archive = "/home/ivan/Downloads/2018-04-18-raspbian-stretch.zip"
	//const archive = "/home/ivan/Downloads/people.zip"
	log.Println("opening", archive)

	f, err := os.Open(archive)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	copyStart := time.Now()
	log.Println("copying/hashing", archive, "open took", copyStart.Sub(start).String())
	w := fs.Writer()
	//buf := make([]byte, 512*1024)
	buf := make([]byte, 32*1024)
	//buf := make([]byte, os.Getpagesize())
	_, err = io.CopyBuffer(w, f, buf)
	//_, err = io.Copy(w, f)
	if err != nil {
		log.Fatalln(err)
	}

	flushStart := time.Now()
	log.Println("flushing", archive, "| copying took", flushStart.Sub(copyStart).String())
	hash, leafs, err := w.Flush()
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("flush took", time.Now().Sub(flushStart).String())

	if err = f.Close(); err != nil {
		log.Fatalln(err)
	}

	fmt.Println("  root:", hash)
	fmt.Printf("chunks: %x\n", leafs)

}
