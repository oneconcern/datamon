// Copyright © 2018 One Concern

package cmd

import (
	"crypto/tls"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sync/atomic"
	"time"

	"github.com/spf13/viper"

	gauth "github.com/oneconcern/datamon/pkg/auth/google"
	"github.com/oneconcern/datamon/pkg/metrics"
	"github.com/oneconcern/datamon/pkg/metrics/exporters/influxdb"
	"github.com/spf13/cobra"
)

// config file content
var config *CLIConfig

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "datamon",
	Short: "Datamon helps build ML pipelines",
	Long: `Datamon helps build ML pipelines by adding versioning, auditing and lineage tracking to cloud storage tools
(e.g. Google GCS, AWS S3).

This is not a replacement for these tools, but rather a way to manage their inputs and outputs.

Datamon works by providing a git like interface to manage data efficiently:
your data buckets are organized in repositories of versioned and tagged bundles of files.
`,
	DisableAutoGenTag: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if datamonFlags.root.upgrade {
			if err := doSelfUpgrade(upgradeFlags{forceUgrade: true}); err != nil {
				infoLogger.Printf("WARN: failed to upgrade datamon. Carrying on with command in the current version: %v", err)
			} else {
				if err := doExecAfterUpgrade(); err != nil {
					wrapFatalln("cannot execute upgraded datamon", err)
				}
			}
		}

		if datamonFlags.root.cpuProf {
			f, err := os.Create("cpu.prof")
			if err != nil {
				log.Fatal(err)
			}
			_ = pprof.StartCPUProfile(f)
		}

		if datamonFlags.root.metrics.IsEnabled() {
			// validate config for metrics
			var err error

			if metricsURL := datamonFlags.root.metrics.URL; metricsURL != "" {
				_, err = url.Parse(metricsURL)
				if err != nil {
					wrapFatalln("the metrics URL should be a valid URL, e.g. https://[user:password@]host[:port]", err)
				}
			}

			version := NewVersionInfo()
			ip := getOutboundIP()

			opts := []influxdb.StoreOption{
				influxdb.WithDatabase("datamon"),
				influxdb.WithURL(datamonFlags.root.metrics.URL),
				influxdb.WithNameAsTag("metrics"), // use metric name as an influxdb tag, with unique time series "metrics"
				influxdb.WithTimeout(30 * time.Second),
				influxdb.WithTLSConfig(&tls.Config{
					MinVersion: tls.VersionTLS13,
					NextProtos: []string{"h2", "http/1.1"},
				}),
				influxdb.WithInsecureSkipVerify(true),
			}

			// override credentials for metrics backend
			user := datamonFlags.root.metrics.User
			if user != "" {
				opts = append(opts, influxdb.WithUser(user))
			}

			password := datamonFlags.root.metrics.Password
			if password != "" {
				opts = append(opts, influxdb.WithPassword(password))
			}

			sink, err := influxdb.NewStore(opts...)
			if err != nil {
				wrapFatalln("cannot register metrics store", err)
			}

			optionInputs := newCliOptionInputs(config, &datamonFlags)
			contributor, err := optionInputs.contributor()
			if err != nil {
				wrapFatalln("populate contributor struct", err)
				return
			}

			var errorReported int64
			exporter := metrics.DefaultExporter(
				influxdb.WithStore(sink),
				influxdb.WithErrorHandler(func(e error) {
					// only report first error on metrics
					firstTime := atomic.CompareAndSwapInt64(&errorReported, 0, 1)
					if firstTime {
						errlog.Printf("warning: metrics export failed: %v", e)
					}
				}),
				influxdb.WithTags(map[string]string{
					"service": "datamon",
					"version": version.Version, // want to track datamon versions in all metrics
					"ip":      ip.String(),     // want to track originating IP in all metrics
					"context": datamonFlags.context.Descriptor.Name,
					"user":    contributor.Email,
				}),
			)

			metrics.Init(
				metrics.WithBasePath("datamon"), // all metrics are organized in a tree rooted by "/datamon/..."
				metrics.WithExporter(exporter),
			)
			// register CLI specific metrics
			datamonFlags.root.metrics.m = metrics.EnsureMetrics("cmd", &M{}).(*M)
		}
	},
	// upstream api note:  *PostRun functions aren't called in case of a panic() in Run
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if datamonFlags.root.cpuProf {
			pprof.StopCPUProfile()
		}

	},
}

