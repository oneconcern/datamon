package metrics

import (
	"context"
	"time"

	"go.opencensus.io/stats/view"
)

// Option defines some options to the metrics initialization
type Option func(*settings)

// WithBasePath defines the root for the registered metrics tree
func WithBasePath(location string) Option {
	return func(m *settings) {
		m.basePath = location
	}
}

// WithContexter sets a context generation function. The default contexter is context.Background
func WithContexter(c func() context.Context) Option {
	return func(m *settings) {
		if c != nil {
			m.contexter = c
		}
	}
}

// WithExporter configures the exporter to convey metrics to some backend collector
func WithExporter(exporter view.Exporter) Option {
	return func(m *settings) {
		if exporter != nil {
			m.exporter = flusher(exporter)
		}
	}
}

// WithReportingPeriod configures how often the exporter is going to upload metrics.
// Durations under 1 sec do not have any effect. The default is 10s.
func WithReportingPeriod(d time.Duration) Option {
	return func(m *settings) {
		m.d = d
	}
}
