// +build !influxdbintegration

package metrics

import (
	mock "github.com/oneconcern/datamon/pkg/metrics/exporters/mock"

	"go.opencensus.io/stats/view"
)

// testExporter adapts metrics sink parameters according to build tag
func testExporter(_ map[string]string) view.Exporter {
	return mock.NewExporter()
}
