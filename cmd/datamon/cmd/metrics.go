package cmd

import (
	"time"

	"github.com/oneconcern/datamon/pkg/metrics"
)

type metricsFlags struct {
	Enabled *bool  `json:"enabled,omitempty" yaml:"enabled,omitempty"` // pointer because we want to distinguish unset from false
	URL     string `json:"url,omitempty" yaml:"url,omitempty"`
	m       *M
}

func (m metricsFlags) IsEnabled() bool {
	return m.Enabled != nil && *m.Enabled
}

// M describes metrics for the cmd package
type M struct {
	Usage metrics.UsageMetrics `group:"telemetry" description:"usage stats for datamon CLI"`

	// more metrics here
}

// cliUsage records a usage metric in the CLI context in a single go.
// This is intended to be used in some defer statement.
//
// Metrics are flushed as soon as the command is done.
func cliUsage(t0 time.Time, command string, err error) {
	if datamonFlags.root.metrics.IsEnabled() {
		datamonFlags.root.metrics.m.Usage.UsedAll(t0, command)(err)
		metrics.Flush()
	}
}
