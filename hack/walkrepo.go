// +build ignore

package main

import (
	"log"
	"os"
	"path/filepath"
)

func main() {
	log.SetFlags(0)
	tgtDir := ".trumpet/hello-there/bundles/objects"
	srcDir := ".trumpet/hello-there/stage/objects"
	filepath.Walk(srcDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}

		rp, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		tgtPth := filepath.Join(tgtDir, rp)

		os.MkdirAll(filepath.Dir(tgtPth), 0700)
		os.Rename(path, tgtPth)
		log.Println(path, "->", tgtPth)
		return nil
	})
}
