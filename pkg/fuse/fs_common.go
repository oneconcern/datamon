package fuse

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"
	"unsafe"

	"go.uber.org/zap"

	iradix "github.com/hashicorp/go-immutable-radix"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"

	"github.com/oneconcern/datamon/pkg/convert"
	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/metrics"
)

type fsCommon struct {
	fuseutil.NotImplementedFileSystem

	// Backing bundle for this FS.
	bundle *core.Bundle

	// Fast lookup of parent iNode id + child name, returns iNode of child. This is a common operation and it's speed is
	// important.
	lookupTree *iradix.Tree

	// logger
	l *zap.Logger

	metrics.Enable
	m *M
}

func (fs *fsCommon) StatFS(
	ctx context.Context,
	op *fuseops.StatFSOp) (err error) {
	t0 := fs.opStart(op)
	defer fs.opEnd(t0, op, err)

	return statFS()
}

func (fs *fsCommon) GetXattr(
	ctx context.Context,
	op *fuseops.GetXattrOp) error {
	// datamon mount ignores extended attributes
	return nil
}

func (fs *fsCommon) ListXattr(
	ctx context.Context,
	op *fuseops.ListXattrOp) error {
	// datamon mount ignores extended attributes
	return nil
}

func statFS() (err error) {
	// TODO: Find the free space on the device and set the attributes accordingly.
	// TODO: Find optimal block size (Default to the one used by underlying FS)
	return
}

func (fs *fsCommon) opStart(op interface{}) time.Time {
	logger := fs.l.With(zap.String("Request", fmt.Sprintf("%T", op)))
	switch t := op.(type) {
	case *fuseops.StatFSOp:
		logger.Debug("Start", zap.Uint64("inodes", t.Inodes), zap.Uint64("blocks", t.Blocks))
	case *fuseops.ReadFileOp:
		logger.Debug("Start", zap.Uint64("inode", uint64(t.Inode)), zap.Int("buffer", len(t.Dst)), zap.Int64("offset", t.Offset))
	case *fuseops.WriteFileOp:
		logger.Debug("Start", zap.Uint64("inode", uint64(t.Inode)))
	case *fuseops.ReadDirOp:
		logger.Debug("Start", zap.Uint64("inode", uint64(t.Inode)))
	case *fuseops.LookUpInodeOp:
		logger.Debug("Start", zap.Uint64("parent", uint64(t.Parent)), zap.String("child", t.Name))
	case *fuseops.GetInodeAttributesOp:
		logger.Debug("Start", zap.Uint64("id", uint64(t.Inode)))
	case *fuseops.SetInodeAttributesOp:
		logger.Debug("Start", zap.Uint64("id", uint64(t.Inode)))
	case *fuseops.ForgetInodeOp:
		logger.Debug("Start", zap.Uint64("id", uint64(t.Inode)))
	case *fuseops.RenameOp:
		logger.Debug("Start", zap.Uint64("oldP", uint64(t.OldParent)), zap.String("oldN", t.OldName),
			zap.Uint64("nP", uint64(t.NewParent)), zap.String("nN", t.NewName))
	case *fuseops.ReleaseDirHandleOp:
		logger.Debug("Start", zap.Uint64("id", uint64(t.Handle)))
	case *fuseops.OpenFileOp:
		logger.Debug("Start", zap.Uint64("id", uint64(t.Inode)))
	case *fuseops.SyncFileOp:
		logger.Debug("Start", zap.Uint64("id", uint64(t.Inode)))
	case *fuseops.FlushFileOp:
		logger.Debug("Start", zap.Uint64("id", uint64(t.Inode)))
	case *fuseops.ReleaseFileHandleOp:
		logger.Debug("Start", zap.Uint64("hndl", uint64(t.Handle)))
	case *fuseops.RmDirOp:
		logger.Debug("Start", zap.Uint64("id", uint64(t.Parent)), zap.String("name", t.Name))
	case *fuseops.UnlinkOp:
		logger.Debug("Start", zap.Uint64("id", uint64(t.Parent)), zap.String("name", t.Name))
	default:
		logger.Debug("Start", zap.Any("op", op))
	}
	return time.Now()
}

func (fs *fsCommon) opEnd(t0 time.Time, op interface{}, err error) {
	opName := fmt.Sprintf("%T", op)
	logger := fs.l.With(zap.String("Request", opName))
	switch t := op.(type) {
	case *fuseops.StatFSOp:
		logger.Debug("End", zap.Uint64("inodes", t.Inodes), zap.Uint64("blocks", t.Blocks), zap.Error(err))
	case *fuseops.ReadFileOp:
		logger.Debug("End", zap.Uint64("inode", uint64(t.Inode)), zap.Int64("offset", t.Offset), zap.Error(err))
		return
	case *fuseops.WriteFileOp:
		logger.Debug("End", zap.Uint64("inode", uint64(t.Inode)), zap.Error(err))
		return
	case *fuseops.ReadDirOp:
		logger.Debug("End", zap.Uint64("inode", uint64(t.Inode)), zap.Error(err))
		return
	case *fuseops.LookUpInodeOp:
		logger.Debug("End", zap.Uint64("parent", uint64(t.Parent)), zap.String("child", t.Name), zap.Error(err))
	case *fuseops.GetInodeAttributesOp:
		logger.Debug("End", zap.Uint64("id", uint64(t.Inode)), zap.Error(err))
	case *fuseops.SetInodeAttributesOp:
		logger.Debug("End", zap.Uint64("id", uint64(t.Inode)), zap.Error(err))
	case *fuseops.ForgetInodeOp:
		logger.Debug("End", zap.Uint64("id", uint64(t.Inode)), zap.Error(err))
	case *fuseops.RenameOp:
		logger.Debug("End", zap.Uint64("oldP", uint64(t.OldParent)), zap.String("oldN", t.OldName),
			zap.Uint64("nP", uint64(t.NewParent)), zap.String("nN", t.NewName), zap.Error(err))
	case *fuseops.ReleaseDirHandleOp:
		logger.Debug("End", zap.Uint64("id", uint64(t.Handle)), zap.Error(err))
	case *fuseops.OpenFileOp:
		logger.Debug("End", zap.Uint64("id", uint64(t.Inode)), zap.Error(err))
	case *fuseops.SyncFileOp:
		logger.Debug("End", zap.Uint64("id", uint64(t.Inode)), zap.Error(err))
	case *fuseops.FlushFileOp:
		logger.Debug("End", zap.Uint64("id", uint64(t.Inode)), zap.Error(err))
	case *fuseops.ReleaseFileHandleOp:
		logger.Debug("End", zap.Uint64("hndl", uint64(t.Handle)), zap.Error(err))
	case *fuseops.RmDirOp:
		logger.Debug("End", zap.Uint64("id", uint64(t.Parent)), zap.String("name", t.Name), zap.Error(err))
	case *fuseops.UnlinkOp:
		logger.Debug("End", zap.Uint64("id", uint64(t.Parent)), zap.String("name", t.Name), zap.Error(err))
	}
	if fs.MetricsEnabled() {
		fs.m.Usage.UsedAll(t0, opName)(err)
	}
	logger.Debug("End", zap.Any("op", op), zap.Error(err))
}

func formLookupKey(id fuseops.InodeID, childName string) []byte {
	i := formKey(id)
	c := convert.UnsafeStringToBytes(childName)
	return append(i, c...)
}

var intSize = unsafe.Sizeof(uint64(0))

func formKey(id fuseops.InodeID) []byte {
	b := make([]byte, intSize)
	binary.BigEndian.PutUint64(b, uint64(id))
	return b
}
