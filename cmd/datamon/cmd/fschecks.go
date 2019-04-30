// Copyright © 2018 One Concern

package cmd

import (
	"os"
)

// DieIfNotAccessible exits the process if the path is not accessible.
func DieIfNotAccessible(path string) {
	_, err := os.Stat(path)
	if err != nil {
		logFatalln(err)
	}
}

func DieIfNotDirectory(path string) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		logFatalln(err)
	}
	if !fileInfo.IsDir() {
		logFatalln("'" + path + "' is not a directory")
	}
}
