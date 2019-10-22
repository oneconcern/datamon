// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"path/filepath"

	"github.com/oneconcern/datamon/pkg/core"

	"github.com/nightlyone/lockfile"
	"github.com/spf13/cobra"
)

var bundleUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a downloaded bundle with a remote bundle.",
	Long: "Update a downloaded bundle with a remote bundle.  " +
		"--destination is a location previously passed to the `bundle download` command.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		destinationStore, err := paramsToDestStore(params, destTNonEmpty, "")
		if err != nil {
			wrapFatalln("create destination store", err)
			return
		}
		/* lockfile to prevent multiple updates to same bundle */
		var cmdLockfile lockfile.Lockfile
		func() {
			var path string
			var cmdLockfilePath string
			path, err = sanitizePath(params.bundle.DataPath)
			if err != nil {
				wrapFatalln("failed path validation", err)
				return
			}
			cmdLockfilePath, err = sanitizePath(filepath.Join(path, ".datamon-lock"))
			if err != nil {
				wrapFatalln("prepare lock file path", err)
				return
			}
			cmdLockfile, err = lockfile.New(cmdLockfilePath)
			if err != nil {
				wrapFatalln("failed to create ui-level lockfile object", err)
				return
			}
		}()
		err = cmdLockfile.TryLock()
		if err != nil {
			wrapFatalln("failed to acquire ui-level lock", err)
			return
		}
		defer func() {
			err = cmdLockfile.Unlock()
			if err != nil {
				wrapFatalln("failed to release ui-level lock", err)
				return
			}
		}()

		err = setLatestOrLabelledBundle(ctx, remoteStores.meta)
		if err != nil {
			wrapFatalln("determine bundle id", err)
			return
		}
		localBundle := core.New(core.NewBDescriptor(),
			core.ConsumableStore(destinationStore),
		)
		remoteBundle := core.New(core.NewBDescriptor(),
			core.Repo(params.repo.RepoName),
			core.MetaStore(remoteStores.meta),
			core.BlobStore(remoteStores.blob),
			core.BundleID(params.bundle.ID),
		)

		err = core.Update(ctx, remoteBundle, localBundle)
		if err != nil {
			wrapFatalln("update bundle", err)
			return
		}

	},
}

func init() {

	// Source
	requiredFlags := []string{addRepoNameOptionFlag(bundleUpdateCmd)}

	// Destination
	requiredFlags = append(requiredFlags, addDataPathFlag(bundleUpdateCmd))

	// Bundle to download
	addBundleFlag(bundleUpdateCmd)
	// Blob bucket
	addBlobBucket(bundleUpdateCmd)
	addBucketNameFlag(bundleUpdateCmd)

	addLabelNameFlag(bundleUpdateCmd)

	addConcurrencyFactorFlag(bundleUpdateCmd)

	for _, flag := range requiredFlags {
		err := bundleUpdateCmd.MarkFlagRequired(flag)
		if err != nil {
			wrapFatalln("mark required flag", err)
			return
		}
	}

	bundleCmd.AddCommand(bundleUpdateCmd)
}
