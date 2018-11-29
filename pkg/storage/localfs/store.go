// Copyright © 2018 One Concern

package localfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/spf13/afero"
)

// New creates a new local file system backed storage model
func New(fs afero.Fs) storage.Store {
	if fs == nil {
		fs = afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(".datamon", "objects"))
	}
	return &localFS{
		fs: fs,
	}
}

type localFS struct {
	fs afero.Fs
}

func (l *localFS) Has(ctx context.Context, key string) (bool, error) {

	fi, err := l.fs.Stat(key)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return !fi.IsDir(), nil
}

func (l *localFS) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return l.fs.Open(key)
}

func (l *localFS) Put(ctx context.Context, key string, source io.Reader) error {
	dir := filepath.Dir(key)
	if dir != "" {
		if err := l.fs.MkdirAll(filepath.Dir(key), 0700); err != nil {
			return fmt.Errorf("ensuring directories for %q: %v", key, err)
		}
	}
	target, err := l.fs.OpenFile(key, os.O_EXCL|os.O_CREATE|os.O_WRONLY|os.O_SYNC, 0600)
	if err != nil {
		return fmt.Errorf("create record for %q: %v", key, err)
	}
	defer target.Close()

	_, err = io.Copy(target, source)
	if err != nil {
		return fmt.Errorf("write record for %q: %v", key, err)
	}

	if err = target.Close(); err != nil {
		return err
	}
	return nil
}

func (l *localFS) Delete(ctx context.Context, key string) error {
	if err := l.fs.Remove(key); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing %q: %v", key, err)
	}
	return nil
}

func (l *localFS) Keys(ctx context.Context) ([]string, error) {
	const root = "."
	var res []string
	e := afero.Walk(l.fs, root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		pth := strings.Split(path, "/")
		if len(pth) == 3 && len(pth[0]) == 3 {
			res = append(res, strings.Join(pth, ""))
		} else {
			res = append(res, path)
		}

		return nil
	})
	if e != nil {
		return nil, e
	}
	return res, nil
}

func (l *localFS) Clear(ctx context.Context) error {
	return l.fs.RemoveAll("/")
}

func (l *localFS) String() string {
	const localfs = "localfs"
	switch fs := l.fs.(type) {
	case *afero.BasePathFs:
		pp, err := fs.RealPath("")
		if err != nil {
			return localfs
		}
		return localfs + "@" + pp
	default:
		return localfs
	}
}
