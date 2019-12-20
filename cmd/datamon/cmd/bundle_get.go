package cmd

import (
	"bytes"
	"context"
	"log"

	"github.com/oneconcern/datamon/pkg/core"
	status "github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/errors"
	"github.com/oneconcern/datamon/pkg/model"

	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

// GetBundleCommand retrieves the metadata for a bundle
var GetBundleCommand = &cobra.Command{
	Use:   "get",
	Short: "Get bundle info",
	Long: `Performs a direct lookup of a bundle.

Prints corresponding bundle metadata if the bundle exists,
exits with ENOENT status otherwise.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		remoteStores, err := paramsToDatamonContext(ctx, datamonFlags)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}

		err = setLatestOrLabelledBundle(ctx, remoteStores)
		if err != nil {
			if errors.Is(err, status.ErrNotFound) {
				wrapFatalWithCodef(int(unix.ENOENT), "didn't find label %q", datamonFlags.label.Name)
				return
			}
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
		if err != nil {
			if errors.Is(err, status.ErrNotFound) {
				wrapFatalWithCodef(int(unix.ENOENT), "didn't find bundle %q", datamonFlags.bundle.ID)
				return
			}
			wrapFatalln("error downloading bundle information", err)
			return
		}

		var labels []model.LabelDescriptor
		if datamonFlags.bundle.WithLabels {
			// optionally starts by retrieving labels on this repo
			labels = getLabels(remoteStores)
		}

		var buf bytes.Buffer
		if labels != nil {
			data := struct {
				model.BundleDescriptor
				Labels string
			}{
				BundleDescriptor: bundle.BundleDescriptor,
			}
			data.Labels = displayBundleLabels(bundle.BundleDescriptor.ID, labels)
			err = bundleDescriptorTemplate(true).Execute(&buf, data)
		} else {
			err = bundleDescriptorTemplate(false).Execute(&buf, bundle.BundleDescriptor)
		}
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
	requireFlags(GetBundleCommand,
		addRepoNameOptionFlag(GetBundleCommand),
	)

	addBundleFlag(GetBundleCommand)
	addLabelNameFlag(GetBundleCommand)
	addWithLabelFlag(GetBundleCommand)

	bundleCmd.AddCommand(GetBundleCommand)
}
