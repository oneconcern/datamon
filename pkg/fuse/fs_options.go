package fuse

import (
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseutil"
	"go.uber.org/zap"
)

// Option for the file system
type Option func(fuseutil.FileSystem)

// Logger for this file system
func Logger(l *zap.Logger) Option {
	return func(mfs fuseutil.FileSystem) {
		if l == nil {
			return
		}
		switch fs := mfs.(type) {
		case *readOnlyFsInternal:
			fs.l = l
		case *fsMutable:
			fs.l = l
		}
	}
}

// Streaming sets the streaming option flag for a bundle (RO mount only)
func Streaming(s bool) Option {
	return func(mfs fuseutil.FileSystem) {
		if fs, ok := mfs.(*readOnlyFsInternal); ok {
			fs.streamed = s
		}
	}
}

// CacheSize tunes the buffer cache size in bytes of streamed FS operations (enabled when Streamed is true).
func CacheSize(size int) Option {
	return func(mfs fuseutil.FileSystem) {
		if fs, ok := mfs.(*readOnlyFsInternal); ok {
			fs.lruSize = size
		}
	}
}

// Prefetch enables prefetching on streamed FS operations (enabled when Streamed is true).
func Prefetch(ahead int) Option {
	return func(mfs fuseutil.FileSystem) {
		if fs, ok := mfs.(*readOnlyFsInternal); ok {
			fs.prefetch = ahead
		}
	}
}

// VerifyHash enables hash verification on streamed FS read perations (enabled when Streamed is true).
func VerifyHash(enabled bool) Option {
	return func(mfs fuseutil.FileSystem) {
		if fs, ok := mfs.(*readOnlyFsInternal); ok {
			fs.withVerifyHash = enabled
		}
	}
}

// WithMetrics toggles metrics on the fuse package
func WithMetrics(enabled bool) Option {
	return func(mfs fuseutil.FileSystem) {
		switch fs := mfs.(type) {
		case *readOnlyFsInternal:
			fs.EnableMetrics(enabled)
		case *fsMutable:
			fs.EnableMetrics(enabled)
		}
	}
}

// MountOption enables options when mounting the file system
//
// TODO plumb additional mount options
type MountOption func(*fuse.MountConfig)