// Execute adds all child commands to the root command and sets datamonFlags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	var err error

	if err = rootCmd.Execute(); err != nil {
		errlog.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	authorizer = gauth.New()
	addConfigFlag(rootCmd)
	addContextFlag(rootCmd)
	addUpgradeFlag(rootCmd)
	addUpgradeForceFlag(rootCmd)
	addLogLevel(rootCmd)
	addMetricsFlag(rootCmd)
	addMetricsURLFlag(rootCmd)
	addMetricsUserFlag(rootCmd)
	addMetricsPasswordFlag(rootCmd)

	addTemplateFlag(repoCmd)
	rootCmd.AddCommand(repoCmd)

	repoCmd.AddCommand(repoDelete)
	requireFlags(repoDelete,
		addRepoNameOptionFlag(repoDelete),
		addContextFlag(repoDelete),
	)
	addForceYesFlag(repoDelete)

	repoCmd.AddCommand(repoRename)
	requireFlags(repoRename,
		addRepoNameOptionFlag(repoRename),
		addContextFlag(repoRename),
	)
	addForceYesFlag(repoRename)

	repoDelete.AddCommand(repoDeleteFiles)
	requireFlags(repoDeleteFiles,
		addRepoNameOptionFlag(repoDeleteFiles),
		addContextFlag(repoDeleteFiles),
	)
	addForceYesFlag(repoDeleteFiles)
	addFileListFlag(repoDeleteFiles)
	addBundleFileFlag(repoDeleteFiles)

	addSkipAuthFlag(purgeCmd, true)
	addPurgeForceFlag(purgeCmd)
	addPurgeLocalPathFlag(purgeCmd)

	addPurgeDryRunFlag(deleteUnusedCmd)
	addConcurrencyFactorFlag(deleteUnusedCmd, 100)
	addConcurrencyFactorFlag(reverseLookupCmd, 100)

	addPurgeResumeFlag(reverseLookupCmd)

	purgeCmd.AddCommand(reverseLookupCmd)
	purgeCmd.AddCommand(deleteUnusedCmd)
	purgeCmd.AddCommand(deleteLookupCmd)
	rootCmd.AddCommand(purgeCmd)
}

// readConfig reads in config file and ENV variables if set.
func readConfig(location string) (*CLIConfig, error) {

	// 1. Defaults: none at the moment (defaults are set together with flags)

	// 2. Override via environment variables
	viper.AutomaticEnv() // read in environment variables that match

	// 3. Read from config file
	if location != "" {
		// use config file from env var
		viper.SetConfigFile(location)
	} else {
		// let viper resolve config file location from some known paths
		viper.SetConfigName(configFile)
		viper.AddConfigPath(".")
		viper.AddConfigPath(filepath.Dir(configFileLocation(true)))
		viper.AddConfigPath("/etc/datamon2")
	}

	// if a config file is found, read it
	handleConfigErrors(viper.ReadInConfig())

	// 4. Initialize config and override via flags
	localConfig := new(CLIConfig)
	if err := viper.Unmarshal(localConfig); err != nil {
		wrapFatalln("config file contains invalid values", err)
		return nil, err
	}

	return localConfig, nil
}

// initConfig reads in config file and ENV variables if set,
// and sets config values based on file, env, cli flags.
func initConfig() {
	var err error
	config, err = readConfig(os.Getenv(envConfigLocation))
	if err != nil {
		wrapFatalln("read config from file and env vars", err)
	}

	// ??? what errors follow if this block is removed?
	if config.Credential != "" {
		// TODO(fred): now handled in paramsToContributor. May be removed
		_ = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", config.Credential)
	}

	if datamonFlags.context.Descriptor.Name == "" {
		datamonFlags.context.Descriptor.Name = viper.GetString("DATAMON_CONTEXT")
	}

	if datamonFlags.core.Config == "" {
		datamonFlags.core.Config = viper.GetString("DATAMON_GLOBAL_CONFIG")
	}

	if config.Metrics.Enabled != nil && datamonFlags.root.metrics.Enabled == nil {
		datamonFlags.root.metrics.Enabled = config.Metrics.Enabled
	}

	if datamonFlags.root.metrics.URL == "" {
		datamonFlags.root.metrics.URL = viper.GetString("DATAMON_METRICS_URL")
	}
	if datamonFlags.root.metrics.URL == "" {
		datamonFlags.root.metrics.URL = config.Metrics.URL
	}

	if datamonFlags.root.metrics.User == "" {
		datamonFlags.root.metrics.User = viper.GetString("DATAMON_METRICS_USER")
	}
	if datamonFlags.root.metrics.User == "" {
		datamonFlags.root.metrics.User = config.Metrics.User
	}

	if datamonFlags.root.metrics.Password == "" {
		datamonFlags.root.metrics.Password = viper.GetString("DATAMON_METRICS_PASSWORD")
	}
	if datamonFlags.root.metrics.Password == "" {
		datamonFlags.root.metrics.Password = config.Metrics.Password
	}

	datamonFlagsPtr := &datamonFlags
	datamonFlagsPtr.setDefaultsFromConfig(config)

	if datamonFlags.context.Descriptor.Name == "" {
		datamonFlags.context.Descriptor.Name = "dev"
	}

	//  do not require config to be set for all commands
}

func handleConfigErrors(err error) {
	if err == nil {
		return
	}
	switch err.(type) {
	case viper.UnsupportedConfigError:
		infoLogger.Println("warning: the config file extension is not of a supported type." +
			"Use a well-known config file extension (.yaml, .json, ...)")
	case *os.PathError:
		// config file was forced but not found: skip
		break
	case viper.ConfigFileNotFoundError:
		// config file resolve attempt, not found: skip
		break
	default:
		// file found but some other error occurred: stop
		wrapFatalln("error reading the config file "+viper.ConfigFileUsed(), err)
		return
	}
}

// getOutboundIP returns the preferred outbound ip of this machine
func getOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80") // doesn't actually connect: only resolves
	if err != nil {
		return net.IP{}
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}
