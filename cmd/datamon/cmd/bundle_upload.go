package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/oneconcern/datamon/pkg/storage"

	"github.com/oneconcern/datamon/pkg/model"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// uploadBundleCmd is the command to upload a bundle from Datamon and model it locally.
var uploadBundleCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a bundle",
	Long:  "Upload a bundle consisting of all files stored in a directory",
	Run: func(cmd *cobra.Command, args []string) {
		if repoParams.ContributorEmail == "" {
			logFatalln(fmt.Errorf("contributor email must be set in config or as a cli param"))
		}
		if repoParams.ContributorName == "" {
			logFatalln(fmt.Errorf("contributor name must be set in config or as a cli param"))
		}

		fmt.Println(config.Credential)
		MetaStore, err := gcs.New(repoParams.MetadataBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}
		blobStore, err := gcs.New(repoParams.BlobBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}
		var sourceStore storage.Store
		if strings.HasPrefix(bundleOptions.DataPath, "gs://") {
			fmt.Println(bundleOptions.DataPath[4:])
			sourceStore, err = gcs.New(bundleOptions.DataPath[5:], config.Credential)
			if err != nil {
				logFatalln(err)
			}
		} else {
			DieIfNotAccessible(bundleOptions.DataPath)
			DieIfNotDirectory(bundleOptions.DataPath)
			sourceStore = localfs.New(afero.NewBasePathFs(afero.NewOsFs(), bundleOptions.DataPath))
		}
		contributors := []model.Contributor{{
			Name:  repoParams.ContributorName,
			Email: repoParams.ContributorEmail,
		}}
		bd := core.NewBDescriptor(
			core.Message(bundleOptions.Message),
			core.Contributors(contributors),
		)
		bundle := core.New(bd,
			core.Repo(repoParams.RepoName),
			core.BlobStore(blobStore),
			core.ConsumableStore(sourceStore),
			core.MetaStore(MetaStore),
			core.SkipMissing(bundleOptions.SkipOnError),
		)

		if bundleOptions.FileList != "" {
			getKeys := func() ([]string, error) {
				var file afero.File
				file, err = os.Open(bundleOptions.FileList)
				if err != nil {
					return nil, fmt.Errorf("failed to open file: %s err:%s", bundleOptions.FileList, err.Error())
				}
				lineScanner := bufio.NewScanner(file)
				files := make([]string, 0)
				for lineScanner.Scan() {
					files = append(files, lineScanner.Text())
				}
				return files, nil
			}
			err = core.UploadSpecificKeys(context.Background(), bundle, getKeys)
		} else {
			err = core.Upload(context.Background(), bundle)
		}
		if err != nil {
			logFatalln(err)
		}

		if labelOptions.Name == "" {
			return
		}
		labelDescriptor := core.NewLabelDescriptor(
			core.LabelContributors(contributors),
		)
		label := core.NewLabel(labelDescriptor,
			core.LabelName(labelOptions.Name),
		)
		err = label.UploadDescriptor(context.Background(), bundle)
		if err != nil {
			logFatalln(err)
		}
	},
}

func init() {

	requiredFlags := []string{addRepoNameOptionFlag(uploadBundleCmd)}
	requiredFlags = append(requiredFlags, addPathFlag(uploadBundleCmd))
	requiredFlags = append(requiredFlags, addCommitMessageFlag(uploadBundleCmd))
	addFileListFlag(uploadBundleCmd)
	addLabelNameFlag(uploadBundleCmd)
	addSkipMissing(uploadBundleCmd)
	for _, flag := range requiredFlags {
		err := uploadBundleCmd.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	bundleCmd.AddCommand(uploadBundleCmd)
}
