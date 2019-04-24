package core

import (
	"context"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/spf13/afero"

	"github.com/hashicorp/go-immutable-radix"
	"github.com/jacobsa/fuse/fuseutil"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"

	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/model"
)

type fsMutable struct {

	// Bundle to commit.
	bundle *Bundle

	// Get fsEntry for an iNode. Speed up stat and other calls keyed by iNode
	iNodeStore *iradix.Tree

	// Fast lookup of parent iNode id + child name, returns iNode of child. This is a common operation and it's speed is
	// important.
	lookupTree *iradix.Tree

	// List of children for a given iNode. Maps inode id to list of children. This stitches the fuse FS together.
	// TODO: This can be based on radix tree as well. Test performance (with locking simplification) and make the change.
	readDirMap map[fuseops.InodeID]map[fuseops.InodeID]*fuseutil.Dirent

	// Cache of backing files.
	backingFiles map[fuseops.InodeID]*afero.File

	// TODO: remove one giant lock with more fine grained locking coupled with the readdir move to radix.
	lock           sync.Mutex
	iNodeGenerator iNodeGenerator

	// local fs cache that mirrors the files.
	localCache afero.Fs

	// Logger
	l *zap.Logger
}

func (fs *fsMutable) StatFS(
	ctx context.Context,
	op *fuseops.StatFSOp) (err error) {
	fs.l.Debug("statfs")
	return statFS()
}

func (fs *fsMutable) atomicGetReferences() (nodeStore *iradix.Tree, lookupTree *iradix.Tree) {
	fs.lock.Lock()
	// Store the old references in case the tree gets updated during lookup. This avoids having the lock for the
	// entire duration of lookup. Semantics for concurrency should hold if the entry is deleted in parallel.
	lookupTree = fs.lookupTree
	nodeStore = fs.iNodeStore
	fs.lock.Unlock()
	return
}

// Lookup against the tree that is referenced. Useful for using an immutable reference of tree.
func lookup(p fuseops.InodeID, c string, lookupTree *iradix.Tree) (le lookupEntry, found bool, lk []byte) {
	lk = formLookupKey(p, c)
	val, found := lookupTree.Get(lk)
	if found {
		le = val.(lookupEntry)
		return le, found, lk
	}
	return lookupEntry{}, found, lk
}

// Lookup an entry and return the right type.
func (fs *fsMutable) lookup(p fuseops.InodeID, c string) (le lookupEntry, found bool, lk []byte) {
	return lookup(p, c, fs.lookupTree)
}

// Delete the entry from namespace only.
func (fs *fsMutable) deleteNSEntry(p fuseops.InodeID, c string) error {
	pn, found := fs.iNodeStore.Get(formKey(p))
	if !found {
		return fuse.ENOENT
	}

	pNode := pn.(*nodeEntry)

	cLE, found, lk := fs.lookup(p, c)
	if !found {
		return fuse.ENOENT
	}

	cn, found := fs.iNodeStore.Get(formKey(cLE.iNode))
	if !found {
		fs.l.Error("Did not find node after lookup", zap.Uint64("childInode", uint64(cLE.iNode)), zap.String("name", c))
		panic(fmt.Sprintf("Did not find node after lookup"))
	}

	cNode := cn.(*nodeEntry)

	if cNode.attr.Mode.IsDir() {
		children := fs.readDirMap[cLE.iNode]
		if len(children) > 0 {
			return fuse.ENOTEMPTY
		}
		// Delete the child dir
		delete(fs.readDirMap, cLE.iNode)
		pNode.attr.Nlink--
	}

	fs.lookupTree, _, _ = fs.lookupTree.Delete(lk)
	children := fs.readDirMap[p]
	// Delete from parent read dir
	delete(children, cLE.iNode)
	return nil
}

func (fs *fsMutable) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) (err error) {

	fs.l.Debug("lookup", zap.Uint64("p", uint64(op.Parent)), zap.String("c", op.Name))

	nodeStore, lookupTree := fs.atomicGetReferences()

	childEntry, found, _ := lookup(op.Parent, op.Name, lookupTree)

	if found {

		v, _ := nodeStore.Get(formKey(childEntry.iNode))

		n := v.(*nodeEntry)
		n.refCount++ // As per LookUpInodeOp spec

		op.Entry.Attributes = n.attr
		op.Entry.Generation = 1
		op.Entry.Child = childEntry.iNode

		// kernel can cache
		op.Entry.AttributesExpiration = time.Now().Add(cacheYearLong)
		op.Entry.EntryExpiration = op.Entry.AttributesExpiration
	} else {
		return fuse.ENOENT
	}
	return nil
}

