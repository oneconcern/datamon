// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	context2 "github.com/oneconcern/datamon/pkg/context"
	gcscontext "github.com/oneconcern/datamon/pkg/context/gcs"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/dlogger"

	"github.com/docker/go-units"
	"github.com/go-openapi/runtime/flagext"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type flagsT struct {
	bundle struct {
		ID                string
		DataPath          string
		Message           string
		File              string
		Daemonize         bool
		FileList          string
		SkipOnError       bool
		ConcurrencyFactor int
		NameFilter        string
	}
	fs struct {
		MountPath      string
		Stream         bool
		CacheSize      flagext.ByteSize
		WithPrefetch   int
		WithVerifyHash bool
	}
	web struct {
		port      int
		noBrowser bool
	}
	label struct {
		Prefix string
		Name   string
	}
	context struct {
		Descriptor model.Context
	}
	repo struct {
		RepoName    string
		Description string
	}
	root struct {
		credFile string
		logLevel string
		cpuProf  bool
		upgrade  bool
		metrics  metricsFlags
	}
	doc struct {
		docTarget string
	}
	core struct {
		Config            string
		ConcurrencyFactor int
		BatchSize         int
		Template          string
	}
	split struct {
		splitID string
		tag     string
	}
	diamond struct {
		diamondID       string
		tag             string
		withConflicts   bool
		withCheckpoints bool
		ignoreConflicts bool
		noConflicts     bool
	}
	upgrade upgradeFlags
}

var datamonFlags = flagsT{}

func addBundleFlag(cmd *cobra.Command) string {
	bundleID := "bundle"
	if cmd != nil {
		cmd.Flags().StringVar(&datamonFlags.bundle.ID, bundleID, "", "The hash id for the bundle, if not specified the latest bundle will be used")
	}
	return bundleID
}

func addDataPathFlag(cmd *cobra.Command) string {
	destination := "destination"
	cmd.Flags().StringVar(&datamonFlags.bundle.DataPath, destination, "",
		"The path to the download dir. Defaults to some random dir /tmp/datamon-mount-destination{xxxxx}")
	return destination
}

func addNameFilterFlag(cmd *cobra.Command) string {
	nameFilter := "name-filter"
	cmd.Flags().StringVar(&datamonFlags.bundle.NameFilter, nameFilter, "",
		"A regular expression (RE2) to match names of bundle entries.")
	return nameFilter
}

func addMountPathFlag(cmd *cobra.Command) string {
	mount := "mount"
	cmd.Flags().StringVar(&datamonFlags.fs.MountPath, mount, "", "The path to the mount dir")
	return mount
}

func addPathFlag(cmd *cobra.Command) string {
	path := "path"
	cmd.Flags().StringVar(&datamonFlags.bundle.DataPath, path, "", "The path to the folder or bucket (gs://<bucket>) for the data")
	return path
}

func addCommitMessageFlag(cmd *cobra.Command) string {
	message := "message"
	cmd.Flags().StringVar(&datamonFlags.bundle.Message, message, "", "The message describing the new bundle")
	return message
}

func addFileListFlag(cmd *cobra.Command) string {
	fileList := "files"
	cmd.Flags().StringVar(&datamonFlags.bundle.FileList, fileList, "", "Text file containing list of files separated by newline.")
	return fileList
}

func addBundleFileFlag(cmd *cobra.Command) string {
	file := "file"
	cmd.Flags().StringVar(&datamonFlags.bundle.File, file, "", "The file to download from the bundle")
	return file
}

func addDaemonizeFlag(cmd *cobra.Command) string {
	daemonize := "daemonize"
	if cmd != nil {
		cmd.Flags().BoolVar(&datamonFlags.bundle.Daemonize, daemonize, false, "Whether to run the command as a daemonized process")
	}
	return daemonize
}

func addStreamFlag(cmd *cobra.Command) string {
	stream := "stream"
	cmd.Flags().BoolVar(&datamonFlags.fs.Stream, stream, true, "Stream in the FS view of the bundle, do not download all files. Default to true.")
	return stream
}

func addSkipMissingFlag(cmd *cobra.Command) string {
	skipOnError := "skip-on-error"
	cmd.Flags().BoolVar(&datamonFlags.bundle.SkipOnError, skipOnError, false, "Skip files encounter errors while reading."+
		"The list of files is either generated or passed in. During upload files can be deleted or encounter an error. Setting this flag will skip those files. Default to false")
	return skipOnError
}

