package cmd

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	storagestatus "github.com/oneconcern/datamon/pkg/storage/status"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// CLIConfig describes the CLI configuration.
type CLIConfig struct {
	// bug in viper? Need to keep names of fields the same as the serialized names..
	Credential string `json:"credential" yaml:"credential"` // Credentials to use for GCS
	Config     string `json:"config" yaml:"config"`         // Config for datamon
	Context    string `json:"context" yaml:"context"`       // Context for datamon
}

func newConfig() (*CLIConfig, error) {
	var config CLIConfig
	err := viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *CLIConfig) setDatamonParams(flags *flagsT) {
	if flags.context.Descriptor.Name == "" {
		flags.context.Descriptor.Name = c.Context
	}
	if flags.core.Config == "" {
		flags.core.Config = c.Config
	}
}

func extraMsg(msg, about string, err error) string {
	// provide extra explanation and guidance about the error
	switch err {
	case storagestatus.ErrInvalidResource:
		return fmt.Sprintf("%s: please check that the config bucket %q is a valid gcs bucket", msg, about)
	case storagestatus.ErrNotExists:
		return fmt.Sprintf("%s: please check that the context has been created in your config", msg)
	}
	return msg
}

func (*CLIConfig) populateRemoteConfig(flags *flagsT) {
	configStore, err := gcs.New(context.Background(), flags.core.Config, config.Credential)
	if err != nil {
		wrapFatalln(extraMsg("failed to get config store", flags.core.Config, err), err)
		return
	}
	rdr, err := configStore.Get(context.Background(), model.GetPathToContext(flags.context.Descriptor.Name))
	if err != nil {
		wrapFatalln(extraMsg("failed to get context details from config store for context "+
			flags.context.Descriptor.Name, flags.core.Config, err), err)
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

// configCmd represents the bundle related commands
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Commands to manage a config",
	Long: `Commands to manage datamon CLI config.

Configuration for datamon is the common set of flags that are needed for most commands and do not change across runs,
analogous to "git config ...". `,
}

func init() {
	rootCmd.AddCommand(configCmd)
}
