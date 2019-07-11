package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"runtime"
	"path/filepath"
	"runtime/pprof"
	"strconv"

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
/*
	fs.l.Info("Start",
		zap.String("Request", fmt.Sprintf("%T", op)),
		zap.String("repo", fs.bundle.RepoID),
		zap.String("bundle", fs.bundle.BundleID),
		zap.Any("op", op),
	)
*/
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
/*
	fs.l.Info("End",
		zap.String("Request", fmt.Sprintf("%T", op)),
		zap.String("repo", fs.bundle.RepoID),
		zap.String("bundle", fs.bundle.BundleID),
		zap.Any("op", op),
		zap.Error(err),
	)
*/
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
//	fs.opStart(op)
//	defer fs.opEnd(op, err)
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
//	fs.opStart(op)

	op.KeepPageCache = true
	op.UseDirectIO = true

/*
	var mstats runtime.MemStats
	runtime.ReadMemStats(&mstats)

	fs.l.Info("open file",
		zap.Uint64("MiB from os", mstats.Sys / 1024 / 1024),
		zap.Uint64("MiB for heap (un-GC)", mstats.Alloc / 1024 / 1024),
		zap.Uint64("MiB for heap (max ever)", mstats.HeapSys / 1024 / 1024),
	)
*/


//	defer fs.opEnd(op, err)
	return
}

// var fuseRoReadFileProfIdx int = 0

var fuseRoReadFileConcurrencyControlC chan struct{}

func init() {
	fuseRoReadFileConcurrencyControlC = make(chan struct{}, 8)
}

func readFileMaybeProf(mstats runtime.MemStats, minAllocMB uint64, minHeapSysMB uint64) {
	const memprofdest = "/home/developer/"
	if mstats.Alloc / 1024 / 1024 < minAllocMB || mstats.HeapSys / 1024 / 1024 < minHeapSysMB {
		return
	}
	if _, err := os.Stat(memprofdest); !os.IsNotExist(err) {
		basePath := filepath.Join(memprofdest, "read_file-" + strconv.Itoa(int(minAllocMB)))
		profPath := basePath + ".mem.prof"
		allocPath := basePath + ".alloc.prof"
		if _, err := os.Stat(profPath); os.IsNotExist(err) {
			var fprof *os.File
			fprof, err = os.Create(profPath)
			if err != nil {
				return
			}
			defer fprof.Close()
			err = pprof.Lookup("heap").WriteTo(fprof, 0)
			if err != nil {
				return
			}
		}
		if _, err := os.Stat(allocPath); os.IsNotExist(err) {
			var falloc *os.File
			falloc, err = os.Create(allocPath)
			if err != nil {
				return
			}
			defer falloc.Close()
			err = pprof.Lookup("allocs").WriteTo(falloc, 0)
			if err != nil {
				return
			}
		}
	}
}

func (fs *readOnlyFsInternal) ReadFile(
	ctx context.Context,
	op *fuseops.ReadFileOp) (err error) {

//	fuseRoReadFileConcurrencyControlC <- struct{}{}
//	defer func() { <-fuseRoReadFileConcurrencyControlC }()

//	fs.opStart(op)
//	defer fs.opEnd(op, err)

	// If file has not been mutated.
	p, found := fs.fsEntryStore.Get(formKey(op.Inode))
	if !found {
		err = fuse.ENOENT
		return
	}
	fe := typeAssertToFsEntry(p)
	fs.l.Debug("reading file",
		zap.String("file", fe.fullPath),
		zap.Uint64("inode", uint64(fe.iNode)),
	)

	n, err := fs.bundle.ReadAt(fe, op.Dst, op.Offset)
	op.BytesRead = n

	var mstats runtime.MemStats
	runtime.ReadMemStats(&mstats)

/*
	fs.l.Info("read file",
		zap.String("file", fe.fullPath),
		zap.String("hash", fe.hash),
		zap.Int("buffer size", len(op.Dst)),
		zap.Int("reported bytes read", n),
		zap.Error(err),
		zap.Uint64("MiB from os", mstats.Sys / 1024 / 1024),
		zap.Uint64("MiB for heap (un-GC)", mstats.Alloc / 1024 / 1024),
		zap.Uint64("MiB for heap (max ever)", mstats.HeapSys / 1024 / 1024),
	)
*/

	for i := 21; i < 25; i++ {
		readFileMaybeProf(mstats, 0, uint64(i * 1000))
	}
	for i := 12; i < 21; i++ {
		readFileMaybeProf(mstats, uint64(i * 1000), 0)
	}

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
//	err = fuse.ENOSYS
	return
}