const concurrencyFactorFlag = "concurrency-factor"

func addConcurrencyFactorFlag(cmd *cobra.Command, defaultConcurrency int) string {
	concurrencyFactor := concurrencyFactorFlag
	cmd.Flags().IntVar(&datamonFlags.bundle.ConcurrencyFactor, concurrencyFactor, defaultConcurrency,
		"Heuristic on the amount of concurrency used by various operations.  "+
			"Turn this value down to use less memory, increase for faster operations.")
	return concurrencyFactor
}

func addCoreConcurrencyFactorFlag(cmd *cobra.Command, defaultConcurrency int) string {
	// this takes the usual "concurrency-factor" flag, but sets non-object specific settings
	concurrencyFactor := concurrencyFactorFlag
	cmd.Flags().IntVar(&datamonFlags.core.ConcurrencyFactor, concurrencyFactor, defaultConcurrency,
		"Heuristic on the amount of concurrency used by core operations. "+
			"Concurrent retrieval of metadata is capped by the 'batch-size' parameter. "+
			"Turn this value down to use less memory, increase for faster operations.")
	return concurrencyFactor
}

func addContextFlag(cmd *cobra.Command) string {
	c := "context"
	cmd.PersistentFlags().StringVar(&datamonFlags.context.Descriptor.Name, c, "", `Set the context for datamon (defaults to "dev")`)
	return c
}

func addConfigFlag(cmd *cobra.Command) string {
	config := "config"
	cmd.PersistentFlags().StringVar(&datamonFlags.core.Config, config, "", "Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')")
	return config
}

func addBatchSizeFlag(cmd *cobra.Command) string {
	batchSize := "batch-size"
	cmd.Flags().IntVar(&datamonFlags.core.BatchSize, batchSize, 1024,
		"Number of bundles streamed together as a batch. This can be tuned for performance based on network connectivity")
	return batchSize
}

func addWebPortFlag(cmd *cobra.Command) string {
	webPort := "port"
	cmd.Flags().IntVar(&datamonFlags.web.port, webPort, 0, "Port number for the web server (defaults to random port)")
	return webPort
}

func addWebNoBrowserFlag(cmd *cobra.Command) string {
	c := "no-browser"
	cmd.Flags().BoolVar(&datamonFlags.web.noBrowser, c, false, "Disable automatic launch of a browser")
	return c
}

func addLabelNameFlag(cmd *cobra.Command) string {
	labelName := "label"
	if cmd != nil { // TODO(fred): quickfix - the actual remedy should be to avoid calling this with nil input
		cmd.Flags().StringVar(&datamonFlags.label.Name, labelName, "", "The human-readable name of a label")
	}
	return labelName
}

func addLabelPrefixFlag(cmd *cobra.Command) string {
	prefixString := "prefix"
	cmd.Flags().StringVar(&datamonFlags.label.Prefix, prefixString, "", "List labels starting with a prefix.")
	return prefixString
}

func addRepoNameOptionFlag(cmd *cobra.Command) string {
	repo := "repo"
	cmd.Flags().StringVar(&datamonFlags.repo.RepoName, repo, "", "The name of this repository")
	return repo
}

func addRepoDescription(cmd *cobra.Command) string {
	description := "description"
	cmd.Flags().StringVar(&datamonFlags.repo.Description, description, "", "The description for the repo")
	return description
}

func addBlobBucket(cmd *cobra.Command) string {
	blob := "blob"
	cmd.Flags().StringVar(&datamonFlags.context.Descriptor.Blob, blob, "", "The name of the bucket hosting the datamon blobs")
	return blob
}

func addMetadataBucket(cmd *cobra.Command) string {
	meta := "meta"
	cmd.Flags().StringVar(&datamonFlags.context.Descriptor.Metadata, meta, "", "The name of the bucket used by datamon metadata")
	return meta
}

func addVMetadataBucket(cmd *cobra.Command) string {
	vm := "vmeta"
	cmd.Flags().StringVar(&datamonFlags.context.Descriptor.VMetadata, vm, "", "The name of the bucket hosting the versioned metadata")
	return vm
}

func addWALBucket(cmd *cobra.Command) string {
	b := "wal"
	cmd.Flags().StringVar(&datamonFlags.context.Descriptor.WAL, b, "", "The name of the bucket hosting the WAL")
	return b
}

