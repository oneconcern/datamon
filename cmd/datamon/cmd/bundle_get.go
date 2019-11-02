package cmd

import (
	"bytes"
	"context"
	"log"

	"github.com/oneconcern/datamon/pkg/core"

	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

var GetBundleCommand = &cobra.Command{
	Use:   "get",
	Short: "Get bundle info by id",
	Long: `Performs a direct lookup of labels by id.
Prints corresponding bundle information if the label exists,
exits with ENOENT status otherwise.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		remoteStores, err := paramsToDatamonContext(ctx, datamonFlags)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}

		err = setLatestOrLabelledBundle(ctx, remoteStores)
		if err == core.ErrNotFound {
			wrapFatalWithCode(int(unix.ENOENT), "didn't find label %q", datamonFlags.label.Name)
			return
		}
		if err != nil {
			wrapFatalln("determine bundle id", err)
			return
		}

		bundleOpts := paramsToBundleOpts(remoteStores)
		bundleOpts = append(bundleOpts, core.BundleID(datamonFlags.bundle.ID))
		bundleOpts = append(bundleOpts, core.Repo(datamonFlags.repo.RepoName))

		bundle := core.NewBundle(core.NewBDescriptor(),
			bundleOpts...,
		)

		err = core.DownloadMetadata(ctx, bundle)
		if err == core.ErrNotFound {
			wrapFatalWithCode(int(unix.ENOENT), "didn't find bundle %q", datamonFlags.bundle.ID)
			return
		}
		if err != nil {
			wrapFatalln("error downloading bundle information", err)
			return
		}

		var buf bytes.Buffer
		err = bundleDescriptorTemplate.Execute(&buf, bundle.BundleDescriptor)
		if err != nil {
			log.Println("executing template:", err)
		}
		log.Println(buf.String())
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		config.populateRemoteConfig(&datamonFlags)
	},
}

func init() {
	requiredFlags := []string{addRepoNameOptionFlag(GetBundleCommand)}

	addBundleFlag(GetBundleCommand)
	addLabelNameFlag(GetBundleCommand)

	for _, flag := range requiredFlags {
		err := GetBundleCommand.MarkFlagRequired(flag)
		if err != nil {
			wrapFatalln("mark required flag", err)
			return
		}
	}

	bundleCmd.AddCommand(GetBundleCommand)
}
