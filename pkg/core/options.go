package core

import (
	"runtime"

	"github.com/oneconcern/datamon/pkg/metrics"
)

// Option sets options for listing core objects
type Option func(*Settings)

// Settings defines various settings for core features
type Settings struct {
	concurrentList   int
	batchSize        int
	doneChannel      chan struct{}
	profilingEnabled bool
	memProfDir       string

	labelPrefix   string
	labelVersions bool

	metrics.Enable
	ignoreCorruptedMetadata bool
	retainTags              bool
	retainSemverTags        bool
	// m *M // TODO(fred): enable metrics for list operations
}

const (
	defaultBatchSize = 1024
)

var (
	defaultListConcurrency = 2 * runtime.NumCPU()
)

// ConcurrentList sets the max level of concurrency to retrieve core objects. It defaults to 2 x #cpus.
func ConcurrentList(concurrentList int) Option {
	return func(s *Settings) {
		if concurrentList == 0 {
			s.concurrentList = defaultListConcurrency
			return
		}
		s.concurrentList = concurrentList
	}
}

// BatchSize sets the batch window to fetch core objects. It defaults to defaultBatchSize
func BatchSize(batchSize int) Option {
	return func(s *Settings) {
		if batchSize == 0 {
			s.batchSize = defaultBatchSize
			return
		}
		s.batchSize = batchSize
	}
}

// WithDoneChan sets a signaling channel controlled by the caller to interrupt ongoing goroutines
func WithDoneChan(done chan struct{}) Option {
	return func(s *Settings) {
		s.doneChannel = done
	}
}

// WithMemProf enables profiling and sets the memory profile directory (defaults to the current working directory).
func WithMemProf(memProfDir string) Option {
	return func(s *Settings) {
		s.profilingEnabled = true
		if memProfDir != "" {
			s.memProfDir = memProfDir
		}
	}
}

// WithMetrics toggles metrics for the core package and its dependencies (e.g. cafs)
func WithMetrics(enabled bool) Option {
	return func(s *Settings) {
		s.EnableMetrics(enabled)
	}
}

// WithLabelVersions makes ListLabels list all versions of labels (requires the vmedata bucket to enable versioning)
func WithLabelVersions(enabled bool) Option {
	return func(s *Settings) {
		s.labelVersions = enabled
	}
}

// WithLabelPrefix is an option for ListLabelsApply, to filter on labels with some given prefix
func WithLabelPrefix(prefix string) Option {
	return func(s *Settings) {
		s.labelPrefix = prefix
	}
}

func WithIgnoreCorruptedMetadata(enabled bool) Option {
	return func(s *Settings) {
		s.ignoreCorruptedMetadata = enabled
	}
}

func WithRetainTags(enabled bool) Option {
	return func(s *Settings) {
		s.retainTags = enabled
	}
}

func WithRetainSemverTags(enabled bool) Option {
	return func(s *Settings) {
		s.retainSemverTags = enabled
	}
}

func defaultSettings() Settings {
	return Settings{
		concurrentList: defaultListConcurrency,
		batchSize:      defaultBatchSize,
		memProfDir:     ".",
	}
}
