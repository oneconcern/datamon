package fuse

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jacobsa/fuse/fuseops"
	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/errors"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	jfuse "github.com/jacobsa/fuse"
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

// MountableFS knows how to mount and unmount a file system
type MountableFS interface {
	Mount(string, ...MountOption) error
	Unmount(string) error
}

var _ MountableFS = &ReadOnlyFS{}
var _ MountableFS = &MutableFS{}

// ReadOnlyFS is the virtual read-only filesystem created on top of a bundle.
type ReadOnlyFS struct {
	mfs        *jfuse.MountedFileSystem // The mounted filesystem
	fsInternal *readOnlyFsInternal      // The core of the filesystem
	server     jfuse.Server             // Fuse server
}

// MutableFS is the virtual mutable filesystem created on top of a bundle.
type MutableFS struct {
	mfs        *jfuse.MountedFileSystem // The mounted filesystem
	fsInternal *fsMutable               // The core of the filesystem
	server     jfuse.Server             // Fuse server
}

func checkBundle(bundle *core.Bundle) error {
	if bundle == nil {
		return fmt.Errorf("bundle is nil")
	}
	return nil
}

// NewReadOnlyFS creates a new instance of the datamon filesystem.
func NewReadOnlyFS(bundle *core.Bundle, opts ...Option) (*ReadOnlyFS, error) {
	if err := checkBundle(bundle); err != nil {
		return nil, err
	}

	fs := defaultReadOnlyFS(bundle)
	for _, bapply := range opts {
		bapply(fs)
	}

	if fs.streamed {
		// prepare the content-addressable backend for this bundle
		cafs, err := cafs.New(
			cafs.LeafSize(bundle.BundleDescriptor.LeafSize),
			cafs.LeafTruncation(bundle.BundleDescriptor.Version < 1),
			cafs.Backend(bundle.BlobStore()),
			cafs.Logger(fs.l),
			cafs.CacheSize(fs.lruSize),
			cafs.Prefetch(fs.prefetch),
			cafs.VerifyHash(fs.withVerifyHash),
		)
		if err != nil {
			return nil, err
		}
		fs.cafs = cafs
	}

	fs.l = fs.l.With(zap.String("repo", bundle.RepoID), zap.String("bundle", bundle.BundleID))

	if fs.streamed {
		// extract the meta information needed: data will be fetched as needed
		err := core.PublishMetadata(context.Background(), fs.bundle)
		if err != nil {
			fs.l.Error("Failed to publish bundle metadata", zap.String("id", bundle.BundleID), zap.Error(err))
			return nil, err
		}
	} else {
		// download the bundle entirely to staging area
		err := core.Publish(context.Background(), fs.bundle)
		if err != nil {
			fs.l.Error("Failed to publish bundle", zap.String("id", bundle.BundleID), zap.Error(err))
			return nil, err
		}
	}

	// Populate the filesystem with medatata
	//
	// NOTE: this fetches the bundle metadata only and builds
	// an in-memory view of the file system structure.
	// This may lead to a large memory footprint for bundles
	// with many files (e.g. thousands)
	//
	// TODO: reduce memory footprint
	return fs.populateFS(bundle)
}

// localPath resolves a consumable store to a local path.
//
// TODO: this should somehow be resolved by the Store API
// and we shouldn't make assumptions here
func localPath(consumable fmt.Stringer) (string, error) {
	fullPath := consumable.String()
	// assume consumable is built with storage/localfs
	parts := strings.Split(fullPath, "@")
	if len(parts) < 2 || parts[0] != "localfs" {
		return "", errors.New("bundle doesn't have localfs consumable store to provide local cache for mutable fs")
	}
	return parts[1], nil
}

// NewMutableFS creates a new instance of the mutable datamon filesystem.
func NewMutableFS(bundle *core.Bundle, opts ...Option) (*MutableFS, error) {
	if err := checkBundle(bundle); err != nil {
		return nil, err
	}

	pathToStaging, err := localPath(bundle.ConsumableStore)
	if err != nil {
		return nil, err
	}

	fs := defaultMutableFS(bundle, pathToStaging)
	for _, bapply := range opts {
		bapply(fs)
	}

	fs.l = fs.l.With(zap.String("repo", bundle.RepoID))
	if bundle.BundleID != "" {
		fs.l = fs.l.With(zap.String("bundle", bundle.BundleID))
	}

	fs.l.Info("mutable mount staging storage", zap.String("path", pathToStaging))

	err = fs.initRoot()
	if err != nil {
		return nil, err
	}
	return &MutableFS{
		mfs:        nil,
		fsInternal: fs,
		server:     fuseutil.NewFileSystemServer(fs),
	}, nil
}