func (fs *fsMutable) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) (err error) {
	fs.l.Info("getAttr", zap.Uint64("id", uint64(op.Inode)))

	nodeStore, _ := fs.atomicGetReferences()

	key := formKey(op.Inode)

	e, found := nodeStore.Get(key)
	if !found {
		err := fuse.ENOENT
		fs.l.Info("getAttr", zap.Uint64("id", uint64(op.Inode)), zap.Error(err))
		return err
	}

	n := e.(*nodeEntry)
	op.AttributesExpiration = time.Now().Add(cacheYearLong)
	n.lock.Lock()
	op.Attributes = n.attr
	n.lock.Unlock()
	return
}

func (fs *fsMutable) SetInodeAttributes(ctx context.Context, op *fuseops.SetInodeAttributesOp) (err error) {
	fs.l.Info("setAttr", zap.Uint64("id", uint64(op.Inode)))

	if op.Mode != nil { // File permissions not supported
		fs.l.Info("set mode", zap.Uint32("mode", uint32(*op.Mode)))
		return fuse.ENOSYS
	}

	nodeStore, _ := fs.atomicGetReferences()

	// Get the node.
	key := formKey(op.Inode)
	e, found := nodeStore.Get(key)
	if !found {
		return fuse.ENOENT
	}

	n := e.(*nodeEntry)

	// lock the entry
	n.lock.Lock()
	defer n.lock.Unlock()

	// Set the values
	if op.Size != nil {
		// File size can be truncated.
		file, err := fs.localCache.OpenFile(fmt.Sprint(op.Inode), os.O_WRONLY|os.O_SYNC, fileDefaultMode)
		if err != nil {
			fs.l.Error("error", zap.Error(err))
			return fuse.EIO
		}
		if *op.Size > math.MaxInt64 {
			fs.l.Error("Received size greater than MaxInt64", zap.Uint64("size", *op.Size), zap.Uint64("inode", uint64(op.Inode)))
			return fuse.EINVAL
		}
		err = file.Truncate(int64(*op.Size))
		if err != nil {
			fs.l.Error("error", zap.Error(err))
			return fuse.EIO
		}
		n.attr.Size = *op.Size
	}

	if op.Atime != nil {
		n.attr.Atime = *op.Atime
	}

	if op.Mtime != nil {
		n.attr.Mtime = *op.Mtime
	}

	op.AttributesExpiration = time.Now().Add(cacheYearLong)

	// Send new attr back
	op.Attributes = n.attr
	return nil
}

func (fs *fsMutable) ForgetInode(
	ctx context.Context,
	op *fuseops.ForgetInodeOp) (err error) {

	fs.l.Info("forgetInode", zap.Uint64("id", uint64(op.Inode)))

	// Check reference count for iNode and remove from iNodeStore
	// Get the node.
	nodeStore, _ := fs.atomicGetReferences()
	key := formKey(op.Inode)
	e, found := nodeStore.Get(key)
	if !found {
		fs.l.Error("ForgetInode inode not found", zap.Uint64("inode", uint64(op.Inode)))
		panic(fmt.Sprintf("not found iNode:%d", op.Inode))
	}

	var del bool
	n := e.(*nodeEntry)

	n.lock.Lock()
	n.refCount--

	if n.refCount < 0 {
		panic(fmt.Sprintf("RefCount below zero %d", op.Inode))
	}

	del = shouldDelete(n)

	//Explicitly release lock.
	n.lock.Unlock()

	if del {

		fs.lock.Lock()
		defer fs.lock.Unlock()

		e, found := fs.iNodeStore.Get(key)
		if !found {
			return
		}
		n := e.(*nodeEntry)
		if shouldDelete(n) {
			fs.iNodeStore, _, _ = fs.iNodeStore.Delete(key)
			fs.l.Info("NodeStore", zap.Int("Size", fs.iNodeStore.Len()))
		}
		fs.iNodeGenerator.freeINode(op.Inode)
	}
	return nil
}

