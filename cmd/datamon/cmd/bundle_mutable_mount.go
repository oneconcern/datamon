// Copyright © 2018 One Concern

package cmd

import (
	"context"

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
		contributor, err := paramsToContributor(params)
		if err != nil {
			wrapFatalln("populate contributor struct", err)
			return
		}
		// cf. comments on runDaemonized in bundle_mount.go
		if params.bundle.Daemonize {
			runDaemonized()
			return
		}
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			onDaemonError("create remote stores", err)
			return
		}
		consumableStore, err := paramsToSrcStore(ctx, params, true)
		if err != nil {
			onDaemonError("create source store", err)
			return
		}

		bd := core.NewBDescriptor(
			core.Message(params.bundle.Message),
			core.Contributor(contributor),
		)
		bundle := core.New(bd,
			core.Repo(params.repo.RepoName),
			core.BlobStore(remoteStores.blob),
			core.ConsumableStore(consumableStore),
			core.MetaStore(remoteStores.meta),
		)
		fs, err := core.NewMutableFS(bundle, params.bundle.DataPath)
		if err != nil {
			onDaemonError("create mutable filesystem", err)
			return
		}
		err = fs.MountMutable(params.bundle.MountPath)
		if err != nil {
			onDaemonError("mount mutable filesystem", err)
			return
		}
		registerSIGINTHandlerMount(params.bundle.MountPath)
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
	},
}

func init() {

	requiredFlags := []string{addRepoNameOptionFlag(mutableMountBundleCmd)}
	addBucketNameFlag(mutableMountBundleCmd)
	addDaemonizeFlag(mutableMountBundleCmd)
	addBlobBucket(mutableMountBundleCmd)
	addDataPathFlag(mutableMountBundleCmd)
	requiredFlags = append(requiredFlags, addMountPathFlag(mutableMountBundleCmd))
	requiredFlags = append(requiredFlags, addCommitMessageFlag(mutableMountBundleCmd))

	for _, flag := range requiredFlags {
		err := mutableMountBundleCmd.MarkFlagRequired(flag)
		if err != nil {
			wrapFatalln("mark required flag", err)
			return
		}
	}

	mountBundleCmd.AddCommand(mutableMountBundleCmd)
}
