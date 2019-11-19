package core

import (
	"context"

	"go.uber.org/zap"

	"github.com/oneconcern/datamon/pkg/cafs"

	"github.com/jacobsa/fuse"
)

func errNotEOF(err error) bool {
	return err != nil && err.Error() != "EOF"
}

// ReadAt reads some bundle data with an optional offset. Streamed bundles ignore the offset.
func (b *Bundle) ReadAt(file *fsEntry, destination []byte, offset int64) (int, error) {
	if !b.Streamed {
		reader, err := b.ConsumableStore.GetAt(context.Background(), file.fullPath)
		if err != nil {
			return 0, fuse.EIO
		}

		n, err := reader.ReadAt(destination, offset)
		if errNotEOF(err) {
			return n, fuse.EIO
		}

		return n, nil
	}

	key, err := cafs.KeyFromString(file.hash)
	if err != nil {
		b.l.Error("failed to create cafs key",
			zap.String("key", file.hash),
			zap.String("bundleID", b.BundleID),
			zap.String("repo", b.RepoID),
			zap.String("file", file.fullPath),
			zap.Error(err),
		)
		return 0, err
	}

	reader, err := b.cafs.GetAt(context.Background(), key)
	if err != nil {
		b.l.Error("filed to getAt",
			zap.String("key", file.hash),
			zap.String("bundleID", b.BundleID),
			zap.String("repo", b.RepoID),
			zap.String("file", file.fullPath),
			zap.Error(err),
		)
		return 0, err
	}

	n, err := reader.ReadAt(destination, offset)
	if errNotEOF(err) {
		b.l.Error("filed to readAt",
			zap.String("key", file.hash),
			zap.String("bundleID", b.BundleID),
			zap.String("repo", b.RepoID),
			zap.String("file", file.fullPath),
			zap.Error(err),
		)
		return n, fuse.EIO
	}
	return n, nil
}
