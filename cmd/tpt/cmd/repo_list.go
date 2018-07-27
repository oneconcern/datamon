// Copyright © 2018 One Concern

package cmd

import (
	"fmt"
	"io"
	"log"

	"github.com/fatih/color"
	"github.com/oneconcern/trumpet/pkg/engine"
	opentracing "github.com/opentracing/opentracing-go"

	"github.com/spf13/cobra"
)

// repoListCmd represents the list command
var repoListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List the known data repositories",
	Long:    `List the known data repositories`,
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := initContext()
		tpt, err := engine.New(&opentracing.NoopTracer{}, "")
		if err != nil {
			log.Fatalln(err)
		}

		repos, err := tpt.ListRepo(ctx)
		if err != nil {
			log.Fatalln(err)
		}

		print(repos)
	},
}

func init() {
	repoCmd.AddCommand(repoListCmd)
	addFormatFlag(repoListCmd, "list", map[string]Formatter{
		"list": FormatterFunc(func(w io.Writer, data interface{}) error {
			repos := data.([]engine.Repo)
			for _, repo := range repos {
				fmt.Fprintf(w, "%s\t%s\n", repo.Name, color.HiBlackString(repo.Description))
			}
			return nil
		}),
	})
}
