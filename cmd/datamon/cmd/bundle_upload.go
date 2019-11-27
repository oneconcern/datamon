package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/dlogger"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	fileUploadsByConcurrencyFactor = 5
)

// uploadBundleCmd is the command to upload a bundle from Datamon and model it locally.
var uploadBundleCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a bundle",
	Long:  "Upload a bundle consisting of all files stored in a directory",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		contributor, err := paramsToContributor(datamonFlags)
		if err != nil {
			wrapFatalln("populate contributor struct", err)
			return
		}
		remoteStores, err := paramsToDatamonContext(ctx, datamonFlags)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		sourceStore, err := paramsToSrcStore(ctx, datamonFlags, false)
		if err != nil {
			wrapFatalln("create source store", err)
			return
		}
		logger, err := dlogger.GetLogger(datamonFlags.root.logLevel)
		if err != nil {
			wrapFatalln("failed to set log level", err)
			return
		}
		bd := core.NewBDescriptor(
			core.Message(datamonFlags.bundle.Message),
			core.Contributor(contributor),
		)

		bundleOpts := paramsToBundleOpts(remoteStores)
		bundleOpts = append(bundleOpts, core.ConsumableStore(sourceStore))
		bundleOpts = append(bundleOpts, core.Repo(datamonFlags.repo.RepoName))
		bundleOpts = append(bundleOpts, core.SkipMissing(datamonFlags.bundle.SkipOnError))
		bundleOpts = append(bundleOpts,
			core.ConcurrentFileUploads(datamonFlags.bundle.ConcurrencyFactor/fileUploadsByConcurrencyFactor))
		bundleOpts = append(bundleOpts, core.Logger(logger))

		bundle := core.NewBundle(bd,
			bundleOpts...,
		)

		if datamonFlags.bundle.FileList != "" {
			getKeys := func() ([]string, error) {
				var file afero.File
				file, err = os.Open(datamonFlags.bundle.FileList)
				if err != nil {
					return nil, fmt.Errorf("failed to open file: %s: %w", datamonFlags.bundle.FileList, err)
				}
				lineScanner := bufio.NewScanner(file)
				files := make([]string, 0)
				for lineScanner.Scan() {
					files = append(files, lineScanner.Text())
				}
				return files, nil
			}
			err = core.UploadSpecificKeys(ctx, bundle, getKeys)
			if err != nil {
				wrapFatalln("upload bundle by filelist", err)
				return
			}
		} else {
			err = core.Upload(ctx, bundle)
			if err != nil {
				wrapFatalln("upload bundle", err)
				return
			}
		}
		log.Printf("Uploaded bundle id:%s ", bundle.BundleID)

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

	requiredFlags := []string{addRepoNameOptionFlag(uploadBundleCmd)}
	requiredFlags = append(requiredFlags, addPathFlag(uploadBundleCmd))
	requiredFlags = append(requiredFlags, addCommitMessageFlag(uploadBundleCmd))
	addFileListFlag(uploadBundleCmd)
	addLabelNameFlag(uploadBundleCmd)
	addSkipMissingFlag(uploadBundleCmd)
	addConcurrencyFactorFlag(uploadBundleCmd, 100)
	addLogLevel(uploadBundleCmd)

	addContextFlag(uploadBundleCmd)

	for _, flag := range requiredFlags {
		err := uploadBundleCmd.MarkFlagRequired(flag)
		if err != nil {
			wrapFatalln("mark required flag", err)
			return
		}
	}

	bundleCmd.AddCommand(uploadBundleCmd)
}
