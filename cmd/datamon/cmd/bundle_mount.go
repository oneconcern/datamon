// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"runtime"
//	"runtime/debug"
	"path/filepath"
	"runtime/pprof"
	"time"
	"go.uber.org/zap"
	"strconv"

	daemonizer "github.com/jacobsa/daemonize"

	"github.com/oneconcern/datamon/pkg/dlogger"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"

	"github.com/spf13/cobra"
)

func mempollMaybeprof(mstats runtime.MemStats, minAllocMB uint64) {
	const memprofdest = "/home/developer/"
	if mstats.Alloc / 1024 / 1024 < minAllocMB {
		return
	}
	if _, err := os.Stat(memprofdest); !os.IsNotExist(err) {
		basePath := filepath.Join(memprofdest, "mem_poll-" + strconv.Itoa(int(minAllocMB)))
		profPath := basePath + ".mem.prof"
		allocPath := basePath + ".alloc.prof"
//		dumpPath := basePath + ".heapdump"
		if _, err := os.Stat(profPath); os.IsNotExist(err) {
			var fprof *os.File
			fprof, err = os.Create(profPath)
			if err != nil {
				return
			}
			defer fprof.Close()
			runtime.GC()
			err = pprof.Lookup("heap").WriteTo(fprof, 0)
			if err != nil {
				return
			}
		}
		if _, err := os.Stat(allocPath); os.IsNotExist(err) {
			var falloc *os.File
			falloc, err = os.Create(allocPath)
			if err != nil {
				return
			}
			defer falloc.Close()
			err = pprof.Lookup("allocs").WriteTo(falloc, 0)
			if err != nil {
				return
			}
		}
/*
		if _, err := os.Stat(dumpPath); os.IsNotExist(err) {
			var fdump *os.File
			fdump, err = os.Create(dumpPath)
			if err != nil {
				return
			}
			defer fdump.Close()
			debug.WriteHeapDump(fdump.Fd())
		}
*/
	}
}

func mempollgoroutine(logger *zap.Logger) {
	var maxHeapThusFar uint64
	var mstats runtime.MemStats
	const pollMs = 50
	const loopLogMs = 1000
	var msSinceLog int
	for {
		runtime.ReadMemStats(&mstats)
		if msSinceLog >= loopLogMs {
			logger.Info("mempoll",
				zap.Uint64("MiB for heap (un-GC)", mstats.Alloc / 1024 / 1024),
				zap.Uint64("MiB for heap (max ever)", mstats.HeapSys / 1024 / 1024),
				zap.Int("num go routines", runtime.NumGoroutine()),
			)
			msSinceLog = 0
		}
		if mstats.HeapSys > maxHeapThusFar {
			maxHeapThusFar = mstats.HeapSys
			logger.Info("grew heap",
				zap.Uint64("MiB for heap (un-GC)", mstats.Alloc / 1024 / 1024),
				zap.Uint64("MiB for heap (max ever)", mstats.HeapSys / 1024 / 1024),
			)
		}
		for i := 12; i < 26; i++ {
			mempollMaybeprof(mstats, uint64(i * 1000))
		}
		time.Sleep(time.Duration(pollMs) * time.Millisecond)
		msSinceLog += pollMs
	}
}

func undaemonizeArgs(args []string) []string {
	foregroundArgs := make([]string, 0)
	for _, arg := range args {
		if arg != "--"+addDaemonizeFlag(nil) {
			foregroundArgs = append(foregroundArgs, arg)
		}
	}
	return foregroundArgs
}

func runDaemonized() {
	var path string
	path, err := os.Executable()
	if err != nil {
		err = fmt.Errorf("os.Executable: %v", err)
		logFatalln(err)
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
		err = fmt.Errorf("daemonize.Run: %v", err)
		logFatalln(err)
	}
}

func onDaemonError(err error) {
	if errSig := daemonizer.SignalOutcome(err); errSig != nil {
		logFatalln(fmt.Errorf("error SignalOutcome: %v, cause: %v", errSig, err))
	}
	logFatalln(err)
}

// Mount a read only view of a bundle
var mountBundleCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount a bundle",
	Long:  "Mount a readonly, non-interactive view of the entire data that is part of a bundle",
	Run: func(cmd *cobra.Command, args []string) {
		if params.bundle.Daemonize {
			runDaemonized()
			return
		}

		var err error

		var consumableStorePath string
		if params.bundle.DataPath == "" {
			consumableStorePath, err = ioutil.TempDir("", "datamon-mount-destination")
			if err != nil {
				log.Fatalf("Couldn't create temporary directory: %v\n", err)
				return
			}
		} else {
			consumableStorePath, err = sanitizePath(params.bundle.DataPath)
			if err != nil {
				log.Fatalf("Failed to sanitize destination: %s\n", params.bundle.DataPath)
				return
			}
			createPath(consumableStorePath)
		}

		metadataSource, err := gcs.New(params.repo.MetadataBucket, config.Credential)
		if err != nil {
			onDaemonError(err)
		}
		blobStore, err := gcs.New(params.repo.BlobBucket, config.Credential)
		if err != nil {
			onDaemonError(err)
		}
		consumableStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), consumableStorePath))

		err = setLatestOrLabelledBundle(metadataSource)
		if err != nil {
			logFatalln(err)
		}
		bd := core.NewBDescriptor()
		bundle := core.New(bd,
			core.Repo(params.repo.RepoName),
			core.BundleID(params.bundle.ID),
			core.BlobStore(blobStore),
			core.ConsumableStore(consumableStore),
			core.MetaStore(metadataSource),
			core.Streaming(params.bundle.Stream),
		)
		logger, err := dlogger.GetLogger(params.root.logLevel)
		if err != nil {
			log.Fatalln("Failed to set log level:" + err.Error())
		}
		fs, err := core.NewReadOnlyFS(bundle, logger)
		if err != nil {
			onDaemonError(err)
		}
		if err = fs.MountReadOnly(params.bundle.MountPath); err != nil {
			onDaemonError(err)
		}

		registerSIGINTHandlerMount(params.bundle.MountPath)
		if err = daemonizer.SignalOutcome(nil); err != nil {
			logFatalln(err)
		}

		mempollgoroutine(logger)

		if err = fs.JoinMount(context.Background()); err != nil {
			logFatalln(err)
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
	// todo: #165 add --cpuprof to all commands via root
	addCPUProfFlag(mountBundleCmd)
	addDataPathFlag(mountBundleCmd)
	requiredFlags = append(requiredFlags, addMountPathFlag(mountBundleCmd))

	for _, flag := range requiredFlags {
		err := mountBundleCmd.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	bundleCmd.AddCommand(mountBundleCmd)
}