func addReadLogBucket(cmd *cobra.Command) string {
	b := "read-log"
	cmd.Flags().StringVar(&datamonFlags.context.Descriptor.ReadLog, b, "", "The name of the bucket hosting the read log")
	return b
}

func addCredentialFile(cmd *cobra.Command) string {
	credential := "credential"
	cmd.Flags().StringVar(&datamonFlags.root.credFile, credential, "", "The path to the credential file")
	return credential
}

func addLogLevel(cmd *cobra.Command) string {
	loglevel := "loglevel"
	cmd.PersistentFlags().StringVar(&datamonFlags.root.logLevel, loglevel, "info", "The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug")
	return loglevel
}

func addCPUProfFlag(cmd *cobra.Command) string {
	c := "cpuprof"
	cmd.Flags().BoolVar(&datamonFlags.root.cpuProf, c, false, "Toggle runtime profiling")
	return c
}

func addUpgradeCheckOnlyFlag(cmd *cobra.Command) string {
	c := "check-version"
	cmd.Flags().BoolVar(&datamonFlags.upgrade.checkOnly, c, false, "Checks if a new version is available but does not upgrade")
	return c
}

func addUpgradeForceFlag(cmd *cobra.Command) string {
	c := "force"
	cmd.Flags().BoolVar(&datamonFlags.upgrade.forceUgrade, c, false, "Forces upgrade even if the current version is not a released version")
	return c
}

const upgradeFlag = "upgrade"

func addUpgradeFlag(cmd *cobra.Command) string {
	cmd.PersistentFlags().BoolVar(&datamonFlags.root.upgrade, upgradeFlag, false, "Upgrades the current version then carries on with the specified command")
	return upgradeFlag
}

func addTargetFlag(cmd *cobra.Command) string {
	c := "target-dir"
	cmd.Flags().StringVar(&datamonFlags.doc.docTarget, c, ".", "The target directory where to generate the markdown documentation")
	return c
}

func addCacheSizeFlag(cmd *cobra.Command) string {
	c := "cache-size"
	datamonFlags.fs.CacheSize = flagext.ByteSize(50 * units.MB)
	cmd.Flags().Var(&datamonFlags.fs.CacheSize, c, "The desired size of the memory cache used (in KB, MB, GB, ...) when streaming is enabled")
	return c
}

func addPrefetchFlag(cmd *cobra.Command) string {
	c := "prefetch"
	cmd.Flags().IntVar(&datamonFlags.fs.WithPrefetch, c, 1, "When greater than 0, specifies the number of fetched-ahead blobs when reading a mounted file (requires Stream enabled)")
	return c
}

func addVerifyHashFlag(cmd *cobra.Command) string {
	c := "verify-hash"
	cmd.Flags().BoolVar(&datamonFlags.fs.WithVerifyHash, c, true, "Enables hash verification on read blobs (requires Stream enabled)")
	return c
}

func addTemplateFlag(cmd *cobra.Command) string {
	c := "format"
	cmd.PersistentFlags().StringVar(&datamonFlags.core.Template, c, "", `Pretty-print datamon objects using a Go template. Use '{{ printf "%#v" . }}' to explore available fields`)
	return c
}

func addMetricsFlag(cmd *cobra.Command) string {
	c := "metrics"
	defaultMetrics := false // TODO(fred): once our concern about metrics backend security is addressed, move back the default to true
	datamonFlags.root.metrics.Enabled = &defaultMetrics
	cmd.PersistentFlags().BoolVar(datamonFlags.root.metrics.Enabled, c, defaultMetrics, `Toggle telemetry and metrics collection`)
	return c
}

func addMetricsURLFlag(cmd *cobra.Command) string {
	c := "metrics-url"
	cmd.PersistentFlags().StringVar(&datamonFlags.root.metrics.URL, c, "", `Fully qualified URL to an influxdb metrics collector, with optional user and password`)
	return c
}

func addMetricsUserFlag(cmd *cobra.Command) string {
	c := "metrics-user"
	cmd.PersistentFlags().StringVar(&datamonFlags.root.metrics.URL, c, "", `User to connect to the metrics collector backend. Overrides any user set in URL`)
	return c
}

