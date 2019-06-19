// Copyright © 2018 One Concern

package cmd

import (
	"github.com/spf13/cobra"
)

type paramsT struct {
	bundle struct {
		ID                    string
		DataPath              string
		Message               string
		ContributorEmail      string
		MountPath             string
		File                  string
		Daemonize             bool
		Stream                bool
		FileList              string
		SkipOnError           bool
		ConcurrentFileUploads int
	}
	web struct {
		port int
	}
	label struct {
		Name string
	}
	repo struct {
		MetadataBucket   string
		RepoName         string
		BlobBucket       string
		Description      string
		ContributorEmail string
		ContributorName  string
	}
	root struct {
		credFile string
		logLevel string
		cpuProf  bool
	}
}

var params = paramsT{}

func addBundleFlag(cmd *cobra.Command) string {
	bundleID := "bundle"
	if cmd != nil {
		cmd.Flags().StringVar(&params.bundle.ID, bundleID, "", "The hash id for the bundle, if not specified the latest bundle will be used")
	}
	return bundleID
}

func addDataPathFlag(cmd *cobra.Command) string {
	destination := "destination"
	cmd.Flags().StringVar(&params.bundle.DataPath, destination, "", "The path to the download dir")
	return destination
}

func addMountPathFlag(cmd *cobra.Command) string {
	mount := "mount"
	cmd.Flags().StringVar(&params.bundle.MountPath, mount, "", "The path to the mount dir")
	return mount
}

func addPathFlag(cmd *cobra.Command) string {
	path := "path"
	cmd.Flags().StringVar(&params.bundle.DataPath, path, "", "The path to the folder or bucket (gs://<bucket>) for the data")
	return path
}

func addCommitMessageFlag(cmd *cobra.Command) string {
	message := "message"
	cmd.Flags().StringVar(&params.bundle.Message, message, "", "The message describing the new bundle")
	return message
}

func addFileListFlag(cmd *cobra.Command) string {
	fileList := "files"
	cmd.Flags().StringVar(&params.bundle.FileList, fileList, "", "Text file containing list of files separated by newline.")
	return fileList
}

func addBundleFileFlag(cmd *cobra.Command) string {
	file := "file"
	cmd.Flags().StringVar(&params.bundle.File, file, "", "The file to download from the bundle")
	return file
}

func addDaemonizeFlag(cmd *cobra.Command) string {
	daemonize := "daemonize"
	if cmd != nil {
		cmd.Flags().BoolVar(&params.bundle.Daemonize, daemonize, false, "Whether to run the command as a daemonized process")
	}
	return daemonize
}

func addStreamFlag(cmd *cobra.Command) string {
	stream := "stream"
	cmd.Flags().BoolVar(&params.bundle.Stream, stream, true, "Stream in the FS view of the bundle, do not download all files. Default to true.")
	return stream
}

func addSkipMissingFlag(cmd *cobra.Command) string {
	skipOnError := "skip-on-error"
	cmd.Flags().BoolVar(&params.bundle.SkipOnError, skipOnError, false, "Skip files encounter errors while reading."+
		"The list of files is either generated or passed in. During upload files can be deleted or encounter an error. Setting this flag will skip those files. Default to false")
	return skipOnError
}

func addConcurrentFileUploadsFlag(cmd *cobra.Command) string {
	concurrentFileUploads := "num-file-uploads"
	cmd.Flags().IntVar(&params.bundle.ConcurrentFileUploads, concurrentFileUploads, 20,
		"Number of files to upload at a time.  "+
			"If uploads consume too much memory, turn this value down.")
	return concurrentFileUploads
}

func addWebPortFlag(cmd *cobra.Command) string {
	cmd.Flags().IntVar(&params.web.port, webPort, 3003, "Port number for the web server")
	return webPort
}

func addLabelNameFlag(cmd *cobra.Command) string {
	labelName := "label"
	cmd.Flags().StringVar(&params.label.Name, labelName, "", "The human-readable name of a label")
	return labelName
}

func addRepoNameOptionFlag(cmd *cobra.Command) string {
	repo := "repo"
	cmd.Flags().StringVar(&params.repo.RepoName, repo, "", "The name of this repository")
	return repo
}

func addBucketNameFlag(cmd *cobra.Command) string {
	meta := "meta"
	cmd.Flags().StringVar(&params.repo.MetadataBucket, meta, "", "The name of the bucket used by datamon metadata")
	_ = cmd.Flags().MarkHidden(meta)
	return meta
}

func addRepoDescription(cmd *cobra.Command) string {
	description := "description"
	cmd.Flags().StringVar(&params.repo.Description, description, "", "The description for the repo")
	return description
}

func addBlobBucket(cmd *cobra.Command) string {
	blob := "blob"
	cmd.Flags().StringVar(&params.repo.BlobBucket, blob, "", "The name of the bucket hosting the datamon blobs")
	_ = cmd.Flags().MarkHidden(blob)
	return blob
}

func addContributorEmail(cmd *cobra.Command) string {
	contributorEmail := "email"
	cmd.Flags().StringVar(&params.repo.ContributorEmail, contributorEmail, "", "The email of the contributor")
	return contributorEmail
}
func addContributorName(cmd *cobra.Command) string {
	contributorName := "name"
	cmd.Flags().StringVar(&params.repo.ContributorName, contributorName, "", "The name of the contributor")
	return contributorName
}

func addCredentialFile(cmd *cobra.Command) string {
	credential := "credential"
	cmd.Flags().StringVar(&params.root.credFile, credential, "", "The path to the credential file")
	return credential
}

func addLogLevel(cmd *cobra.Command) string {
	loglevel := "loglevel"
	cmd.Flags().StringVar(&params.root.logLevel, loglevel, "info", "The logging level")
	return loglevel
}

func addCPUProfFlag(cmd *cobra.Command) string {
	cpuprof := "cpuprof"
	cmd.Flags().BoolVar(&params.root.cpuProf, cpuprof, false, "Toggle runtime profiling")
	return cpuprof
}
