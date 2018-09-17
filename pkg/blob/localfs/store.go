package localfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/oneconcern/trumpet/pkg/blob"
	"github.com/spf13/afero"
)

// New creates a new local file system backed blob store
func New(fs afero.Fs) blob.Store {
	if fs == nil {
		fs = afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(".trumpet", "objects"))
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

func (l *localFS) Put(ctx context.Context, key string, rdr io.Reader) error {
	dir := filepath.Dir(key)
	if dir != "" {
		if err := l.fs.MkdirAll(filepath.Dir(key), 0700); err != nil {
			return fmt.Errorf("ensuring directories for %q: %v", key, err)
		}
	} else {
		dir = "."
	}

	fi, err := afero.TempFile(l.fs, dir, "tpt-put")
	if err != nil {
		return fmt.Errorf("create record for %q: %v", key, err)
	}
	defer fi.Close()

	_, err = io.Copy(fi, rdr)
	if err != nil {
		return fmt.Errorf("write record for %q: %v", key, err)
	}

	if err = fi.Close(); err != nil {
		return err
	}

	return l.fs.Rename(fi.Name(), filepath.Base(key))
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
