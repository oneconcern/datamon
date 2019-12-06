// Copyright © 2018 One Concern

package cmd

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"

	"github.com/spf13/viper"

	"github.com/oneconcern/datamon/pkg/auth"
	gauth "github.com/oneconcern/datamon/pkg/auth/google"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "datamon",
	Short: "Datamon helps building ML pipelines",
	Long: `Datamon helps building ML pipelines by adding versioning, auditing and security to cloud storage tools
(e.g. Google GCS, AWS S3).

This is not a replacement for these tools, but rather a way to manage their inputs and outputs.

Datamon works by providing a git like interface to manage data efficiently:
your data buckets are organized in repositories of versioned and tagged bundles of files.
`,
	DisableAutoGenTag: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if datamonFlags.root.upgrade {
			if err := doSelfUpgrade(upgradeFlags{forceUgrade: true}); err != nil {
				log.Printf("WARN: failed to upgrade datamon. Carrying on with command in the current version: %v", err)
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

var config *CLIConfig

// used to patch over calls to os.Exit() during test
var logFatalln = log.Fatalln
var logFatalf = log.Fatalf
var osExit = os.Exit

// used to patch over calls to Authable.Principal() during test
var authorizer auth.Authable

// infoLogger wraps informative messages to os.Stdout without cluttering expected output in tests.
// To be used instead on fmt.Printf(os.Stdout, ...)
var infoLogger = log.New(os.Stdout, "", 0)

func wrapFatalln(msg string, err error) {
	if err == nil {
		logFatalln(msg)
	} else {
		logFatalf("%v", fmt.Errorf(msg+": %w", err))
	}
}

func wrapFatalWithCode(code int, format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	osExit(code)
}

// Execute adds all child commands to the root command and sets datamonFlags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	var err error
	// Check OAuth
	if err = rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	log.SetFlags(0)
	cobra.OnInitialize(initConfig)
	authorizer = gauth.New()
	addConfigFlag(rootCmd)
	addUpgradeFlag(rootCmd)
	addUpgradeForceFlag(rootCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// 1. Set defaults
	// 2. Read in config file
	// 3. Override via environment variable
	// 4. Override via flags.

	// 2. Config file
	if os.Getenv("DATAMON_CONFIG") != "" {
		// Use config file from the flag.
		viper.SetConfigFile(os.Getenv("DATAMON_CONFIG"))
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/" + datamonDir)
		viper.AddConfigPath("/etc/datamon2")
		viper.SetConfigName("datamon2")
	}

	// If a config file is found, read it in.
	_ = viper.ReadInConfig() // nolint:errcheck
	// `viper.ConfigFileUsed()` returns path to config file if error is nil
	var err error
	// 2. Initialize config
	config, err = newConfig()
	if err != nil {
		wrapFatalln("populate config struct", err)
		return
	}

	if config.Credential != "" {
		_ = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", config.Credential)
	}

	viper.AutomaticEnv() // read in environment variables that match
	if datamonFlags.context.Descriptor.Name == "" {
		datamonFlags.context.Descriptor.Name = viper.GetString("DATAMON_CONTEXT")
	}
	if datamonFlags.core.Config == "" {
		datamonFlags.core.Config = viper.GetString("DATAMON_GLOBAL_CONFIG")
	}

	config.setDatamonParams(&datamonFlags)

	if datamonFlags.context.Descriptor.Name == "" {
		datamonFlags.context.Descriptor.Name = "datamon-dev"
	}

	if datamonFlags.core.Config == "" {
		wrapFatalln("set environment variable $DATAMON_GLOBAL_CONFIG or create config file", nil)
		return
	}
}

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
