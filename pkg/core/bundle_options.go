package core

import (
	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"go.uber.org/zap"
)

// BundleOption is a functor to build a bundle with some options
type BundleOption func(*Bundle)

// BundleDescriptor sets the descriptor for this bundle
func BundleDescriptor(r *model.BundleDescriptor) BundleOption {
	return func(b *Bundle) {
		if r != nil {
			b.BundleDescriptor = *r
		}
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

// BundleWithMetrics toggles metrics on a core Bundle object
func BundleWithMetrics(enabled bool) BundleOption {
	return func(b *Bundle) {
		b.EnableMetrics(enabled)
	}
}

// BundleWithRetry toggles exponential backoff retry logic on upload of core Bundle object
func BundleWithRetry(enabled bool) BundleOption {
	return func(b *Bundle) {
		b.Retry = enabled
	}
}
