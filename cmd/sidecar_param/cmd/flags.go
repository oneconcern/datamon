package cmd

import (
//	"github.com/spf13/cobra"
)

type paramsT struct {
	parse struct {
	}
}

var params = paramsT{}

/*
func addPathFlag(cmd *cobra.Command) string {
	path := "path"
	cmd.Flags().StringVar(&params.bundle.DataPath, path, "", "The path to the folder or bucket (gs://<bucket>) for the data")
	return path
}
*/
