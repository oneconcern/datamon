package core

import (
	"github.com/oneconcern/datamon/pkg/metrics"
)

// M describes metrics for the core package
type M struct {
	Volume struct {
		Bundles metrics.FilesMetrics `group:"bundles" description:"metrics about datasets (bundles)"`
	} `group:"volumetry" description:""`
	Usage metrics.UsageMetrics `group:"telemetry" description:"usage stats for the core package"`

	// more metrics here
}
