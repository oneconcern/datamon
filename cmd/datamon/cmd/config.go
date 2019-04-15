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

func (c *Config) setContributor(params *RepoParams) {
	if params.ContributorEmail == "" {
		params.ContributorEmail = config.Email
		if params.ContributorEmail == "" {
			log.Fatalln(fmt.Errorf("contributor email must be set in config or as a cli param"))
		}
	}

	if params.ContributorName == "" {
		params.ContributorName = config.Name
		if params.ContributorName == "" {
			log.Fatalln(fmt.Errorf("contributor name must be set in config or as a cli param"))
		}
	}
}

func (c *Config) setRepoParams(params *RepoParams) {
	c.setContributor(params)
	if params.MetadataBucket == "" {
		params.MetadataBucket = config.Metadata
		if params.MetadataBucket == "" {
			log.Fatalln(fmt.Errorf("metadata bucket not set in config or as a cli param"))
		}
	}
	if params.BlobBucket == "" {
		params.BlobBucket = config.Blob
		if params.BlobBucket == "" {
			log.Fatalln(fmt.Errorf("blob bucket not set in config or as a cli param"))
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
