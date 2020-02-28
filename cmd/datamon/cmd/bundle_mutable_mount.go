// Copyright © 2018 One Concern

package cmd

import (
	"context"

	daemonizer "github.com/jacobsa/daemonize"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/fuse"
	"github.com/oneconcern/datamon/pkg/metrics"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/spf13/cobra"
)

// Mount a mutable view of a bundle
var mutableMountBundleCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a bundle incrementally with filesystem operations",
	Long: `Write directories and files to the mountpoint.  Unmount or send SIGINT to this process to save.
The destination path is a temporary staging area for write operations.`,
	Run: func(cmd *cobra.Command, args []string) {
		if datamonFlags.root.metrics.IsEnabled() {
			// do not record timings or failures for long running or daemonized commands, do not wait for completion to report
			datamonFlags.root.metrics.m.Usage.Inc("bundle new")
			metrics.Flush()
		}

		ctx := context.Background()

		// cf. comments on runDaemonized in bundle_mount.go
		if datamonFlags.bundle.Daemonize {
			runDaemonized()
			return
		}
		optionInputs := newCliOptionInputs(config, &datamonFlags)
		contributor, err := optionInputs.contributor()
		if err != nil {
			onDaemonError("populate contributor struct", err)
			return
		}
		consumableStore, err := optionInputs.srcStore(ctx, true)
		if err != nil {
			onDaemonError("create source store", err)
			return
		}

		bd := model.NewBundleDescriptor(
			model.Message(datamonFlags.bundle.Message),
			model.BundleContributor(contributor),
		)
		bundleOpts, err := optionInputs.bundleOpts(ctx)
		if err != nil {
			onDaemonError("failed to initialize bundle options", err)
			return
		}
		bundleOpts = append(bundleOpts, core.BundleDescriptor(bd))
		bundleOpts = append(bundleOpts, core.Repo(datamonFlags.repo.RepoName))
		bundleOpts = append(bundleOpts, core.ConsumableStore(consumableStore))
		bundleOpts = append(bundleOpts, core.BundleID(datamonFlags.bundle.ID))
		logger, err := optionInputs.getLogger()
		if err != nil {
			onDaemonError("get logger", err)
			return
		}
		bundleOpts = append(bundleOpts, core.Logger(logger))
		bundleOpts = append(bundleOpts, core.BundleWithMetrics(datamonFlags.root.metrics.IsEnabled()))

		bundle := core.NewBundle(bundleOpts...)

		var fsOpts []fuse.Option
		fsOpts = append(fsOpts, fuse.Logger(logger))
		fsOpts = append(fsOpts, fuse.WithMetrics(datamonFlags.root.metrics.IsEnabled()))

		fs, err := fuse.NewMutableFS(bundle, fsOpts...)
		if err != nil {
			onDaemonError("create mutable filesystem", err)
			return
		}
		err = fs.MountMutable(datamonFlags.fs.MountPath)
		if err != nil {
			onDaemonError("mount mutable filesystem", err)
			return
		}
		registerSIGINTHandlerMount(datamonFlags.fs.MountPath)
		if err = daemonizer.SignalOutcome(nil); err != nil {
			wrapFatalln("send event from possibly daemonized process", err)
			return
		}
		if err = fs.JoinMount(ctx); err != nil {
			wrapFatalln("block on os mount", err)
			return
		}
		if err = fs.Commit(); err != nil {
			wrapFatalln("upload bundle from mutable fs", err)
			return
		}
		log.Printf("bundle: %v", bundle.BundleID)
		if datamonFlags.label.Name != "" {
			label := core.NewLabel(
				core.LabelWithMetrics(datamonFlags.root.metrics.IsEnabled()),
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
			log.Printf("set label '%v'", datamonFlags.label.Name)
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

func init() {
	requireFlags(mutableMountBundleCmd,
		addRepoNameOptionFlag(mutableMountBundleCmd),
		addMountPathFlag(mutableMountBundleCmd),
		addCommitMessageFlag(mutableMountBundleCmd),
	)

	addDaemonizeFlag(mutableMountBundleCmd)
	addDataPathFlag(mutableMountBundleCmd)
	addLabelNameFlag(mutableMountBundleCmd)

	mountBundleCmd.AddCommand(mutableMountBundleCmd)
}
