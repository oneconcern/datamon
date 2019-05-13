// Copyright © 2018 One Concern

package localfs

import (
	"context"
	"errors"
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

func (l *localFS) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	has, err := l.Has(ctx, key)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, storage.ErrNotFound
	}
	t, err := l.fs.Open(key)
	return localReader{
		objectReader: t,
	}, err
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

func (l *localFS) Put(ctx context.Context, key string, source io.Reader, exclusive bool) error {
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
	s := readCloser{
		reader: source,
	}
	// If reader implements writeto use it.
	wt := source.(io.WriterTo)
	if wt != nil {
		_, err = wt.WriteTo(target)
		if err != nil {
			return fmt.Errorf("write record for %q: %v", key, err)
		}
	} else {
		_, err = storage.PipeIO(target, s)
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

func (l *localFS) GetAt(ctx context.Context, key string) (io.ReaderAt, error) {
	// could be wrapped in a custom type as with Get()
	return l.fs.Open(key)
}

/* thread-safe local storage implementation.
 * use a decorator pattern to implement atomic Put()s via atomicity of afero.Fs.Rename()
 * for those filesystems where Rename() is thread-safe:  files are placed in a staging area,
 * then Rename()d into place.
 */

/* staging area key prefix and helper functions */
const (
	nestedPutStageName = ".put-stage"
)

func maybeInvalidKey(key string) error {
	const pathSepString = string(os.PathSeparator)
	pathComponents := strings.Split(strings.TrimLeft(key, pathSepString), pathSepString)
	if len(pathComponents) == 0 {
		return nil
	}
	if pathComponents[0] == nestedPutStageName {
		return fmt.Errorf("key '%v' conflicts with put staging area name '%v'", key, nestedPutStageName)
	}
	return nil
}

func filterInvalidKeys(ks []string) []string {
	/* https://github.com/golang/go/wiki/SliceTricks#filtering-without-allocating */
	ksFiltered := ks[:0]
	for _, key := range ks {
		if err := maybeInvalidKey(key); err == nil {
			ksFiltered = append(ksFiltered, key)
		}
	}
	for i := len(ksFiltered); i < len(ks); i++ {
		ks[i] = ""
	}
	return ksFiltered
}

func NewAtomic(fs afero.Fs) (storage.Store, error) {
	if fs == nil {
		fs = afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(".datamon", "objects"))
	}
	/* the staging area exists within the afero.Fs itself */
	if err := fs.MkdirAll(nestedPutStageName, 0700); err != nil {
		return nil, fmt.Errorf("ensuring put staging directory for %q: %v", nestedPutStageName, err)
	}
	return &localFSAtomic{
		storeImpl: localFS{fs: fs},
	}, nil
}

type localFSAtomic struct {
	storeImpl localFS
}

/* implementing the Store interface is mostly a matter of wrapping the decorated localFs's
 * interface with helper functions.
 */

func (l *localFSAtomic) Has(ctx context.Context, key string) (bool, error) {
	if err := maybeInvalidKey(key); err != nil {
		return false, err
	}
	return l.storeImpl.Has(ctx, key)
}

func (l *localFSAtomic) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := maybeInvalidKey(key); err != nil {
		return nil, err
	}
	return l.storeImpl.Get(ctx, key)
}

func (l *localFSAtomic) Delete(ctx context.Context, key string) error {
	if err := maybeInvalidKey(key); err != nil {
		return err
	}
	return l.storeImpl.Delete(ctx, key)
}

func (l *localFSAtomic) GetAt(ctx context.Context, key string) (io.ReaderAt, error) {
	if err := maybeInvalidKey(key); err != nil {
		return nil, err
	}
	return l.storeImpl.GetAt(ctx, key)
}

func (l *localFSAtomic) Keys(ctx context.Context) ([]string, error) {
	ks, err := l.storeImpl.Keys(ctx)
	if err != nil {
		return ks, err
	}
	return filterInvalidKeys(ks), nil
}

func (l *localFSAtomic) KeysPrefix(ctx context.Context, token, prefix, delimiter string, count int) ([]string, string, error) {
	ks, pageToken, err := l.storeImpl.KeysPrefix(ctx, token, prefix, delimiter, count)
	if err != nil {
		return ks, pageToken, err
	}
	return filterInvalidKeys(ks), pageToken, nil
}

func (l *localFSAtomic) Clear(ctx context.Context) error {
	return l.storeImpl.Clear(ctx)
}

/* the Put() implementation is the only part of the Store interface implemented
 * outside of the functional wrap design pattern
 */
func (l *localFSAtomic) Put(ctx context.Context, key string, source io.Reader, exclusive bool) error {
	if err := maybeInvalidKey(key); err != nil {
		return err
	}
	putStageKey := filepath.Join(nestedPutStageName, key)
	if err := l.storeImpl.Put(ctx, putStageKey, source, exclusive); err != nil {
		return err
	}
	/* Rename() doesn't create directories automatically */
	dir := filepath.Dir(key)
	if dir != "" {
		if err := l.storeImpl.fs.MkdirAll(filepath.Dir(key), 0700); err != nil {
			return fmt.Errorf("ensuring directories for %q: %v", key, err)
		}
	}
	return l.storeImpl.fs.Rename(putStageKey, key)
}

// dupe: localFs.String
func (l *localFSAtomic) String() string {
	const localfs = "localfs-atomic"
	switch fs := l.storeImpl.fs.(type) {
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
