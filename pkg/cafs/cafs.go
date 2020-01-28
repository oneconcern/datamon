package cafs

import (
	"context"
	"fmt"
	"io"
	"sync"

	lru "github.com/hashicorp/golang-lru"

	"go.uber.org/zap"

	"github.com/oneconcern/datamon/pkg/dlogger"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/localfs"

	"github.com/docker/go-units"
)

const (
	// DeduplicationBlake is the deduplication scheme using the blake hash
	// https://en.wikipedia.org/wiki/BLAKE_(hash_function).
	//
	// The implementation of the Blake hash we use (https://github.com/minio/blake2b-simd)
	// is 3 to 5 times faster than usual hashes such as MD5 or SHA's.
	DeduplicationBlake = "blake"

	// DefaultCacheSize sets the default target LRU buffer cache in bytes.
	//
	// This defines the number of leaf buffers allocated to the cache (rounded up)
	DefaultCacheSize = 50 * units.MiB

	// DefaultLeafSize sets the the default size of a blob leaf (2 MB). It cannot exceed MaxLeafSize.
	// The actual leaf size used is usually specified by each bundle.
	DefaultLeafSize uint32 = 2 * units.MiB

	// MaxLeafSize is the maximum size of a buffer in the memory pool
	MaxLeafSize = 5 * units.MiB

	// DefaultKeysCacheSize is the default size of the cache for resolved keys for root hashes.
	//
	// Corresponds to the number of files for which a root key is checked only once
	DefaultKeysCacheSize = 10000
)

// PutRes holds the result from a Put operation
type PutRes struct {
	Written int64  // bytes written
	Key     Key    // the new root hash of the written object
	Keys    []byte // the sequence of leaf keys of this object (NOTE(fred): don't quite get why we don't have []Key)
	Found   bool   // the root hash was already existing
}

// Fs implementations provide content-addressable filesystem operations
type Fs interface {
	Get(context.Context, Key) (io.ReadCloser, error)
	GetAt(context.Context, Key) (io.ReaderAt, error)
	Put(context.Context, io.Reader) (PutRes, error)
	Delete(context.Context, Key) error
	Clear(context.Context) error
	Keys(context.Context) ([]Key, error)
	RootKeys(context.Context) ([]Key, error)
	Has(context.Context, Key, ...HasOption) (bool, []Key, error)
	GetAddressingScheme() string
}

var _ Fs = &defaultFs{}

func defaultsForFs() *defaultFs {
	return &defaultFs{
		store:                       cafsStore{backend: localfs.New(nil)},
		leafSize:                    DefaultLeafSize,
		concurrentFlushes:           10,
		readerConcurrentChunkWrites: 3,
		deduplicationScheme:         DeduplicationBlake,
		keysCacheSize:               DefaultKeysCacheSize,
		lruSize:                     DefaultCacheSize,
		l:                           dlogger.MustGetLogger("info"),
		withVerifyHash:              true,
		withPrefetch:                0, // prefetching disabled by default
	}
}

// New creates a new instance of a content-addressable file system
func New(opts ...Option) (Fs, error) {
	f := defaultsForFs()
	for _, apply := range opts {
		apply(f)
	}

	if f.leafSize > MaxLeafSize {
		return nil, fmt.Errorf("%v exceeds maximum cafs leaf size %v", f.leafSize, MaxLeafSize)
	}
	if f.leafSize < KeySize {
		return nil, fmt.Errorf("%v is smaller than the key size %v", f.leafSize, KeySize)
	}

	const buffersForparallelReaders = 3
	cacheBuffers := BytesToBuffers(f.lruSize, f.leafSize)
	f.leafPool = newLeafFreelist(f.leafSize, cacheBuffers+buffersForparallelReaders)

	var err error
	f.lru, err = lru.NewWithEvict(cacheBuffers, func(_ interface{}, lruVal interface{}) {
		f.leafPool.Release(lruVal.(LeafBuffer)) // relinquish buffers to the freelist
	})
	if err != nil {
		return nil, err
	}

	f.keysCache, err = lru.New(f.keysCacheSize)
	if err != nil {
		return nil, err
	}

	f.pather = func(lks Key) string { return lks.StringWithPrefix(f.prefix) }

	return f, nil
}

type cafsStore struct {
	backend storage.Store // CAFS backing store
}

type defaultFs struct {
	store    cafsStore
	leafSize uint32
	l        *zap.Logger

	// prefix determines a namespace for keys
	prefix string
	pather func(Key) string // pathing logic

	// buffer cache (atm only supported for ReadAt)
	lru      *lru.Cache // this holds leaf data in cache
	lruLatch sync.Mutex // this ensures consistent LRU buffer pinning
	leafPool FreeList
	lruSize  int

	// root key cache of resolved leaf keys
	keysCache     *lru.Cache // this holds leaf keys in cache to avoid resolving root keys again
	keysCacheSize int

	// options
	leafTruncation              bool
	concurrentFlushes           int
	readerConcurrentChunkWrites int
	deduplicationScheme         string
	withPrefetch                int
	withVerifyHash              bool
}

func (d *defaultFs) GetAddressingScheme() string {
	return DeduplicationBlake
}

