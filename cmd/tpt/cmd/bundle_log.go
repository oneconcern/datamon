// Copyright © 2018 One Concern

package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/oneconcern/trumpet/pkg/store"
	"github.com/spf13/cobra"
)

// bundleLogCmd represents the log command
var bundleLogCmd = &cobra.Command{
	Use:   "log",
	Short: "Get commit history",
	Long:  `Displays a list of commits with their messages`,
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo()
		if err != nil {
			log.Fatalln(err)
		}

		commits, err := repo.ListCommits()
		if err != nil {
			log.Fatalln(err)
		}

		if err := formatters[format].Format(os.Stdout, commits); err != nil {
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
		for _, c := range data.([]store.Bundle) {
			fmt.Fprint(w, "     ID: ")
			fmt.Fprintln(w, color.MagentaString(c.ID))
			fmt.Fprint(w, "Authors: ")
			for i, v := range c.Committers {
				if i > 0 {
					fmt.Fprint(w, ", ")
				}
				fmt.Fprint(w, color.YellowString(v.String()))
			}
			fmt.Fprintln(w)
			fmt.Fprint(w, "   Date: ")
			fmt.Fprintln(w, color.YellowString(c.Timestamp.Format(time.RFC3339)))
			fmt.Fprintln(w)
			fmt.Fprintln(w, c.Message)
		}
		return nil
	}
}
