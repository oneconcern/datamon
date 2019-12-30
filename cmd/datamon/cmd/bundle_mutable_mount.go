// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"log"

	daemonizer "github.com/jacobsa/daemonize"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/spf13/cobra"
)

// Mount a mutable view of a bundle
var mutableMountBundleCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a bundle incrementally with filesystem operations",
	Long:  "Write directories and files to the mountpoint.  Unmount or send SIGINT to this process to save.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		contributor, err := paramsToContributor(datamonFlags)
		if err != nil {
			wrapFatalln("populate contributor struct", err)
			return
		}
		// cf. comments on runDaemonized in bundle_mount.go
		if datamonFlags.bundle.Daemonize {
			runDaemonized()
			return
		}
		remoteStores, err := paramsToDatamonContext(ctx, datamonFlags)
		if err != nil {
			onDaemonError("create remote stores", err)
			return
		}
		consumableStore, err := paramsToSrcStore(ctx, datamonFlags, true)
		if err != nil {
			onDaemonError("create source store", err)
			return
		}

		bd := core.NewBDescriptor(
			core.Message(datamonFlags.bundle.Message),
			core.Contributor(contributor),
		)
		bundleOpts := paramsToBundleOpts(remoteStores)
		bundleOpts = append(bundleOpts, core.Repo(datamonFlags.repo.RepoName))
		bundleOpts = append(bundleOpts, core.ConsumableStore(consumableStore))
		bundleOpts = append(bundleOpts, core.BundleID(datamonFlags.bundle.ID))
		bundle := core.NewBundle(bd,
			bundleOpts...,
		)
		fs, err := core.NewMutableFS(bundle, datamonFlags.bundle.DataPath)
		if err != nil {
			onDaemonError("create mutable filesystem", err)
			return
		}
		err = fs.MountMutable(datamonFlags.bundle.MountPath)
		if err != nil {
			onDaemonError("mount mutable filesystem", err)
			return
		}
		registerSIGINTHandlerMount(datamonFlags.bundle.MountPath)
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
		infoLogger.Printf("bundle: %v", bundle.BundleID)
		if datamonFlags.label.Name != "" {
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
			log.Printf("set label '%v'", datamonFlags.label.Name)
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		config.populateRemoteConfig(&datamonFlags)
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
