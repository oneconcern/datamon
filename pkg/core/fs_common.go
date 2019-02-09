package core

import (
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"time"
	"unsafe"

	"go.uber.org/zap"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"

	"github.com/oneconcern/datamon/pkg/model"
)

func statFS() (err error) {
	// TODO: Find the free space on the device and set the attributes accordingly.
	// TODO: Find optimal block size (Default to the one used by underlying FS)
	return
}

func formLookupKey(id fuseops.InodeID, childName string) []byte {
	i := formKey(id)
	c := model.UnsafeStringToBytes(childName)
	return append(i, c...)
}

func formKey(ID fuseops.InodeID) []byte {
	b := make([]byte, unsafe.Sizeof(uint64(0)))
	binary.BigEndian.PutUint64(b, uint64(ID))
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
	fs.lookupTree, _, _ = fs.lookupTree.Insert(lk, lookupEntry{iNode: fuseops.InodeID(iNodeID)})

	// Default to common case of create file
	var linkCount = fileLinkCount
	var defaultMode os.FileMode = fileDefaultMode
	var defaultSize uint64 = 0

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
			fs.l.Error("failed to create backing file",
				zap.Error(err),
				zap.String("child", childName),
				zap.Uint64("parent", uint64(parentINode)))
		}
	}

	d := &fuseutil.Dirent{
		Inode: fuseops.InodeID(iNodeID),
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
		entry.Child = fuseops.InodeID(iNodeID)
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
