package cmd

import (
	"log"

	"github.com/oneconcern/kubeless/cmd/kubeless/function"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "Command to list models on kubernetes",
	Long:    "Command to list models on Kubernetes",

	Run: func(cmd *cobra.Command, args []string) {
		output, err := cmd.Flags().GetString("out")
		if err != nil {
			log.Fatal(err.Error())
		}
		ns, err := cmd.Flags().GetString("namespace")
		if err != nil {
			log.Fatal(err.Error())
		}

		err = function.ListAdapter(output, ns, args)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	listCmd.Flags().StringP("out", "o", "", "Output format. One of: json|yaml")
	listCmd.Flags().StringP("namespace", "n", "", "Specify namespace for the model")

	modelCmd.AddCommand(listCmd)
}
