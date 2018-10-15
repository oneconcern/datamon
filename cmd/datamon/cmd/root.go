// Copyright © 2018 One Concern

package cmd

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

var (
	cfgFile string
	format  string
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .datamon.yaml)")
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
		viper.SetConfigName(".datamon")
	}

	viper.AutomaticEnv() // read in environment variables that match
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func print(data interface{}) error {
	return formatters[format].Format(os.Stdout, data)
}

func printe(data interface{}) error {
	return formatters[format].Format(os.Stderr, data)
}

// A Formatter is used to render output. They take an interface and output a byte array
//
// This byte array should be suitable for writing to a stream directly.
type Formatter interface {
	Format(io.Writer, interface{}) error
}

// FormatterFunc provides a way to use functions as a formatter interface
type FormatterFunc func(io.Writer, interface{}) error

// Format the data with the function
func (f FormatterFunc) Format(w io.Writer, data interface{}) error {
	return f(w, data)
}

// JSONFormatter for printing as pretified json
func JSONFormatter() FormatterFunc {
	return func(w io.Writer, data interface{}) error {
		enc := jsoniter.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}
}

// CompactJSONFormatter for priting as compact json
func CompactJSONFormatter() FormatterFunc {
	return func(w io.Writer, data interface{}) error {
		return jsoniter.NewEncoder(w).Encode(data)
	}
}

// YAMLFormatter for printing as yaml
func YAMLFormatter() FormatterFunc {
	return func(w io.Writer, data interface{}) error {
		enc := yaml.NewEncoder(w)
		defer enc.Close()
		if err := enc.Encode(data); err != nil {
			return err
		}
		return enc.Close()
	}
}

var formatters map[string]Formatter

func initDefaultFormatters() {
	if formatters == nil {
		formatters = make(map[string]Formatter)
		formatters["json"] = JSONFormatter()
		formatters["compactjson"] = CompactJSONFormatter()
		formatters["yaml"] = YAMLFormatter()
	}
}

func knownFormatters() []string {
	res := make([]string, len(formatters))
	var i int
	for k := range formatters {
		res[i] = k
		i++
	}
	return res
}

func addFormatFlag(cmd *cobra.Command, defaultValue string, extraFormatters ...map[string]Formatter) error {
	initDefaultFormatters()
	if defaultValue == "" {
		defaultValue = "yaml"
	}
	cmd.Flags().StringVarP(&format, "output", "o", "", "the output format to use")
	prevPreRunE := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if format == "" {
			format = defaultValue
		}
		for _, ef := range extraFormatters {
			for k, v := range ef {
				formatters[k] = v
			}
		}
		if prevPreRunE != nil {
			if err := prevPreRunE(cmd, args); err != nil {
				return err
			}
		}
		if _, ok := formatters[format]; !ok {
			return fmt.Errorf("%q is not a known output format, use one of: %v", format, knownFormatters())
		}
		return nil
	}
	return nil
}
