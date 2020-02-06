package core

import (
	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/model"
	"go.uber.org/zap"
)

// DiamondOption is a functor to build a diamond with some options.
//
// Since Diamond extends bundle, it gets most of its options.
type DiamondOption func(*Diamond)

// DiamondDescriptor defines the descriptor of the diamond
func DiamondDescriptor(desc *model.DiamondDescriptor) DiamondOption {
	return func(b *Diamond) {
		if desc != nil {
			b.DiamondDescriptor = *desc
		}
	}
}

// DiamondRepo defines the repo a diamond belongs to
func DiamondRepo(r string) DiamondOption {
	return func(b *Diamond) {
		b.RepoID = r
	}
}

// DiamondContextStores defines the set of stores to build a context for a diamond
func DiamondContextStores(cs context2.Stores) DiamondOption {
	return func(b *Diamond) {
		b.contextStores = cs
	}
}

// DiamondLogger injects a logging facility into diamond core operations
func DiamondLogger(l *zap.Logger) DiamondOption {
	return func(b *Diamond) {
		b.l = l
	}
}

// DiamondConcurrentFileUploads tunes the level of concurrency when uploading diamond files
func DiamondConcurrentFileUploads(concurrentFileUploads int) DiamondOption {
	return func(b *Diamond) {
		b.concurrentFileUploads = concurrentFileUploads
	}
}

// DiamondConcurrentFileDownloads tunes the level of concurrency when downloading diamond files
func DiamondConcurrentFileDownloads(concurrentFileDownloads int) DiamondOption {
	return func(b *Diamond) {
		b.concurrentFileDownloads = concurrentFileDownloads
	}
}

// DiamondConcurrentFilelistDownloads tunes the level of concurrency when retrieving the list of files in a diamond
func DiamondConcurrentFilelistDownloads(concurrentFilelistDownloads int) DiamondOption {
	return func(b *Diamond) {
		b.concurrentFilelistDownloads = concurrentFilelistDownloads
	}
}

// DiamondMessage defines the message of the bundle descriptor created by the diamond
func DiamondMessage(m string) DiamondOption {
	return func(b *Diamond) {
		b.BundleDescriptor.Message = m
	}
}

// DiamondBundleID defines the ID for the bundle produced by the diamond
//
// NOTE: this require the package to be built with the bundle_preserve tag to enable this feature
func DiamondBundleID(bundleID string) DiamondOption {
	return func(b *Diamond) {
		if enableBundlePreserve && bundleID != "" {
			b.setBundleID(bundleID)
		}
	}
}
