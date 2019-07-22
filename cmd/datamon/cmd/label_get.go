package cmd

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/oneconcern/datamon/pkg/core"

	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

var GetLabelCommand = &cobra.Command{
	Use:   "get",
	Short: "Get bundle info by label",
	Long: `Performs a direct lookup of labels by name.
Prints corresponding bundle information if the label exists,
exits with ENOENT status otherwise.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			logFatalln(err)
		}
		bundle := core.New(core.NewBDescriptor(),
			core.Repo(params.repo.RepoName),
			core.MetaStore(remoteStores.meta),
		)
		label := core.NewLabel(core.NewLabelDescriptor(),
			core.LabelName(params.label.Name),
		)
		err = label.DownloadDescriptor(ctx, bundle, true)
		if err == core.ErrNotFound {
			fmt.Fprintf(os.Stderr, "didn't find label '%v'\n", params.label.Name)
			osExit(int(unix.ENOENT))
			return
		}
		if err != nil {
			logFatalf("error downloading label information: %v\n", err)
		}

		var buf bytes.Buffer
		err = labelDescriptorTemplate.Execute(&buf, label.Descriptor)
		if err != nil {
			log.Println("executing template:", err)
		}
		log.Println(buf.String())

	},
}

func init() {
	requiredFlags := []string{addRepoNameOptionFlag(GetLabelCommand)}

	requiredFlags = append(requiredFlags, addLabelNameFlag(GetLabelCommand))

	for _, flag := range requiredFlags {
		err := GetLabelCommand.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	labelCmd.AddCommand(GetLabelCommand)
}
