// Copyright © 2018 One Concern

package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/viper"

	"github.com/spf13/cobra"
)

// Global list of flags
const (
	bundleID         = "bundle"
	destination      = "destination"
	mount            = "mount"
	folder           = "folder"
	message          = "message"
	repo             = "repo"
	meta             = "meta"
	blob             = "blob"
	description      = "description"
	contributorEmail = "email"
	contributorName  = "name"
	credential       = "credential"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "datamon",
	Short: "Datamon helps building ML pipelines",
	Long: `Datamon helps building ML pipelines by adding versioning, auditing and security to existing tools.

This is not a replacement for existing tools, but rather a way to manage their inputs and outputs.

Datamon works by providing a git like interface to manage data efficiently.
It executes pipelines by scheduling the processors as serverless functions on either AWS lambda or on kubeless.

`,
}

var config *Config
var credFile string

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	var err error
	if err = rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	log.SetFlags(0)
	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if os.Getenv("DATAMON_CONFIG") != "" {
		// Use config file from the flag.
		viper.SetConfigFile(os.Getenv("DATAMON_CONFIG"))
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.datamon")
		viper.AddConfigPath("/etc/datamon")
		viper.SetConfigName("datamon")
	}

	viper.AutomaticEnv() // read in environment variables that match
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	}
	var err error
	err, config = newConfig()
	if err != nil {
		log.Fatalln(err)
	}
	config.setRepoParams(&repoParams)
	if config.Credential != "" {
		// Always pick the config file. There can be a duplicate bucket name in a different project, avoid wrong environment
		// variable from dev testing from screwing things up..
		_ = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", config.Credential)
	}
}

func addCredentialFile(cmd *cobra.Command) string {
	cmd.Flags().StringVar(&credFile, credential, "", "The path to the credential file")
	return contributorName
}
