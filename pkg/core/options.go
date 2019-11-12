package core

import "runtime"

// ListOption sets options for listing core objects
type ListOption func(*Settings)

// Settings defines various settings for core features
type Settings struct {
	concurrentList int
	batchSize      int
	doneChannel    chan struct{}
}

const (
	defaultBatchSize = 1024
)

var (
	defaultListConcurrency = 2 * runtime.NumCPU()
)

// ConcurrentList sets the max level of concurrency to retrieve core objects. It defaults to 2 x #cpus.
func ConcurrentList(concurrentList int) ListOption {
	return func(s *Settings) {
		if concurrentList == 0 {
			s.concurrentList = defaultListConcurrency
			return
		}
		s.concurrentList = concurrentList
	}
}

// BatchSize sets the batch window to fetch core objects. It defaults to defaultBatchSize
func BatchSize(batchSize int) ListOption {
	return func(s *Settings) {
		if batchSize == 0 {
			s.batchSize = defaultBatchSize
			return
		}
		s.batchSize = batchSize
	}
}

// WithDoneChan sets a signaling channel controlled by the caller to interrupt ongoing goroutines
func WithDoneChan(done chan struct{}) ListOption {
	return func(s *Settings) {
		s.doneChannel = done
	}
}

func defaultSettings() Settings {
	return Settings{
		concurrentList: defaultListConcurrency,
		batchSize:      defaultBatchSize,
	}
}
