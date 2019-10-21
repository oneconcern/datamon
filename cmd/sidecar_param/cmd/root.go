package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sidecar_param",
	Short: "Parameter munging for datamon sidecars",
	Long: `Conversion and extraction of various serialization formats.

Read and write to stdio.
`,
}

func terminate(err error) {
	fmt.Println(err)
	os.Exit(1)
}

func Execute() {
	var err error
	if err = rootCmd.Execute(); err != nil {
		terminate(err)
	}
}
