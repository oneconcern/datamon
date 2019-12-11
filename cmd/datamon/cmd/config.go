package cmd

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	storagestatus "github.com/oneconcern/datamon/pkg/storage/status"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

const (
	// environment variable to tune the configuration file location
	envConfigLocation = "DATAMON_CONFIG"

	// default config file location under $HOME
	datamonDir = ".datamon2"

	// default config name (without extension for viper to be able to recognize different serializations)
	configFile = "datamon"
)

// resolve default absolute directory where to find the config file (may be overridden by viper)
func configFileLocation(expandEnv bool) string {
	var home string
	if expandEnv {
		if location := os.Getenv(envConfigLocation); location != "" {
			return location
		}

		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			wrapFatalln("could not get home directory for user", err)
			return ""
		}
	} else {
		home = "$HOME"
	}
	// default config file with YAML serialization used when generating a config file
	return filepath.Join(home, datamonDir, configFile+".yaml")
}

// CLIConfig describes the CLI local configuration file.
type CLIConfig struct {
	// bug in viper? Need to keep names of fields the same as the serialized names..
	Credential string `json:"credential" yaml:"credential"` // Credentials to use for GCS
	Config     string `json:"config" yaml:"config"`         // Config bucket for datamon contexts and metadata
	Context    string `json:"context" yaml:"context"`       // Current context for datamon
}

func (c *CLIConfig) setDatamonParams(flags *flagsT) {
	if flags.context.Descriptor.Name == "" {
		flags.context.Descriptor.Name = c.Context
	}
	if flags.core.Config == "" {
		flags.core.Config = c.Config
	}
}

// MarshalConfig produces a CLI config as a YAML document
func (c *CLIConfig) MarshalConfig() ([]byte, error) {
	return yaml.Marshal(c)
}

func (*CLIConfig) populateRemoteConfig(flags *flagsT) {
	if flags.core.Config == "" {
		wrapFatalln("set environment variable $DATAMON_GLOBAL_CONFIG or define remote config in the config file", nil)
		return
	}
	configStore, err := handleRemoteConfigErr(gcs.New(context.Background(), flags.core.Config, config.Credential))
	if err != nil {
		wrapFatalln("failed to get config store", err)
		return
	}
	rdr, err := handleContextErr(configStore.Get(context.Background(), model.GetPathToContext(flags.context.Descriptor.Name)))
	if err != nil {
		wrapFatalln("failed to get context details from config store for context "+flags.context.Descriptor.Name, err)
		return
	}
	b, err := ioutil.ReadAll(rdr)
	if err != nil {
		wrapFatalln("failed to read context details", err)
		return
	}
	contextDescriptor := model.Context{}
	err = yaml.Unmarshal(b, &contextDescriptor)
	if err != nil {
		wrapFatalln("failed to unmarshal", err)
		return
	}
	flags.context.Descriptor = contextDescriptor
}

func handleRemoteConfigErr(store storage.Store, err error) (storage.Store, error) {
	// provide extra explanation and guidance about the error
	switch err {
	case storagestatus.ErrInvalidResource:
		return nil, fmt.Errorf("please check that the config bucket %v is a valid gcs bucket: %w", store, err)
	case storagestatus.ErrNotExists:
		return nil, fmt.Errorf("please check that the config bucket has been created in your remote config at %v: %w", store, err)
	}
	return store, err
}

func handleContextErr(r io.ReadCloser, err error) (io.ReadCloser, error) {
	// provide extra explanation and guidance about the error
	switch err {
	case storagestatus.ErrInvalidResource:
		return nil, fmt.Errorf("please check that the config bucket is a valid gcs bucket: %w", err)
	case storagestatus.ErrNotExists:
		return nil, fmt.Errorf("please check that the context has been created in your remote config: %w", err)
	}
	return r, err
}

// configCmd represents the bundle related commands
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Commands to manage the config file",
	Long: `Commands to manage datamon local CLI config file.

The local datamon configuration file contains the common set of flags that are needed for most commands and do not change across runs,
analogous to "git config ...".

You may force a specific local config file using the $` + envConfigLocation + ` environment variable (must be some yaml or json file).
`,
}

func init() {
	rootCmd.AddCommand(configCmd)
}
