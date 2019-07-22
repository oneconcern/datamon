// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"fmt"

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
			logFatalln(err)
		}
		// cf. comments on runDaemonized in bundle_mount.go
		if params.bundle.Daemonize {
			runDaemonized()
			return
		}
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			onDaemonError(err)
		}
		consumableStore, err := paramsToSrcStore(ctx, params, true)
		if err != nil {
			onDaemonError(err)
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
			onDaemonError(err)
		}
		err = fs.MountMutable(params.bundle.MountPath)
		if err != nil {
			onDaemonError(err)
		}
		registerSIGINTHandlerMount(params.bundle.MountPath)
		if err = daemonizer.SignalOutcome(nil); err != nil {
			logFatalln(err)
		}
		if err = fs.JoinMount(ctx); err != nil {
			logFatalln(err)
		}
		if err = fs.Commit(); err != nil {
			logFatalln(err)
		}
		fmt.Printf("bundle: %v\n", bundle.BundleID)
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
			logFatalln(err)
		}
	}

	mountBundleCmd.AddCommand(mutableMountBundleCmd)
}
