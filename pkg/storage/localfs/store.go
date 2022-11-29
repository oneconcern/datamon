// Copyright © 2018 One Concern

// Package localfs implements datamon Store for a local file system
package localfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/oneconcern/datamon/pkg/dlogger"
	"github.com/oneconcern/datamon/pkg/storage"
	storagestatus "github.com/oneconcern/datamon/pkg/storage/status"
	"github.com/spf13/afero"
	"go.uber.org/zap"
)

// New creates a new local file system backed storage model
func New(fs afero.Fs, opts ...Option) storage.Store {
	if fs == nil {
		fs = afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(".datamon", "objects"))
	}
	local := &localFS{
		fs:    fs,
		glob:  make(map[string][]string),
		retry: true,
	}

	if local.l == nil {
		local.l, _ = dlogger.GetLogger("info")
	}

	for _, apply := range opts {
		apply(local)
	}
	return local
}

// Option for the local FS store
type Option func(*localFS)

// WithLock prevents concurrent writes or concurrent read/writes on this local FS
func WithLock(flag bool) Option {
	return func(fs *localFS) {
		fs.lock = flag
	}
}

// WithRetry enables exponential backoff retry logic to be enabled on put operations
func WithRetry(enabled bool) Option {
	return func(fs *localFS) {
		fs.retry = enabled
	}
}

// WithLogger adds a logger to the localfs object
func WithLogger(logger *zap.Logger) Option {
	return func(fs *localFS) {
		fs.l = logger
	}
}

type localFS struct {
	fs        afero.Fs
	glob      map[string][]string // current state of KeyPrefix matches
	exclusive sync.Mutex          // mutex on glob access
	lock      bool
	rw        sync.RWMutex
	retry     bool
	l         *zap.Logger
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
		return storagestatus.ErrNotExists.Wrap(err)
	}
	return err
}

func (l *localFS) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if l.lock {
		l.rw.RLock()
		defer l.rw.RUnlock()
	}
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

func (l *localFS) Put(ctx context.Context, key string, source io.Reader, exclusive bool) (err error) {
	var (
		retryPolicy backoff.BackOff
		target      afero.File
	)

	l.l.Debug("Start Put", zap.String("key", key))
	defer func() {
		l.l.Debug("End Put", zap.String("key", key), zap.Error(err))
	}()

	if l.retry {
		r := backoff.NewExponentialBackOff()
		r.MaxElapsedTime = 30 * time.Second
		r.Reset()
		retryPolicy = r
	} else {
		retryPolicy = &backoff.StopBackOff{}
	}

	if l.lock {
		l.rw.Lock()
		defer l.rw.Unlock()
	}
	// TODO: Change this implementation to use rename to put file into place.
	dir := filepath.Dir(key)
	if dir != "" {
		if e := l.fs.MkdirAll(filepath.Dir(key), 0700); e != nil {
			return fmt.Errorf("ensuring directories for %q: %v", key, e)
		}
	}
	flag := os.O_CREATE | os.O_WRONLY | os.O_SYNC | os.O_TRUNC
	if exclusive {
		flag |= os.O_EXCL
	}
	// If reader implements writeto use it.
	wt, ok := source.(io.WriterTo)
	if ok {
		// wrapping WriteTo execution so it can be retried
		operation := func() error {
			target, err = l.fs.OpenFile(key, flag, 0600)
			if err != nil {
				return fmt.Errorf("create record for %q: %v", key, err)
			}

			_, err = wt.WriteTo(target)
			if err != nil {
				l.l.Error("write error, retrying",
					zap.String("key", key),
					zap.Error(err),
				)
			}
			err = target.Close()
			if err != nil {
				l.l.Error("write error, retrying",
					zap.String("key", key),
					zap.Error(err),
				)

			}

			return err
		}
		err = backoff.Retry(operation, retryPolicy)
		if err != nil {
			return fmt.Errorf("write record for %q: %v", key, err)
		}
	} else {
		// wrapping PipeIO execution so it can be retried
		operation := func() error {
			target, err = l.fs.OpenFile(key, flag, 0600)
			if err != nil {
				return fmt.Errorf("create record for %q: %v", key, err)
			}
			_, err = storage.PipeIO(target, readCloser{reader: source})
			if err != nil {
				l.l.Error("write error, retrying",
					zap.String("key", key),
					zap.Error(err),
				)
			}

			err = target.Close()
			if err != nil {
				l.l.Error("write error, retrying",
					zap.String("key", key),
					zap.Error(err),
				)
			}

			return err
		}
		err = backoff.Retry(operation, retryPolicy)
		if err != nil {
			return fmt.Errorf("write record for %q: %v", key, err)
		}
	}

	return nil
}

