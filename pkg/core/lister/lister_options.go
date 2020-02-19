package lister

// Option is a functor to define er settings
type Option func(*Lister)

// WithDoneChan sets a signaling channel controlled by the caller to interrupt ongoing goroutines
func WithDoneChan(done chan struct{}) Option {
	return func(l *Lister) {
		l.doneChan = done
	}
}

// Concurrency sets the max level of concurrency to retrieve core objects. It defaults to 2 x #cpus.
func Concurrency(concurrent int) Option {
	return func(l *Lister) {
		if concurrent != 0 {
			l.concurrent = concurrent
		}
	}
}

// Checker sets a function to check for prerequisites to a listing operation, e.g. the repo exists, etc.
func Checker(checker func() error) Option {
	return func(l *Lister) {
		if checker != nil {
			l.checker = checker
		}
	}
}

// Iterator sets a function to fetch metadata keys on store
func Iterator(iterator func(string) ([]string, string, error)) Option {
	return func(l *Lister) {
		if iterator != nil {
			l.iterator = iterator
		}
	}
}

// Downloader sets a function to fetch and unmarshal a metadata descriptor
func Downloader(downloader func(string) (Listable, error)) Option {
	return func(l *Lister) {
		if downloader != nil {
			l.downloader = downloader
		}
	}
}
