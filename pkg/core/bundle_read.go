package core

import (
	"context"
	"io"

	"go.uber.org/zap"

	"github.com/oneconcern/datamon/pkg/cafs"

	"github.com/jacobsa/fuse"
)

const EOF = "EOF"

func (b *Bundle) ReadAt(file *fsEntry, destination []byte, offset int64) (n int, err error) {
	if !b.Streamed {

		var reader io.ReaderAt

		reader, err = b.ConsumableStore.GetAt(context.Background(), file.fullPath)
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

		key, err = cafs.KeyFromString(file.hash)
		if err != nil {
			b.l.Error("failed to create cafs key",
				zap.String("key", file.hash),
				zap.String("bundleID", b.BundleID),
				zap.String("repo", b.RepoID),
				zap.String("file", file.fullPath),
			)
			return
		}

		reader, err = b.cafs.GetAt(context.Background(), key)
		if err != nil {
			b.l.Error("filed to getAt",
				zap.String("key", file.hash),
				zap.String("bundleID", b.BundleID),
				zap.String("repo", b.RepoID),
				zap.String("file", file.fullPath),
			)
			return
		}

		n, err = reader.ReadAt(destination, offset)
		if err != nil && err.Error() != EOF {
			err = fuse.EIO
			b.l.Error("filed to readAt",
				zap.String("key", file.hash),
				zap.String("bundleID", b.BundleID),
				zap.String("repo", b.RepoID),
				zap.String("file", file.fullPath),
			)
			return
		}
		return
	}
}
