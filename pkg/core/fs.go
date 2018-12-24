package core

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/oneconcern/datamon/pkg/model"

	"github.com/hashicorp/go-immutable-radix"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
)

// Cache duration
var cacheYearLong = 365 * 24 * time.Hour
var dirLinkCount uint32 = 2
var fileLinkCount uint32 = 1
var rootPath = "/"
var firstINode uint64 = 1023

// DatamonFS is the virtual filesystem created on top of a bundle.
type DatamonFS struct {
	mfs        *fuse.MountedFileSystem // The mounted filesystem
	fsInternal *fsInternal             // The core of the filesystem
	server     fuse.Server             // Fuse server
}

// NewDatamonFS creates a new instance of the datamon filesystem.
func NewDatamonFS(bundle *Bundle) (*DatamonFS, error) {

	fs := &fsInternal{
		bundle:       bundle,
		readDirMap:   make(map[uint64][]fuseutil.Dirent),
		fsEntryStore: iradix.New(),
		lookupTree:   iradix.New(),
		fsDirStore:   iradix.New(),
	}

	// Extract the meta information needed.
	err := PublishMetadata(context.Background(), fs.bundle)
	if err != nil {
		return nil, err
	}

	// Populate the filesystem.
	return fs.populateFS(bundle)
}

type fsInternal struct {

	// Backing bundle for this FS.
	bundle *Bundle

	// Get iNode for path. This is needed to generate directory entries without imposing a strict order of traversal.
	fsDirStore *iradix.Tree

	// Get fsEntry for an iNode. Speed up stat and other calls keyed by iNode
	fsEntryStore *iradix.Tree

	// Fast lookup of parent iNode id + child name, returns iNode of child. This is a common operation and it's speed is
	// important.
	lookupTree *iradix.Tree

	// List of children for a given iNode. Maps inode id to list of children. This stitches the fuse FS together.
	readDirMap map[uint64][]fuseutil.Dirent

	// readonly
	isReadOnly bool
}

// fsEntry is a node in the filesystem.
type fsEntry struct {
	hash string // Set for files, empty for directories

	// iNode ID is generated on the fly for a bundle that is committed. Since the file list
	// for a bundle is static and the list of files is frozen, multiple mounts of the same
	// bundle will preserve a fixed iNode for a file provided the order of reading the files
	// remains fixed.
	iNode      uint64                  // Unique ID for Fuse
	attributes fuseops.InodeAttributes // Fuse Attributes
}

type fsNodeToAdd struct {
	parentINode uint64
	fullPath    string
	fsEntry     fsEntry
}

func (fs *fsInternal) insertDatamonFSDirEntry(
	dirStoreTxn *iradix.Txn,
	lookupTreeTxn *iradix.Txn,
	fsEntryStoreTxn *iradix.Txn,
	fullPath string,
	parentInode uint64,
	dirFsEntry fsEntry) error {

	_, update := dirStoreTxn.Insert([]byte(fullPath), dirFsEntry)
	if update {
		return errors.New("dirStore updates are not expected: /" + fullPath)
	}

	key := formKey(dirFsEntry.iNode)

	_, update = fsEntryStoreTxn.Insert(key, dirFsEntry)
	if update {
		return errors.New("fsEntryStore updates are not expected: /")
	}

	if dirFsEntry.iNode != fuseops.RootInodeID {
		key = formLookupKey(parentInode, path.Base(fullPath))

		_, update = lookupTreeTxn.Insert(key, dirFsEntry)
		if update {
			return errors.New("lookupTree updates are not expected: " + fullPath)
		}

		childEntries := fs.readDirMap[parentInode]
		childEntries = append(childEntries, fuseutil.Dirent{
			Offset: fuseops.DirOffset(len(childEntries) + 1),
			Inode:  fuseops.InodeID(dirFsEntry.iNode),
			Name:   path.Base(fullPath),
			Type:   fuseutil.DT_Directory,
		})
		fs.readDirMap[parentInode] = childEntries
	}

	return nil
}

func (fs *fsInternal) insertDatamonFSEntry(
	lookupTreeTxn *iradix.Txn,
	fsEntryStoreTxn *iradix.Txn,
	fullPath string,
	parentInode uint64,
	fsEntry fsEntry) error {

	key := formKey(fsEntry.iNode)

	_, update := fsEntryStoreTxn.Insert(key, fsEntry)
	if update {
		return errors.New("fsEntryStore updates are not expected: " + fullPath)
	}

	key = formLookupKey(parentInode, path.Base(fullPath))

	_, update = lookupTreeTxn.Insert(key, fsEntry)
	if update {
		return errors.New("lookupTree updates are not expected: " + fullPath)
	}

	childEntries := fs.readDirMap[parentInode]
	childEntries = append(childEntries, fuseutil.Dirent{
		Offset: fuseops.DirOffset(len(childEntries) + 1),
		Inode:  fuseops.InodeID(fsEntry.iNode),
		Name:   path.Base(fullPath),
		Type:   fuseutil.DT_File,
	})
	fs.readDirMap[parentInode] = childEntries

	return nil
}

