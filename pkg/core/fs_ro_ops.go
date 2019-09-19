package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	iradix "github.com/hashicorp/go-immutable-radix"
	"go.uber.org/zap"

	"github.com/oneconcern/datamon/pkg/model"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
)

func (fs *readOnlyFsInternal) StatFS(
	ctx context.Context,
	op *fuseops.StatFSOp) (err error) {
	return statFS()
}

func (fs *readOnlyFsInternal) opStart(op interface{}) {
	switch t := op.(type) {
	case *fuseops.ReadFileOp:
		fs.l.Info("Start",
			zap.String("Request", fmt.Sprintf("%T", op)),
			zap.String("repo", fs.bundle.RepoID),
			zap.String("bundle", fs.bundle.BundleID),
			zap.Uint64("inode", uint64(t.Inode)),
			zap.Int("buffer", len(t.Dst)),
			zap.Int64("offset", t.Offset),
		)
		return
	case *fuseops.WriteFileOp:
		fs.l.Info("Start",
			zap.String("Request", fmt.Sprintf("%T", op)),
			zap.String("repo", fs.bundle.RepoID),
			zap.String("bundle", fs.bundle.BundleID),
			zap.Uint64("inode", uint64(t.Inode)),
		)
		return
	case *fuseops.ReadDirOp:
		fs.l.Info("Start",
			zap.String("Request", fmt.Sprintf("%T", op)),
			zap.String("repo", fs.bundle.RepoID),
			zap.String("bundle", fs.bundle.BundleID),
			zap.Uint64("inode", uint64(t.Inode)),
		)
		return
	}
	fs.l.Info("Start",
		zap.String("Request", fmt.Sprintf("%T", op)),
		zap.String("repo", fs.bundle.RepoID),
		zap.String("bundle", fs.bundle.BundleID),
		zap.Any("op", op),
	)
}
func (fs *readOnlyFsInternal) opEnd(op interface{}, err error) {
	switch t := op.(type) {
	case *fuseops.ReadFileOp:
		fs.l.Info("End",
			zap.String("Request", fmt.Sprintf("%T", op)),
			zap.String("repo", fs.bundle.RepoID),
			zap.String("bundle", fs.bundle.BundleID),
			zap.Uint64("inode", uint64(t.Inode)),
			zap.Int64("offset", t.Offset),
			zap.Error(err),
		)
		return
	case *fuseops.WriteFileOp:
		fs.l.Info("End",
			zap.String("Request", fmt.Sprintf("%T", op)),
			zap.String("repo", fs.bundle.RepoID),
			zap.String("bundle", fs.bundle.BundleID),
			zap.Uint64("inode", uint64(t.Inode)),
			zap.Error(err),
		)
		return
	case *fuseops.ReadDirOp:
		fs.l.Info("End",
			zap.String("Request", fmt.Sprintf("%T", op)),
			zap.String("repo", fs.bundle.RepoID),
			zap.String("bundle", fs.bundle.BundleID),
			zap.Uint64("inode", uint64(t.Inode)),
			zap.Error(err),
		)
		return
	}
	fs.l.Info("End",
		zap.String("Request", fmt.Sprintf("%T", op)),
		zap.String("repo", fs.bundle.RepoID),
		zap.String("bundle", fs.bundle.BundleID),
		zap.Any("op", op),
		zap.Error(err),
	)
}
func typeAssertToFsEntry(p interface{}) *fsEntry {
	fe := p.(fsEntry)
	return &fe
}

func (fs *readOnlyFsInternal) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	lookupKey := formLookupKey(op.Parent, op.Name)
	val, found := fs.lookupTree.Get(lookupKey)

	if found {
		childEntry := typeAssertToFsEntry(val)
		op.Entry.Attributes = childEntry.attributes
		if fs.isReadOnly {
			op.Entry.AttributesExpiration = time.Now().Add(cacheYearLong)
			op.Entry.EntryExpiration = op.Entry.AttributesExpiration
		}
		op.Entry.Child = childEntry.iNode
		op.Entry.Generation = 1

	} else {
		err = fuse.ENOENT
		return
	}
	defer fs.opEnd(op, err)
	return nil
}

func (fs *readOnlyFsInternal) GetInodeAttributes(
	ctx context.Context,
	op *fuseops.GetInodeAttributesOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	key := formKey(op.Inode)
	e, found := fs.fsEntryStore.Get(key)
	if !found {
		err = fuse.ENOENT
		return
	}
	fe := typeAssertToFsEntry(e)
	op.AttributesExpiration = time.Now().Add(cacheYearLong)
	op.Attributes = fe.attributes
	return nil
}

