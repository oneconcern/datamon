package trumpet

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/oneconcern/trumpet/pkg/blob"
	"github.com/oneconcern/trumpet/pkg/fingerprint"
	"github.com/oneconcern/trumpet/pkg/store"
	"github.com/oneconcern/trumpet/pkg/store/localfs"
)

// NewStage creates a new stage instance
func newStage(baseDir string) (*Stage, error) {
	if baseDir == "" {
		baseDir = ".trumpet/stage"
	}

	meta := localfs.NewObjectMeta(baseDir)
	if err := meta.Initialize(); err != nil {
		return nil, err
	}

	return &Stage{
		objects: blob.LocalFS(filepath.Join(baseDir, "objects")),
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
	}, nil
}

// UnstagedStream as blob to add to stage
func UnstagedStream(path string, reader io.Reader, mtime time.Time, mode os.FileMode) AddBlob {
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
	objects blob.Store
	meta    store.ObjectMeta
	hasher  *fingerprint.Maker
}

// Add a file to stage
func (s *Stage) Add(addBlob AddBlob) (string, bool, error) {
	// TODO: encode and write file in a single pass
	fp, err := s.hasher.Process(addBlob.Path)
	if err != nil {
		return "", false, err
	}
	hash := fmt.Sprintf("%x", fp)

	var isNew bool
	_, err = s.meta.Get(hash)
	if err == store.ObjectNotFound {
		isNew = true
	}

	if isNew {
		defer addBlob.Close()
		if err = s.objects.Put(hash, addBlob.Stream); err != nil {
			return "", false, err
		}
		if err = addBlob.Close(); err != nil {
			return "", false, err
		}
	}

	err = s.meta.Add(store.Entry{
		Path:  addBlob.Path,
		Hash:  hash,
		Mtime: addBlob.Mtime,
		Mode:  addBlob.Mode,
	})
	if err != nil {
		return "", false, err
	}

	return hash, isNew, nil
}

// Remove a file from the stage
func (s *Stage) Remove(path string) error {
	hash, err := s.meta.HashFor(path)
	if err != nil {
		return err
	}

	if err := s.meta.Remove(hash); err != nil {
		return err
	}

	return s.objects.Delete(hash)
}

// Clear the stage
func (s *Stage) Clear() error {
	if err := s.meta.Clear(); err != nil {
		return err
	}
	return s.objects.Clear()
}