func addMetricsPasswordFlag(cmd *cobra.Command) string {
	c := "metrics-password"
	cmd.PersistentFlags().StringVar(&datamonFlags.root.metrics.URL, c, "", `Password to connect to the metrics collector backend. Overrides any password set in URL`)
	return c
}

func addDiamondFlag(cmd *cobra.Command) string {
	c := "diamond"
	if cmd != nil {
		cmd.Flags().StringVar(&datamonFlags.diamond.diamondID, c, "", `The diamond to use`)
	}
	return c
}

func addSplitFlag(cmd *cobra.Command) string {
	c := "split"
	if cmd != nil {
		cmd.Flags().StringVar(&datamonFlags.split.splitID, c, "", `The split to use`)
	}
	return c
}

func addWithConflictsFlag(cmd *cobra.Command) string {
	c := "with-conflicts"
	if cmd != nil {
		cmd.Flags().BoolVar(&datamonFlags.diamond.withConflicts, c, true, `Diamond commit handles conflicts and keeps them in store`+
			` Conflicting versions of your uploaded files are located in the .conflicts folder`)
	}
	return c
}

func addNoConflictsFlag(cmd *cobra.Command) string {
	c := "no-conflicts"
	if cmd != nil {
		cmd.Flags().BoolVar(&datamonFlags.diamond.noConflicts, c, false, `Diamond commit fails if any conflict is detected`)
	}
	return c
}

func addIgnoreConflictsFlag(cmd *cobra.Command) string {
	c := "ignore-conflicts"
	if cmd != nil {
		cmd.Flags().BoolVar(&datamonFlags.diamond.ignoreConflicts, c, false, `Diamond commit ignores conflicts and does not report about them`)
	}
	return c
}

func addWithCheckpointFlag(cmd *cobra.Command) string {
	c := "with-checkpoints"
	if cmd != nil {
		cmd.Flags().BoolVar(&datamonFlags.diamond.withCheckpoints, c, false, `Diamond commit handles conflicts and keeps them as intermediate checkpoints rather than conflicts.`+
			` Intermediate versions of your uploaded files are located in the .checkpoints folder`)
	}
	return c
}

func addSplitTagFlag(cmd *cobra.Command) string {
	c := "split-tag"
	if cmd != nil {
		cmd.Flags().StringVar(&datamonFlags.split.tag, c, "", `A custom tag to identify your split in logs or datamon reports. Example: "pod-1"`)
	}
	return c
}

func addDiamondTagFlag(cmd *cobra.Command) string {
	c := "diamond-tag"
	if cmd != nil {
		cmd.Flags().StringVar(&datamonFlags.diamond.tag, c, "", `A custom tag to identify your diamond in logs or datamon reports. Example: "coordinator-pod-A"`)
	}
	return c
}

/** parameters struct from other formats */

// apply config file + env vars to structure used to parse cli flags
func (flags *flagsT) setDefaultsFromConfig(c *CLIConfig) {
	if flags.context.Descriptor.Name == "" {
		flags.context.Descriptor.Name = c.Context
	}
	if flags.core.Config == "" {
		flags.core.Config = c.Config
	}
}

/** combined config (file + env var) and parameters (pflags) */

type cliOptionInputs struct {
	config *CLIConfig
	params *flagsT
}

func newCliOptionInputs(config *CLIConfig, params *flagsT) *cliOptionInputs {
	return &cliOptionInputs{
		config: config,
		params: params,
	}
}

/** combined config and parameters to internal objects */

// DestT defines the nomenclature for allowed destination types (e.g. Empty/NonEmpty)
type DestT uint

const (
	destTEmpty = iota
	destTMaybeNonEmpty
	destTNonEmpty
)

