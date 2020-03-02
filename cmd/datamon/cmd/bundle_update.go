// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"path/filepath"
	"time"

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
		var err error

		defer func(t0 time.Time) {
			cliUsage(t0, "bundle update", err)
		}(time.Now())

		ctx := context.Background()
		optionInputs := newCliOptionInputs(config, &datamonFlags)
		remoteStores, err := optionInputs.datamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		destinationStore, err := optionInputs.destStore(destTNonEmpty, "")
		if err != nil {
			wrapFatalln("create destination store", err)
			return
		}
		/* lockfile to prevent multiple updates to same bundle */
		var cmdLockfile lockfile.Lockfile
		func() {
			var path string
			var cmdLockfilePath string
			path, err = sanitizePath(datamonFlags.bundle.DataPath)
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

		err = setLatestOrLabelledBundle(ctx, remoteStores)
		if err != nil {
			wrapFatalln("determine bundle id", err)
			return
		}
		localBundle := core.NewBundle(
			core.ConsumableStore(destinationStore),
			core.BundleWithMetrics(datamonFlags.root.metrics.IsEnabled()),
		)
		bundleOpts, err := optionInputs.bundleOpts(ctx)
		if err != nil {
			wrapFatalln("failed to initialize bundle options", err)
		}
		bundleOpts = append(bundleOpts, core.Repo(datamonFlags.repo.RepoName))
		bundleOpts = append(bundleOpts, core.BundleID(datamonFlags.bundle.ID))
		bundleOpts = append(bundleOpts, core.BundleWithMetrics(datamonFlags.root.metrics.IsEnabled()))
		remoteBundle := core.NewBundle(
			bundleOpts...,
		)

		err = core.Update(ctx, remoteBundle, localBundle)
		if err != nil {
			wrapFatalln("update bundle", err)
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
	requireFlags(bundleUpdateCmd,
		// Source
		addRepoNameOptionFlag(bundleUpdateCmd),
		// Destination
		addDataPathFlag(bundleUpdateCmd),
	)

	// Bundle to download
	addBundleFlag(bundleUpdateCmd)
	addLabelNameFlag(bundleUpdateCmd)
	addConcurrencyFactorFlag(bundleUpdateCmd, 100)

	bundleCmd.AddCommand(bundleUpdateCmd)
}
