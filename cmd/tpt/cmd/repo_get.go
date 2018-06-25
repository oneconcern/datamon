// Copyright © 2018 One Concern

package cmd

import (
	"fmt"
	"log"

	"github.com/oneconcern/trumpet/pkg/store"
	"github.com/oneconcern/trumpet/pkg/store/localfs"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

type DataRepo struct {
	store.Repo    `json:",inline" yaml:",inline"`
	CurrentBranch string
}

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "get the details for a repository",
	Long:  `get the details for a repository as json`,
	Run: func(cmd *cobra.Command, args []string) {
		repodb := localfs.New()
		if err := repodb.Initialize(); err != nil {
			log.Fatalln(err)
		}

		if repo, err := repodb.Get(repoOptions.Name); err != nil {
			log.Fatalln(err)
		} else {
			d := DataRepo{
				Repo:          *repo,
				CurrentBranch: "master",
			}
			b, err := yaml.Marshal(d)
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Println(string(b))
		}

	},
}

func init() {
	repoCmd.AddCommand(getCmd)
	addRepoNameOption(getCmd)
}
