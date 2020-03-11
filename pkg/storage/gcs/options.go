package gcs

import (
	"github.com/oneconcern/datamon/pkg/storage"
	"go.uber.org/zap"
)

// Option is a functor to pass optional parameters to the gcs store
type Option func(*gcs)

// Logger specifies a logger for this store
func Logger(logger *zap.Logger) Option {
	return func(g *gcs) {
		if logger != nil {
			g.l = logger
		}
	}
}

func WithVersion(version storage.Version) Option {
	return func(g *gcs) {
		g.s.Version = version
	}
}
