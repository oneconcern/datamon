// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"log"
	"path/filepath"

	"github.com/oneconcern/datamon/pkg/engine"
	"github.com/spf13/cobra"
)

// bundleAddCmd represents the add command
var bundleAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a file to a bundle for commit",
	Long: `Add a file or group of files to a bundle for commit.

This command supports providing one or more glob patterns
`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo(initContext())
		if err != nil {
			log.Fatalln(err)
		}

		for _, arg := range args {
			// TODO: validate that the files that are being added are underneath the base directory for the repository.
			pths, err := filepath.Glob(arg)
			if err != nil {
				log.Fatalln(err)
			}
			for _, pth := range pths {
				addBlob, err := engine.UnstagedFilePath(pth)
				if err != nil {
					log.Fatalln(err)
				}

				hash, isNew, err := repo.Stage().Add(context.Background(), addBlob)
				if err != nil {
					log.Fatalln(err)
				}

				log.Println("added file", hash, "is new:", isNew)
			}
		}
	},
}

func init() {
	bundleCmd.AddCommand(bundleAddCmd)
	//#nosec
	addRepoFlag(bundleAddCmd)

	for i := 1; i < 100; i++ {
		//#nosec
		bundleAddCmd.MarkZshCompPositionalArgumentFile(i, "*")
	}
}