func (fs *fsInternal) populateFS(bundle *Bundle) (*DatamonFS, error) {
	dirStoreTxn := fs.fsDirStore.Txn()
	lookupTreeTxn := fs.lookupTree.Txn()
	fsEntryStoreTxn := fs.fsEntryStore.Txn()

	// Add root.
	dirFsEntry := newDatamonFSEntry(generateBundleDirEntry(rootPath), bundle.BundleDescriptor.Timestamp, fuseops.RootInodeID, dirLinkCount)
	err := fs.insertDatamonFSDirEntry(
		dirStoreTxn,
		lookupTreeTxn,
		fsEntryStoreTxn,
		rootPath,
		fuseops.RootInodeID, // Root points to itself
		*dirFsEntry)
	if err != nil {
		return nil, err
	}

	// For a Bundle Entry there might be intermediate directories that need adding.
	var nodesToAdd []fsNodeToAdd
	// iNode for fs entries
	var iNode = firstINode

	generateNextINode := func(iNode *uint64) uint64 {
		*iNode++
		return *iNode
	}

	for _, bundleEntry := range fs.bundle.GetBundleEntries() {

		// Generate the fsEntry
		newFsEntry := newDatamonFSEntry(&bundleEntry, bundle.BundleDescriptor.Timestamp, generateNextINode(&iNode), fileLinkCount)

		// Add parents if first visit
		// If a parent has been visited, all the parent's parents in the path have been visited
		nameWithPath := bundleEntry.NameWithPath
		for {
			parentPath := path.Dir(nameWithPath)
			// entry under root
			if parentPath == "" || parentPath == "." || parentPath == "/" {
				nodesToAdd = append(nodesToAdd, fsNodeToAdd{
					parentINode: fuseops.RootInodeID,
					fsEntry:     *newFsEntry,
					fullPath:    nameWithPath,
				})
				if len(nodesToAdd) > 1 {
					// If more than one node is to be added populate the parent iNode.
					nodesToAdd[len(nodesToAdd)-2].parentINode = nodesToAdd[len(nodesToAdd)-1].fsEntry.iNode
				}
				break
			}

			// Copy into queue
			nodesToAdd = append(nodesToAdd, fsNodeToAdd{
				parentINode: 0, // undefined
				fsEntry:     *newFsEntry,
				fullPath:    nameWithPath,
			})

			if len(nodesToAdd) > 1 {
				// If more than one node is to be added populate the parent iNode.
				nodesToAdd[len(nodesToAdd)-2].parentINode = nodesToAdd[len(nodesToAdd)-1].fsEntry.iNode
			}

			p, found := dirStoreTxn.Get([]byte(parentPath))
			if !found {

				newFsEntry = newDatamonFSEntry(generateBundleDirEntry(parentPath), bundle.BundleDescriptor.Timestamp, generateNextINode(&iNode), dirLinkCount)

				// Continue till we hit root or found
				nameWithPath = parentPath
				continue
			} else {
				parentDirEntry := p.(fsEntry)
				if len(nodesToAdd) == 1 {
					nodesToAdd[len(nodesToAdd)-1].parentINode = parentDirEntry.iNode
				}
			}
			break
		}

		for _, nodeToAdd := range nodesToAdd {
			if nodeToAdd.fsEntry.attributes.Nlink == dirLinkCount {
				err = fs.insertDatamonFSDirEntry(
					dirStoreTxn,
					lookupTreeTxn,
					fsEntryStoreTxn,
					nodeToAdd.fullPath,
					nodeToAdd.parentINode,
					nodeToAdd.fsEntry,
				)

			} else {
				err = fs.insertDatamonFSEntry(
					lookupTreeTxn,
					fsEntryStoreTxn,
					nodeToAdd.fullPath,
					nodeToAdd.parentINode,
					nodeToAdd.fsEntry,
				)
			}
			if err != nil {
				return nil, err
			}
			nodesToAdd = nodesToAdd[:0]
		}
	} // End walking bundle entries q2

	fs.fsEntryStore = fsEntryStoreTxn.Commit()
	fs.lookupTree = lookupTreeTxn.Commit()
	fs.fsDirStore = fsEntryStoreTxn.Commit()
	fs.isReadOnly = true
	return &DatamonFS{
		fsInternal: fs,
		server:     fuseutil.NewFileSystemServer(fs),
	}, nil
}

