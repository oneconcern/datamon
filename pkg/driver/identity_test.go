package driver

import (
	"context"
	"testing"

	csi "github.com/container-storage-interface/spec/lib/go/csi/v0"
)

const (
	testDriver  = "test-driver"
	testVersion = "test-version"
	testNodeId  = "test-node-id"
)

func initTestIdentityServer(t *testing.T) csi.IdentityServer {
	return newIdentityServer(initTestDriver(t))
}

func TestGetPluginInfo(t *testing.T) {
	s := initTestIdentityServer(t)

	resp, err := s.GetPluginInfo(context.TODO(), &csi.GetPluginInfoRequest{})
	if err != nil {
		t.Fatalf("GetPluginInfo failed: %v", err)
	}

	if resp == nil {
		t.Fatalf("GetPluginInfo resp is nil")
	}

	if resp.Name != testDriver {
		t.Errorf("got driver name %v", resp.Name)
	}

	if resp.VendorVersion != testVersion {
		t.Errorf("got driver version %v", resp.VendorVersion)
	}
}

func TestGetPluginCapabilities(t *testing.T) {
	s := initTestIdentityServer(t)

	resp, err := s.GetPluginCapabilities(context.TODO(), &csi.GetPluginCapabilitiesRequest{})
	if err != nil {
		t.Fatalf("GetPluginCapabilities failed: %v", err)
	}

	if resp == nil {
		t.Fatalf("GetPluginCapabilities resp is nil")
	}

	if len(resp.Capabilities) != 2 {
		t.Fatalf("returned %v capabilities", len(resp.Capabilities))
	}

	if resp.Capabilities[0].Type == nil {
		t.Fatalf("returned nil capability type")
	}

	service := resp.Capabilities[0].GetService()
	if service == nil {
		t.Fatalf("returned nil capability service")
	}

	if serviceType := service.GetType(); serviceType != csi.PluginCapability_Service_CONTROLLER_SERVICE {
		t.Fatalf("returned %v capability service", serviceType)
	}
}

func TestProbe(t *testing.T) {
	s := initTestIdentityServer(t)

	resp, err := s.Probe(context.TODO(), &csi.ProbeRequest{})
	if err != nil {
		t.Fatalf("Probe failed: %v", err)
	}

	if resp == nil {
		t.Fatalf("Probe resp is nil")
	}
}