func (fs *fsMutable) MkDir(
	ctx context.Context,
	op *fuseops.MkDirOp) (err error) {
	fs.l.Info("mkdir", zap.Uint64("id", uint64(op.Parent)), zap.String("name", op.Name))

	fs.lock.Lock()
	defer fs.lock.Unlock()

	lk := formLookupKey(op.Parent, op.Name)

	err = fs.preCreateCheck(op.Parent, lk)
	if err != nil {
		fs.lock.Unlock()
		return
	}

	err = fs.createNode(lk, op.Parent, op.Name, &op.Entry, fuseutil.DT_Directory, false)
	return
}

// TODO: Should file and dir node be supported via this call? So far no..
func (fs *fsMutable) MkNode(
	ctx context.Context,
	op *fuseops.MkNodeOp) (err error) {
	fs.l.Info("mknode", zap.Uint64("id", uint64(op.Parent)), zap.String("name", op.Name))
	err = fuse.ENOSYS
	return
}

func (fs *fsMutable) CreateFile(
	ctx context.Context,
	op *fuseops.CreateFileOp) (err error) {

	fs.l.Info("createFile", zap.Uint64("id", uint64(op.Parent)), zap.String("name", op.Name))

	// TODO: Implement a CAFS friendly store. That will chunk file at leaf size and on commit, move the chunks into
	// blob cache.
	// ??? chunking occurs on disk or in memory?

	fs.lock.Lock()
	defer fs.lock.Unlock()

	lk := formLookupKey(op.Parent, op.Name)

	err = fs.preCreateCheck(op.Parent, lk)

	if err != nil {
		return
	}

	err = fs.createNode(lk, op.Parent, op.Name, &op.Entry, fuseutil.DT_File, false)
	return
}

// No sym link support in datamon
func (fs *fsMutable) CreateSymlink(
	ctx context.Context,
	op *fuseops.CreateSymlinkOp) (err error) {
	fs.l.Info("createSymLink", zap.Uint64("id", uint64(op.Parent)), zap.String("name", op.Name))
	err = fuse.ENOSYS
	return
}

// no create link support in datamon
func (fs *fsMutable) CreateLink(
	ctx context.Context,
	op *fuseops.CreateLinkOp) (err error) {
	fs.l.Info("createLink", zap.Uint64("id", uint64(op.Parent)), zap.String("name", op.Name))
	err = fuse.ENOSYS
	return
}

// From man 2 rename:
// If newpath exists but the operation fails for some reason, rename() guarantees to leave an instance of newpath in place.
// oldpath can specify a directory.  In this case, newpath must either not exist, or it must specify an empty directory.
func (fs *fsMutable) Rename(ctx context.Context, op *fuseops.RenameOp) (err error) {

	fs.l.Info("rename", zap.Uint64("oldP", uint64(op.OldParent)), zap.String("oldN", op.OldName),
		zap.Uint64("nP", uint64(op.NewParent)), zap.String("nN", op.NewName))

	fs.lock.Lock()
	defer fs.lock.Unlock()

	// Find the old child
	oldChild, found, _ := fs.lookup(op.OldParent, op.OldName)
	if !found {
		return fuse.ENOENT
	}
	newChild, found, _ := fs.lookup(op.NewParent, op.NewName)
	if found {
		if newChild.mode.IsDir() {
			return fuse.ENOSYS
		}
		// Delete new child, ignore if not present
		_ = fs.deleteNSEntry(op.NewParent, op.NewName)
	}

	// Insert iNode into new readDir and lookup and remove from old.
	rC := fs.readDirMap[op.OldParent][oldChild.iNode]

	newRC := fuseutil.Dirent{
		Inode: rC.Inode,
		Name:  op.NewName,
		Type:  rC.Type,
	}

	// Delete from old parent
	delete(fs.readDirMap[op.OldParent], rC.Inode)
	var l interface{}
	fs.lookupTree, l, _ = fs.lookupTree.Delete(formLookupKey(op.OldParent, op.OldName)) // lookupEntry remains the same

	// Insert into new.
	fs.insertReadDirEntry(op.NewParent, &newRC)
	fs.insertLookupEntry(op.NewParent, op.NewName, l.(lookupEntry))

	return nil
}

func (fs *fsMutable) RmDir(
	ctx context.Context,
	op *fuseops.RmDirOp) (err error) {
	fs.l.Info("rmdir", zap.Uint64("id", uint64(op.Parent)), zap.String("name", op.Name))

	return fs.deleteNSEntry(op.Parent, op.Name)
}

