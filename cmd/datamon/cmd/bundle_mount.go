// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"fmt"
	"os"

	daemonizer "github.com/jacobsa/daemonize"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/dlogger"

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
func onDaemonError(err error) {
	if errSig := daemonizer.SignalOutcome(err); errSig != nil {
		wrapFatalln(fmt.Sprintf("error SignalOutcome: %v", errSig), err)
		return
	}
	logFatalln(err)
}

// Mount a read only view of a bundle
var mountBundleCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount a bundle",
	Long:  "Mount a readonly, non-interactive view of the entire data that is part of a bundle",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		// cf. comments on runDaemonized
		if params.bundle.Daemonize {
			runDaemonized()
			return
		}
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			onDaemonError(err)
			return
		}
		consumableStore, err := paramsToDestStore(params, destTEmpty, "datamon-mount-destination")
		if err != nil {
			onDaemonError(err)
			return
		}

		err = setLatestOrLabelledBundle(ctx, remoteStores.meta)
		if err != nil {
			logFatalln(err)
			return
		}
		bd := core.NewBDescriptor()
		bundle := core.New(bd,
			core.Repo(params.repo.RepoName),
			core.BundleID(params.bundle.ID),
			core.BlobStore(remoteStores.blob),
			core.ConsumableStore(consumableStore),
			core.MetaStore(remoteStores.meta),
			core.Streaming(params.bundle.Stream),
			core.ConcurrentFilelistDownloads(
				params.bundle.ConcurrencyFactor/filelistDownloadsByConcurrencyFactor),
		)
		logger, err := dlogger.GetLogger(params.root.logLevel)
		if err != nil {
			wrapFatalln("failed to set log level", err)
			return
		}
		fs, err := core.NewReadOnlyFS(bundle, logger)
		if err != nil {
			onDaemonError(err)
			return
		}
		if err = fs.MountReadOnly(params.bundle.MountPath); err != nil {
			onDaemonError(err)
			return
		}

		registerSIGINTHandlerMount(params.bundle.MountPath)
		if err = daemonizer.SignalOutcome(nil); err != nil {
			logFatalln(err)
			return
		}
		if err = fs.JoinMount(ctx); err != nil {
			logFatalln(err)
			return
		}
	},
}

func init() {

	requiredFlags := []string{addRepoNameOptionFlag(mountBundleCmd)}
	addBucketNameFlag(mountBundleCmd)
	addDaemonizeFlag(mountBundleCmd)
	addBlobBucket(mountBundleCmd)
	addBundleFlag(mountBundleCmd)
	addLogLevel(mountBundleCmd)
	addStreamFlag(mountBundleCmd)
	addLabelNameFlag(mountBundleCmd)
	addConcurrencyFactorFlag(mountBundleCmd)
	// todo: #165 add --cpuprof to all commands via root
	addCPUProfFlag(mountBundleCmd)
	addDataPathFlag(mountBundleCmd)
	requiredFlags = append(requiredFlags, addMountPathFlag(mountBundleCmd))

	for _, flag := range requiredFlags {
		err := mountBundleCmd.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
			return
		}
	}

	bundleCmd.AddCommand(mountBundleCmd)
}
