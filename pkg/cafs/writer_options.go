package cafs

import "go.uber.org/zap"

// WriterOption is a functor to provide the writer with options
type WriterOption func(writer *fsWriter)

// WriterPrefix assigns some prefix to the keys of the written objects
func WriterPrefix(prefix string) WriterOption {
	return func(writer *fsWriter) {
		writer.prefix = prefix
	}
}

// WriterConcurrentFlushes defines the degree of parallelism to flush write operations to storage
func WriterConcurrentFlushes(concurrentFlushes int) WriterOption {
	if concurrentFlushes < 1 {
		concurrentFlushes = 1
	}
	return func(writer *fsWriter) {
		writer.maxGoRoutines = make(chan struct{}, concurrentFlushes)
	}
}

// WriterLogger injects a logger in the writer
func WriterLogger(l *zap.Logger) WriterOption {
	return func(writer *fsWriter) {
		if l != nil {
			writer.l = l
		}
	}
}

// WriterPather injects some path prefixing logics in the writer
func WriterPather(fn func(Key) string) WriterOption {
	return func(writer *fsWriter) {
		if fn != nil {
			writer.pather = fn
		}
	}
}

// WriterWithMetrics enables metrics collection on this writer
func WriterWithMetrics(enabled bool) WriterOption {
	return func(writer *fsWriter) {
		if enabled {
			writer.EnableMetrics(enabled)
		}
	}
}
