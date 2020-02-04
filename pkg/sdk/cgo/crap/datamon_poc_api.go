package crap

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/oneconcern/datamon/cmd/datamon/cmd"
	context2 "github.com/oneconcern/datamon/pkg/context"
	gcscontext "github.com/oneconcern/datamon/pkg/context/gcs"
	"github.com/oneconcern/datamon/pkg/dlogger"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// ALL THAT IS HERE HAS TO BE SCRATCHED AND REPLACED BY A CONFIG/FLAGS COMMON API

// Config is the same as the CLI config
type Config = cmd.CLIConfig

func ParseConfigAndFlagsEtc(data []byte) (*Config, error) {
	var config Config
	err := json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config from json: %s", err)
	}
	// defaults for the POC
	if config.Config == "" {
		config.Config = "workshop-config"
	}
	if config.Credential == "" {
		config.Credential = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}
	return &config, nil
}

func SetupStoresEtc(config *Config) (storage.Store, context2.Stores, *zap.Logger, error) {
	ctx := context.Background()
	logger := dlogger.MustGetLogger("info")

	configStore, err := gcs.New(ctx, config.Config, config.Credential, gcs.Logger(logger))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get config store: %w", err)
	}

	rdr, err := configStore.Get(ctx, model.GetPathToContext("dev"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get context details from config store for context: %w ", err)
	}

	b, err := ioutil.ReadAll(rdr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read context details: %w", err)
	}

	var contextDescriptor model.Context
	err = yaml.Unmarshal(b, &contextDescriptor)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	//config.populateRemoteConfig(&datamonFlags)
	//remoteStores, err := paramsToDatamonContext(ctx, datamonFlags)

	remoteStores, err := gcscontext.MakeContext(ctx, contextDescriptor, config.Credential, gcs.Logger(logger))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create remote stores: %w", err)
	}

	return configStore, remoteStores, logger, nil
}
