package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats"
)

func TestStructTags(t *testing.T) {
	s := newSettings()
	m := &exampleMetrics{}

	scanStruct("parent", s.addMetric, m)

	assert.Nil(t, m.Telemetry.UsageCounts)   // ignored slice
	assert.Nil(t, m.Telemetry.FailureCounts) // ignored slice

	assert.NotNil(t, m.Telemetry.TestCount)
	assert.NotNil(t, m.Volumetry.Metadata.FileCount)
	assert.NotNil(t, m.Volumetry.Metadata.FileSize)
	assert.NotNil(t, m.Network.Requests.Count)
	assert.NotNil(t, m.Network.Requests.Timing)
	assert.NotNil(t, m.Network.Requests.Failures)
	assert.NotNil(t, m.Network.Requests.IOSize)

	require.NotNil(t, m.Network.Requests.IOThroughput)
	assert.IsType(t, &stats.Float64Measure{}, m.Network.Requests.IOThroughput)
	assert.Len(t, s.allMetrics, 8)
	assert.Len(t, s.allViews, 11)
}