func (fs *readOnlyFsInternal) SetInodeAttributes(
	ctx context.Context,
	op *fuseops.SetInodeAttributesOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	err = fuse.ENOSYS
	return
}

func (fs *readOnlyFsInternal) ForgetInode(
	ctx context.Context,
	op *fuseops.ForgetInodeOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	return
}

func (fs *readOnlyFsInternal) MkDir(
	ctx context.Context,
	op *fuseops.MkDirOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	err = fuse.ENOSYS
	return
}

func (fs *readOnlyFsInternal) MkNode(
	ctx context.Context,
	op *fuseops.MkNodeOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	err = fuse.ENOSYS
	return
}

func (fs *readOnlyFsInternal) CreateFile(
	ctx context.Context,
	op *fuseops.CreateFileOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	err = fuse.ENOSYS
	return
}

func (fs *readOnlyFsInternal) CreateSymlink(
	ctx context.Context,
	op *fuseops.CreateSymlinkOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	err = fuse.ENOSYS
	return
}

// Hard links are not supported in datamon.
func (fs *readOnlyFsInternal) CreateLink(
	ctx context.Context,
	op *fuseops.CreateLinkOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	err = fuse.ENOSYS
	return
}

func (fs *readOnlyFsInternal) Rename(
	ctx context.Context,
	op *fuseops.RenameOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	err = fuse.ENOSYS
	return
}

func (fs *readOnlyFsInternal) RmDir(
	ctx context.Context,
	op *fuseops.RmDirOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	err = fuse.ENOSYS
	return
}

func (fs *readOnlyFsInternal) Unlink(
	ctx context.Context,
	op *fuseops.UnlinkOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	err = fuse.ENOSYS
	return
}

func (fs *readOnlyFsInternal) OpenDir(ctx context.Context, op *fuseops.OpenDirOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	p, found := fs.fsEntryStore.Get(formKey(op.Inode))
	if !found {
		err = fuse.ENOENT
		return
	}
	fe := typeAssertToFsEntry(p)
	if isDir(fe) {
		err = fuse.ENOENT
		return
	}
	return nil
}

func (fs *readOnlyFsInternal) ReadDir(ctx context.Context, op *fuseops.ReadDirOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	offset := int(op.Offset)
	iNode := op.Inode

	children, found := fs.readDirMap[iNode]

	if !found {
		err = fuse.ENOENT
		return
	}

	if offset > len(children) {
		err = fuse.ENOENT
		return
	}

	for i := offset; i < len(children); i++ {
		n := fuseutil.WriteDirent(op.Dst[op.BytesRead:], children[i])
		if n == 0 {
			break
		}
		op.BytesRead += n
	}
	return nil
}

func (fs *readOnlyFsInternal) ReleaseDirHandle(
	ctx context.Context,
	op *fuseops.ReleaseDirHandleOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	return
}

func (fs *readOnlyFsInternal) OpenFile(
	ctx context.Context,
	op *fuseops.OpenFileOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	return
}

func (fs *readOnlyFsInternal) ReadFile(
	ctx context.Context,
	op *fuseops.ReadFileOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)

	// If file has not been mutated.
	p, found := fs.fsEntryStore.Get(formKey(op.Inode))
	if !found {
		err = fuse.ENOENT
		return
	}
	fe := typeAssertToFsEntry(p)
	fs.l.Debug("reading file", zap.String("file", fe.fullPath), zap.Uint64("inode", uint64(fe.iNode)))

	n, err := fs.bundle.ReadAt(fe, op.Dst, op.Offset)
	op.BytesRead = n
	return err
}

func (fs *readOnlyFsInternal) WriteFile(
	ctx context.Context,
	op *fuseops.WriteFileOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	err = fuse.ENOSYS
	return
}

func (fs *readOnlyFsInternal) SyncFile(
	ctx context.Context,
	op *fuseops.SyncFileOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	err = fuse.ENOSYS
	return
}

func (fs *readOnlyFsInternal) FlushFile(
	ctx context.Context,
	op *fuseops.FlushFileOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	err = fuse.ENOSYS
	return
}

func (fs *readOnlyFsInternal) ReleaseFileHandle(
	ctx context.Context,
	op *fuseops.ReleaseFileHandleOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	return
}

