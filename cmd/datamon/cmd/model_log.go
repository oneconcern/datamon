package cmd

import (
	"log"

	"github.com/oneconcern/kubeless/cmd/kubeless/function"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <function_name> FLAG",
	Short: "get logs from a running model",
	Long:  `get logs from a running model`,

	Run: func(cmd *cobra.Command, args []string) {

		if len(args) != 1 {
			log.Fatal("Need exactly one argument - model name")
		}
		model := args[0]
		follow, err := cmd.Flags().GetBool("follow")
		if err != nil {
			log.Fatal(err)
		}
		ns, err := cmd.Flags().GetString("namespace")
		if err != nil {
			log.Fatal(err)
		}

		function.LogAdapter(model, ns, follow)
	},
}

func init() {
	logsCmd.Flags().BoolP("follow", "f", false, "Specify if the logs should be streamed.")
	logsCmd.Flags().StringP("namespace", "n", "", "Specify namespace for the model")

	modelCmd.AddCommand(logsCmd)
}
