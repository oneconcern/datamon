package driver

import (
	"context"
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/kubernetes/pkg/util/mount"
	"log"
	"os"
	"os/exec"
)

type nodeServer struct {
	driver *Driver
}

func newNodeServer(driver *Driver) csi.NodeServer {
	return &nodeServer{driver: driver}
}

func (d *nodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	log.Printf("NodeStageVolume: called with args %+v", *req)

	volumeID := req.GetVolumeId()

	stagingTargetPath := req.GetStagingTargetPath()

	// Check arguments
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	if req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Volume Capability must be provided")
	}

	log.Printf("NodeStageVolume: checking mount on stage target path %s ", stagingTargetPath)
	notMnt, err := checkMount(stagingTargetPath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !notMnt {
		return &csi.NodeStageVolumeResponse{}, nil
	}

	log.Printf("NodeStageVolume: executing GCSFuse command")
	//out, err := exec.Command("gcsfuse", volumeID, stagingTargetPath).Output()
	out, err := exec.Command("gcsfuse", "onec-gcsfuse", stagingTargetPath).Output()
	if err != nil {
		log.Printf("NodeStageVolume: gcsfuse error %v ", err)
		log.Printf("NodeStageVolume: Outout gcsfuse %s ", string(out))
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Printf("NodeStageVolume: gcsfuse command run output %s ", out)

	return &csi.NodeStageVolumeResponse{}, nil

}

func (d *nodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	log.Printf("NodeUnstageVolume: called with args %+v", *req)
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	target := req.GetStagingTargetPath()
	if len(target) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Staging target not provided")
	}

	log.Printf("NodeUnstageVolume: unmounting %s", target)
	err := d.driver.mounter.Interface.Unmount(target)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmount target %q: %v", target, err)
	}

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (d *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	log.Printf("NodePublishVolume: called with args %+v", *req)
	volumeID := req.GetVolumeId()
	targetPath := req.GetTargetPath()
	stagingTargetPath := req.GetStagingTargetPath()

	// Check arguments
	if req.GetVolumeCapability() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability missing in request")
	}
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Staging Target path missing in request")
	}
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	options := []string{"bind"}
	if req.GetReadonly() {
		options = append(options, "ro")
	}

	log.Printf("NodePublishVolume: creating dir %s", targetPath)
	if err := d.driver.mounter.Interface.MakeDir(targetPath); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not create dir %q: %v", targetPath, err)
	}

	log.Printf("NodePublishVolume: mounting %s at %s", stagingTargetPath, targetPath)
	if err := d.driver.mounter.Interface.Mount(stagingTargetPath, targetPath, "ext4", options); err != nil {
		os.Remove(targetPath)
		return nil, status.Errorf(codes.Internal, "Could not mount %q at %q: %v", stagingTargetPath, targetPath, err)
	}

	log.Printf("NodePublishVolume: bucket %s successfuly mounted to %s", volumeID, targetPath)

	return &csi.NodePublishVolumeResponse{}, nil
}

func (d *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	log.Printf("NodeUnpublishVolume: called with args %+v", *req)
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	target := req.GetTargetPath()
	if len(target) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path not provided")
	}

	log.Printf("NodeUnpublishVolume: unmounting %s", target)
	err := d.driver.mounter.Interface.Unmount(target)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmount %q: %v", target, err)
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (d *nodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	log.Printf("NodeGetCapabilities: called with args %+v", *req)
	var caps []*csi.NodeServiceCapability
	for _, cap := range d.driver.nodeCaps {
		c := &csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: cap,
				},
			},
		}
		caps = append(caps, c)
	}
	return &csi.NodeGetCapabilitiesResponse{Capabilities: caps}, nil
}

func (d *nodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	log.Printf("NodeGetInfo: called with args %+v", *req)

	return &csi.NodeGetInfoResponse{
		NodeId: d.driver.nodeID,

		// make sure that the driver works on this particular region only
		AccessibleTopology: &csi.Topology{
			Segments: map[string]string{
				"region": "us-west-2",
			},
		},
	}, nil
	return &csi.NodeGetInfoResponse{}, nil
}

func (d *nodeServer) NodeGetId(ctx context.Context, req *csi.NodeGetIdRequest) (*csi.NodeGetIdResponse, error) {
	log.Printf("NodeGetId: called with args %+v", *req)
	return &csi.NodeGetIdResponse{
		NodeId: d.driver.nodeID,
	}, nil
}

func checkMount(targetPath string) (bool, error) {
	notMnt, err := mount.New("").IsLikelyNotMountPoint(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(targetPath, 0750); err != nil {
				return false, err
			}
			notMnt = true
		} else {
			return false, err
		}
	}
	return notMnt, nil
}
