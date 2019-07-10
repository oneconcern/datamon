package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
	"path/filepath"

	"github.com/jacobsa/fuse/fuseops"

	"github.com/spf13/afero"
	"go.uber.org/zap"

	iradix "github.com/hashicorp/go-immutable-radix"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseutil"
)

const (
	// Cache duration
	cacheYearLong                    = 365 * 24 * time.Hour
	dirLinkCount     uint32          = 2
	fileLinkCount    uint32          = 1
	rootPath                         = "/"
	firstINode       fuseops.InodeID = 1023
	dirDefaultMode                   = 0777 | os.ModeDir
	fileDefaultMode                  = 0666
	dirReadOnlyMode                  = 0555 | os.ModeDir
	fileReadOnlyMode                 = 0444
	defaultUID                       = 0
	defaultGID                       = 0
	dirInitialSize                   = 64
)

// ReadOnlyFS is the virtual filesystem created on top of a bundle.
type ReadOnlyFS struct {
	mfs        *fuse.MountedFileSystem // The mounted filesystem
	fsInternal *readOnlyFsInternal     // The core of the filesystem
	server     fuse.Server             // Fuse server
}

// ReadOnlyFS is the virtual filesystem created on top of a bundle.
type MutableFS struct {
	mfs        *fuse.MountedFileSystem // The mounted filesystem
	fsInternal *fsMutable              // The core of the filesystem
	server     fuse.Server             // Fuse server
}

// NewReadOnlyFS creates a new instance of the datamon filesystem.
func NewReadOnlyFS(bundle *Bundle, l *zap.Logger) (*ReadOnlyFS, error) {
	if l == nil {
		return nil, fmt.Errorf("logger is nil")
	}
	if bundle == nil {
		err := fmt.Errorf("bundle is nil")
		l.Error("bundle is nil", zap.Error(err))
		return nil, err
	}
	fsInternal := &readOnlyFsInternal{
		bundle:       bundle,
		readDirMap:   make(map[fuseops.InodeID][]fuseutil.Dirent),
		fsEntryStore: iradix.New(),
		lookupTree:   iradix.New(),
		l:            l,
	}

	// Extract the meta information needed.
	err := Publish(context.Background(), fsInternal.bundle)
	if err != nil {
		l.Error("Failed to publish bundle", zap.String("id", bundle.BundleID),
			zap.Error(err))
		return nil, err
	}
	// TODO: Introduce streaming and caching
	// Populate the filesystem.
	l.Info("initializing read-only fs internal")
	fsExternal, err := fsInternal.populateFS(bundle)
	if err != nil {
		return nil, err
	}
	// nil is the preferred idiom for empty slices
	// https://github.com/golang/go/wiki/CodeReviewComments#declaring-empty-slices
	// todo: verify whether this line successfully frees memory
	bundle.BundleEntries = nil
	return fsExternal, nil
}

// NewMutableFS creates a new instance of the datamon filesystem.
func NewMutableFS(bundle *Bundle, pathToStaging string) (*MutableFS, error) {
	logger, _ := zap.NewProduction()
	fs := &fsMutable{
		bundle:       bundle,
		readDirMap:   make(map[fuseops.InodeID]map[fuseops.InodeID]*fuseutil.Dirent),
		iNodeStore:   iradix.New(),
		lookupTree:   iradix.New(),
		backingFiles: make(map[fuseops.InodeID]*afero.File),
		lock:         sync.Mutex{},
		iNodeGenerator: iNodeGenerator{
			lock:         sync.Mutex{},
			highestInode: firstINode,
			freeInodes:   make([]fuseops.InodeID, 0, 65536),
		},
		localCache: afero.NewBasePathFs(afero.NewOsFs(), pathToStaging),
		l:          logger.With(zap.String("bundle", bundle.BundleID)),
	}
	err := fs.initRoot()
	if err != nil {
		return nil, err
	}
	return &MutableFS{
		mfs:        nil,
		fsInternal: fs,
		server:     fuseutil.NewFileSystemServer(fs),
	}, err
}

func prepPath(path string) error {
	err := os.MkdirAll(path, dirDefaultMode)
	if err != nil && strings.Contains(err.Error(), "file exists") {
		return nil
	}
	return err
}

func (dfs *ReadOnlyFS) MountReadOnly(path string) error {
	err := prepPath(path)
	if err != nil {
		return err
	}

	const logdest = "/home/developer/"

	var errorLogger *log.Logger
	var debugLogger *log.Logger

	if _, err := os.Stat(logdest); !os.IsNotExist(err) {
		var f *os.File
		f, err = os.Create(filepath.Join(logdest, "fuseerr.log"))
		if err != nil {
			return err
		}
		errorLogger = log.New(f, "", log.Flags())
		f, err = os.Create(filepath.Join(logdest, "fusedbg.log"))
		if err != nil {
			return err
		}
		debugLogger = log.New(f, "", log.Flags())
	} else {
		errorLogger = log.New(os.Stderr, "fuse: ", log.Flags())
	}

	// Reminder: Options are OS specific
	// options := make(map[string]string)
	// options["allow_other"] = ""
	mountCfg := &fuse.MountConfig{
		FSName:      dfs.fsInternal.bundle.RepoID,
		VolumeName:  dfs.fsInternal.bundle.BundleID,
		ErrorLogger: errorLogger,
		DebugLogger: debugLogger,
		// Options:     debugLogger,
	}
	dfs.mfs, err = fuse.Mount(path, dfs.server, mountCfg)
	return err
}

func (dfs *MutableFS) MountMutable(path string) error {
	err := prepPath(path)
	if err != nil {
		return err
	}
	// TODO plumb additional mount options
	mountCfg := &fuse.MountConfig{
		FSName:      dfs.fsInternal.bundle.RepoID,
		VolumeName:  dfs.fsInternal.bundle.BundleID,
		ErrorLogger: log.New(os.Stderr, "fuse: ", log.Flags()),
	}
	dfs.mfs, err = fuse.Mount(path, dfs.server, mountCfg)
	return err
}

func (dfs *ReadOnlyFS) Unmount(path string) error {
	// On unmount, walk the FS and create a bundle
	return fuse.Unmount(path)
}

func (dfs *ReadOnlyFS) JoinMount(ctx context.Context) error {
	return dfs.mfs.Join(ctx)
}

func (dfs *MutableFS) Unmount(path string) error {
	// On unmount, walk the FS and create a bundle
	_ = dfs.fsInternal.Commit()
	//if err != nil {
	// dump the metadata to the local FS to manually recover.
	//}
	return fuse.Unmount(path)
}

func (dfs *MutableFS) JoinMount(ctx context.Context) error {
	return dfs.mfs.Join(ctx)
}

func (dfs *MutableFS) Commit() error {
	return dfs.fsInternal.Commit()
}
