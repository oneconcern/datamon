package csi

import (
	"context"
	"fmt"
	"math"

	"github.com/oneconcern/datamon/pkg/model"

	"go.uber.org/zap"

	"github.com/oneconcern/datamon/pkg/core"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/container-storage-interface/spec/lib/go/csi/v0"
)

type controllerServer struct {
	config *controllerServerConfig
}

type controllerServerConfig struct {
	driver *Driver
}

func newControllerServer(config *controllerServerConfig) csi.ControllerServer {
	return &controllerServer{
		config: config,
	}
}

func (s *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	s.config.driver.config.Logger.Info("received create volume",
		zap.String("name", req.Name))
	// Check if volume==repo is there
	repo := req.GetParameters()["repo"]
	if repo == "" {
		s.config.driver.config.Logger.Error("Repo not set", zap.String("name", req.Name))
		return nil, fmt.Errorf("repo not set for req name:%s", req.Name)
	}
	hash := req.GetParameters()["release"]
	if hash == "" {
		s.config.driver.config.Logger.Error("Release not set", zap.String("name", req.Name))
		return nil, fmt.Errorf("release not set for req name:%s", req.Name)
	}
	rd, err := core.GetRepo(repo, s.config.driver.metadataStore)
	if err != nil {
		s.config.driver.config.Logger.Error("requested repo could not be found", zap.Error(err))
		return nil, fmt.Errorf("requested volume not found. err:" + err.Error())
	}
	attrs := map[string]string{
		"repo":        repo,
		"hash":        hash,
		"timestamp":   rd.Timestamp.String(),
		"contributor": rd.Contributor.Name,
		"email":       rd.Contributor.Email,
		"description": rd.Description,
	}
	volume := csi.Volume{
		CapacityBytes: math.MaxInt64,
		Id:            repo + "-" + hash,
		Attributes:    attrs,
	}
	s.config.driver.config.Logger.Info("Volume created",
		zap.Strings("attrs", []string{volume.Attributes["repo"], volume.Attributes["hash"]}))
	return &csi.CreateVolumeResponse{
		Volume: &volume,
	}, nil
}

func (s *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	return &csi.DeleteVolumeResponse{}, nil
}

func (s *controllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	return &csi.ValidateVolumeCapabilitiesResponse{
		Supported: true,
	}, nil
}

func (s *controllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	s.config.driver.config.Logger.Sugar().Debugf("ControllerGetCapabilities: %+v", s.config.driver.controllerServiceCapabilities)
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: s.config.driver.controllerServiceCapabilities,
	}, nil
}

func (s *controllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ControllerPublishVolume unsupported")
}

func (s *controllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ControllerUnpublishVolume unsupported")
}

func (s *controllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	s.config.driver.config.Logger.Debug("ListVolumes request", zap.Int32("max", req.MaxEntries), zap.String("token", req.StartingToken))
	// skip to the token passed in.
	repos, err := core.ListReposPaginated(s.config.driver.metadataStore, req.StartingToken)
	if err != nil {
		s.config.driver.config.Logger.Error("ListVolumes: failed to list repos", zap.Error(err))
		return nil, fmt.Errorf("datamon failed to list repos, err:%v", err)
	}
	volumes, nextToken := getVolumeEntries(repos, req.MaxEntries)
	// return a list of repos as volumes.
	s.config.driver.config.Logger.Debug("ListVolumes request done", zap.Int("count", len(volumes)), zap.String("nextToken", nextToken))
	return &csi.ListVolumesResponse{
		Entries:   volumes,
		NextToken: nextToken,
	}, nil
}

func getVolumeEntries(repos []model.RepoDescriptor, max int32) ([]*csi.ListVolumesResponse_Entry, string) {
	volumes := make([]*csi.ListVolumesResponse_Entry, max)
	for _, repo := range repos {
		if max == 0 {
			return volumes, repo.Name
		}
		volume := csi.ListVolumesResponse_Entry{
			Volume: &csi.Volume{
				CapacityBytes: math.MaxInt64,
				Id:            repo.Name,
				Attributes:    getAttributes(repo),
			},
		}
		volumes = append(volumes, &volume)
		max--
	}
	return volumes, ""
}

func getAttributes(repo model.RepoDescriptor) map[string]string {

	attributes := make(map[string]string)
	attributes["description"] = repo.Description
	attributes["Contributor"] = repo.Contributor.Name
	attributes["Email"] = repo.Contributor.Email
	attributes["timestamp"] = repo.Timestamp.String()
	return attributes
}

func (s *controllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return &csi.GetCapacityResponse{
		AvailableCapacity: math.MinInt64,
	}, nil
}

func (s *controllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (s *controllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (s *controllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}
