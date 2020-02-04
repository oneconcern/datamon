package fuse

import (
	"context"

	"github.com/jacobsa/fuse"
	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/fuse/status"
	"go.uber.org/zap"
)

func errNotEOF(err error) bool {
	return err != nil && err.Error() != "EOF"
}

// readAtBundle reads some bundle data with an optional offset
func (fs *readOnlyFsInternal) readAtBundle(file *FsEntry, destination []byte, offset int64) (int, error) {
	logger := fs.l.With(
		zap.String("key", file.hash),
		zap.String("bundleID", fs.bundle.BundleID),
		zap.String("repo", fs.bundle.RepoID),
		zap.String("file", file.fullPath),
	)

	if !fs.streamed {
		// just consumes the file from staging ("consumable store")
		logger.Debug("unstreamed ReadAt", zap.Int("asked bytes", len(destination)))
		reader, err := fs.bundle.ConsumableStore.GetAt(context.Background(), file.fullPath)
		if err != nil {
			logger.Error("error in unstreamed GetAt", zap.String("hash", file.hash), zap.Error(err))
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

	reader, err := fs.cafs.GetAt(context.Background(), key)
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
