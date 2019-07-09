package cafs

import (
	"context"
	"io"
	"log"
	"sync"

	lru "github.com/hashicorp/golang-lru"

	"go.uber.org/zap"

	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/localfs"

	"github.com/docker/go-units"
)

const (
	DefaultLeafSize = 2 * 1024 * 1024
)

// LeafSize configuration for the blake2b hashes
func LeafSize(sz uint32) Option {
	return func(w *defaultFs) {
		w.leafSize = sz
	}
}

type HasOption func(*hasOpts)

func HasOnlyRoots() HasOption {
	return func(opts *hasOpts) {
		opts.OnlyRoots = true
	}
}

func HasGatherIncomplete() HasOption {
	return func(opts *hasOpts) {
		opts.OnlyRoots = true
		opts.GatherIncomplete = true
	}
}

func LeafTruncation(a bool) Option {
	return func(w *defaultFs) {
		w.leafTruncation = a
	}
}

func Prefix(prefix string) Option {
	return func(w *defaultFs) {
		w.prefix = prefix
	}
}

func Backend(store storage.Store) Option {
	return func(w *defaultFs) {
		w.store.backend = store
	}
}

func ConcurrentFlushes(concurrentFlushes int) Option {
	return func(w *defaultFs) {
		w.concurrentFlushes = concurrentFlushes
	}
}

func ReaderConcurrentChunkWrites(readerConcurrentChunkWrites int) Option {
	return func(w *defaultFs) {
		w.readerConcurrentChunkWrites = readerConcurrentChunkWrites
	}
}

type hasOpts struct {
	OnlyRoots, GatherIncomplete bool
	_                           struct{} // disallow unkeyed usage
}

// Option to configure content addressable FS components
type Option func(*defaultFs)

type PutRes struct {
	Written int64
	Key     Key
	Keys    []byte
	Found   bool
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
}

// New creates a new file system operations instance for a repository
func New(opts ...Option) (Fs, error) {

	f := &defaultFs{
		store:                       cafsStore{backend: localfs.New(nil)},
		leafSize:                    uint32(5 * units.MiB),
		concurrentFlushes:           10,
		readerConcurrentChunkWrites: 3,
	}
	f.leafPool = newLeafFreelist()
	f.lru, _ = lru.NewWithEvict(10, func(lruKey interface{}, lruVal interface{}) {
		lbuf := lruVal.(*leafBuffer)
		f.leafPool.put(lbuf)
	})
	for _, apply := range opts {
		apply(f)
	}
	return f, nil
}

type cafsStore struct {
	backend storage.Store // CAFS backing store
}

type leafBuffer struct{
	buf [2 * DefaultLeafSize]byte
	slice []byte
}

type leafFreelist struct{
	list []*leafBuffer
	mu sync.Mutex
}

func newLeafFreelist() *leafFreelist {
	return &leafFreelist{
		list: make([]*leafBuffer, 0),
	}
}

func (l* leafFreelist) get() *leafBuffer {
	var x *leafBuffer
	l.mu.Lock()
	ll := len(l.list)
	if ll != 0 {
		x = l.list[ll-1]
		l.list = l.list[:ll-1]
	}
	l.mu.Unlock()
	if x == nil {
		x = new(leafBuffer)
	}
	x.slice = x.buf[:]
	x.slice = x.slice[:0]
	return x
}

func (l* leafFreelist) put(lb *leafBuffer) {
	l.mu.Lock()
	l.list = append(l.list, lb)
	l.mu.Unlock()
}

type defaultFs struct {
	store                       cafsStore
	leafSize                    uint32
	prefix                      string
	zl                          zap.Logger //nolint:structcheck,unused
	l                           log.Logger //nolint:structcheck,unused
	leafTruncation              bool
	lru                         *lru.Cache
	leafPool *leafFreelist
	concurrentFlushes           int
	readerConcurrentChunkWrites int
}

func (d *defaultFs) Put(ctx context.Context, src io.Reader) (PutRes, error) {
	w := d.writer(d.prefix)
	defer w.Close()
	written, err := io.Copy(w, src)
	if err != nil {
		return PutRes{}, err
	}
	key, keys, err := w.Flush()
	if err != nil {
		return PutRes{}, err
	}
	if err = w.Close(); err != nil {
		return PutRes{}, err
	}
	destinations := make([]storage.MultiStoreUnit, 0)

	found, _ := d.store.backend.Has(ctx, d.prefix+key.String())
	if !found {
		destinations = append(destinations, storage.MultiStoreUnit{
			Store:           d.store.backend,
			TolerateFailure: false,
		})
	}
	buffer := append(keys, key[:]...)
	err = storage.MultiPut(ctx, destinations, key.String(), buffer, storage.OverWrite)
	if err != nil {
		return PutRes{Found: found}, err
	}
	return PutRes{
		Written: written,
		Key:     key,
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
	return newReader(d.store.backend, hash, d.leafSize, d.prefix,
		TruncateLeaf(d.leafTruncation),
		VerifyHash(true),
		ConcurrentChunkWrites(d.readerConcurrentChunkWrites),
		SetCache(d.lru),
		SetLeafPool(d.leafPool),
	)
}

func (d *defaultFs) writer(prefix string) Writer {
	maxGoRoutines := d.concurrentFlushes
	if maxGoRoutines < 1 {
		maxGoRoutines = 1
	}
	w := &fsWriter{
		store:               d.store.backend,
		leafSize:            d.leafSize,
		buf:                 make([]byte, d.leafSize),
		prefix:              prefix,
		flushChan:           make(chan blobFlush),
		errC:                make(chan error),
		flushThreadDoneChan: make(chan struct{}),
		maxGoRoutines:       make(chan struct{}, maxGoRoutines),
		blobFlushes:         make([]blobFlush, 0),
		errors:              make([]error, 0),
	}
	go w.flushThread()
	return w
}

func (d *defaultFs) Delete(ctx context.Context, hash Key) error {
	keys, err := LeafsForHash(d.store.backend, hash, d.leafSize, d.prefix)
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

	ks, err := LeafsForHash(d.store.backend, key, d.leafSize, d.prefix)
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

func IsRootKey(fs storage.Store, key Key, leafSize uint32) bool {
	keys, err := LeafsForHash(fs, key, leafSize, "")
	if err != nil {
		return false
	}
	return len(keys) > 0
}
