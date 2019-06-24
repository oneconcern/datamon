// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"fmt"
	"regexp"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/spf13/cobra"
)

const (
	fileDownloadsByConcurrencyFactor     = 10
	filelistDownloadsByConcurrencyFactor = 10
)

var BundleDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download a bundle",
	Long: "Download a readonly, non-interactive view of the entire data that is part of a bundle. If --bundle is not specified" +
		" the latest bundle will be downloaded",
	Run: func(cmd *cobra.Command, args []string) {
		var nameFilterRe *regexp.Regexp

		ctx := context.Background()
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			logFatalln(err)
		}
		destinationStore, err := paramsToDestStore(params, destTEmpty, "")
		if err != nil {
			logFatalln(err)
		}

		err = setLatestOrLabelledBundle(ctx, remoteStores.meta)
		if err != nil {
			logFatalln(err)
		}
		bd := core.NewBDescriptor()
		bundle := core.New(bd,
			core.Repo(params.repo.RepoName),
			core.MetaStore(remoteStores.meta),
			core.ConsumableStore(destinationStore),
			core.BlobStore(remoteStores.blob),
			core.BundleID(params.bundle.ID),
			core.ConcurrentFileDownloads(
				params.bundle.ConcurrencyFactor/fileDownloadsByConcurrencyFactor),
			core.ConcurrentFilelistDownloads(
				params.bundle.ConcurrencyFactor/filelistDownloadsByConcurrencyFactor),
		)

		if params.bundle.NameFilter != "" {
			nameFilterRe, err = regexp.Compile(params.bundle.NameFilter)
			if err != nil {
				logFatalln(fmt.Errorf("name filter regexp %s didn't build: %v",
					params.bundle.NameFilter, err))
			}
			err = core.PublishSelectBundleEntries(ctx, bundle, func(name string) (bool, error) {
				return nameFilterRe.MatchString(name), nil
			})
			if err != nil {
				logFatalln(err)
			}
		} else {
			err = core.Publish(ctx, bundle)
			if err != nil {
				logFatalln(err)
			}
		}
	},
}

func init() {

	// Source
	requiredFlags := []string{addRepoNameOptionFlag(BundleDownloadCmd)}

	// Destination
	requiredFlags = append(requiredFlags, addDataPathFlag(BundleDownloadCmd))

	// Bundle to download
	addBundleFlag(BundleDownloadCmd)
	// Blob bucket
	addBlobBucket(BundleDownloadCmd)
	addBucketNameFlag(BundleDownloadCmd)

	addLabelNameFlag(BundleDownloadCmd)

	addConcurrencyFactorFlag(BundleDownloadCmd)

	addNameFilterFlag(BundleDownloadCmd)

	for _, flag := range requiredFlags {
		err := BundleDownloadCmd.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	bundleCmd.AddCommand(BundleDownloadCmd)
}
