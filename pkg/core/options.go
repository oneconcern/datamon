package core

import (
	"github.com/oneconcern/datamon/pkg/metrics"
)

const (
	defaultBatchSize = 1024
)

// Option sets options for listing core objects
type Option func(*settings)

// Settings defines various settings for core features
type settings struct {
	profilingEnabled bool
	memProfDir       string

	batchSize      int
	concurrentList int

	metrics.Enable
	//m *M // TODO(fred): enable metrics for list operations
}

func defaultSettings() *settings {
	return &settings{
		memProfDir: ".",
		batchSize:  defaultBatchSize,
	}
}

func newSettings(opts ...Option) *settings {
	s := defaultSettings()
	for _, apply := range opts {
		apply(s)
	}
	return s
}

// WithMemProf enables profiling and sets the memory profile directory (defaults to the current working directory).
// Currently, extra
func WithMemProf(memProfDir string) Option {
	return func(s *settings) {
		s.profilingEnabled = true
		if memProfDir != "" {
			s.memProfDir = memProfDir
		}
	}
}

// WithMetrics toggles metrics for the core package and its dependencies (e.g. cafs)
func WithMetrics(enabled bool) Option {
	return func(s *settings) {
		s.EnableMetrics(enabled)
	}
}

// BatchSize sets the batch window to fetch lists of core objects. It defaults to defaultBatchSize
func BatchSize(batchSize int) Option {
	return func(s *settings) {
		if batchSize != 0 {
			s.batchSize = batchSize
		}
	}
}

// ConcurrentList sets the max level of concurrency to retrieve lists of core objects
func ConcurrentList(concurrent int) Option {
	return func(s *settings) {
		if concurrent != 0 {
			s.concurrentList = concurrent
		}
	}
}
