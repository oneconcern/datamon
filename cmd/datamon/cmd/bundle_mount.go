// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"fmt"
	"os"

	daemonizer "github.com/jacobsa/daemonize"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/fuse"
	"github.com/oneconcern/datamon/pkg/metrics"

	"github.com/spf13/cobra"
)

func undaemonizeArgs(args []string) []string {
	foregroundArgs := make([]string, 0)
	for _, arg := range args {
		if arg != "--"+addDaemonizeFlag(nil) {
			foregroundArgs = append(foregroundArgs, arg)
		}
	}
	return foregroundArgs
}

/**
 * call this function followed by return at any point in a Run: func in order to run the command as a pseudo-daemonized process.
 * conceptually, pseudo-daemonization is akin to usual daemon processes in C wherein fork() does the job
 * of splitting the process in two within the control-flow of the given process, copying or sharing memory segments as needed.
 *
 * Go doesn't fork() because of the runtime.  only exec() is available.  so what pseudo-daemonization means is exec()ing
 * the selfsame binary in a goroutine with some additional IPC communication via pipes to simulate a meaningful fork()-like
 * return value indicating whether the process started successfully without blocking on the exec()ed process's exit code.
 *
 * specifically, `daemonizer.SignalOutcome(nil)` is used to in Run() to bracket the daemonized process and the end of the
 * initialization code.
 */
func runDaemonized() {
	var path string
	path, err := os.Executable()
	if err != nil {
		wrapFatalln("os.Executable", err)
		return
	}

	foregroundArgs := undaemonizeArgs(os.Args[1:])

	// Pass along PATH so that the daemon can find fusermount on Linux.
	env := []string{
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
	}

	// Pass along GOOGLE_APPLICATION_CREDENTIALS
	if p, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS"); ok {
		env = append(env, fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", p))
	}

	// Run.
	err = daemonizer.Run(path, foregroundArgs, env, os.Stdout)
	if err != nil {
		wrapFatalln("daemonize.Run", err)
		return
	}
}

/**
 * in between runDaemonized() and SignalOutcome(), call this function instead of logFatalln() or similar
 * in case of errors
 */
func onDaemonError(msg string, err error) {
	if errSig := daemonizer.SignalOutcome(fmt.Errorf("%v: %v", msg, err)); errSig != nil {
		wrapFatalln(fmt.Sprintf("message '%v' SignalOutcome '%v'", msg, errSig), err)
		return
	}
	wrapFatalln(msg, err)
}

// Mount a read only view of a bundle
var mountBundleCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount a bundle",
	Long:  "Mount a readonly, non-interactive view of the entire data that is part of a bundle",
	Run: func(cmd *cobra.Command, args []string) {
		if datamonFlags.root.metrics.IsEnabled() {
			// do not record timings or failures for long running or daemonized commands, do not wait for completion to report
			datamonFlags.root.metrics.m.Usage.Inc("bundle mount")
			metrics.Flush()
		}

		ctx := context.Background()

		// cf. comments on runDaemonized
		if datamonFlags.bundle.Daemonize {
			runDaemonized()
			return
		}
		optionInputs := newCliOptionInputs(config, &datamonFlags)
		remoteStores, err := optionInputs.datamonContext(ctx, ReadOnlyContext())
		if err != nil {
			onDaemonError("create remote stores", err)
			return
		}
		consumableStore, err := optionInputs.destStore(destTEmpty, "datamon-mount-destination")
		if err != nil {
			onDaemonError("create destination store", err)
			return
		}

		err = setLatestOrLabelledBundle(ctx, remoteStores)
		if err != nil {
			onDaemonError("determine bundle id", err)
			return
		}
		bundleOpts, err := optionInputs.bundleOpts(ctx, ReadOnlyContext())
		if err != nil {
			wrapFatalln("failed to initialize bundle options", err)
		}
		bundleOpts = append(bundleOpts, core.Repo(datamonFlags.repo.RepoName))
		bundleOpts = append(bundleOpts, core.ConsumableStore(consumableStore))
		bundleOpts = append(bundleOpts, core.BundleID(datamonFlags.bundle.ID))
		bundleOpts = append(bundleOpts, core.ConcurrentFilelistDownloads(getConcurrencyFactor(filelistDownloadsByConcurrencyFactor)))
		logger, err := optionInputs.getLogger()
		if err != nil {
			onDaemonError("get logger", err)
			return
		}
		bundleOpts = append(bundleOpts, core.Logger(logger))
		bundleOpts = append(bundleOpts, core.BundleWithMetrics(datamonFlags.root.metrics.IsEnabled()))

		bundle := core.NewBundle(bundleOpts...)
		var fsOpts []fuse.Option
		fsOpts = append(fsOpts, fuse.Streaming(datamonFlags.fs.Stream))
		fsOpts = append(fsOpts, fuse.Logger(logger))
		if datamonFlags.fs.Stream {
			fsOpts = append(fsOpts, fuse.CacheSize(int(datamonFlags.fs.CacheSize)))
			fsOpts = append(fsOpts, fuse.Prefetch(datamonFlags.fs.WithPrefetch))
			fsOpts = append(fsOpts, fuse.VerifyHash(datamonFlags.fs.WithVerifyHash))
			fsOpts = append(fsOpts, fuse.WithMetrics(datamonFlags.root.metrics.IsEnabled()))
		}
		fs, err := fuse.NewReadOnlyFS(bundle, fsOpts...)
		if err != nil {
			onDaemonError("create read only filesystem", err)
			return
		}
		if err = fs.MountReadOnly(datamonFlags.fs.MountPath); err != nil {
			onDaemonError("mount read only filesystem", err)
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
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

func init() {
	requireFlags(mountBundleCmd,
		addRepoNameOptionFlag(mountBundleCmd),
		addMountPathFlag(mountBundleCmd),
	)

	addDaemonizeFlag(mountBundleCmd)
	addBundleFlag(mountBundleCmd)
	addStreamFlag(mountBundleCmd)
	addLabelNameFlag(mountBundleCmd)
	addConcurrencyFactorFlag(mountBundleCmd, 100)
	// todo: #165 add --cpuprof to all commands via root
	addCPUProfFlag(mountBundleCmd)
	addDataPathFlag(mountBundleCmd)
	addCacheSizeFlag(mountBundleCmd)
	addPrefetchFlag(mountBundleCmd)
	addVerifyHashFlag(mountBundleCmd)

	bundleCmd.AddCommand(mountBundleCmd)
}
