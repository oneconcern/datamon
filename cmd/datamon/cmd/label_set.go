package cmd

import (
	"context"
	"fmt"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/spf13/cobra"
)

var SetLabelCommand = &cobra.Command{
	Use:   "set",
	Short: "Set labels",
	Long:  "Set the label corresponding to a bundle",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		contributor, err := paramsToContributor(params)
		if err != nil {
			wrapFatalln("populate contributor struct", err)
			return
		}
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		bundle := core.New(core.NewBDescriptor(),
			core.Repo(params.repo.RepoName),
			core.MetaStore(remoteStores.meta),
			core.BundleID(params.bundle.ID),
		)
		bundleExists, err := bundle.Exists(ctx)
		if err != nil {
			wrapFatalln("poll for bundle existence", err)
			return
		}
		if !bundleExists {
			wrapFatalln(fmt.Sprintf("bundle %v not found", bundle), nil)
			return
		}
		labelDescriptor := core.NewLabelDescriptor(
			core.LabelContributor(contributor),
		)
		label := core.NewLabel(labelDescriptor,
			core.LabelName(params.label.Name),
		)
		err = label.UploadDescriptor(ctx, bundle)
		if err != nil {
			wrapFatalln("upload label", err)
			return
		}
	},
}

func init() {
	requiredFlags := []string{addRepoNameOptionFlag(SetLabelCommand)}

	requiredFlags = append(requiredFlags, addLabelNameFlag(SetLabelCommand))
	requiredFlags = append(requiredFlags, addBundleFlag(SetLabelCommand))

	for _, flag := range requiredFlags {
		err := SetLabelCommand.MarkFlagRequired(flag)
		if err != nil {
			wrapFatalln("mark required flag", err)
			return
		}
	}

	labelCmd.AddCommand(SetLabelCommand)
}
