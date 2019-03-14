// Copyright © 2018 One Concern

package cmd

import (
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
}

func init() {
	rootCmd.AddCommand(bundleCmd)
	addBucketNameFlag(bundleCmd)
	addBlobBucket(bundleCmd)
}

func addBundleFlag(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&bundleOptions.ID, bundleID, "", "The hash id for the bundle")
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

func addFolderPathFlag(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&bundleOptions.DataPath, folder, "", "The path to the folder of the bundle")
	return folder
}

func addCommitMessageFlag(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&bundleOptions.Message, message, "", "The message describing the new bundle")
	return message
}

func addBundleFileFlag(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&bundleOptions.File, file, "", "The file to download from the bundle")
	return file
}
