package driver

import (
	"context"
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"k8s.io/kubernetes/pkg/util/mount"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	DriverName = "com.oc.cmd.datamonfuse"
)

type Driver struct {
	nodeID     string
	version    string
	driverName string

	srv *grpc.Server

	mounter *mount.SafeFormatAndMount

	volumeCaps     []csi.VolumeCapability_AccessMode
	controllerCaps []csi.ControllerServiceCapability_RPC_Type
	nodeCaps       []csi.NodeServiceCapability_RPC_Type
}

func NewDriver(nodeId, version, driverName string, mounter *mount.SafeFormatAndMount) (*Driver, error) {
	if driverName == "" {
		return nil, fmt.Errorf("driver name is missing")
	}
	if version == "" {
		return nil, fmt.Errorf("driver version is missing")
	}
	if nodeId == "" {
		return nil, fmt.Errorf("node id is missing")
	}

	if mounter == nil {
		mounter = newSafeMounter()
	}

	return &Driver{
		nodeID:     nodeId,
		version:    version,
		driverName: driverName,

		mounter: mounter,

		volumeCaps: []csi.VolumeCapability_AccessMode{
			{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
			{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
			},
			{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
			},
			{
				Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_SINGLE_WRITER,
			},
			{
				Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
			},
		},
		controllerCaps: []csi.ControllerServiceCapability_RPC_Type{
			csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
			csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		},
		nodeCaps: []csi.NodeServiceCapability_RPC_Type{
			csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
		},
	}, nil
}

func newSafeMounter() *mount.SafeFormatAndMount {
	return &mount.SafeFormatAndMount{
		Interface: mount.New(""),
		Exec:      mount.NewOsExec(),
	}
}

func (d *Driver) Run(endpoint string) error {
	if endpoint == ""{
		 return fmt.Errorf("GRPC endpoint is empty ")
	}
	scheme, addr, err := parseEndpoint(endpoint)
	if err != nil {
		return err
	}

	listener, err := net.Listen(scheme, addr)
	if err != nil {
		return err
	}

	logErr := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			glog.Errorf("GRPC error: %v", err)
		}
		return resp, err
	}
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logErr),
	}
	d.srv = grpc.NewServer(opts...)

	csi.RegisterIdentityServer(d.srv, newIdentityServer(d))
	csi.RegisterControllerServer(d.srv, newControllerServer(d))
	csi.RegisterNodeServer(d.srv, newNodeServer(d))

	glog.Infof("Listening for connections on address: %#v", listener.Addr())
	return d.srv.Serve(listener)
}

func parseEndpoint(endpoint string) (string, string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", "", fmt.Errorf("could not parse endpoint: %v", err)
	}

	addr := path.Join(u.Host, filepath.FromSlash(u.Path))

	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "tcp":
	case "unix":
		addr = path.Join("/", addr)
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			return "", "", fmt.Errorf("could not remove unix domain socket %q: %v", addr, err)
		}
	default:
		return "", "", fmt.Errorf("unsupported protocol: %s", scheme)
	}

	return scheme, addr, nil
}