func (fs *fsMutable) Unlink(
	ctx context.Context,
	op *fuseops.UnlinkOp) (err error) {
	fs.l.Info("unlink", zap.Uint64("id", uint64(op.Parent)), zap.String("name", op.Name))
	// TODO: remove from lookup and readdir
	return fs.deleteNSEntry(op.Parent, op.Name)
}

func (fs *fsMutable) OpenDir(
	ctx context.Context,
	op *fuseops.OpenDirOp) (err error) {
	fs.l.Info("openDir", zap.Uint64("id", uint64(op.Inode)))
	return
}

func (fs *fsMutable) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) (err error) {
	fs.l.Info("readDir", zap.Uint64("id", uint64(op.Inode)))

	offset := int(op.Offset)
	iNode := op.Inode

	fs.lock.Lock()
	defer fs.lock.Unlock()
	children, found := fs.readDirMap[iNode]

	if !found {
		return fuse.ENOENT
	}

	if offset > len(children) {
		return
	}

	var i uint64 = 1
	for _, c := range children {
		i++
		if i < uint64(offset) {
			continue
		}
		child := *c
		child.Offset = fuseops.DirOffset(i) // This is where dirOffset matters..
		n := fuseutil.WriteDirent(op.Dst[op.BytesRead:], child)
		if n == 0 {
			break
		}
		op.BytesRead += n
	}
	return nil
}

func (fs *fsMutable) ReleaseDirHandle(
	ctx context.Context,
	op *fuseops.ReleaseDirHandleOp) (err error) {
	fs.l.Info("releaseDir", zap.Uint64("id", uint64(op.Handle)))
	return
}

func (fs *fsMutable) OpenFile(
	ctx context.Context,
	op *fuseops.OpenFileOp) (err error) {
	fs.l.Info("openFile", zap.Uint64("id", uint64(op.Inode)))
	return
}

func (fs *fsMutable) ReadFile(
	ctx context.Context,
	op *fuseops.ReadFileOp) (err error) {
	fs.l.Info("readFile", zap.Uint64("id", uint64(op.Inode)))
	file, err := fs.localCache.OpenFile(getPathToBackingFile(op.Inode), os.O_RDONLY|os.O_SYNC, fileDefaultMode)
	if err != nil {
		return fuse.EIO
	}
	fs.backingFiles[op.Inode] = &file
	op.BytesRead, err = file.ReadAt(op.Dst, op.Offset)
	if err != nil {
		return fuse.EIO
	}
	return
}

func (fs *fsMutable) WriteFile(
	ctx context.Context,
	op *fuseops.WriteFileOp) (err error) {
	fs.l.Info("writeFile", zap.Uint64("id", uint64(op.Inode)))
	file, err := fs.localCache.OpenFile(getPathToBackingFile(op.Inode), os.O_WRONLY|os.O_SYNC, fileDefaultMode)
	if err != nil {
		return fuse.EIO
	}
	fs.backingFiles[op.Inode] = &file
	_, err = file.WriteAt(op.Data, op.Offset)
	if err != nil {
		return fuse.EIO
	}
	ne, found := fs.iNodeStore.Get(formKey(op.Inode))
	if !found {
		panic("Invalid state inode: not found" + fmt.Sprint(uint64(op.Inode)))
	}
	nodeEntry := ne.(*nodeEntry)
	nodeEntry.lock.Lock()
	s, _ := file.Stat()
	nodeEntry.attr.Size = uint64(s.Size())
	nodeEntry.lock.Unlock()
	return
}

func (fs *fsMutable) SyncFile(
	ctx context.Context,
	op *fuseops.SyncFileOp) (err error) {
	fs.l.Info("syncFile", zap.Uint64("id", uint64(op.Inode)))
	file := *fs.backingFiles[op.Inode]
	if file != nil {
		err := file.Sync()
		if err != nil {
			return fuse.EIO
		}
	}
	return
}

func (fs *fsMutable) FlushFile(
	ctx context.Context,
	op *fuseops.FlushFileOp) (err error) {
	fs.l.Info("syncFile", zap.Uint64("id", uint64(op.Inode)))
	f := fs.backingFiles[op.Inode]
	if f != nil {
		file := *f
		err := file.Sync()
		if err != nil {
			return fuse.EIO
		}
	}
	return
}

