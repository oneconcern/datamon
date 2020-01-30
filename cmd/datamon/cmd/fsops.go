// Copyright © 2018 One Concern

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

// DieIfNotAccessible exits the process if the path is not accessible.
func DieIfNotAccessible(path string) {
	_, err := os.Stat(path)
	if err != nil {
		wrapFatalln(fmt.Sprintf("couldn't stat %q", path), err)
		return
	}
}

func createPath(path string) {
	// todo: determine proper permission bits.  previously 0700.
	err := os.MkdirAll(path, 0777)
	if err != nil {
		errlog.Println(err)
	}
}

func sanitizePath(path string) (string, error) {
	return filepath.Abs(filepath.Clean(path))
}

func DieIfNotDirectory(path string) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		wrapFatalln(fmt.Sprintf("couldn't stat %q", path), err)
		return
	}
	if !fileInfo.IsDir() {
		wrapFatalln("incorrect file info", fmt.Errorf("%q is not a directory", path))
	}
}
