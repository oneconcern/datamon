package csi

import (
	"context"
	"os"
	"sync"

	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"

	"github.com/oneconcern/datamon/pkg/core"

	"github.com/oneconcern/datamon/pkg/storage"

	"go.uber.org/zap"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/container-storage-interface/spec/lib/go/csi/v0"
)

type downloadedBundle struct {
	repo     string
	bundleID string
	path     string
	refCount int
	mounted  bool
	fs       *core.ReadOnlyFS
}

type nodeServer struct {
	l         *zap.Logger
	meta      storage.Store
	blob      storage.Store
	localFS   string
	bundleMap map[string]*downloadedBundle
	lock      sync.Mutex
	driver    *Driver
}

func newNodeServer(driver *Driver) *nodeServer {
	return &nodeServer{
		driver:    driver,
		l:         driver.config.Logger,
		meta:      driver.metadataStore,
		blob:      driver.blobStore,
		localFS:   driver.config.LocalFS,
		bundleMap: make(map[string]*downloadedBundle),
		lock:      sync.Mutex{},
	}
}

func (n *nodeServer) NodeStageVolume(context context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	repo, ok := req.VolumeAttributes["repo"]
	if !ok {
		n.l.Error("repo not set for volume", zap.String("req", req.String()))
		return nil, status.Error(codes.InvalidArgument, "datamon repo not set, req="+req.String())
	}
	bundle, ok := req.VolumeAttributes["hash"]
	if !ok {
		n.l.Info("latest commit for main branch", zap.String("repo", repo), zap.String("req", req.String()))
	}

	n.l.Info("Stage volume done",
		zap.String("volume", req.VolumeId),
		zap.String("repo", repo),
		zap.String("bundle", bundle))
	return &csi.NodeStageVolumeResponse{}, nil
}

func (n *nodeServer) prepBundle(repo string, bundle string, volumeID string) error {
	// Check if the bundle has been downloaded, if not download it.
	_, ok := n.bundleMap[volumeID]
	if !ok {
		path := getPathToLocalFS(n.localFS, bundle, repo)
		err := os.MkdirAll(path, 0777)
		if err != nil {
			n.l.Error("failed to create mount path", zap.Error(err))
			return err
		}
		localFS := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), path))
		bd := core.NewBDescriptor()
		b := core.New(bd,
			core.Repo(repo),
			core.BundleID(bundle),
			core.BlobStore(n.blob),
			core.ConsumableStore(localFS),
			core.MetaStore(n.meta),
		)
		fs, err := core.NewReadOnlyFS(b, n.l)
		if err != nil {
			n.l.Error("failed to initialize bundle",
				zap.String("repo", repo),
				zap.String("bundle", bundle),
				zap.Error(err))
			return status.Error(codes.Internal, "failed to initialize repo:bundle "+repo+":"+bundle)
		}
		downloadedBundle := downloadedBundle{
			repo:     repo,
			bundleID: bundle,
			path:     "", // TODO
			refCount: 1,
			fs:       fs,
			mounted:  false,
		}

		n.bundleMap[volumeID] = &downloadedBundle
		n.l.Info("volume ready to be published",
			zap.String("volumeID", volumeID),
			zap.String("repo", repo),
			zap.String("bundle", bundle))
	}
	n.l.Info("Prep Bundle finished")
	return nil
}

func (n *nodeServer) NodeUnstageVolume(context.Context, *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (n *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	n.lock.Lock()
	defer n.lock.Unlock()

	repo, ok := req.VolumeAttributes["repo"]
	if !ok {
		n.l.Error("repo not set for volume", zap.String("req", req.String()))
		return nil, status.Error(codes.InvalidArgument, "repo not set for volume seen first time")
	}
	bundle, ok := req.VolumeAttributes["hash"]
	if !ok {
		n.l.Info("latest commit for main branch", zap.String("repo", repo), zap.String("req", req.String()))
	}

	downloadedBundle, ok := n.bundleMap[req.VolumeId]
	if !ok {
		err := n.prepBundle(repo, bundle, req.VolumeId)
		if err != nil {
			return nil, err
		}
	} else {
		n.l.Info("mounting an existing bundle", zap.String("req", req.String()))
	}
	downloadedBundle, ok = n.bundleMap[req.VolumeId]
	if !ok {
		return nil, status.Error(codes.Internal, "fsMap missing entry: "+req.String())
	}
	if !downloadedBundle.mounted {
		err := downloadedBundle.fs.MountReadOnly(req.TargetPath)
		if err != nil {
			return nil, err
		}
		downloadedBundle.mounted = true
	}
	downloadedBundle.refCount++
	n.l.Info("Publish volume done",
		zap.String("volume", req.VolumeId),
		zap.String("id", downloadedBundle.bundleID),
		zap.String("targetPath", req.TargetPath))
	return &csi.NodePublishVolumeResponse{}, nil
}

func getPathToLocalFS(basePath string, repo string, bundle string) string {
	path := basePath + "/" + repo + "/" + bundle
	return path
}

func (n *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	n.lock.Lock()
	defer n.lock.Unlock()
	downloadedBundle, ok := n.bundleMap[req.VolumeId]
	if ok {
		n.l.Info("unMounting",
			zap.String("volumeId", req.VolumeId),
			zap.String("targetPath", req.TargetPath),
			zap.String("repo", downloadedBundle.repo),
			zap.String("bundleID", downloadedBundle.bundleID),
		)
		err := downloadedBundle.fs.Unmount(req.TargetPath)
		if err != nil {
			n.l.Error("Failed to unmount",
				zap.Error(err),
			)
			return nil, err
		}
		// last mount
		if downloadedBundle.refCount == 1 {
			path := getPathToLocalFS(n.localFS, downloadedBundle.bundleID, downloadedBundle.repo)
			err = os.RemoveAll(path)
			if err != nil {
				n.l.Error("failed to remove staging folder",
					zap.String("path", path),
					zap.String("repo", downloadedBundle.repo),
					zap.String("bundle", downloadedBundle.bundleID),
				)
				return nil, err
			}
			delete(n.bundleMap, req.VolumeId)
		} else {
			downloadedBundle.refCount--
		}
	} else {
		n.l.Warn("NodeUnpublishVolume for non mounted FS",
			zap.String("volumeId", req.VolumeId),
			zap.String("targetPath", req.TargetPath),
		)
	}
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (n *nodeServer) NodeGetId(ctx context.Context, req *csi.NodeGetIdRequest) (*csi.NodeGetIdResponse, error) {
	return &csi.NodeGetIdResponse{
		NodeId: n.driver.config.NodeID,
	}, nil
}

func (n *nodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{
		NodeId: n.driver.config.NodeID,
	}, nil
}

func (n *nodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	cap := csi.NodeServiceCapability{
		Type: &csi.NodeServiceCapability_Rpc{
			Rpc: &csi.NodeServiceCapability_RPC{
				Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
			},
		},
	}
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{&cap},
	}, nil
}