func prepPath(path string) error {
	return os.MkdirAll(path, dirDefaultMode)
}

// Mount a ReadOnlyFS
func (dfs *ReadOnlyFS) Mount(path string, opts ...MountOption) error {
	return dfs.MountReadOnly(path, opts...)
}

func defaultMountConfig(bundle *core.Bundle, readOnly bool, subType string) *jfuse.MountConfig {
	return &jfuse.MountConfig{
		Subtype:    subType, // mount appears as "fuse.{subType}"
		ReadOnly:   readOnly,
		FSName:     bundle.RepoID,
		VolumeName: bundle.BundleID, // NOTE: OSX only option
		// Options:     options,
		// Reminder: Options are OS specific
		// options := make(map[string]string)
		// options["allow_other"] = ""
	}

}

// MountReadOnly a ReadOnlyFS
func (dfs *ReadOnlyFS) MountReadOnly(path string, opts ...MountOption) error {
	err := prepPath(path)
	if err != nil {
		return err
	}

	mountCfg := defaultMountConfig(dfs.fsInternal.bundle, true, "datamon")
	for _, bapply := range opts {
		bapply(mountCfg)
	}

	el, _ := zap.NewStdLogAt(dfs.fsInternal.l.
		With(zap.String("fuse", "read-only mount"), zap.String("mountpoint", path)), zapcore.ErrorLevel)
	dl, _ := zap.NewStdLogAt(dfs.fsInternal.l.
		With(zap.String("fuse-debug", "read-only mount"), zap.String("mountpoint", path)), zapcore.DebugLevel)
	mountCfg.ErrorLogger = el
	mountCfg.DebugLogger = dl

	dfs.mfs, err = jfuse.Mount(path, dfs.server, mountCfg)
	if err == nil {
		dfs.fsInternal.l.Info("mounting", zap.String("mountpoint", path))
	}
	return err
}

// Unmount a ReadOnlyFS
func (dfs *ReadOnlyFS) Unmount(path string) error {
	dfs.fsInternal.l.Info("unmounting", zap.String("mountpoint", path))
	return jfuse.Unmount(path)
}

// JoinMount blocks until a mounted file system has been unmounted.
// It does not return successfully until all ops read from the connection have been responded to
// (i.e. the file system server has finished processing all in-flight ops).
func (dfs *ReadOnlyFS) JoinMount(ctx context.Context) error {
	return dfs.mfs.Join(ctx)
}

// Mount a MutableFS as mutable (read-write)
func (dfs *MutableFS) Mount(path string, opts ...MountOption) error {
	return dfs.MountMutable(path, opts...)
}

// MountMutable mounts a MutableFS as mutable (read-write)
func (dfs *MutableFS) MountMutable(path string, opts ...MountOption) error {
	err := prepPath(path)
	if err != nil {
		return err
	}
	mountCfg := defaultMountConfig(dfs.fsInternal.bundle, false, "datamon-mutable")
	for _, bapply := range opts {
		bapply(mountCfg)
	}

	el, _ := zap.NewStdLogAt(dfs.fsInternal.l.
		With(zap.String("fuse", "mutable mount"), zap.String("mountpoint", path)), zapcore.ErrorLevel)
	dl, _ := zap.NewStdLogAt(dfs.fsInternal.l.
		With(zap.String("fuse-debug", "mutable mount"), zap.String("mountpoint", path)), zapcore.DebugLevel)
	mountCfg.ErrorLogger = el
	mountCfg.DebugLogger = dl

	dfs.mfs, err = jfuse.Mount(path, dfs.server, mountCfg)
	if err == nil {
		dfs.fsInternal.l.Info("mounting", zap.String("mountpoint", path))
	}
	return err
}

// Unmount a MutableFS
func (dfs *MutableFS) Unmount(path string) error {
	// On unmount, walk the FS and create a bundle
	_ = dfs.fsInternal.Commit()
	//if err != nil {
	// dump the metadata to the local FS to manually recover.
	//}
	dfs.fsInternal.l.Info("unmounting", zap.String("mountpoint", path))
	return jfuse.Unmount(path)
}

// JoinMount blocks until a mounted file system has been unmounted.
// It does not return successfully until all ops read from the connection have been responded to
// (i.e. the file system server has finished processing all in-flight ops).
func (dfs *MutableFS) JoinMount(ctx context.Context) error {
	return dfs.mfs.Join(ctx)
}
