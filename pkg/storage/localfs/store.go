// Copyright © 2018 One Concern

package localfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/status"
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

type localReader struct {
	objectReader io.ReadCloser
}

func (r *localReader) WriteTo(writer io.Writer) (n int64, err error) {
	return storage.PipeIO(writer, r.objectReader)
}

func (r localReader) Close() error {
	return r.objectReader.Close()
}

func (r localReader) Read(p []byte) (n int, err error) {
	return r.objectReader.Read(p)
}

func toSentinelErrors(err error) error {
	// return sentinel errors defined by the status package
	if os.IsNotExist(err) {
		return status.ErrNotExists
	}
	return err
}

func (l *localFS) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	t, err := l.fs.Open(key)
	return localReader{
		objectReader: t,
	}, toSentinelErrors(err)
}

type readCloser struct {
	reader io.Reader
}

func (rc readCloser) Read(p []byte) (n int, err error) {
	return rc.reader.Read(p)
}
func (rc readCloser) Close() error {
	return nil
}

func (l *localFS) Put(ctx context.Context, key string, source io.Reader, exclusive storage.NewKey) error {
	// TODO: Change this implementation to use rename to put file into place.
	dir := filepath.Dir(key)
	if dir != "" {
		if err := l.fs.MkdirAll(filepath.Dir(key), 0700); err != nil {
			return fmt.Errorf("ensuring directories for %q: %v", key, err)
		}
	}
	flag := os.O_CREATE | os.O_WRONLY | os.O_SYNC | 0600
	if exclusive {
		flag |= os.O_EXCL
	}
	target, err := l.fs.OpenFile(key, flag, 0600)
	if err != nil {
		return fmt.Errorf("create record for %q: %v", key, err)
	}
	// If reader implements writeto use it.
	wt, ok := source.(io.WriterTo)
	if ok {
		_, err = wt.WriteTo(target)
		if err != nil {
			return fmt.Errorf("write record for %q: %v", key, err)
		}
	} else {
		_, err = storage.PipeIO(target, readCloser{reader: source})
		if err != nil {
			return fmt.Errorf("write record for %q: %v", key, err)
		}
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
		fileInfo, err := l.fs.Stat(path)
		if err != nil {
			return err
		}
		if fileInfo.IsDir() {
			return nil
		}
		res = append(res, path)
		return nil
	})
	if e != nil {
		return nil, e
	}
	return res, nil
}

//TODO discuss the implementation with @Ivan & @Ritesh
func (l *localFS) KeysPrefix(ctx context.Context, token, prefix, delimiter string, count int) ([]string, string, error) {
	return nil, "", errors.New("unimplemented")
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

func (l *localFS) GetAt(ctx context.Context, objectName string) (io.ReaderAt, error) {
	return l.fs.Open(objectName)
}

func (l *localFS) Touch(ctx context.Context, objectName string) error {
	err := l.fs.Chtimes(objectName, time.Now(), time.Now())
	return err
}

func (l *localFS) GetAttr(ctx context.Context, objectName string) (storage.Attributes, error) {
	stat, err := l.fs.Stat(objectName)
	if err != nil {
		return storage.Attributes{}, err
	}
	sys, ok := stat.Sys().(syscall.Stat_t)
	if !ok {
		return storage.Attributes{}, fmt.Errorf("failed to convert sys to Stat_t for object:%s", objectName)
	}
	return storage.Attributes{
		Created: stat.ModTime(), // Fix me: need a platform independent way to extracting timestamps
		Updated: stat.ModTime(),
		Owner:   fmt.Sprint(sys.Uid),
	}, nil

}
