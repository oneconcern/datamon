// Copyright © 2018 One Concern

package cmd

import (
	"fmt"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/spf13/cobra"
)

// bundleCmd represents the bundle related commands
var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Commands to manage bundles for a repo",
	Long: `Commands to manage bundles for a repo.

A bundle is a group of files that are tracked and changed together.
Every bundle is an entry in the history of a repository at a point in time.
`,
}

var bundleOptions struct {
	ID               string
	DataPath         string
	Message          string
	ContributorEmail string
	MountPath        string
	File             string
	Daemonize        bool
	Stream           bool
	FileList         string
	SkipOnError      bool
}

func init() {
	rootCmd.AddCommand(bundleCmd)
	addBucketNameFlag(bundleCmd)
	addBlobBucket(bundleCmd)
}

func addBundleFlag(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&bundleOptions.ID, bundleID, "", "The hash id for the bundle, if not specified the latest bundle will be used")
	return bundleID
}

func addDataPathFlag(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&bundleOptions.DataPath, destination, "", "The path to the download dir")
	return destination
}

func addMountPathFlag(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&bundleOptions.MountPath, mount, "", "The path to the mount dir")
	return mount
}

func addPathFlag(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&bundleOptions.DataPath, path, "", "The path to the folder or bucket (gs://<bucket>) for the data")
	return path
}

func addCommitMessageFlag(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&bundleOptions.Message, message, "", "The message describing the new bundle")
	return message
}

func addFileListFlag(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&bundleOptions.FileList, fileList, "", "Text file containing list of files separated by newline.")
	return fileList
}

func addBundleFileFlag(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&bundleOptions.File, file, "", "The file to download from the bundle")
	return file
}

func addDaemonizeFlag(cmd *cobra.Command) string {
	cmd.Flags().BoolVar(&bundleOptions.Daemonize, daemonize, false, "Whether to run the command as a daemonized process")
	return daemonize
}

func addStreamFlag(cmd *cobra.Command) string {
	cmd.Flags().BoolVar(&bundleOptions.Stream, stream, true, "Stream in the FS view of the bundle, do not download all files. Default to true.")
	return stream
}

func addSkipMissing(cmd *cobra.Command) string {
	cmd.Flags().BoolVar(&bundleOptions.SkipOnError, skipOnError, false, "Skip files encounter errors while reading."+
		"The list of files is either generated or passed in. During upload files can be deleted or encounter an error. Setting this flag will skip those files. Default to false")
	return stream
}

func setLatestBundle(store storage.Store) error {
	if bundleOptions.ID == "" {
		key, err := core.GetLatestBundle(repoParams.RepoName, store)
		if err != nil {
			return err
		}
		bundleOptions.ID = key
	}
	fmt.Printf("Using bundle: %s\n", bundleOptions.ID)
	return nil
}
