// Copyright © 2018 One Concern

package cmd

import (
	"os"
)

// DieIfNotAccessible exits the process if the path is not accessible.
func DieIfNotAccessible(path string) {
	_, err := os.Stat(path)
	if err != nil {
		log_Fatalln(err)
	}
}
