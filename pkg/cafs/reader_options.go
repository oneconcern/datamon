package cafs

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"go.uber.org/zap"
)

// ReaderOption is a functor to provide the reader with options
type ReaderOption func(reader *chunkReader)

// TruncateLeaf enables buffer truncation on WriteTo operations.
//
// This option is used to work around a bug in blob prior to bundle model V1.
//
// Older blobs uploaded to the blobs bucket are less than leaf size by 32k.
// This has been fixed but while downloading, this needs to be detected and
// truncation logic has to be applied.
//
// The WriteTo staging buffer size corresponds to the
// size of the internal temporary buffer used by io.Copy.
func TruncateLeaf(t bool) ReaderOption {
	return func(reader *chunkReader) {
		reader.leafTruncation = t
	}
}

// Keys sets the hash keys to be read from the store
func Keys(keys []Key) ReaderOption {
	return func(reader *chunkReader) {
		reader.keys = keys
	}
}

// ReaderVerifyHash enables checksum verification of blob objects in store.
func ReaderVerifyHash(t bool) ReaderOption {
	return func(reader *chunkReader) {
		reader.withVerifyHash = t
	}
}

// SetCache sets the LRU buffer cache for read pages.
// TODO(fred): nice - should externalize all this caching thing with an interface
func SetCache(lru *lru.Cache, latch sync.Locker) ReaderOption {
	return func(reader *chunkReader) {
		reader.lru = lru
		reader.lruLatch = latch
	}
}

// ConcurrentChunkWrites sets the number of parallel write operations allowed. Defaults to 1.
func ConcurrentChunkWrites(concurrentChunkWrites int) ReaderOption {
	return func(reader *chunkReader) {
		reader.concurrentChunkWrites = concurrentChunkWrites
	}
}

// SetLeafPool provides a memory pool to the reader.
//
// If none is provided, buffers are allocated and relinquished to the garbage collector.
func SetLeafPool(leafPool FreeList) ReaderOption {
	return func(reader *chunkReader) {
		reader.leafPool = leafPool
	}
}

// ReaderLogger overrides the default logger for this reader
func ReaderLogger(l *zap.Logger) ReaderOption {
	return func(reader *chunkReader) {
		if l != nil {
			reader.l = l
		}
	}
}

// ReaderPrefetch enables prefetching of that many leaf blobs ahead of the current one
func ReaderPrefetch(ahead int) ReaderOption {
	return func(reader *chunkReader) {
		if ahead >= 0 {
			reader.maxFetchAhead = ahead
		}
	}
}

// ReaderPrefix sets a prefix for the keys used by this reader
func ReaderPrefix(prefix string) ReaderOption {
	return func(reader *chunkReader) {
		reader.prefix = prefix
	}
}

// ReaderPather injects some path prefixing logics in the reader
func ReaderPather(fn func(Key) string) ReaderOption {
	return func(reader *chunkReader) {
		if fn != nil {
			reader.pather = fn
		}
	}
}
