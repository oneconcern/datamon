package cafs

import (
	"github.com/oneconcern/datamon/pkg/storage"
	"go.uber.org/zap"
)

type hasOpts struct {
	OnlyRoots, GatherIncomplete bool
	_                           struct{} // disallow unkeyed usage
}

// HasOption is a functor used to set some keying options for the Has operation on this FS
type HasOption func(*hasOpts)

func HasOnlyRoots() HasOption {
	return func(opts *hasOpts) {
		opts.OnlyRoots = true
	}
}

func HasGatherIncomplete() HasOption {
	return func(opts *hasOpts) {
		opts.OnlyRoots = true
		opts.GatherIncomplete = true
	}
}

// Option to configure content addressable FS components
type Option func(*defaultFs)

// LeafSize specifies the leaf size used to split blobs and compute key hashes
func LeafSize(sz uint32) Option {
	return func(w *defaultFs) {
		w.leafSize = sz
	}
}

func LeafTruncation(a bool) Option {
	return func(w *defaultFs) {
		w.leafTruncation = a
	}
}

// Prefix sets a prefix on keys
func Prefix(prefix string) Option {
	return func(w *defaultFs) {
		w.prefix = prefix
	}
}

// Backend specifies the backend store
func Backend(store storage.Store) Option {
	return func(w *defaultFs) {
		w.store.backend = store
	}
}

func ConcurrentFlushes(concurrentFlushes int) Option {
	return func(w *defaultFs) {
		w.concurrentFlushes = concurrentFlushes
	}
}

func ReaderConcurrentChunkWrites(readerConcurrentChunkWrites int) Option {
	return func(w *defaultFs) {
		w.readerConcurrentChunkWrites = readerConcurrentChunkWrites
	}
}

// Logger sets a logger for this store
func Logger(l *zap.Logger) Option {
	return func(w *defaultFs) {
		w.l = l
	}
}

// CacheSize sets the target size of the LRU buffer cache in bytes
func CacheSize(size int) Option {
	return func(w *defaultFs) {
		if size < 1 {
			size = DefaultCacheSize
		}
		w.lruSize = size
	}
}

// Prefetch enables prefetching on read operations
func Prefetch(ahead int) Option {
	return func(w *defaultFs) {
		w.withPrefetch = ahead
	}
}

// VerifyHash enables hash verification on blob objects
func VerifyHash(enabled bool) Option {
	return func(w *defaultFs) {
		w.withVerifyHash = enabled
	}
}

// KeysCacheSize sets the size of the LRU cache for root keys in number of keys
func KeysCacheSize(keys int) Option {
	return func(w *defaultFs) {
		if keys > 0 {
			w.keysCacheSize = keys
		}
	}
}
