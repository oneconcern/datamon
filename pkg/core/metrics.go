package core

import (
	"github.com/oneconcern/datamon/pkg/metrics"
	"go.opencensus.io/stats"
)

// M describes metrics for the core package
type M struct {
	Volume struct {
		Bundles metrics.FilesMetrics `group:"bundles" description:"metrics about datasets (bundles)"`
		IO      BundleIOMetrics      `group:"io" description:"metrics about bundle IO operations"`
	} `group:"volumetry" description:""`
	Usage metrics.UsageMetrics `group:"telemetry" description:"usage stats for the core package"`

	// more metrics here
}

// BundleIOMetrics extends IOMetrics to capture bundle file count and metadata index files count
type BundleIOMetrics struct {
	metrics.IOMetrics
	FileCount  *stats.Int64Measure `metric:"fileCount" description:"number of files in a bundle" tags:"kind,operation"`
	IndexCount *stats.Int64Measure `metric:"indexCount" description:"number of metadata index files in a bundle" tags:"kind,operation"`
}

// BundleFiles recors metrics about files in a bundle
func (b *BundleIOMetrics) BundleFiles(files, indices int64, operation string) {
	tags := map[string]string{"kind": "io", "operation": operation}
	metrics.Int64(b.FileCount, files, tags)
	metrics.Int64(b.IndexCount, indices, tags)
}
