package gcs

import "go.uber.org/zap"

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

// ReadOnly indicates that only the read-only client is going to be used
func ReadOnly() Option {
	return func(g *gcs) {
		g.isReadOnly = true
	}
}

// WithRetry enables exponential backoff retry logic to be enabled on put operations
func WithRetry(enabled bool) Option {
	return func(g *gcs) {
		g.retry = enabled
	}
}

// WithRetry enables exponential backoff retry logic to be enabled on put operations
func KeyPrefix(keyPrefix string) Option {
	return func(g *gcs) {
		g.keyPrefix = keyPrefix
	}
}
