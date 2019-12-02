package cmd

import (
	"bytes"
	"context"
	"log"

	"github.com/oneconcern/datamon/pkg/core"
	status "github.com/oneconcern/datamon/pkg/core/status"

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
		remoteStores, err := paramsToDatamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		bundle := core.NewBundle(core.NewBDescriptor(),
			core.Repo(datamonFlags.repo.RepoName),
			core.ContextStores(remoteStores),
		)
		label := core.NewLabel(core.NewLabelDescriptor(),
			core.LabelName(datamonFlags.label.Name),
		)
		err = label.DownloadDescriptor(ctx, bundle, true)
		if err == status.ErrNotFound {
			wrapFatalWithCode(int(unix.ENOENT), "didn't find label %q", datamonFlags.label.Name)
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
	PreRun: func(cmd *cobra.Command, args []string) {
		populateRemoteConfig()
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
