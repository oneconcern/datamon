package metrics

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixtureRequires(t testing.TB, m *exampleMetrics) {
	require.NotNil(t, m.Telemetry.TestCount)
	require.NotNil(t, m.Volumetry.Metadata.FileCount)
	require.NotNil(t, m.Network.Requests.Count)
}

func exerciseAPI(t testing.TB, m *exampleMetrics) {
	Inc(m.Telemetry.TestCount)
	Inc(m.Volumetry.Metadata.FileCount)
	Int64(m.Network.Requests.Count, 10)
}

func TestMetrics(t *testing.T) {
	testMetrics := &exampleMetrics{}
	Init(
		WithExporter(testExporter(map[string]string{"testing": "testingvalue"})),
	)
	_ = EnsureMetrics("example", testMetrics)

	fixtureRequires(t, testMetrics)

	exerciseAPI(t, testMetrics)
}

func TestRegister(t *testing.T) {
	testMetrics := &exampleMetrics{}
	Init(
		WithExporter(testExporter(map[string]string{"registerTesting": "testingvalue"})),
	)

	// lazy registration
	x := EnsureMetrics("registerExample", testMetrics)
	fixtureRequires(t, testMetrics)
	exerciseAPI(t, testMetrics)

	// retry registration
	y := EnsureMetrics("registerExample", testMetrics)
	require.Equal(t, x, y)
}

func TestModules(t *testing.T) {
	s := newSettings(
		WithBasePath("root"),
		WithExporter(testExporter(map[string]string{"author": "fred", "run": "test"})),
	)
	testMetrics := &exampleMetrics{}
	_ = s.EnsureMetrics("moduleTesting", testMetrics)

	require.Len(t, s.modules, 1)
	assert.Len(t, s.allMetrics, 8)
	assert.Len(t, s.allViews, 11)

	fixtureRequires(t, testMetrics)
	mp = s

	// helper object level API
	t0 := time.Now()

	testMetrics.IncTest()

	testMetrics.Network.Requests.IO(time.Now(), "read")
	testMetrics.Network.Requests.Size(100, "write")
	testMetrics.Network.Requests.Failed("delete")
	testMetrics.Network.Requests.Throughput(t0, time.Now(), 100, "read")
	testMetrics.Network.Requests.Throughput(t0, t0, 100, "nop")
	testMetrics.Network.Requests.Throughput(t0, t0, 0, "nop")

	testMetrics.Network.Requests.IORecord(t0, "nop")(0, nil)
	testMetrics.Network.Requests.IORecord(t0, "read")(100, nil)
	testMetrics.Network.Requests.IORecord(t0, "error")(0, fmt.Errorf("failure"))
	testMetrics.Network.Requests.IORecord(t0, "write")(100, nil)

	testMetrics.Volumetry.Metadata.Inc("read")
	testMetrics.Volumetry.Metadata.Size(100, "write")
	testMetrics.Volumetry.Metadata.Size(0, "download")

	s.Flush()
}
