// Copyright © 2018 One Concern

package cmd

import (
	"log"
	"os"
)

func DieIfNotAccessible(path string) {
	_, err := os.Stat(path)
	if err != nil {
		log.Fatalln(err)
	}
}
