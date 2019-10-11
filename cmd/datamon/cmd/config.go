package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

type Config struct {
	// bug in viper? Need to keep names of fields the same as the serialized names..
	Metadata   string `json:"metadata" yaml:"metadata"`
	Blob       string `json:"blob" yaml:"blob"`
	Email      string `json:"email" yaml:"email"`
	Name       string `json:"name" yaml:"name"`
	Credential string `json:"credential" yaml:"credential"`
}

func newConfig() (*Config, error) {
	var config Config
	err := viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *Config) setContributor(params *paramsT) {
	if params.repo.ContributorEmail == "" {
		params.repo.ContributorEmail = config.Email
	}

	if params.repo.ContributorName == "" {
		params.repo.ContributorName = config.Name
	}
}

func (c *Config) setRepoParams(params *paramsT) {
	c.setContributor(params)
	if params.repo.MetadataBucket == "" {
		params.repo.MetadataBucket = config.Metadata
		if params.repo.MetadataBucket == "" {
			logFatalln(fmt.Errorf("metadata bucket not set in config or as a cli param"))
		}
	}
	if params.repo.BlobBucket == "" {
		params.repo.BlobBucket = config.Blob
		if params.repo.BlobBucket == "" {
			logFatalln(fmt.Errorf("blob bucket not set in config or as a cli param"))
		}
	}
}

// configCmd represents the bundle related commands
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Commands to manage a config",
	Long: `Commands to manage datamon cli config.

Configuration for datamon is the common set of params that are needed for most commands and do not change.
`,
}

func init() {
	rootCmd.AddCommand(configCmd)
}