func (in *cliOptionInputs) destStore(destT DestT,
	tmpdirPrefix string,
) (storage.Store, error) {
	var params *flagsT
	var err error
	var consumableStorePath string
	var destStore storage.Store

	params = in.params

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

func (in *cliOptionInputs) datamonContext(ctx context.Context) (context2.Stores, error) {
	logger, err := in.getLogger()
	if err != nil {
		return context2.New(), fmt.Errorf("get logger: %v", err)
	}
	// here we select a 100% gcs backend strategy (more elaborate strategies could be defined by the context pkg)
	return gcscontext.MakeContext(ctx,
		in.params.context.Descriptor,
		in.config.Credential,
		gcs.Logger(logger))
}

func (in *cliOptionInputs) srcStore(ctx context.Context, create bool) (storage.Store, error) {
	var (
		err                 error
		consumableStorePath string
	)
	switch {
	case in.params.bundle.DataPath == "":
		consumableStorePath, err = ioutil.TempDir("", "datamon-mount-destination")
		if err != nil {
			return nil, fmt.Errorf("couldn't create temporary directory: %w", err)
		}
	case strings.HasPrefix(in.params.bundle.DataPath, "gs://"):
		consumableStorePath = in.params.bundle.DataPath
	default:
		consumableStorePath, err = sanitizePath(in.params.bundle.DataPath)
		if err != nil {
			return nil, fmt.Errorf("failed to sanitize destination: %v: %w",
				in.params.bundle.DataPath, err)
		}
	}

	if create {
		createPath(consumableStorePath)
	}

	var sourceStore storage.Store
	if strings.HasPrefix(consumableStorePath, "gs://") {
		logger, err := in.getLogger()
		if err != nil {
			return sourceStore, fmt.Errorf("get logger: %v", err)
		}
		infoLogger.Println(consumableStorePath[4:])
		sourceStore, err = gcs.New(ctx,
			consumableStorePath[5:],
			in.config.Credential,
			gcs.Logger(logger))
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

func (in *cliOptionInputs) bundleOpts(ctx context.Context) ([]core.BundleOption, error) {
	stores, err := in.datamonContext(ctx)
	if err != nil {
		return nil, err
	}
	ops := []core.BundleOption{
		core.ContextStores(stores),
	}
	return ops, nil
}

func (in *cliOptionInputs) getLogger() (*zap.Logger, error) {
	var err error
	in.config.onceLogger.Do(func() {
		in.config.logger, err = dlogger.GetLogger(in.params.root.logLevel)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set log level: %v", err)
	}
	return in.config.logger, nil
}

func (in *cliOptionInputs) populateRemoteConfig() error {
	flags := in.params
	if flags.core.Config == "" {
		return fmt.Errorf("set environment variable $DATAMON_GLOBAL_CONFIG or define remote config in the config file")
	}
	logger, err := in.getLogger()
	if err != nil {
		return fmt.Errorf("get logger: %v", err)
	}
	configStore, err := handleRemoteConfigErr(
		gcs.New(context.Background(),
			flags.core.Config,
			config.Credential,
			gcs.Logger(logger)))
	if err != nil {
		return fmt.Errorf("failed to get config store: %v", err)
	}
	rdr, err := handleContextErr(configStore.Get(context.Background(), model.GetPathToContext(flags.context.Descriptor.Name)))
	if err != nil {
		return fmt.Errorf("failed to get context details from config store for context %q: %v",
			flags.context.Descriptor.Name, err)
	}
	b, err := ioutil.ReadAll(rdr)
	if err != nil {
		return fmt.Errorf("failed to read context details: %v", err)
	}
	contextDescriptor := model.Context{}
	err = yaml.Unmarshal(b, &contextDescriptor)
	if err != nil {
		return fmt.Errorf("failed to unmarshal: %v", err)
	}
	flags.context.Descriptor = contextDescriptor
	return nil
}

func (in *cliOptionInputs) contributor() (model.Contributor, error) {
	flags := in.params
	var credentials string
	switch {
	case flags.root.credFile != "":
		credentials = flags.root.credFile
	case config.Credential != "":
		credentials = config.Credential
	}
	contributor, err := authorizer.Principal(credentials)
	if err != nil {
		return model.Contributor{},
			fmt.Errorf("could not resolve credentials: must be present as --credential flag, or in local config or as GOOGLE_APPLICATION_CREDENTIALS environment. err: %s", err)
	}
	return contributor, err
}

func (in *cliOptionInputs) dumpContext() string {
	return "using config:" + in.config.Config + " context:" + in.params.context.Descriptor.Name
}

/** misc util */

// requireFlags sets a flag (local to the command or inherited) as required
func requireFlags(cmd *cobra.Command, flags ...string) {
	for _, flag := range flags {
		err := cmd.MarkFlagRequired(flag)
		if err != nil {
			err = cmd.MarkPersistentFlagRequired(flag)
		}
		if err != nil {
			wrapFatalln(fmt.Sprintf("error attempting to mark the required flag %q", flag), err)
			return
		}
	}
}
