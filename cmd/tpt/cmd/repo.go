// Copyright © 2018 One Concern


package cmd

import (
	"github.com/spf13/cobra"
)

var repoOptions struct {
	Name string
	Description string
}



// repoCmd represents the repo command
var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Data Repo management related operations",
	Long: `Data repository management related operations for trumpet.

Repositories don't carry much content until a commit is made.
`,
}

func init() {
	rootCmd.AddCommand(repoCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// repoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// repoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func addRepoOptions(cmd *cobra.Command) error {
	fls := cmd.Flags()
	if err := addRepoNameOption(cmd); err != nil {
		return err
	}
	fls.StringVar(&repoOptions.Description, "description", "", "A description of this repository")
	return nil
}

func addRepoNameOption(cmd *cobra.Command) error {
	fls := cmd.Flags()
	fls.StringVar(&repoOptions.Name, "name", "", "The name of this repository")
	return cmd.MarkFlagRequired("name")
}