func (d *defaultFs) Put(ctx context.Context, src io.Reader) (PutRes, error) {
	d.l.Debug("Start cafs Put")
	defer d.l.Debug("End cafs Put")

	w := d.writer()

	// write leaf blobs
	written, err := io.Copy(w, src)
	if err != nil {
		w.Close()
		return PutRes{}, err
	}
	root, keys, err := w.Flush()
	if err != nil {
		w.Close()
		return PutRes{}, err
	}

	if err = w.Close(); err != nil {
		return PutRes{}, err
	}

	// write the root key as a blob containing all leaf keys, if it does not exist already
	found, _ := d.store.backend.Has(ctx, d.pather(root))
	if !found {
		// MultiPut exposes backend.PutCRC with a []byte input
		d.l.Debug("cafs writing the root hash blob",
			zap.String("prefix", d.prefix),
			zap.Stringer("root hash", root),
		)
		destinations := []storage.MultiStoreUnit{
			{
				Store:           d.store.backend,
				TolerateFailure: false,
			},
		}
		buffer := append(keys, root[:]...) // the root key trails the sequence
		err = storage.MultiPut(ctx, destinations, d.pather(root), buffer, storage.OverWrite)
		if err != nil {
			return PutRes{Found: found}, err
		}
	}
	if d.keysCache != nil {
		_, _ = d.keysCache.ContainsOrAdd(root, UnverifiedLeafKeys(keys, d.leafSize))
	}
	return PutRes{
		Written: written,
		Key:     root,
		Keys:    keys,
		Found:   found,
	}, nil
}

func (d *defaultFs) Get(ctx context.Context, hash Key) (io.ReadCloser, error) {
	return d.reader(hash)
}

func (d *defaultFs) GetAt(ctx context.Context, hash Key) (io.ReaderAt, error) {
	return d.reader(hash)
}

func (d *defaultFs) reader(hash Key) (Reader, error) {
	var (
		keys []Key
		err  error
	)

	if d.keysCache != nil {
		if b, ok := d.keysCache.Get(hash); ok {
			keys = b.([]Key)
		}
	}
	if keys == nil {
		d.l.Debug("cafs retrieving blob keys", zap.String("prefix", d.prefix))
		keys, err = LeavesForHash(d.store.backend, hash, d.leafSize, d.prefix)
		if err != nil {
			return nil, err
		}
		_, _ = d.keysCache.ContainsOrAdd(hash, keys)
	}

	rdr, err := newReader(d.store.backend, hash, d.leafSize,
		Keys(keys),
		TruncateLeaf(d.leafTruncation),
		ReaderVerifyHash(d.withVerifyHash),
		ConcurrentChunkWrites(d.readerConcurrentChunkWrites),
		SetCache(d.lru, &d.lruLatch),
		SetLeafPool(d.leafPool),
		ReaderPrefix(d.prefix),
		ReaderLogger(d.l),
		ReaderPrefetch(d.withPrefetch),
	)
	if err != nil {
		return nil, err
	}

	return rdr, nil
}

func (d *defaultFs) writer() Writer {
	return newWriter(d.store.backend, d.leafSize,
		WriterPrefix(d.prefix),
		WriterConcurrentFlushes(d.concurrentFlushes),
		WriterLogger(d.l),
		WriterPather(d.pather),
	)
}

func (d *defaultFs) Delete(ctx context.Context, hash Key) error {
	keys, err := LeavesForHash(d.store.backend, hash, d.leafSize, d.prefix)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err = d.store.backend.Delete(ctx, key.String()); err != nil {
			return err
		}
	}

	return d.store.backend.Delete(ctx, hash.String())
}

func (d *defaultFs) Clear(ctx context.Context) error {
	return d.store.backend.Clear(ctx)
}

// Keys returns all the keys from the backend store, with some optional matching filter.
//
// TODO(fred): nice - since the FS is configured with some prefix, one should only
// return prefixed keys.
func (d *defaultFs) Keys(ctx context.Context) ([]Key, error) {
	return d.keys(ctx, matchAnyKey)
}

func (d *defaultFs) keys(ctx context.Context, matches func(Key) bool) ([]Key, error) {
	v, err := d.store.backend.Keys(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]Key, 0, len(v))
	for _, k := range v {
		kk, err := KeyFromString(k)
		if err != nil {
			return nil, err
		}

		if matches(kk) {
			result = append(result, kk)
		}
	}
	return result, nil
}

func (d *defaultFs) RootKeys(ctx context.Context) ([]Key, error) {
	return d.keys(ctx, d.matchOnlyObjectRoots)
}

func (d *defaultFs) matchOnlyObjectRoots(key Key) bool {
	return IsRootKey(d.store.backend, key, d.leafSize)
}

func (d *defaultFs) Has(ctx context.Context, key Key, cfgs ...HasOption) (bool, []Key, error) {
	var opts hasOpts
	for _, apply := range cfgs {
		apply(&opts)
	}

	has, err := d.store.backend.Has(ctx, key.String())
	if err != nil {
		return false, nil, err
	}

	if !has {
		return false, nil, nil
	}

	if !opts.GatherIncomplete && !opts.OnlyRoots {
		return has, nil, nil
	}

	ks, err := LeavesForHash(d.store.backend, key, d.leafSize, d.prefix)
	if err != nil {
		return false, nil, nil
	}
	if len(ks) == 0 {
		return false, nil, nil
	}

	var keys []Key
	if opts.GatherIncomplete {
		for _, k := range ks {
			if ok, err := d.store.backend.Has(ctx, k.String()); err != nil || !ok {
				keys = append(keys, k)
			}
		}
	}
	return true, keys, nil
}

func matchAnyKey(_ Key) bool { return true }
