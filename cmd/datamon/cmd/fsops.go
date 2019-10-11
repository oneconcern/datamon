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
		logFatalln(err)
	}
}

func createPath(path string) {
	// todo: determine proper permission bits.  previously 0700.
	err := os.MkdirAll(path, 0777)
	if err != nil {
		fmt.Println(err)
	}
}

func sanitizePath(path string) (string, error) {
	return filepath.Abs(filepath.Clean(path))
}

func DieIfNotDirectory(path string) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		logFatalln(err)
	}
	if !fileInfo.IsDir() {
		logFatalln(fmt.Errorf("%q is not a directory", path))
	}
}
