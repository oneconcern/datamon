// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	daemonizer "github.com/jacobsa/daemonize"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"

	"github.com/spf13/cobra"
)

// Mount a mutable view of a bundle
var mutableMountBundleCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a bundle incrementally with filesystem operations",
	Long:  "Write directories and files to the mountpoint.  Unmount or send SIGINT to this process to save.",
	Run: func(cmd *cobra.Command, args []string) {
		if params.repo.ContributorEmail == "" {
			logFatalln(fmt.Errorf("contributor email must be set in config or as a cli param"))
		}
		if params.repo.ContributorName == "" {
			logFatalln(fmt.Errorf("contributor name must be set in config or as a cli param"))
		}

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

		bd := core.NewBDescriptor(
			core.Message(params.bundle.Message),
			core.Contributors([]model.Contributor{{
				Name:  params.repo.ContributorName,
				Email: params.repo.ContributorEmail,
			},
			}),
		)
		bundle := core.New(bd,
			core.Repo(params.repo.RepoName),
			core.BlobStore(blobStore),
			core.ConsumableStore(consumableStore),
			core.MetaStore(metadataSource),
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
		if err = fs.JoinMount(context.Background()); err != nil {
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