func (fs *readOnlyFsInternal) ReadSymlink(
	ctx context.Context,
	op *fuseops.ReadSymlinkOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	err = fuse.ENOSYS
	return
}

func (fs *readOnlyFsInternal) RemoveXattr(
	ctx context.Context,
	op *fuseops.RemoveXattrOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	err = fuse.ENOSYS
	return
}

func (fs *readOnlyFsInternal) GetXattr(
	ctx context.Context,
	op *fuseops.GetXattrOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	err = fuse.ENOSYS
	return
}

func (fs *readOnlyFsInternal) ListXattr(
	ctx context.Context,
	op *fuseops.ListXattrOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	err = fuse.ENOSYS
	return
}

func (fs *readOnlyFsInternal) SetXattr(
	ctx context.Context,
	op *fuseops.SetXattrOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
	err = fuse.ENOSYS
	return
}

func (fs *readOnlyFsInternal) Destroy() {
	fs.l.Info("Destroy",
		zap.String("repo", fs.bundle.RepoID),
		zap.String("bundle", fs.bundle.BundleID),
	)
}

func isDir(fsEntry *fsEntry) bool {
	return fsEntry.hash != ""
}

func newDatamonFSEntry(bundleEntry *model.BundleEntry, time time.Time, id fuseops.InodeID, linkCount uint32) *fsEntry {
	var mode os.FileMode = fileReadOnlyMode
	if bundleEntry.Hash == "" {
		mode = dirReadOnlyMode
	}
	return &fsEntry{
		fullPath: bundleEntry.NameWithPath,
		hash:     bundleEntry.Hash,
		iNode:    id,
		attributes: fuseops.InodeAttributes{
			Size:   bundleEntry.Size,
			Nlink:  linkCount,
			Mode:   mode,
			Atime:  time,
			Mtime:  time,
			Ctime:  time,
			Crtime: time,
			Uid:    1020, // TODO: Set to uid gid usable by container..
			Gid:    2000, // TODO: Same as above
		},
	}
}

func generateBundleDirEntry(nameWithPath string) *model.BundleEntry {
	return &model.BundleEntry{
		Hash:         "", // Directories do not have datamon backed hash
		NameWithPath: nameWithPath,
		FileMode:     dirReadOnlyMode,
		Size:         2048, // TODO: Increase size of directory with file count when mount is mutable.
	}
}

/* all the radix trees used during initialization */
type populateFSTxns struct {
	dirStore     *iradix.Txn
	lookupTree   *iradix.Txn
	fsEntryStore *iradix.Txn
}

func (txns *populateFSTxns) commitToFS(fs *readOnlyFsInternal) {
	fs.fsEntryStore = txns.fsEntryStore.Commit()
	fs.lookupTree = txns.lookupTree.Commit()
	fs.fsDirStore = txns.dirStore.Commit()
}