func newDatamonFSEntry(bundleEntry *model.BundleEntry, time time.Time, id uint64, linkCount uint32) *fsEntry {
	var mode os.FileMode = 0775
	if bundleEntry.Hash == "" {
		mode = 0777 | os.ModeDir
	}
	return &fsEntry{
		hash:  bundleEntry.Hash,
		iNode: id,
		attributes: fuseops.InodeAttributes{
			Size:   bundleEntry.Size,
			Nlink:  linkCount,
			Mode:   mode,
			Atime:  time,
			Mtime:  time,
			Ctime:  time,
			Crtime: time,
			Uid:    0, // TODO: Set to uid gid usable by container..
			Gid:    0, // TODO: Same as above
		},
	}
}

func generateBundleDirEntry(nameWithPath string) *model.BundleEntry {
	return &model.BundleEntry{
		Hash:         "", // Directories do not have datamon backed hash
		NameWithPath: nameWithPath,
		FileMode:     0777 | os.ModeDir,
		Size:         2048, // TODO: Increase size of directory with file count when mount is mutable.
	}
}

func (dfs *DatamonFS) Mount(path string) error {
	// TODO plumb additional mount options
	mountCfg := &fuse.MountConfig{
		FSName:      dfs.fsInternal.bundle.RepoID,
		VolumeName:  dfs.fsInternal.bundle.BundleID,
		ErrorLogger: log.New(os.Stderr, "fuse: ", log.Flags()),
	}
	var err error
	dfs.mfs, err = fuse.Mount(path, dfs.server, mountCfg)
	return err
}

func (dfs *DatamonFS) Unmount(path string) error {
	return fuse.Unmount(path)
}

func formLookupKey(id uint64, childName string) []byte {
	iNode := model.Uint64ToBytes(&id)
	childNameBuf := model.UnsafeStringToBytes(childName)
	return append(iNode, childNameBuf...)
}

func formKey(ID uint64) []byte {
	return model.Uint64ToBytes(&ID)
}

func isDir(fsEntry fsEntry) bool {
	if fsEntry.hash != "" {
		return true
	}
	return false
}

func (fs *fsInternal) StatFS(
	ctx context.Context,
	op *fuseops.StatFSOp) (err error) {
	// TODO: Find the free space on the device and set the attributes accordingly.
	// TODO: Find optimal block size (Default to the one used by underlying FS)
	log.Print("Stat fs")
	return
}

func (fs *fsInternal) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	log.Print(fmt.Printf("lookup parent id:%d, child: %s", op.Parent, op.Name))
	parentINode := uint64(op.Parent)
	lookupKey := formLookupKey(parentINode, op.Name)
	val, found := fs.lookupTree.Get(lookupKey)
	if found {
		childEntry := val.(fsEntry)
		op.Entry.Attributes = childEntry.attributes
		if fs.isReadOnly {
			op.Entry.AttributesExpiration = time.Now().Add(cacheYearLong)
			op.Entry.EntryExpiration = op.Entry.AttributesExpiration
		}
		op.Entry.Child = fuseops.InodeID(childEntry.iNode)
		op.Entry.Generation = 1
	} else {
		return fuse.ENOENT
	}
	return nil
}

func (fs *fsInternal) GetInodeAttributes(
	ctx context.Context,
	op *fuseops.GetInodeAttributesOp) (err error) {
	log.Print(fmt.Printf("iNode attr id:%d ", op.Inode))
	key := formKey(uint64(op.Inode))
	e, found := fs.fsEntryStore.Get(key)
	if !found {
		return fuse.ENOENT
	}
	fsEntry := e.(fsEntry)
	op.AttributesExpiration = time.Now().Add(cacheYearLong)
	op.Attributes = fsEntry.attributes
	return nil
}

