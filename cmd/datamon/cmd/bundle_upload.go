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
		if params.repo.ContributorEmail == "" {
			logFatalln(fmt.Errorf("contributor email must be set in config or as a cli param"))
		}
		if params.repo.ContributorName == "" {
			logFatalln(fmt.Errorf("contributor name must be set in config or as a cli param"))
		}

		fmt.Println(config.Credential)
		MetaStore, err := gcs.New(params.repo.MetadataBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}
		blobStore, err := gcs.New(params.repo.BlobBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}
		var sourceStore storage.Store
		if strings.HasPrefix(params.bundle.DataPath, "gs://") {
			fmt.Println(params.bundle.DataPath[4:])
			sourceStore, err = gcs.New(params.bundle.DataPath[5:], config.Credential)
			if err != nil {
				logFatalln(err)
			}
		} else {
			DieIfNotAccessible(params.bundle.DataPath)
			DieIfNotDirectory(params.bundle.DataPath)
			sourceStore = localfs.New(afero.NewBasePathFs(afero.NewOsFs(), params.bundle.DataPath))
		}
		contributors := []model.Contributor{{
			Name:  params.repo.ContributorName,
			Email: params.repo.ContributorEmail,
		}}
		bd := core.NewBDescriptor(
			core.Message(params.bundle.Message),
			core.Contributors(contributors),
		)
		bundle := core.New(bd,
			core.Repo(params.repo.RepoName),
			core.BlobStore(blobStore),
			core.ConsumableStore(sourceStore),
			core.MetaStore(MetaStore),
			core.SkipMissing(params.bundle.SkipOnError),
			core.ConcurrentFileUploads(params.bundle.ConcurrentFileUploads),
		)

		if params.bundle.FileList != "" {
			getKeys := func() ([]string, error) {
				var file afero.File
				file, err = os.Open(params.bundle.FileList)
				if err != nil {
					return nil, fmt.Errorf("failed to open file: %s err:%s", params.bundle.FileList, err.Error())
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

		if params.label.Name == "" {
			return
		}
		labelDescriptor := core.NewLabelDescriptor(
			core.LabelContributors(contributors),
		)
		label := core.NewLabel(labelDescriptor,
			core.LabelName(params.label.Name),
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
	addSkipMissingFlag(uploadBundleCmd)
	addConcurrentFileUploadsFlag(uploadBundleCmd)
	for _, flag := range requiredFlags {
		err := uploadBundleCmd.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	bundleCmd.AddCommand(uploadBundleCmd)
}