func (l *localFS) Delete(ctx context.Context, key string) error {
	// unlink is atomic: no need to lock the in-memory object for that
	if err := l.fs.Remove(key); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing %q: %v", key, err)
	}
	return nil
}

func (l *localFS) Keys(ctx context.Context) ([]string, error) {
	if l.lock {
		l.rw.RLock()
		defer l.rw.RUnlock()
	}
	const root = "."
	var res []string
	e := afero.Walk(l.fs, root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		if info.IsDir() {
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

// KeyPrefix provides a paginated key iterator using "pageToken" as the next starting point
//
// NOTE: this cursory implementation is at the moment only used by mocks in test. A more thorough approach
// is required to make KeyPrefix a first class citizen for localfs.
//
// TODO(known limitations):
//   - this implementation does not really scale up, but it is quite workable for our testcases using localfs.
//   - this implementation is not meant for parallel use with mutable FS.
func (l *localFS) KeysPrefix(_ context.Context, token, prefix, delimiter string, count int) ([]string, string, error) {
	l.exclusive.Lock()
	defer l.exclusive.Unlock()

	noRoot := !strings.HasPrefix(prefix, "/")
	prefix = path.Clean("/" + prefix)

	// we cache the result for the duration of the fetch loop: during this period, localfs updates are not seen
	search, ok := l.glob[prefix]
	if !ok {
		// NOTE: Glob is not workable, fall back to Walk
		matches := make([]string, 0, 50)
		err := afero.Walk(l.fs, path.Dir(prefix), func(pth string, info os.FileInfo, err error) error {
			if info.IsDir() || err != nil {
				return nil
			}
			if strings.HasPrefix(pth, prefix) {
				if delimiter != "" && len(pth) > len(prefix) {
					if cut := strings.Index(pth[len(prefix):], delimiter); cut > -1 {
						pth = pth[0 : len(prefix)+cut+1]
					}
				}
				if noRoot {
					pth = strings.TrimPrefix(pth, "/")
				}
				matches = append(matches, pth)
			}
			return nil
		})
		if err != nil {
			return nil, "", err
		}
		if delimiter != "" {
			// dedupe truncated matches
			deduped := make([]string, 0, len(matches))
			for _, match := range matches {
				dupe := false
				for _, lookup := range deduped {
					if match == lookup {
						dupe = true
						break
					}
				}
				if !dupe {
					deduped = append(deduped, match)
				}
			}
			matches = deduped
		}
		l.glob[prefix], search = matches, matches
	}

	var (
		start, end int
		next       string
	)

	if token == "" {
		start = 0
	} else {
		found := false
		for i, lookup := range search {
			if token != lookup {
				continue
			}
			found = true
			start = i
			break
		}
		if !found {
			delete(l.glob, prefix)
			return []string{}, "", nil
		}
	}

	if len(search) > start+count {
		next = search[start+count]
		end = start + count
	} else {
		next = ""
		end = len(search)
		delete(l.glob, prefix)
	}

	return search[start:end], next, nil
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

	// do not block when UID is not available on local os
	var owner string
	sys, ok := stat.Sys().(syscall.Stat_t)
	if ok {
		owner = fmt.Sprint(sys.Uid)
	}

	return storage.Attributes{
		Created: stat.ModTime(), // Fix me: need a platform independent way to extracting timestamps
		Updated: stat.ModTime(),
		Owner:   owner,
		Size:    stat.Size(),
		// CRC32C not supported on localfs
	}, nil

}
