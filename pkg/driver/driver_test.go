package driver

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func initTestDriver(t *testing.T) *Driver {

	driver, err := NewDriver(testNodeId, testVersion, testDriver, nil)
	if err != nil {
		require.NoError(t, err)
	}

	if driver == nil {
		t.Fatalf("driver is nil")
	}
	return driver
}
