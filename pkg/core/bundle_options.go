package core

import (
	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"go.uber.org/zap"
)

// BundleOption is a functor to build a bundle with some options
type BundleOption func(*Bundle)

// BundleDescriptorOption is a functor to build a bundle descriptor with some options
type BundleDescriptorOption func(descriptor *model.BundleDescriptor)

// Message defines the message of the bundle descriptor
func Message(m string) BundleDescriptorOption {
	return func(b *model.BundleDescriptor) {
		b.Message = m
	}
}

// Contributors defines the list of contributors for a bundle descriptor
func Contributors(c []model.Contributor) BundleDescriptorOption {
	return func(b *model.BundleDescriptor) {
		b.Contributors = c
	}
}

// Contributor defines a single contributor for a bundle descriptor
func Contributor(c model.Contributor) BundleDescriptorOption {
	return Contributors([]model.Contributor{c})
}

// Parents defines the parents for a bundle descriptor
func Parents(p []string) BundleDescriptorOption {
	return func(b *model.BundleDescriptor) {
		b.Parents = p
	}
}

// Deduplication defines the deduplication scheme for a bundle descriptor
func Deduplication(d string) BundleDescriptorOption {
	return func(b *model.BundleDescriptor) {
		b.Deduplication = d
	}
}

// Repo defines the repo a bundle belongs to
func Repo(r string) BundleOption {
	return func(b *Bundle) {
		b.RepoID = r
	}
}

// ConsumableStore defines the consumable storage for a bundle
func ConsumableStore(store storage.Store) BundleOption {
	return func(b *Bundle) {
		b.ConsumableStore = store
	}
}

// ContextStores defines the set of stores to build a context for a bundle
func ContextStores(cs context2.Stores) BundleOption {
	return func(b *Bundle) {
		b.contextStores = cs
	}
}

// BundleID defines the ID for a bundle
func BundleID(bID string) BundleOption {
	return func(b *Bundle) {
		b.BundleID = bID
	}
}

// Streaming sets the streaming option flag for a bundle
func Streaming(s bool) BundleOption {
	return func(b *Bundle) {
		b.Streamed = s
	}
}

// SkipMissing indicates that bundle retrieval errors should be ignored. Currently not implementated.
func SkipMissing(s bool) BundleOption {
	return func(b *Bundle) {
		b.SkipOnError = s
	}
}

// Logger injects a logging facility into bundle core operations
func Logger(l *zap.Logger) BundleOption {
	return func(b *Bundle) {
		b.l = l
	}
}

// ConcurrentFileUploads tunes the level of concurrency when uploading bundle files
func ConcurrentFileUploads(concurrentFileUploads int) BundleOption {
	return func(b *Bundle) {
		b.concurrentFileUploads = concurrentFileUploads
	}
}

// ConcurrentFileDownloads tunes the level of concurrency when downloading bundle files
func ConcurrentFileDownloads(concurrentFileDownloads int) BundleOption {
	return func(b *Bundle) {
		b.concurrentFileDownloads = concurrentFileDownloads
	}
}

// ConcurrentFilelistDownloads tunes the level of concurrency when retrieving the list of files in a bundle
func ConcurrentFilelistDownloads(concurrentFilelistDownloads int) BundleOption {
	return func(b *Bundle) {
		b.concurrentFilelistDownloads = concurrentFilelistDownloads
	}
}

// CacheSize tunes the buffer cache size in bytes of streamed FS operations (enabled when Streamed is true).
func CacheSize(size int) BundleOption {
	return func(b *Bundle) {
		b.lruSize = size
	}
}

// Prefetch enables prefetching on streamed FS operations (enabled when Streamed is true).
func Prefetch(ahead int) BundleOption {
	return func(b *Bundle) {
		b.withPrefetch = ahead
	}
}

// VerifyHash enables hash verification on streamed FS read perations (enabled when Streamed is true).
func VerifyHash(enabled bool) BundleOption {
	return func(b *Bundle) {
		b.withVerifyHash = enabled
	}
}
