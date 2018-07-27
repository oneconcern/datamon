// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gosuri/uitable"
	"github.com/oneconcern/trumpet/pkg/store"
	"github.com/spf13/cobra"
)

// bundleLogCmd represents the log command
var bundleLogCmd = &cobra.Command{
	Use:   "log",
	Short: "Get commit history",
	Long:  `Displays a list of commits with their messages`,
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo(initContext())
		if err != nil {
			log.Fatalln(err)
		}

		commits, err := repo.ListCommits(context.Background())
		if err != nil {
			log.Fatalln(err)
		}

		if err := print(commits); err != nil {
			log.Fatalln(err)
		}
	},
}

func init() {
	bundleCmd.AddCommand(bundleLogCmd)
	addRepoFlag(bundleLogCmd)
	addFormatFlag(bundleLogCmd, "long", map[string]Formatter{
		"short": shortLogFormatter(),
		"long":  longLogFormatter(),
	})
}

func shortLogFormatter() FormatterFunc {
	return func(w io.Writer, data interface{}) error {
		for _, c := range data.([]store.Bundle) {
			fmt.Fprint(w, color.MagentaString(c.ID[:7])+"..."+color.MagentaString(c.ID[len(c.ID)-7:])+" ")
			fmt.Fprintln(w, c.Message)
		}
		return nil
	}
}

func longLogFormatter() FormatterFunc {
	return func(w io.Writer, data interface{}) error {
		table := uitable.New()
		table.MaxColWidth = 80
		table.Wrap = true

		for _, c := range data.([]store.Bundle) {
			table.AddRow("ID:", color.MagentaString(c.ID[:70])+"...")
			var authors []string
			for _, v := range c.Committers {
				authors = append(authors, color.YellowString(v.String()))
			}
			table.AddRow("Authors:", strings.Join(authors, ", "))
			table.AddRow("Date:", color.YellowString(c.Timestamp.Format(time.RFC3339)))
			table.AddRow("")
			table.AddRow("", c.Message)
		}
		fmt.Fprintln(w, table.String())
		return nil
	}
}
