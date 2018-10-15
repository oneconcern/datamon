// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/spf13/cobra"
)

// tagListCmd represents the list command
var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the known tags for this repo",
	Long:  `List the known tags for this repository.`,
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo(initContext())
		if err != nil {
			log.Fatalln(err)
		}

		ls, err := repo.ListTags(context.Background())
		if err != nil {
			log.Fatalln(err)
		}

		print(ls)
	},
}

func init() {
	tagCmd.AddCommand(tagListCmd)
	addFormatFlag(tagListCmd, "list", map[string]Formatter{
		"list": FormatterFunc(func(w io.Writer, data interface{}) error {
			val := data.([]string)
			for _, tag := range val {
				fmt.Fprintf(w, "%s\n", tag)
			}
			return nil
		}),
	})
}
