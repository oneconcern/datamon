// Copyright © 2018 One Concern

package cmd

import (
	"fmt"
	"io"
	"log"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// branchListCmd represents the list command
var branchListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the known branches for this repo",
	Long:  `List the known branches for this repository.`,
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo()
		if err != nil {
			log.Fatalln(err)
		}

		ls, err := repo.ListBranches()
		if err != nil {
			log.Fatalln(err)
		}

		print(branchListResult{Names: ls, Active: repo.CurrentBranch})
	},
}

func init() {
	branchCmd.AddCommand(branchListCmd)
	addRepoNameOption(branchListCmd)
	addFormatFlag(branchListCmd, "list", map[string]Formatter{
		"list": branchListFormatter(),
	})
}

type branchListResult struct {
	Names  []string `json:"names" yaml:"names"`
	Active string   `json:"active" yaml:"active"`
}

func branchListFormatter() FormatterFunc {
	return func(w io.Writer, data interface{}) error {
		val := data.(branchListResult)
		for _, v := range val.Names {
			if v != val.Active {
				fmt.Fprintln(w, " ", v)
			} else {
				fmt.Fprintln(w, color.YellowString("*"), v)
			}
		}
		return nil
	}
}
