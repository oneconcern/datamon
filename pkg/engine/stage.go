package engine

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/oneconcern/pipelines/pkg/log"
	"github.com/oneconcern/trumpet/pkg/store/instrumented"
	opentracing "github.com/opentracing/opentracing-go"

	"github.com/oneconcern/trumpet/pkg/blob"
	bloblocalfs "github.com/oneconcern/trumpet/pkg/blob/localfs"
	"github.com/oneconcern/trumpet/pkg/fingerprint"
	"github.com/oneconcern/trumpet/pkg/store"
	"github.com/oneconcern/trumpet/pkg/store/localfs"
	"github.com/spf13/afero"
)

// NewStage creates a new stage instance
func newStage(repo, baseDir string, tr opentracing.Tracer, logs log.Factory, bundles store.BundleStore) (*Stage, error) {
	meta := instrumented.NewObjectMeta(repo, tr, localfs.NewObjectMeta(baseDir))
	if err := meta.Initialize(); err != nil {
		return nil, err
	}

	return &Stage{
		logs:    logs,
		bundles: bundles,
		objects: blob.Instrument(tr, logs, bloblocalfs.New(afero.NewBasePathFs(afero.NewOsFs(), baseDir))),
		meta:    meta,
		hasher:  fingerprint.New(),
	}, nil
}

// UnstagedFilePath as blob to add to stage
func UnstagedFilePath(pth string) (AddBlob, error) {
	f, err := os.Open(pth)
	if err != nil {
		return AddBlob{}, err
	}

	fi, err := f.Stat()
	if err != nil {
		return AddBlob{}, err
	}

	return AddBlob{
		Path:   f.Name(),
		Stream: f,
		Mtime:  fi.ModTime(),
		Mode:   fi.Mode(),
	}, nil
}

// UnstagedFile as blob to add to stage
func UnstagedFile(f *os.File) (AddBlob, error) {
	fi, err := f.Stat()
	if err != nil {
		return AddBlob{}, err
	}

	return AddBlob{
		Path:   f.Name(),
		Stream: f,
		Mtime:  fi.ModTime(),
		Mode:   fi.Mode(),
		Size:   fi.Size(),
	}, nil
}

// UnstagedStream as blob to add to stage
func UnstagedStream(path string, reader io.Reader, mtime time.Time, mode os.FileMode) AddBlob {
	if mtime.IsZero() {
		mtime = time.Now()
	}
	return AddBlob{
		Path:   path,
		Stream: reader,
		Mtime:  mtime,
		Mode:   mode,
	}
}

// AddBlob arguments for adding a new blob to stage
type AddBlob struct {
	Path   string
	Stream io.Reader
	Mtime  time.Time
	Mode   os.FileMode
	Size   int64

	_ struct{} // avoid unkeyed usage
}

// Close the stream when if it can be closed
func (a *AddBlob) Close() error {
	if closer, ok := a.Stream.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Stage contains the information to manage staged changes
type Stage struct {
	logs    log.Factory
	bundles store.BundleStore
	objects blob.Store
	meta    store.StageMeta
	hasher  *fingerprint.Maker
}

// Add a file to stage
func (s *Stage) Add(ctx context.Context, addBlob AddBlob) (string, bool, error) {
	// TODO: encode and write file in a single pass
	fp, err := s.hasher.Process(addBlob.Path)
	if err != nil {
		return "", false, err
	}
	hash := fmt.Sprintf("%x", fp)

	var isNew bool
	_, err = s.meta.Get(ctx, hash)
	if err == store.ObjectNotFound {
		isNew = true
	}

	if isNew {
		defer addBlob.Close()
		if err = s.objects.Put(ctx, hash, addBlob.Stream); err != nil {
			return "", false, err
		}
		if err = addBlob.Close(); err != nil {
			return "", false, err
		}
	}

	err = s.meta.Add(ctx, store.Entry{
		Path:  addBlob.Path,
		Hash:  hash,
		Mtime: addBlob.Mtime,
		Mode:  store.FileMode(addBlob.Mode),
	})
	if err != nil {
		return "", false, err
	}

	return hash, isNew, nil
}

// Remove a file from the stage
func (s *Stage) Remove(ctx context.Context, path string) error {
	// also look up hash in the committed bundles
	// when there is a hash found in the committed bundles
	// then instead of deleting we'll mark it for delete on the stage
	entry, err := s.bundles.GetObjectForPath(ctx, path)
	if err == nil {
		return s.meta.MarkDelete(ctx, &entry)
	}

	hash, err := s.meta.HashFor(ctx, path)
	if err != nil {
		return err
	}

	if err := s.meta.Remove(ctx, hash); err != nil {
		return err
	}

	return s.objects.Delete(ctx, hash)
}

// Clear the stage
func (s *Stage) Clear(ctx context.Context) error {
	if err := s.meta.Clear(ctx); err != nil {
		return err
	}
	return s.objects.Clear(ctx)
}

// Status of the stage, returns a changeset
func (s *Stage) Status(ctx context.Context) (store.ChangeSet, error) {
	return s.meta.List(ctx)
}
