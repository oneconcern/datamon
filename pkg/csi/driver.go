package csi

import (
	"fmt"

	"github.com/oneconcern/datamon/pkg/storage"

	"go.uber.org/zap"

	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"k8s.io/kubernetes/pkg/util/mount"
)

type Config struct {
	Name          string
	Version       string
	NodeID        string
	RunController bool
	RunNode       bool
	Mounter       mount.Interface
	Logger        *zap.Logger
	LocalFS       string
}

type Driver struct {
	config *Config

	ids              csi.IdentityServer
	nodeServer       csi.NodeServer
	controllerServer csi.ControllerServer

	blobStore     storage.Store
	metadataStore storage.Store

	controllerServiceCapabilities []*csi.ControllerServiceCapability
}

func NewDatamonDriver(config *Config, blobStore storage.Store, metadataStore storage.Store) (*Driver, error) {
	if config.Name == "" {
		return nil, fmt.Errorf("driver name missing")
	}
	if config.Version == "" {
		return nil, fmt.Errorf("driver version missing")
	}
	if config.NodeID == "" {
		return nil, fmt.Errorf("node id missing")
	}
	if !config.RunController && !config.RunNode {
		return nil, fmt.Errorf("must run at least one controller or node service")
	}
	if blobStore == nil {
		return nil, fmt.Errorf("must set the backing blob store")
	}
	if metadataStore == nil {
		return nil, fmt.Errorf("must set the metadata store")
	}

	driver := &Driver{
		config:        config,
		blobStore:     blobStore,
		metadataStore: metadataStore,
	}

	// Setup RPC servers
	driver.ids = newIdentityServer(driver)
	if config.RunController {
		csc := []csi.ControllerServiceCapability_RPC_Type{
			csi.ControllerServiceCapability_RPC_LIST_VOLUMES,         // List the repos
			csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME, // List the repos
		}
		driver.addControllerServiceCapabilities(csc)

		// Configure controller server
		driver.controllerServer = newControllerServer(&controllerServerConfig{
			driver: driver,
		})
	}
	if config.RunNode {
		if config.LocalFS == "" {
			return nil, fmt.Errorf("localFS to use is missing")
		}
		driver.nodeServer = newNodeServer(driver)
	}

	return driver, nil
}

func (driver *Driver) addControllerServiceCapabilities(cl []csi.ControllerServiceCapability_RPC_Type) {
	driver.config.Logger.Debug("addControllerServiceCapabilities")
	csc := make([]*csi.ControllerServiceCapability, 0)
	for _, c := range cl {
		driver.config.Logger.Info("Enabling controller service capability", zap.String("cap", c.String()))
		csc = append(csc, NewControllerServiceCapability(c))
	}
	driver.controllerServiceCapabilities = csc
	driver.config.Logger.Debug("addControllerServiceCapabilities done")
}

func NewControllerServiceCapability(cap csi.ControllerServiceCapability_RPC_Type) *csi.ControllerServiceCapability {
	return &csi.ControllerServiceCapability{
		Type: &csi.ControllerServiceCapability_Rpc{
			Rpc: &csi.ControllerServiceCapability_RPC{
				Type: cap,
			},
		},
	}
}

func (driver *Driver) Run(endpoint string) {
	driver.config.Logger.Info("Running driver", zap.String("name", driver.config.Name))

	s := NewNonBlockingGRPCServer(driver.config.Logger)
	s.Start(endpoint, driver.ids, driver.controllerServer, driver.nodeServer)
	s.Wait()
}
