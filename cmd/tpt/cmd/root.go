// Copyright © 2018 One Concern

package cmd

import (
	"fmt"
	"os"

	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tpt",
	Short: "Trumpet helps building ML pipelines",
	Long: `Trumpet helps building ML pipelines by adding versioning, auditing and security to existing tools.

This is not a replacement for existing tools, but rather a way to manage their inputs and outputs.

Trumpet works by providing a git like interface to manage data efficiently.
It executes pipelines by scheduling the processors as serverless functions on either AWS lambda or on kubeless.

`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	log.SetFlags(0)
	cobra.OnInitialize(initConfig)
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.trumpet.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if os.Getenv("TRUMPET_CONFIG") != "" {
		// Use config file from the flag.
		viper.SetConfigFile(os.Getenv("TRUMPET_CONFIG"))
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName(".trumpet")
	}

	viper.AutomaticEnv() // read in environment variables that match
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
