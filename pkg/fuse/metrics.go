package fuse

import (
	"github.com/oneconcern/datamon/pkg/metrics"
)

// M describes metrics for the fuse package
type M struct {
	Volume struct {
		Files metrics.FilesMetrics `group:"files" description:"metrics about fuse files (mount)"`
		IO    metrics.IOMetrics    `group:"io" description:"metrics about fuse IO operations"`
	} `group:"volumetry" description:""`
	Usage metrics.UsageMetrics `group:"telemetry" description:"usage stats for the fuse package"`

	// more metrics here
}
