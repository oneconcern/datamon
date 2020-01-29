// Copyright © 2018 One Concern

package cmd

import (
	"os"
	"path/filepath"
	"runtime/pprof"

	"github.com/spf13/viper"

	gauth "github.com/oneconcern/datamon/pkg/auth/google"
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
	// Check OAuth
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

	datamonFlagsPtr := &datamonFlags
	datamonFlagsPtr.setDefaultsFromConfig(config)

	//	config.setDatamonParams(&datamonFlags)

	if datamonFlags.context.Descriptor.Name == "" {
		datamonFlags.context.Descriptor.Name = "datamon-dev"
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
