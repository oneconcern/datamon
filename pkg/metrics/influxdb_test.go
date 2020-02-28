// +build influxdbintegration

package metrics

import (
	"github.com/oneconcern/datamon/pkg/metrics/exporters/influxdb"

	"go.opencensus.io/stats/view"
)

// testExporter adapts metrics sink parameters according to build tag
func testExporter(tags map[string]string) view.Exporter {
	return DefaultExporter(influxdb.WithTags(tags))
}