func (fs *fsMutable) ReleaseFileHandle(
	ctx context.Context,
	op *fuseops.ReleaseFileHandleOp) (err error) {
	fs.l.Info("releaseFileHandle", zap.Uint64("hndl", uint64(op.Handle)))
	return
}

func (fs *fsMutable) ReadSymlink(
	ctx context.Context,
	op *fuseops.ReadSymlinkOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *fsMutable) RemoveXattr(
	ctx context.Context,
	op *fuseops.RemoveXattrOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *fsMutable) GetXattr(
	ctx context.Context,
	op *fuseops.GetXattrOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *fsMutable) ListXattr(
	ctx context.Context,
	op *fuseops.ListXattrOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *fsMutable) SetXattr(
	ctx context.Context,
	op *fuseops.SetXattrOp) (err error) {
	err = fuse.ENOSYS
	return
}

func (fs *fsMutable) Destroy() {
}

type commit_chans struct {
	// recv data from goroutines about uploaded files
	bundleEntry chan<- model.BundleEntry
	error       chan<- error
	// broadcast to all goroutines not to block by closing this channel
	done <-chan struct{}
}

type commit_uploadTask struct {
	inodeID fuseops.InodeID
	name    string
}

// todo: pre-chunk as mentioned in  `CreateFile` TODO
func commit_fileUpload(
	fs *fsMutable,
	ctx context.Context,
	chans commit_chans,
	bundleUploadWaitGroup *sync.WaitGroup,
	caFs cafs.Fs,
	uploadTask commit_uploadTask) {
	defer bundleUploadWaitGroup.Done()
	file, err := fs.localCache.OpenFile(getPathToBackingFile(uploadTask.inodeID),
		os.O_RDONLY|os.O_SYNC, fileDefaultMode)
	if err != nil {
		select {
		case chans.error <- err:
		case <-chans.done:
		}
		return
	}
	// written, key, keys, duplicate, err =
	written, key, _, _, err := caFs.Put(ctx, file)
	if err != nil {
		select {
		case chans.error <- err:
		case <-chans.done:
		}
		return
	}
	be := model.BundleEntry{
		Hash:         key.String(),
		NameWithPath: uploadTask.name,
		FileMode:     0, // #TODO: #35 file mode support
		Size:         uint64(written),
	}
	select {
	case chans.bundleEntry <- be:
	case <-chans.done:
	}

}

/* these are the concurrency primitives used to get bounded concurrency in the
 * directory upload.  the idea of using a buffered channel to set a bounds on concurrency is
 * from, for example, TestTCPSpuriousConnSetupCompletionWithCancel in the stdlib net package.
 */
type commit_dirUploadSync struct {
	waitGroup       *sync.WaitGroup
	bufferedChanSem chan struct{}
}

func commit_dirUpload(
	fs *fsMutable,
	ctx context.Context,
	chans commit_chans,
	bundleUploadWaitGroup *sync.WaitGroup,
	caFs cafs.Fs,
	dirUploadSync commit_dirUploadSync,
	uploadTask commit_uploadTask) {
	defer dirUploadSync.waitGroup.Done()
	var directoryUploadTasks []commit_uploadTask
	func() {
		defer func() { <-dirUploadSync.bufferedChanSem }()
		directoryUploadTasks = make([]commit_uploadTask, 0)
		for currInode, currEnt := range fs.readDirMap[uploadTask.inodeID] {
			tsk := commit_uploadTask{inodeID: currInode, name: uploadTask.name + "/" + currEnt.Name}
			if currEnt.Type == fuseutil.DT_File {
				bundleUploadWaitGroup.Add(1)
				go commit_fileUpload(
					fs,
					ctx,
					chans,
					bundleUploadWaitGroup,
					caFs,
					tsk)
			} else if currEnt.Type == fuseutil.DT_Directory {
				directoryUploadTasks = append(directoryUploadTasks, tsk)
			} else {
				fs.l.Warn("unexpected file type")
			}
		}
	}()
	for _, dutsk := range directoryUploadTasks {
		select {
		case dirUploadSync.bufferedChanSem <- struct{}{}:
		case <-chans.done:
			return
		}
		dirUploadSync.waitGroup.Add(1)
		go commit_dirUpload(
			fs,
			ctx,
			chans,
			bundleUploadWaitGroup,
			caFs,
			dirUploadSync,
			dutsk,
		)
	}
}

