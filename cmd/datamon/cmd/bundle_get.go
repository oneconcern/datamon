package cmd

import (
	"bytes"
	"context"
	"time"

	"github.com/oneconcern/datamon/pkg/core"
	status "github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/errors"

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
		var err error

		defer func(t0 time.Time) {
			cliUsage(t0, "bundle get", err)
		}(time.Now())

		ctx := context.Background()
		datamonFlagsPtr := &datamonFlags
		optionInputs := newCliOptionInputs(config, datamonFlagsPtr)
		remoteStores, err := optionInputs.datamonContext(ctx, ReadOnlyContext())
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

		bundleOpts, err := optionInputs.bundleOpts(ctx, ReadOnlyContext())
		if err != nil {
			wrapFatalln("failed to initialize bundle options", err)
		}
		bundleOpts = append(bundleOpts, core.BundleID(datamonFlags.bundle.ID))
		bundleOpts = append(bundleOpts, core.Repo(datamonFlags.repo.RepoName))
		bundleOpts = append(bundleOpts, core.BundleWithMetrics(datamonFlags.root.metrics.IsEnabled()))

		bundle := core.NewBundle(
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

		var buf bytes.Buffer
		err = bundleDescriptorTemplate(datamonFlags).Execute(&buf, bundle.BundleDescriptor)
		if err != nil {
			wrapFatalln("executing template", err)
		}
		log.Println(buf.String())
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

func init() {
	requireFlags(GetBundleCommand,
		addRepoNameOptionFlag(GetBundleCommand),
	)

	addBundleFlag(GetBundleCommand)
	addLabelNameFlag(GetBundleCommand)

	bundleCmd.AddCommand(GetBundleCommand)
}
