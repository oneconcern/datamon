// Copyright © 2018 One Concern

package cmd

import (
	"log"

	"github.com/oneconcern/trumpet/pkg/engine"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete a data repository",
	Long:    `Delete a data repository. This will only succeed when the repository is an orphan`,
	Aliases: []string{"del", "rm"},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := initContext()
		tpt, err := engine.New(&opentracing.NoopTracer{}, logger, "")
		if err != nil {
			log.Fatalln(err)
		}

		if err := tpt.DeleteRepo(ctx, repoOptions.Name); err != nil {
			log.Fatalln(err)
		}
		log.Printf("%s has been deleted", repoOptions.Name)
	},
}

func init() {
	repoCmd.AddCommand(deleteCmd)
	addRepoNameOption(deleteCmd)
}