func (fs *readOnlyFsInternal) ReleaseFileHandle(
	ctx context.Context,
	op *fuseops.ReleaseFileHandleOp) (err error) {
//	fs.opStart(op)

/*
	var mstats runtime.MemStats
	runtime.ReadMemStats(&mstats)

	fs.l.Info("close file",
		zap.Uint64("MiB from os", mstats.Sys / 1024 / 1024),
		zap.Uint64("MiB for heap (un-GC)", mstats.Alloc / 1024 / 1024),
		zap.Uint64("MiB for heap (max ever)", mstats.HeapSys / 1024 / 1024),
	)
*/

//	defer fs.opEnd(op, err)
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
//	err = fuse.ENOSYS
	return
}

func (fs *readOnlyFsInternal) ListXattr(
	ctx context.Context,
	op *fuseops.ListXattrOp) (err error) {
	fs.opStart(op)
	defer fs.opEnd(op, err)
//	err = fuse.ENOSYS
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

func newDatamonFSEntry(
	bundleEntry *model.BundleEntry,
	time time.Time,
	id fuseops.InodeID,
	linkCount uint32,
	streamed bool,
) (*fsEntry, string) {
	var mode os.FileMode = fileReadOnlyMode
	if bundleEntry.Hash == "" {
		mode = dirReadOnlyMode
	}
	entry := fsEntry{
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
			Uid:    1020, // TODO: Set to uid gid usable by container..
			Gid:    2000, // TODO: Same as above
		},
	}
	if !streamed {
		entry.fullPath = bundleEntry.NameWithPath
	}
	return &entry, bundleEntry.NameWithPath
}

func generateBundleDirEntry(nameWithPath string) *model.BundleEntry {
	return &model.BundleEntry{
		Hash:         "", // Directories do not have datamon backed hash
		NameWithPath: nameWithPath,
		FileMode:     dirReadOnlyMode,
		Size:         2048, // TODO: Increase size of directory with file count when mount is mutable.
	}
}

