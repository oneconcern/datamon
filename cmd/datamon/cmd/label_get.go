package cmd

import (
	"bytes"
	"context"
	"log"

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
			wrapFatalln("create remote stores", err)
			return
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
			wrapFatalWithCode(int(unix.ENOENT), "didn't find label %q", params.label.Name)
			return
		}
		if err != nil {
			wrapFatalln("error downloading label information", err)
			return
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
			wrapFatalln("mark required flag", err)
			return
		}
	}

	labelCmd.AddCommand(GetLabelCommand)
}