const maxDirUploadTasks = 4 // approximate number of cores
func commit_walkReadDirMap(
	fs *fsMutable,
	ctx context.Context,
	chans commit_chans,
	caFs cafs.Fs) {
	// bundle upload wait group: used to wait for all file upload operations to complete
	bundleUploadWaitGroup := new(sync.WaitGroup)
	// directory upload wait group: used to wait for all directory upload operations to complete
	dirUploadSync := commit_dirUploadSync{
		waitGroup:       new(sync.WaitGroup),
		bufferedChanSem: make(chan struct{}, maxDirUploadTasks),
	}
	defer func() {
		dirUploadSync.waitGroup.Wait()
		bundleUploadWaitGroup.Wait()
		close(chans.bundleEntry)
	}()
	dirUploadSync.bufferedChanSem <- struct{}{}
	dirUploadSync.waitGroup.Add(1)
	commit_dirUpload(
		fs,
		ctx,
		chans,
		bundleUploadWaitGroup,
		caFs,
		dirUploadSync,
		commit_uploadTask{inodeID: fuseops.RootInodeID, name: ""})
}

// starting from root, find each file and upload using go routines.
func (fs *fsMutable) Commit() error {
	/* some sync setup */
	if fs.bundle.BundleID == "" {
		if err := fs.bundle.InitializeBundleID(); err != nil {
			return err
		}
	}
	// ??? allocate caFs here or in each goroutine?
	caFs, err := cafs.New(
		cafs.LeafSize(fs.bundle.BundleDescriptor.LeafSize),
		cafs.Backend(fs.bundle.BlobStore),
	)
	if err != nil {
		return err
	}
	ctx := context.Background() // ??? is this the correct context?
	/* `commit_chans` includes rules about directionality that apply to threads only,
	 * so we keep channels without directionality restriction separately.
	 */
	bundleEntryC := make(chan model.BundleEntry)
	errorC := make(chan error)
	doneC := make(chan struct{})
	/* closing the done channel broadcasts to all threads and is particularly important to prevent
	 * goroutine leaks in the case of an error from any particular thread.
	 * see the "Explicit cancellation" section of https://blog.golang.org/pipelines
	 * for more detailed description on using closure of a channel to broadcast errors.
	 *
	 * using defer to cleanup concurrency and similar resource usage is preferred throughout,
	 * both as a stylistic hint that what's going on is cleanup as well as for the practical reason
	 * that deferred calls still occur, even in the case of a panic(), as described in the blog post
	 * https://blog.golang.org/defer-panic-and-recover
	 */
	defer close(doneC)
	/* `commit_walkReadDirMap` signals that it's done to the caller by closing the channel containing
	 * `model.BundleEntry` instances.  since the goal of the commit is to produce a sequence of bundle uploads,
	 * and since the second parameter to reading from a channel is false when the channel is both empty and closed,
	 * this thread can use reading from the bundle entry channel to detect whether the walk is finished.
	 */
	go commit_walkReadDirMap(fs, ctx, commit_chans{
		bundleEntry: bundleEntryC,
		error:       errorC,
		done:        doneC,
	}, caFs)
	fileList := make([]model.BundleEntry, 0)
	for {
		var bundleEntry model.BundleEntry
		var moreBundleEntries bool
		select {
		case bundleEntry, moreBundleEntries = <-bundleEntryC:
		case err := <-errorC:
			// one of the threads has had an error.
			return err
		}
		if !moreBundleEntries {
			break
		}
		fileList = append(fileList, bundleEntry)
	}
	for i := 0; i*bundleEntriesPerFile < len(fileList); i++ {
		firstIdx := i * bundleEntriesPerFile
		nextFirstIdx := (i + 1) * bundleEntriesPerFile
		if nextFirstIdx < len(fileList) {
			if err := uploadBundleEntriesFileList(ctx, fs.bundle, fileList[firstIdx:nextFirstIdx]); err != nil {
				return err
			}
		} else {
			if err := uploadBundleEntriesFileList(ctx, fs.bundle, fileList[firstIdx:]); err != nil {
				return err
			}
		}
	}
	if err := uploadBundleDescriptor(ctx, fs.bundle); err != nil {
		return err
	}
	return nil
}