func (fs *readOnlyFsInternal) populateFS(bundle *Bundle) (*ReadOnlyFS, error) {
	// Get iNode for path. This is needed to generate directory entries without imposing a strict order of traversal.
	fsDirStore := iradix.New()

	dirStoreTxn := fsDirStore.Txn()
	lookupTreeTxn := fs.lookupTree.Txn()
	fsEntryStoreTxn := fs.fsEntryStore.Txn()

	// Add root.
	dirFsEntry, dirFsFullPath := newDatamonFSEntry(generateBundleDirEntry(rootPath),
		bundle.BundleDescriptor.Timestamp, fuseops.RootInodeID, dirLinkCount,
		bundle.Streamed)
	err := fs.insertDatamonFSDirEntry(
		dirStoreTxn,
		lookupTreeTxn,
		fsEntryStoreTxn,
		fuseops.RootInodeID, // Root points to itself
		*dirFsEntry,
		dirFsFullPath,
	)
	if err != nil {
		return nil, err
	}

	// For a Bundle Entry there might be intermediate directories that need adding.
	var nodesToAdd []fsNodeToAdd
	// iNode for fs entries
	var iNode = firstINode

	generateNextINode := func(iNode *fuseops.InodeID) fuseops.InodeID {
		*iNode++
		return *iNode
	}

	fs.l.Info("top populateFS()",
		zap.String("repo", fs.bundle.RepoID),
		zap.String("bundle ID", fs.bundle.BundleID),
		zap.Int("entryCount", len(fs.bundle.BundleEntries)),
	)
	var count int
	for _, bundleEntry := range fs.bundle.BundleEntries {
		be := bundleEntry
		// Generate the fsEntry
		newFsEntry, fsNodeToAddFullPath := newDatamonFSEntry(&be, bundle.BundleDescriptor.Timestamp,
			generateNextINode(&iNode), fileLinkCount, bundle.Streamed)

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
					fullPath:    fsNodeToAddFullPath,
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
				fullPath:    fsNodeToAddFullPath,
			})

			if len(nodesToAdd) > 1 {
				// If more than one node is to be added populate the parent iNode.
				nodesToAdd[len(nodesToAdd)-2].parentINode = nodesToAdd[len(nodesToAdd)-1].fsEntry.iNode
			}

			p, found := dirStoreTxn.Get([]byte(parentPath))
			if !found {
				fs.l.Debug("parentPath not found",
					zap.String("parent", parentPath))
				newFsEntry, fsNodeToAddFullPath = newDatamonFSEntry(generateBundleDirEntry(parentPath),
					bundle.BundleDescriptor.Timestamp, generateNextINode(&iNode), dirLinkCount,
					bundle.Streamed)

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
			fs.l.Debug("last node", zap.String("path", nodesToAdd[len(nodesToAdd)-1].fullPath),
				zap.Uint64("childInode", uint64(nodesToAdd[len(nodesToAdd)-1].fsEntry.iNode)),
				zap.Uint64("parentInode", uint64(nodesToAdd[len(nodesToAdd)-1].parentINode)))
			break
		}

		fs.l.Debug("Nodes added",
			zap.String("repo", fs.bundle.RepoID),
			zap.String("bundle ID", fs.bundle.BundleID),
			zap.Int("count", len(nodesToAdd)),
		)
		for _, nodeToAdd := range nodesToAdd {
			if nodeToAdd.fsEntry.attributes.Nlink == dirLinkCount {
				err = fs.insertDatamonFSDirEntry(
					dirStoreTxn,
					lookupTreeTxn,
					fsEntryStoreTxn,
					nodeToAdd.parentINode,
					nodeToAdd.fsEntry,
					nodeToAdd.fullPath,
				)

			} else {
				count++
				err = fs.insertDatamonFSEntry(
					lookupTreeTxn,
					fsEntryStoreTxn,
					nodeToAdd.parentINode,
					nodeToAdd.fsEntry,
					nodeToAdd.fullPath,
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
	fsDirStore = dirStoreTxn.Commit() // nolint: staticcheck
	fs.isReadOnly = true
	fs.l.Info("Populating fs done",
		zap.String("repo", fs.bundle.RepoID),
		zap.String("bundle ID", fs.bundle.BundleID),
		zap.Int("entryCount", count),
	)
	return &ReadOnlyFS{
		fsInternal: fs,
		server:     fuseutil.NewFileSystemServer(fs),
	}, nil
}

func (fs *readOnlyFsInternal) insertDatamonFSDirEntry(
	dirStoreTxn *iradix.Txn,
	lookupTreeTxn *iradix.Txn,
	fsEntryStoreTxn *iradix.Txn,
	parentInode fuseops.InodeID,
	dirFsEntry fsEntry,
	fullPath string,
) error {
	fs.l.Debug("Inserting FSDirEntry",
		zap.String("fullPath", fullPath),
		zap.Uint64("parentInode", uint64(parentInode)))
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
			fs.l.Error("lookupTree updates are not expected",
				zap.String("fullPath", fullPath),
				zap.Uint64("parent iNode", uint64(parentInode)))
			return errors.New("lookupTree updates are not expected: " + fullPath)
		}

		childEntries := fs.readDirMap[parentInode]
		childEntries = append(childEntries, fuseutil.Dirent{
			Offset: fuseops.DirOffset(len(childEntries) + 1),
			Inode:  dirFsEntry.iNode,
			Name:   path.Base(fullPath),
			Type:   fuseutil.DT_Directory,
		})
		fs.readDirMap[parentInode] = childEntries
	}

	return nil
}

func (fs *readOnlyFsInternal) insertDatamonFSEntry(
	lookupTreeTxn *iradix.Txn,
	fsEntryStoreTxn *iradix.Txn,
	parentInode fuseops.InodeID,
	fsEntry fsEntry,
	fullPath string,
) error {
	fs.l.Debug("adding",
		zap.Uint64("parent", uint64(parentInode)),
		zap.String("fullPath", fullPath),
		zap.Uint64("childInode", uint64(fsEntry.iNode)),
		zap.String("base", path.Base(fullPath)))
	key := formKey(fsEntry.iNode)

	_, update := fsEntryStoreTxn.Insert(key, fsEntry)
	if update {
		return errors.New("fsEntryStore updates are not expected: " + fullPath)
	}

	key = formLookupKey(parentInode, path.Base(fullPath))

	_, update = lookupTreeTxn.Insert(key, fsEntry)
	if update {
		fs.l.Error("lookupTree updates are not expected",
			zap.String("fullPath", fullPath),
			zap.Uint64("parent iNode", uint64(parentInode)))
		return errors.New("lookupTree updates are not expected: " + fullPath)
	}

	childEntries := fs.readDirMap[parentInode]
	childEntries = append(childEntries, fuseutil.Dirent{
		Offset: fuseops.DirOffset(len(childEntries) + 1),
		Inode:  fsEntry.iNode,
		Name:   path.Base(fullPath),
		Type:   fuseutil.DT_File,
	})
	fs.readDirMap[parentInode] = childEntries

	return nil
}

type readOnlyFsInternal struct {

	// Backing bundle for this FS.
	bundle *Bundle

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
	fullPath    string
}
