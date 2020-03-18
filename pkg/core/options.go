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

	prefix string
	label  string

	metrics.Enable
	//m *M // TODO(fred): enable metrics for list operations
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
// Currently, extra
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

// WithLabel is an option for ListLabelsApply along with WithPrefix.
// Precisely one of these options is expected to be applied.
func WithLabel(label string) Option {
	return func(s *Settings) {
		s.label = label
	}
}

// WithPrefix is an option for ListLabelsApply along with WithLabel.
// Precisely one of these options is expected to be applied.
func WithPrefix(prefix string) Option {
	return func(s *Settings) {
		s.prefix = prefix
	}
}

func defaultSettings() Settings {
	return Settings{
		concurrentList: defaultListConcurrency,
		batchSize:      defaultBatchSize,
		memProfDir:     ".",
	}
}
