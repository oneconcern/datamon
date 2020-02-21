package cmd

import (
	"context"
	"fmt"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/model"
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

		optionInputs := newCliOptionInputs(config, &datamonFlags)
		contributor, err := optionInputs.contributor()
		if err != nil {
			wrapFatalln("populate contributor struct", err)
			return
		}
		remoteStores, err := optionInputs.datamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}

		bundle := core.NewBundle(
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
			wrapFatalln(fmt.Sprintf("bundle:%s not found in repo:%s %s ", bundle.BundleID, bundle.RepoID, optionInputs.dumpContext()), nil)
			return
		}

		label := core.NewLabel(
			core.LabelDescriptor(
				model.NewLabelDescriptor(
					model.LabelContributor(contributor),
					model.LabelName(datamonFlags.label.Name),
				),
			))

		err = label.UploadDescriptor(ctx, bundle)
		if err != nil {
			wrapFatalln("upload label", err)
			return
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

func init() {
	requireFlags(SetLabelCommand,
		addRepoNameOptionFlag(SetLabelCommand),
		addLabelNameFlag(SetLabelCommand),
		addBundleFlag(SetLabelCommand),
	)

	labelCmd.AddCommand(SetLabelCommand)
}
