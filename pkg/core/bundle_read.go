package core

import (
	"context"

	"go.uber.org/zap"

	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/core/status"

	"github.com/jacobsa/fuse"
)

func errNotEOF(err error) bool {
	return err != nil && err.Error() != "EOF"
}

// ReadAt reads some bundle data with an optional offset
func (b *Bundle) ReadAt(file *FsEntry, destination []byte, offset int64) (int, error) {
	logger := b.l.With(
		zap.String("key", file.hash),
		zap.String("bundleID", b.BundleID),
		zap.String("repo", b.RepoID),
		zap.String("file", file.fullPath),
	)

	if !b.Streamed {
		// just consumes the file from staging ("consumable store")
		logger.Debug("unstreamed ReadAt", zap.Int("asked bytes", len(destination)))
		reader, err := b.ConsumableStore.GetAt(context.Background(), file.fullPath)
		if err != nil {
			return 0, fuse.EIO
		}

		n, err := reader.ReadAt(destination, offset)
		if errNotEOF(err) {
			logger.Error("error in unstreamed readAt", zap.String("hash", file.hash), zap.Error(err))
			return n, fuse.EIO
		}

		logger.Debug("unstreamed filed ReadAt", zap.Int("bytes", n))
		return n, nil
	}

	// when "streaming" is enabled, consumes the file from a content-addressable FS
	logger.Debug("streamed ReadAt", zap.Int("asked bytes", cap(destination)))
	key, err := cafs.KeyFromString(file.hash)
	if err != nil {
		return 0, status.ErrCafsKey.
			WrapWithLog(logger, err, zap.String("hash", file.hash))
	}

	reader, err := b.cafs.GetAt(context.Background(), key)
	if err != nil {
		return 0, status.ErrReadAt.
			WrapWithLog(logger, err, zap.String("hash", file.hash))
	}

	n, err := reader.ReadAt(destination, offset)
	if errNotEOF(err) {
		logger.Error("error in stream ReadAt", zap.String("hash", file.hash), zap.Error(err))
		return n, fuse.EIO
	}
	logger.Debug("filed ReadAt", zap.Int("bytes", n))
	return n, nil
}
