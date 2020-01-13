package core

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"time"
	"unsafe"

	"go.uber.org/zap"

	iradix "github.com/hashicorp/go-immutable-radix"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"

	"github.com/oneconcern/datamon/pkg/convert"
)

type fsCommon struct {
	fuseutil.NotImplementedFileSystem

	// Backing bundle for this FS.
	bundle *Bundle

	// Fast lookup of parent iNode id + child name, returns iNode of child. This is a common operation and it's speed is
	// important.
	lookupTree *iradix.Tree

	// logger
	l *zap.Logger
}

func (fs *fsCommon) StatFS(
	ctx context.Context,
	op *fuseops.StatFSOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)

	return statFS()
}

func statFS() (err error) {
	// TODO: Find the free space on the device and set the attributes accordingly.
	// TODO: Find optimal block size (Default to the one used by underlying FS)
	return
}

func (fs *fsCommon) opStart(op interface{}) {
	logger := fs.l.With(zap.String("Request", fmt.Sprintf("%T", op)))
	switch t := op.(type) {
	case *fuseops.StatFSOp:
		logger.Debug("Start", zap.Uint64("inodes", t.Inodes), zap.Uint64("blocks", t.Blocks))
	case *fuseops.ReadFileOp:
		logger.Debug("Start", zap.Uint64("inode", uint64(t.Inode)), zap.Int("buffer", len(t.Dst)), zap.Int64("offset", t.Offset))
		return
	case *fuseops.WriteFileOp:
		logger.Debug("Start", zap.Uint64("inode", uint64(t.Inode)))
		return
	case *fuseops.ReadDirOp:
		logger.Debug("Start", zap.Uint64("inode", uint64(t.Inode)))
		return
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
	}
	logger.Debug("Start", zap.Any("op", op))
}

func (fs *fsCommon) opEnd(op interface{}, err error) {
	logger := fs.l.With(zap.String("Request", fmt.Sprintf("%T", op)))
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
	logger.Debug("End", zap.Any("op", op), zap.Error(err))
}

func formLookupKey(id fuseops.InodeID, childName string) []byte {
	i := formKey(id)
	c := convert.UnsafeStringToBytes(childName)
	return append(i, c...)
}

func formKey(id fuseops.InodeID) []byte {
	b := make([]byte, unsafe.Sizeof(uint64(0)))
	binary.BigEndian.PutUint64(b, uint64(id))
	return b
}

// Add the root of the FS into the FS.
func (fs *fsMutable) initRoot() (err error) {
	_, found := fs.lookupTree.Get(formKey(fuseops.RootInodeID))
	if found {
		return
	}
	err = fs.createNode(
		formLookupKey(fuseops.RootInodeID, rootPath),
		fuseops.RootInodeID,
		rootPath,
		nil,
		fuseutil.DT_Directory,
		true)
	return
}

// Run validations before creating a node. Need to take locks before calling.
func (fs *fsMutable) preCreateCheck(parentInode fuseops.InodeID, lk []byte) error {
	// check parent exists
	key := formKey(parentInode)
	e, found := fs.iNodeStore.Get(key)
	if !found {
		return fuse.ENOENT
	}

	// parent is a directory
	n := e.(*nodeEntry)
	if !n.attr.Mode.IsDir() {
		return fuse.ENOTDIR
	}

	// check child name not taken
	_, found = fs.lookupTree.Get(lk)
	if found {
		return fuse.EEXIST
	}
	return nil
}

func (fs *fsMutable) insertReadDirEntry(id fuseops.InodeID, dirEnt *fuseutil.Dirent) {

	if fs.readDirMap[id] == nil {
		fs.readDirMap[id] = make(map[fuseops.InodeID]*fuseutil.Dirent)
	}
	fs.readDirMap[id][dirEnt.Inode] = dirEnt
}

func (fs *fsMutable) insertLookupEntry(id fuseops.InodeID, child string, entry lookupEntry) {
	fs.lookupTree, _, _ = fs.lookupTree.Insert(formLookupKey(id, child), entry)
}

// Create a node. Need to hold the locks before calling.
func (fs *fsMutable) createNode(lk []byte, parentINode fuseops.InodeID, childName string,
	entry *fuseops.ChildInodeEntry, nodeType fuseutil.DirentType, isRoot bool) error {

	// Create lookup key if not already created.
	if lk == nil {
		lk = formLookupKey(parentINode, childName)
	}

	var iNodeID fuseops.InodeID
	if !isRoot {
		iNodeID = fs.iNodeGenerator.allocINode()
	} else {
		iNodeID = parentINode
	}

	// lookup
	fs.lookupTree, _, _ = fs.lookupTree.Insert(lk, lookupEntry{iNode: iNodeID})

	// Default to common case of create file
	var linkCount = fileLinkCount
	var defaultMode os.FileMode = fileDefaultMode
	var defaultSize uint64

	if nodeType == fuseutil.DT_Directory {
		linkCount = dirLinkCount
		defaultMode = dirDefaultMode
		defaultSize = dirInitialSize
		fs.readDirMap[iNodeID] = make(map[fuseops.InodeID]*fuseutil.Dirent)
	} else {
		// dont return error as open file will retry this.
		file, err := fs.localCache.Create(fmt.Sprint(iNodeID))
		if err != nil {
			fs.backingFiles[iNodeID] = &file
		} else {
			fs.l.Warn("failed to create backing file: open file will retry this",
				zap.Error(err),
				zap.String("child", childName),
				zap.Uint64("parent", uint64(parentINode)))
		}
	}

	d := &fuseutil.Dirent{
		Inode: iNodeID,
		Name:  childName,
		Type:  nodeType,
	}
	if !isRoot {
		fs.insertReadDirEntry(parentINode, d)
	}

	ts := time.Now()
	attr := fuseops.InodeAttributes{
		Size:   defaultSize,
		Nlink:  linkCount,
		Mode:   defaultMode,
		Atime:  ts,
		Mtime:  ts,
		Ctime:  ts,
		Crtime: ts,
		Uid:    defaultGID,
		Gid:    defaultUID,
	}

	//iNode Store
	fs.iNodeStore, _, _ = fs.iNodeStore.Insert(formKey(iNodeID), &nodeEntry{
		lock:              sync.Mutex{},
		refCount:          1, // As per spec CreateFileOp
		pathToBackingFile: getPathToBackingFile(iNodeID),
		attr:              attr,
	})

	if nodeType == fuseutil.DT_Directory {
		// Increment parent ref count.
		p, _ := fs.iNodeStore.Get(formKey(parentINode))
		parentNodeEntry := p.(*nodeEntry)
		parentNodeEntry.attr.Nlink++
	}

	// If return is expected
	if entry != nil {
		entry.Attributes = attr
		entry.EntryExpiration = time.Now().Add(cacheYearLong)
		entry.AttributesExpiration = time.Now().Add(cacheYearLong)
		entry.Child = iNodeID
	}
	return nil
}

func getPathToBackingFile(iNode fuseops.InodeID) string {
	return fmt.Sprint(uint64(iNode))
}

func shouldDelete(n *nodeEntry) bool {
	// LookupCount should be zero.
	if n.attr.Mode.IsDir() {
		if n.refCount == 0 {
			return true
		}
	} else {
		if n.refCount == 0 && n.attr.Nlink == 0 {
			return true
		}
	}
	return false
}
