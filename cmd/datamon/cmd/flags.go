// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

type paramsT struct {
	bundle struct {
		ID                string
		DataPath          string
		Message           string
		MountPath         string
		File              string
		Daemonize         bool
		Stream            bool
		FileList          string
		SkipOnError       bool
		ConcurrencyFactor int
		NameFilter        string
	}
	web struct {
		port int
	}
	label struct {
		Name string
	}
	repo struct {
		MetadataBucket string
		RepoName       string
		BlobBucket     string
		Description    string
	}
	root struct {
		credFile string
		logLevel string
		cpuProf  bool
	}
	core struct {
		ConcurrencyFactor int
		BatchSize         int
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

func addNameFilterFlag(cmd *cobra.Command) string {
	nameFilter := "name-filter"
	cmd.Flags().StringVar(&params.bundle.NameFilter, nameFilter, "",
		"A regular expression (RE2) to match names of bundle entries.")
	return nameFilter
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

const concurrencyFactorFlag = "concurrency-factor"

func addConcurrencyFactorFlag(cmd *cobra.Command) string {
	concurrencyFactor := concurrencyFactorFlag
	cmd.Flags().IntVar(&params.bundle.ConcurrencyFactor, concurrencyFactor, 100,
		"Heuristic on the amount of concurrency used by various operations.  "+
			"Turn this value down to use less memory, increase for faster operations.")
	return concurrencyFactor
}

func addCoreConcurrencyFactorFlag(cmd *cobra.Command) string {
	// this takes the usual "concurrency-factor" flag, but sets non-object specific settings
	concurrencyFactor := concurrencyFactorFlag
	cmd.Flags().IntVar(&params.core.ConcurrencyFactor, concurrencyFactor, 100,
		"Heuristic on the amount of concurrency used by core operations (e.g. bundle list). "+
			"Concurrent retrieval of bundle metadata is capped by the 'batch-size' parameter. "+
			"Turn this value down to use less memory, increase for faster operations.")
	return concurrencyFactor
}

func addBatchSizeFlag(cmd *cobra.Command) string {
	batchSize := "batch-size"
	cmd.Flags().IntVar(&params.core.BatchSize, batchSize, 1024,
		"Number of bundles streamed together as a batch. This can be tuned for performance based on network connectivity")
	return batchSize
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

/** parameters struct to other formats */

type cmdStoresRemote struct {
	meta storage.Store
	blob storage.Store
}

func paramsToRemoteCmdStores(ctx context.Context, params paramsT) (cmdStoresRemote, error) {
	stores := cmdStoresRemote{}
	meta, err := gcs.New(ctx, params.repo.MetadataBucket, config.Credential)
	if err != nil {
		return cmdStoresRemote{}, err
	}
	stores.meta = meta
	if params.repo.BlobBucket != "" {
		blob, err := gcs.New(ctx, params.repo.BlobBucket, config.Credential)
		if err != nil {
			return cmdStoresRemote{}, err
		}
		stores.blob = blob
	}
	return stores, nil
}

func paramsToSrcStore(ctx context.Context, params paramsT, create bool) (storage.Store, error) {
	var err error
	var consumableStorePath string

	switch {
	case params.bundle.DataPath == "":
		consumableStorePath, err = ioutil.TempDir("", "datamon-mount-destination")
		if err != nil {
			return nil, fmt.Errorf("couldn't create temporary directory: %w", err)
		}
	case strings.HasPrefix(params.bundle.DataPath, "gs://"):
		consumableStorePath = params.bundle.DataPath
	default:
		consumableStorePath, err = sanitizePath(params.bundle.DataPath)
		if err != nil {
			return nil, fmt.Errorf("failed to sanitize destination: %v: %w",
				params.bundle.DataPath, err)
		}
	}

	if create {
		createPath(consumableStorePath)
	}

	var sourceStore storage.Store
	if strings.HasPrefix(consumableStorePath, "gs://") {
		fmt.Println(consumableStorePath[4:])
		sourceStore, err = gcs.New(ctx, consumableStorePath[5:], config.Credential)
		if err != nil {
			return sourceStore, err
		}
	} else {
		DieIfNotAccessible(consumableStorePath)
		DieIfNotDirectory(consumableStorePath)
		sourceStore = localfs.New(afero.NewBasePathFs(afero.NewOsFs(), consumableStorePath))
	}
	return sourceStore, nil
}

// DestT defines the nomenclature for allowed destination types (e.g. Empty/NonEmpty)
type DestT uint

const (
	destTEmpty = iota
	destTMaybeNonEmpty
	destTNonEmpty
)

func paramsToDestStore(params paramsT,
	destT DestT,
	tmpdirPrefix string,
) (storage.Store, error) {
	var err error
	var consumableStorePath string
	var destStore storage.Store

	if tmpdirPrefix != "" && params.bundle.DataPath != "" {
		tmpdirPrefix = ""
	}

	if tmpdirPrefix != "" {
		if destT == destTNonEmpty {
			return nil, fmt.Errorf("can't specify temp dest path and non-empty dir mutually exclusive")
		}
		consumableStorePath, err = ioutil.TempDir("", tmpdirPrefix)
		if err != nil {
			return nil, fmt.Errorf("couldn't create temporary directory: %w", err)
		}
	} else {
		consumableStorePath, err = sanitizePath(params.bundle.DataPath)
		if err != nil {
			return nil, fmt.Errorf("failed to sanitize destination: %s: %w", params.bundle.DataPath, err)
		}
		createPath(consumableStorePath)
	}

	fs := afero.NewBasePathFs(afero.NewOsFs(), consumableStorePath)

	if destT == destTEmpty {
		var empty bool
		empty, err = afero.IsEmpty(fs, "/")
		if err != nil {
			return nil, fmt.Errorf("failed path validation: %w", err)
		}
		if !empty {
			return nil, fmt.Errorf("%s should be empty", consumableStorePath)
		}
	} else if destT == destTNonEmpty {
		/* fail-fast impl.  partial dupe of model pkg, encoded here to more fully encode intent
		 * of this package independently.
		 */
		var ok bool
		ok, err = afero.DirExists(fs, ".datamon")
		if err != nil {
			return nil, fmt.Errorf("failed to look for metadata dir: %w", err)
		}
		if !ok {
			return nil, fmt.Errorf("failed to find metadata dir in %v", consumableStorePath)
		}
	}
	destStore = localfs.New(fs)
	return destStore, nil
}

func paramsToContributor(_ paramsT) (model.Contributor, error) {
	return authorizer.Principal(config.Credential)
}
