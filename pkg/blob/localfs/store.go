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

func (l *localFS) fpath(key string) string {
	return filepath.Join(key[:2], key[2:4], key[4:])
}

func (l *localFS) Has(ctx context.Context, key string) (bool, error) {
	fp := l.fpath(key)
	fi, err := l.fs.Stat(fp)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return !fi.IsDir(), nil
}

func (l *localFS) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return l.fs.Open(l.fpath(key))
}

func (l *localFS) Put(ctx context.Context, key string, rdr io.Reader) error {
	fp := l.fpath(key)
	if err := l.fs.MkdirAll(filepath.Dir(fp), 0700); err != nil {
		return fmt.Errorf("ensuring directories for %q: %v", key, err)
	}

	fi, err := afero.TempFile(l.fs, filepath.Dir(fp), "tpt-put")
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

	return l.fs.Rename(fi.Name(), fp)
}

func (l *localFS) Delete(ctx context.Context, key string) error {
	if err := l.fs.Remove(l.fpath(key)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing %q: %v", key, err)
	}
	return nil
}

func (l *localFS) Keys(ctx context.Context) ([]string, error) {
	fis, err := afero.Glob(l.fs, "*/*/*")
	if err != nil {
		return nil, err
	}

	res := make([]string, len(fis))
	for i, v := range fis {
		res[i] = strings.Join(strings.Split(v, "/"), "")
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