func (fs *fsInternal) SetInodeAttributes(
	ctx context.Context,
	op *fuseops.SetInodeAttributesOp) (err error) {
	log.Print(fmt.Printf("SetInodeAttributes iNode id:%d ", op.Inode))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) ForgetInode(
	ctx context.Context,
	op *fuseops.ForgetInodeOp) (err error) {
	log.Print(fmt.Printf("ForgetInode iNode id:%d ", op.Inode))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) MkDir(
	ctx context.Context,
	op *fuseops.MkDirOp) (err error) {
	log.Print(fmt.Printf("Mkdie parent iNode id:%d ", op.Parent))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) MkNode(
	ctx context.Context,
	op *fuseops.MkNodeOp) (err error) {
	log.Print(fmt.Printf("MkNode parent iNode id:%d ", op.Parent))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) CreateFile(
	ctx context.Context,
	op *fuseops.CreateFileOp) (err error) {
	log.Print(fmt.Printf("CreateFile parent iNode id:%d name: %s", op.Parent, op.Name))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) CreateSymlink(
	ctx context.Context,
	op *fuseops.CreateSymlinkOp) (err error) {
	log.Print(fmt.Printf("CreateSymLink"))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) CreateLink(
	ctx context.Context,
	op *fuseops.CreateLinkOp) (err error) {
	log.Print(fmt.Printf("CreateLink"))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) Rename(
	ctx context.Context,
	op *fuseops.RenameOp) (err error) {
	log.Print(fmt.Printf("Rename new name:"+op.NewName+" oldname:"+op.OldName+" new parent %d, old parent %d", op.NewParent, op.OldParent))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) RmDir(
	ctx context.Context,
	op *fuseops.RmDirOp) (err error) {
	log.Print(fmt.Printf("RmDir iNode id:%d ", op.Parent))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) Unlink(
	ctx context.Context,
	op *fuseops.UnlinkOp) (err error) {
	log.Print(fmt.Printf("Unlink child: "+op.Name+" parent: %d", op.Parent))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) OpenDir(ctx context.Context, openDirOp *fuseops.OpenDirOp) error {
	log.Print(fmt.Printf("openDir iNode id:%d ", openDirOp.Inode))
	p, found := fs.fsEntryStore.Get(formKey(uint64(openDirOp.Inode)))
	if !found {
		return fuse.ENOENT
	}
	fsEntry := p.(fsEntry)
	if isDir(fsEntry) {
		return fuse.ENOTDIR
	}
	return nil
}

func (fs *fsInternal) ReadDir(ctx context.Context, readDirOp *fuseops.ReadDirOp) error {

	offset := int(readDirOp.Offset)
	iNode := uint64(readDirOp.Inode)

	children, found := fs.readDirMap[iNode]

	if !found {
		return fuse.ENOENT
	}

	if offset > len(children) {
		return fuse.EIO
	}

	for i := offset; i < len(children); i++ {
		n := fuseutil.WriteDirent(readDirOp.Dst[readDirOp.BytesRead:], children[i])
		if n == 0 {
			break
		}
		readDirOp.BytesRead += n
	}
	log.Print(fmt.Printf("readDir iNode id:%d offset: %d bytes: %d ", readDirOp.Inode, readDirOp.Offset, readDirOp.BytesRead))
	return nil
}

func (fs *fsInternal) ReleaseDirHandle(
	ctx context.Context,
	op *fuseops.ReleaseDirHandleOp) (err error) {
	log.Print(fmt.Printf("ReleaseDirHandle iNode id:%d ", op.Handle))
	return
}

func (fs *fsInternal) OpenFile(
	ctx context.Context,
	op *fuseops.OpenFileOp) (err error) {
	log.Print(fmt.Printf("OpenFile iNode id:%d ", op.Inode))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) ReadFile(
	ctx context.Context,
	op *fuseops.ReadFileOp) (err error) {
	log.Print(fmt.Printf("ReadFile iNode id:%d ", op.Inode))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) WriteFile(
	ctx context.Context,
	op *fuseops.WriteFileOp) (err error) {
	log.Print(fmt.Printf("WriteFile iNode id:%d ", op.Inode))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) SyncFile(
	ctx context.Context,
	op *fuseops.SyncFileOp) (err error) {
	log.Print(fmt.Printf("SyncFile iNode id:%d ", op.Inode))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) FlushFile(
	ctx context.Context,
	op *fuseops.FlushFileOp) (err error) {
	log.Print(fmt.Printf("FlushFile iNode id:%d ", op.Inode))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) ReleaseFileHandle(
	ctx context.Context,
	op *fuseops.ReleaseFileHandleOp) (err error) {
	log.Print(fmt.Printf("ReleaseFileHandle iNode id:%d ", op.Handle))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) ReadSymlink(
	ctx context.Context,
	op *fuseops.ReadSymlinkOp) (err error) {
	log.Print(fmt.Printf("ReadSymlink iNode id:%d ", op.Inode))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) RemoveXattr(
	ctx context.Context,
	op *fuseops.RemoveXattrOp) (err error) {
	log.Print(fmt.Printf("RemoveXattr iNode id:%d ", op.Inode))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) GetXattr(
	ctx context.Context,
	op *fuseops.GetXattrOp) (err error) {
	log.Print(fmt.Printf("GetXattr iNode id:%d ", op.Inode))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) ListXattr(
	ctx context.Context,
	op *fuseops.ListXattrOp) (err error) {
	log.Print(fmt.Printf("ListXattr iNode id:%d ", op.Inode))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) SetXattr(
	ctx context.Context,
	op *fuseops.SetXattrOp) (err error) {
	log.Print(fmt.Printf("SetXattr iNode id:%d ", op.Inode))
	err = fuse.ENOSYS
	return
}

func (fs *fsInternal) Destroy() {
	log.Print(fmt.Printf("Destroy"))
}
