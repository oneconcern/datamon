package cmd

import (
	"context"
	"fmt"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/spf13/cobra"
)

// SetLabelCommand is a command to set a label on a bundle
var SetLabelCommand = &cobra.Command{
	Use:   "set",
	Short: "Set labels",
	Long: `Set the label corresponding to a bundle.

Setting a label is analogous to the git command "git tag {label}".`,
	Example: `% datamon label set --repo ritesh-test-repo --label anotherlabel --bundle 1ISwIzeAR6m3aOVltAsj1kfQaml
`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		contributor, err := paramsToContributor(datamonFlags)
		if err != nil {
			wrapFatalln("populate contributor struct", err)
			return
		}

		remoteStores, err := paramsToDatamonContext(ctx, datamonFlags)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}

		bundle := core.NewBundle(core.NewBDescriptor(),
			core.Repo(datamonFlags.repo.RepoName),
			core.ContextStores(remoteStores),
			core.BundleID(datamonFlags.bundle.ID),
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
			core.LabelName(datamonFlags.label.Name),
		)

		err = label.UploadDescriptor(ctx, bundle)
		if err != nil {
			wrapFatalln("upload label", err)
			return
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		config.populateRemoteConfig(&datamonFlags)
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
