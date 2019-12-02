package cmd

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	context2 "github.com/oneconcern/datamon/pkg/context"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// CLIConfig describes the CLI configuration.
type CLIConfig struct {
	// bug in viper? Need to keep names of fields the same as the serialized names..
	Credential string `json:"credential" yaml:"credential"` // Credentials to use for GCS
	Config     string `json:"config" yaml:"config"`         // Config for datamon
	Context    string `json:"context" yaml:"context"`       // Context for datamon
}

func populateRemoteConfig() {
	var flags *flagsT
	flags = &datamonFlags
	if flags.core.Config == "" {
		wrapFatalln("set environment variable $DATAMON_GLOBAL_CONFIG or create config file", nil)
		return
	}
	configStore, err := gcs.New(context.Background(), flags.core.Config, config.Credential)
	if err != nil {
		wrapFatalln("failed to get context details", err)
		return
	}
	contextName := flags.context.Name
	if contextName == "" {
		contextName = config.Context
	}
	rdr, err := configStore.Get(context.Background(), model.GetPathToContext(contextName))
	if err != nil {
		wrapFatalln("failed to get context details from config store for "+flags.context.Descriptor.Name, err)
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

	paramsToDatamonContext = func(ctx context.Context) (context2.Stores, error) {
		stores := context2.Stores{}
		meta, err := gcs.New(ctx, contextDescriptor.Metadata, config.Credential)
		if err != nil {
			return context2.Stores{}, fmt.Errorf("failed to initialize metadata store, err:%s", err)
		}
		stores.SetMetadata(meta)
		blob, err := gcs.New(ctx, contextDescriptor.Blob, config.Credential)
		if err != nil {
			return context2.Stores{}, fmt.Errorf("failed to initialize blob store, err:%s", err)
		}
		stores.SetBlob(blob)
		v, err := gcs.New(ctx, contextDescriptor.VMetadata, config.Credential)
		if err != nil {
			return context2.Stores{}, fmt.Errorf("failed to initialize vmetadata store, err:%s", err)
		}
		stores.SetVMetadata(v)
		w, err := gcs.New(ctx, contextDescriptor.WAL, config.Credential)
		if err != nil {
			return context2.Stores{}, fmt.Errorf("failed to initialize wal store, err:%s", err)
		}
		stores.SetWal(w)
		r, err := gcs.New(ctx, contextDescriptor.ReadLog, config.Credential)
		if err != nil {
			return context2.Stores{}, fmt.Errorf("failed to initialize read log store, err:%s", err)
		}
		stores.SetReadLog(r)
		return stores, nil
	}

}

// configCmd represents the bundle related commands
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Commands to manage a config",
	Long: `Commands to manage datamon cli config.

Configuration for datamon is the common set of datamonFlags that are needed for most commands and do not change.
`,
}

func init() {
	rootCmd.AddCommand(configCmd)
}
