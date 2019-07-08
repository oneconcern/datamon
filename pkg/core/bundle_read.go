package core

import (
	"context"
	"io"

	"go.uber.org/zap"

	"github.com/oneconcern/datamon/pkg/cafs"

	"github.com/jacobsa/fuse"
)

const EOF = "EOF"

// public in order to gather profiling metrics
func BundleReadAtImpl(b *Bundle,
	fileFullPath string, fileHash string,
	destination []byte, offset int64) (n int, err error) {
	if !b.Streamed {

		var reader io.ReaderAt

		reader, err = b.ConsumableStore.GetAt(context.Background(), fileFullPath)
		if err != nil {
			err = fuse.EIO
			return
		}

		n, err = reader.ReadAt(destination, offset)
		if err != nil && err.Error() != EOF {
			err = fuse.EIO
			return
		}

		return

	} else {

		var reader io.ReaderAt
		var key cafs.Key

		key, err = cafs.KeyFromString(fileHash)
		if err != nil {
			b.l.Error("failed to create cafs key",
				zap.String("key", fileHash),
				zap.String("bundleID", b.BundleID),
				zap.String("repo", b.RepoID),
				zap.String("file", fileFullPath),
			)
			return
		}

		reader, err = b.cafs.GetAt(context.Background(), key)
		if err != nil {
			b.l.Error("filed to getAt",
				zap.String("key", fileHash),
				zap.String("bundleID", b.BundleID),
				zap.String("repo", b.RepoID),
				zap.String("file", fileFullPath),
			)
			return
		}

		n, err = reader.ReadAt(destination, offset)
		if err != nil && err.Error() != EOF {
			err = fuse.EIO
			b.l.Error("filed to readAt",
				zap.String("key", fileHash),
				zap.String("bundleID", b.BundleID),
				zap.String("repo", b.RepoID),
				zap.String("file", fileFullPath),
			)
			return
		}
		return
	}
}

func (b *Bundle) ReadAt(file *fsEntry, destination []byte, offset int64) (n int, err error) {
	return BundleReadAtImpl(b,
		file.fullPath, file.hash,
		destination, offset)
}