/* unwound recursion to build a list of ents terminating at the first extant parent */
// consider winding up recursion for clarity.(?).
func populateFSBundleEntryToNodes(
	fs *readOnlyFsInternal,
	bundle *Bundle,
	txns *populateFSTxns,
	nodesToAdd []fsNodeToAdd,
	iNode *fuseops.InodeID,
	bundleEntry model.BundleEntry,
) []fsNodeToAdd {

	generateNextINode := func(iNode *fuseops.InodeID) fuseops.InodeID {
		*iNode++
		return *iNode
	}

	be := bundleEntry
	// Generate the fsEntry
	newFsEntry := newDatamonFSEntry(
		&be,
		bundle.BundleDescriptor.Timestamp,
		generateNextINode(iNode),
		fileLinkCount,
	)

	// Add parents if first visit
	// If a parent has been visited, all the parent's parents in the path have been visited
	nameWithPath := be.NameWithPath
	for {
		parentPath := path.Dir(nameWithPath)
		fs.l.Debug("Processing parent",
			zap.String("parentPath", parentPath),
			zap.String("fullPath", be.NameWithPath))
		// entry under root
		if parentPath == "" || parentPath == "." || parentPath == "/" {
			nodesToAdd = append(nodesToAdd, fsNodeToAdd{
				parentINode: fuseops.RootInodeID,
				fsEntry:     *newFsEntry,
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
		})

		if len(nodesToAdd) > 1 {
			// If more than one node is to be added populate the parent iNode.
			nodesToAdd[len(nodesToAdd)-2].parentINode = nodesToAdd[len(nodesToAdd)-1].fsEntry.iNode
		}

		p, found := txns.dirStore.Get([]byte(parentPath))
		if !found {
			fs.l.Debug("parentPath not found",
				zap.String("parent", parentPath))
			newFsEntry = newDatamonFSEntry(
				generateBundleDirEntry(parentPath),
				bundle.BundleDescriptor.Timestamp,
				generateNextINode(iNode),
				dirLinkCount,
			)
			// Continue till we hit root or found
			nameWithPath = parentPath
			continue
		} else {
			fs.l.Debug("parentPath found",
				zap.String("parent", parentPath))
			parentDirEntry := typeAssertToFsEntry(p)
			if len(nodesToAdd) >= 1 {
				nodesToAdd[len(nodesToAdd)-1].parentINode = parentDirEntry.iNode
			}
		}
		fs.l.Debug("last node", zap.String("path", nodesToAdd[len(nodesToAdd)-1].fsEntry.fullPath),
			zap.Uint64("childInode", uint64(nodesToAdd[len(nodesToAdd)-1].fsEntry.iNode)),
			zap.Uint64("parentInode", uint64(nodesToAdd[len(nodesToAdd)-1].parentINode)))
		break
	}
	return nodesToAdd
}

func populateFSAddNodes(
	fs *readOnlyFsInternal,
	txns *populateFSTxns,
	nodesToAdd []fsNodeToAdd,
) error {
	var err error
	for _, nodeToAdd := range nodesToAdd {
		if nodeToAdd.fsEntry.attributes.Nlink == dirLinkCount {
			err = fs.insertDatamonFSDirEntry(
				txns,
				nodeToAdd.parentINode,
				nodeToAdd.fsEntry,
			)

		} else {
			err = fs.insertDatamonFSEntry(
				txns,
				nodeToAdd.parentINode,
				nodeToAdd.fsEntry,
			)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func populateFSAddBundleEntry(
	fs *readOnlyFsInternal,
	bundle *Bundle,
	txns *populateFSTxns,
	nodesToAdd []fsNodeToAdd,
	iNode *fuseops.InodeID,
	bundleEntry model.BundleEntry,
) error {

	nodesToAdd = populateFSBundleEntryToNodes(
		fs,
		bundle,
		txns,
		nodesToAdd,
		iNode,
		bundleEntry,
	)

	fs.l.Debug("Nodes added",
		zap.String("repo", fs.bundle.RepoID),
		zap.String("bundle ID", fs.bundle.BundleID),
		zap.Int("count", len(nodesToAdd)),
	)
	if err := populateFSAddNodes(
		fs,
		txns,
		nodesToAdd,
	); err != nil {
		return err
	}
	return nil
}

func populateFSAddBundleEntries(
	fs *readOnlyFsInternal,
	bundle *Bundle,
	txns *populateFSTxns,
) error {

	// For a Bundle Entry there might be intermediate directories that need adding.
	var nodesToAdd []fsNodeToAdd
	// iNode for fs entries
	var iNode = firstINode

	for _, bundleEntry := range fs.bundle.GetBundleEntries() {
		if err := populateFSAddBundleEntry(
			fs,
			bundle,
			txns,
			nodesToAdd,
			&iNode,
			bundleEntry,
		); err != nil {
			return err
		}
		nodesToAdd = nodesToAdd[:0]
	} // End walking bundle entries q2
	return nil

}

func (fs *readOnlyFsInternal) populateFS(bundle *Bundle) (*ReadOnlyFS, error) {
	txns := new(populateFSTxns)
	txns.dirStore = fs.fsDirStore.Txn()
	txns.lookupTree = fs.lookupTree.Txn()
	txns.fsEntryStore = fs.fsEntryStore.Txn()

	// Add root.
	dirFsEntry := newDatamonFSEntry(
		generateBundleDirEntry(rootPath),
		bundle.BundleDescriptor.Timestamp,
		fuseops.RootInodeID,
		dirLinkCount,
	)
	if err := fs.insertDatamonFSDirEntry(
		txns,
		fuseops.RootInodeID, // Root points to itself
		*dirFsEntry,
	); err != nil {
		return nil, err
	}

	fs.l.Info("Populating fs",
		zap.String("repo", fs.bundle.RepoID),
		zap.String("bundle ID", fs.bundle.BundleID),
		zap.Int("entryCount", len(fs.bundle.BundleEntries)),
	)

	if err := populateFSAddBundleEntries(
		fs,
		bundle,
		txns,
	); err != nil {
		return nil, err
	}

	txns.commitToFS(fs)

	fs.isReadOnly = true
	fs.l.Info("Populating fs done",
		zap.String("repo", fs.bundle.RepoID),
		zap.String("bundle ID", fs.bundle.BundleID),
	)
	return &ReadOnlyFS{
		fsInternal: fs,
		server:     fuseutil.NewFileSystemServer(fs),
	}, nil
}

func (fs *readOnlyFsInternal) insertDatamonFSDirEntry(
	txns *populateFSTxns,
	parentInode fuseops.InodeID,
	dirFsEntry fsEntry) error {

	fs.l.Debug("Inserting FSDirEntry",
		zap.String("fullPath", dirFsEntry.fullPath),
		zap.Uint64("parentInode", uint64(parentInode)))
	_, update := txns.dirStore.Insert([]byte(dirFsEntry.fullPath), dirFsEntry)

	if update {
		return errors.New("dirStore updates are not expected: /" + dirFsEntry.fullPath)
	}

	key := formKey(dirFsEntry.iNode)

	_, update = txns.fsEntryStore.Insert(key, dirFsEntry)
	if update {
		return errors.New("fsEntryStore updates are not expected: /")
	}

	if dirFsEntry.iNode != fuseops.RootInodeID {
		key = formLookupKey(parentInode, path.Base(dirFsEntry.fullPath))

		_, update = txns.lookupTree.Insert(key, dirFsEntry)
		if update {
			fs.l.Error("lookupTree updates are not expected",
				zap.String("fullPath", dirFsEntry.fullPath),
				zap.Uint64("parent iNode", uint64(parentInode)))
			return errors.New("lookupTree updates are not expected: " + dirFsEntry.fullPath)
		}

		childEntries := fs.readDirMap[parentInode]
		childEntries = append(childEntries, fuseutil.Dirent{
			Offset: fuseops.DirOffset(len(childEntries) + 1),
			Inode:  dirFsEntry.iNode,
			Name:   path.Base(dirFsEntry.fullPath),
			Type:   fuseutil.DT_Directory,
		})
		fs.readDirMap[parentInode] = childEntries
	}

	return nil
}

func (fs *readOnlyFsInternal) insertDatamonFSEntry(
	txns *populateFSTxns,
	parentInode fuseops.InodeID,
	fsEntry fsEntry) error {

	fs.l.Debug("adding",
		zap.Uint64("parent", uint64(parentInode)),
		zap.String("fullPath", fsEntry.fullPath),
		zap.Uint64("childInode", uint64(fsEntry.iNode)),
		zap.String("base", path.Base(fsEntry.fullPath)))
	key := formKey(fsEntry.iNode)

	_, update := txns.fsEntryStore.Insert(key, fsEntry)
	if update {
		return errors.New("fsEntryStore updates are not expected: " + fsEntry.fullPath)
	}

	key = formLookupKey(parentInode, path.Base(fsEntry.fullPath))

	_, update = txns.lookupTree.Insert(key, fsEntry)
	if update {
		fs.l.Error("lookupTree updates are not expected",
			zap.String("fullPath", fsEntry.fullPath),
			zap.Uint64("parent iNode", uint64(parentInode)))
		return errors.New("lookupTree updates are not expected: " + fsEntry.fullPath)
	}

	childEntries := fs.readDirMap[parentInode]
	childEntries = append(childEntries, fuseutil.Dirent{
		Offset: fuseops.DirOffset(len(childEntries) + 1),
		Inode:  fsEntry.iNode,
		Name:   path.Base(fsEntry.fullPath),
		Type:   fuseutil.DT_File,
	})
	fs.readDirMap[parentInode] = childEntries

	return nil
}

type readOnlyFsInternal struct {

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
	readDirMap map[fuseops.InodeID][]fuseutil.Dirent

	// readonly
	isReadOnly bool

	// logger
	l *zap.Logger
}

// fsEntry is a node in the filesystem.
type fsEntry struct {
	hash string // Set for files, empty for directories

	// iNode ID is generated on the fly for a bundle that is committed. Since the file list
	// for a bundle is static and the list of files is frozen, multiple mounts of the same
	// bundle will preserve a fixed iNode for a file provided the order of reading the files
	// remains fixed.
	iNode      fuseops.InodeID         // Unique ID for Fuse
	attributes fuseops.InodeAttributes // Fuse Attributes
	fullPath   string
}

type fsNodeToAdd struct {
	parentINode fuseops.InodeID
	fsEntry     fsEntry
}
