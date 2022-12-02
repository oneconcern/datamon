package core

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/docker/go-units"
	"github.com/oneconcern/datamon/pkg/cafs"
	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/errors"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"github.com/oneconcern/datamon/pkg/storage/status"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	layout    = time.RFC3339Nano
	batchSize = 1024
)

type (
	PurgeIndex struct {
		IndexTime  time.Time
		NumEntries uint64
	}

	PurgeBlobs struct {
		IndexTime         time.Time
		ScannedEntries    uint64
		IndexedEntries    uint64
		MoreRecentEntries uint64
		DeletedEntries    uint64
		DeletedSize       uint64
		DryRun            bool
	}
)

// PurgeBuildReverseIndex creates or update a reverse-lookip index
// of all used blob keys.
//
// This operation can take quite a long time: there is some extra logging to keep track of the progress.
func PurgeBuildReverseIndex(stores context2.Stores, opts ...PurgeOption) (*PurgeIndex, error) {
	// 1. scan all repos, all bundles
	// 2. Fetch root key, explode root key
	// 3. Add root key and children keys to index
	// 4. Write index file (with overwrite)

	options := defaultPurgeOptions(opts)
	ctx := context.Background() // no timeout here
	indexTime := time.Now().UTC()

	db, err := makeKV(options.localStorePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = db.Close()
	}()

	// about to write the index in the current context
	indexStore := getMetaStore(stores)
	indexPath := model.ReverseIndex()
	blob := getBlobStore(stores)
	logger := options.l.With(
		zap.String("index_path", path.Join(indexStore.String(), indexPath)),
		zap.String("local_index_path", options.localStorePath),
	)
	logger.Info("copying index entries to local KV store",
		zap.Time("index_recording_time", indexTime),
		zap.Stringer("blob_store", blob),
	)

	contextStores := []context2.Stores{stores}
	// include extra context stores to scan additional metadata
	contextStores = append(contextStores, options.extraStores...)

	// announce all contexts to be scanned
	for _, contextStore := range contextStores {
		logger.Info("about to scan metadata for context",
			zap.Stringer("context metadata", contextStore.Metadata()),
		)
	}

	// iterate over all contexts that share the same blob store
	var allKeysCount uint64
	for _, toPin := range contextStores {
		contextStore := toPin
		clogger := logger.With(
			zap.Stringer("context metadata", contextStore.Metadata()),
		)
		clogger.Info("scanning metadata for context", zap.Int("max_parallel", options.maxParallel))

		repos, erl := ListRepos(contextStore, ConcurrentList(options.maxParallel))
		if erl != nil {
			return nil, erl
		}

		clogger.Info("scanning repos", zap.Int("num_repos_in_context", len(repos)))

		reposGroup, gctx := errgroup.WithContext(ctx)
		reposGroup.SetLimit(options.maxParallel)
		monitorGroup, mctx := errgroup.WithContext(gctx)

		var count uint64
		// progress status reporting: report about progress and KV store size every 5 minutes
		monitorGroup.Go(func() error {
			interval := 5 * time.Minute
			ticker := time.NewTicker(interval)
			defer func() {
				ticker.Stop()
			}()

			var lastSeen uint64
			for {
				select {
				case <-ticker.C:
					keys := atomic.LoadUint64(&count)
					lsmSize, logSize := db.Size()
					dbSize := lsmSize + logSize
					throughput := int64(float64(keys-lastSeen) / interval.Seconds())
					clogger.Info("keys scanned so far for this context",
						zap.Uint64("keys", keys),
						zap.Uint64("keys_since_last_report", keys-lastSeen),
						zap.Int64("throuhput keys/s", throughput),
						zap.String("db_size", units.HumanSize(float64(dbSize))),
					)
					lastSeen = keys
				case <-mctx.Done():
					return nil
				}
			}
		})

		// iterate over all objects referred to by metadata in this context.
		// * for all repos, all bundles, all files, all keys in the root key
		for i, repoToPin := range repos {
			repo := repoToPin
			clogger.Info("in-progress percent of current context",
				zap.Int("percent_repos_in_context", int(float64(i)/float64(len(repos))*100.00)),
			)

			// scan repos in parallel
			reposGroup.Go(func() error {
				lg := clogger.With(
					zap.String("repo", repo.Name),
				)
				lg.Info("start scanning repo entries")
				var repoCount uint64

				erb := ListBundlesApply(repo.Name, contextStore, func(bundle model.BundleDescriptor) error {
					b := NewBundle(
						BundleID(bundle.ID),
						Repo(repo.Name),
						BundleDescriptor(&bundle),
						ContextStores(contextStore),
						Logger(zap.NewNop()), // mute verbosity on retrieving bundle details
					)

					keys, erk := bundleKeys(gctx, b, bundle.LeafSize, lg)
					if erk != nil {
						return erk
					}
					atomic.AddUint64(&count, uint64(len(keys)))
					atomic.AddUint64(&repoCount, uint64(len(keys)))

					for _, key := range keys {
						select {
						case <-gctx.Done():
							return gctx.Err()
						default:
						}

						eru := db.Update(func(txn *badger.Txn) error {
							return txn.Set([]byte(key), []byte{})
						})
						if eru != nil {
							return eru
						}
					}

					return nil
				},
					ConcurrentList(options.maxParallel),
					WithIgnoreCorruptedMetadata(true), // ignore when bundle.yaml is corrupted (e.g. empty file)
				)

				if erb != nil {
					lg.Error("could not retrieve keys for repo",
						zap.Uint64("repo_keys_so_far", repoCount),
						zap.Error(erb),
					)

					return erb
				}

				lg.Info("finished scanning repo entries",
					zap.Uint64("repo_keys", repoCount),
				)

				return nil
			})
		}

		erw := reposGroup.Wait()
		_ = monitorGroup.Wait()
		if erw != nil {
			return nil, erw
		}

		logger.Info("keys scanned over all repos for this context",
			zap.Stringer("context metadata", contextStore.Metadata()),
			zap.Uint64("num_keys", count),
		)
		allKeysCount += count
	}

	logger.Info("uploading index file to metadata",
		zap.String("index_file", indexPath),
		zap.Uint64("total_keys", allKeysCount),
	)

	dbReader := newDBReader(ctx, db, indexTime, logger)
	defer func() {
		_ = dbReader.Close()
	}()

	// make sure the overwrite does not leaves some trailing stuff
	err = indexStore.Delete(ctx, indexPath)
	if err != nil && !errors.Is(err, status.ErrNotExists) {
		return nil, err
	}

	// iterate over all deduplicated keys from KV and upload the index file
	// NOTE: we don't compute CRC here.
	err = indexStore.Put(ctx, indexPath, dbReader, storage.NoOverWrite)
	if err != nil {
		return nil, err
	}

	logger.Info("done uploading index file to metadata",
		zap.String("index_file", indexPath),
	)

	return &PurgeIndex{
		IndexTime:  indexTime,
		NumEntries: dbReader.Count(),
	}, nil
}

func bundleKeys(ctx context.Context, b *Bundle, size uint32, logger *zap.Logger) ([]string, error) {
	if err := unpackBundleFileList(ctx, b, false, defaultBundleEntriesPerFile); err != nil {
		logger.Error("the metadata for bundle is invalid", zap.String("bundle_id", b.BundleID), zap.Error(err))

		return nil, err
	}

	keys := make([]string, 0, 1024)

	logger.Info("unpacked file entries for bundle", zap.Int("num_entries", len(b.BundleEntries)))
	for _, entry := range b.BundleEntries {
		root, err := cafs.KeyFromString(entry.Hash)
		if err != nil {
			// The root key contains invalid bytes. We've never seen this issue so far: block the process.
			logger.Error("the root key is invalid", zap.String("key", entry.Hash), zap.Error(err))

			return nil, err
		}
		keys = append(keys, root.String())

		leaves, err := cafs.LeavesForHash(b.BlobStore(), root, size, "")
		if err != nil {
			// The root key is somehow corrupted. This might happen with objects created with previous versions of datamon:
			// ignore the leaves and just return the root key.
			logger.Warn("the root key is corrupted: indexing the root, skipping unavailable leaves",
				zap.String("entry", entry.NameWithPath),
				zap.String("key", entry.Hash), zap.Error(err))

			continue
		}

		for _, leaf := range leaves {
			keys = append(keys, leaf.String())
		}
	}

	return keys, nil
}

// PurgeDeleteUnused deletes blob entries that are not referenced by the reserve-lookup index.
func PurgeDeleteUnused(stores context2.Stores, opts ...PurgeOption) (*PurgeBlobs, error) {
	options := defaultPurgeOptions(opts)
	indexStore := getMetaStore(stores)
	indexPath := model.ReverseIndex()
	logger := options.l.With(
		zap.String("index_path", path.Join(indexStore.String(), indexPath)),
		zap.String("local_index_path", options.localStorePath),
	)
	ctx := context.Background() // no timeout here

	db, err := makeKV(options.localStorePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = db.Close()
	}()

	// 1. Download index and store it on a local badgerdb KV store
	r, err := indexStore.Get(ctx, indexPath)
	if err != nil {
		return nil, fmt.Errorf("open index in metadata: %w", err)
	}
	defer func() {
		_ = r.Close()
	}()

	logger.Info("copying index entries to local KV store")

	indexTime, numKeys, err := copyIndex(db, r)
	if err != nil {
		return nil, fmt.Errorf("copy index: %w", err)
	}

	// 2. Scan all keys in blob store
	blob := getBlobStore(stores)
	logger = logger.With(
		zap.Timep("index_creation_time", indexTime),
		zap.Stringer("blob_store", blob),
	)

	logger.Info("done with dumping index entries to local KV store", zap.Uint64("num_keys", numKeys))

	iterator := func(next string) ([]string, string, error) {
		return blob.KeysPrefix(ctx, next, "", "", batchSize)
	}

	logger.Info("scanning blob entries against index")

	// 3. Remove blob keys that are not indexed
	descriptor, err := scanBlob(ctx, blob, iterator, db, *indexTime, logger, options.dryRun, options.maxParallel)
	if err != nil {
		return nil, fmt.Errorf("scan blob: %w", err)
	}

	logger.Info("done with purging unused blobs",
		zap.Timep("index_creation_time", indexTime),
		zap.Stringer("blob", blob),
	)

	return descriptor, nil
}

func makeKV(pth string) (*badger.DB, error) {
	err := os.MkdirAll(pth, 0700)
	if err != nil {
		return nil, fmt.Errorf("makeKV: mkdir: %w", err)
	}

	db, err := badger.Open(
		badger.LSMOnlyOptions(pth).
			WithLoggingLevel(badger.WARNING).
			WithIndexCacheSize(100 << 20). // 100MB
			WithMetricsEnabled(false),
	)
	if err != nil {
		return nil, fmt.Errorf("open KV: %w", err)
	}

	//  scratch any pre-existing local index
	if err = db.DropAll(); err != nil {
		return nil, fmt.Errorf("scrach KV: %w", err)
	}

	return db, nil
}

func scanBlob(ctx context.Context, blob storage.Store, iterator func(string) ([]string, string, error), db *badger.DB, indexTime time.Time, logger *zap.Logger, dryRun bool, maxParallel int) (*PurgeBlobs, error) {
	var wg sync.WaitGroup
	doneWithKeysChan := make(chan struct{}, 1)
	keysChan := make(chan keyBatchEvent, 1)
	lookupGroup, gctx := errgroup.WithContext(ctx)
	lookupGroup.SetLimit(maxParallel + 1)
	monitorGroup, mctx := errgroup.WithContext(gctx)

	var scannedKeys, indexedEntries, moreRecentEntries, deletedEntries, deletedSize uint64

	monitorGroup.Go(func() error {
		interval := 5 * time.Minute
		ticker := time.NewTicker(interval)
		defer func() {
			ticker.Stop()
		}()

		var lastSeen uint64
		for {
			select {
			case <-ticker.C:
				keys := atomic.LoadUint64(&scannedKeys)
				throughput := int64(float64(keys-lastSeen) / interval.Seconds())
				logger.Info("keys scanned so far from blob store",
					zap.Uint64("keys", keys),
					zap.Uint64("keys_since_last_report", keys-lastSeen),
					zap.Int64("throuhput keys/s", throughput),
				)
				lastSeen = keys
			case <-mctx.Done():
				return nil
			}
		}
	})

	// fetch a blob keys asynchronously, in batches
	wg.Add(1)
	defer wg.Wait()
	go fetchKeys(iterator, keysChan, doneWithKeysChan, &wg) // scan for key batches

	// check against KV store for the existing of the key in the index: if not found, delete the blob key
	lookupGroup.Go(func() error {
		for {
			select {
			case <-gctx.Done():
				return gctx.Err()

			case batch, isOpen := <-keysChan:
				if !isOpen {
					// done with keys
					return nil
				}

				if batch.err != nil {
					logger.Error("fetching blob keys", zap.Error(batch.err))

					return batch.err
				}
				atomic.AddUint64(&scannedKeys, uint64(len(batch.keys)))

				// run up to maxParallel lookup & delete routines
				lookupGroup.Go(func() error {
					for _, key := range batch.keys {
						select {
						case <-gctx.Done():
							return gctx.Err()
						default:
						}

						if err := checkAndDeleteKey(gctx, db,
							indexTime, key, blob,
							logger, dryRun,
							&indexedEntries, &moreRecentEntries, &deletedEntries, &deletedSize,
						); err != nil {
							return err
						}
					}

					return nil
				})
			}
		}
	})

	if err := lookupGroup.Wait(); err != nil {
		logger.Error("waiting on index lookup", zap.Error(err))
		close(doneWithKeysChan) // interrupt background key scanning

		return nil, err
	}
	_ = monitorGroup.Wait()

	return &PurgeBlobs{
		IndexTime:         indexTime,
		ScannedEntries:    scannedKeys,
		IndexedEntries:    indexedEntries,
		MoreRecentEntries: moreRecentEntries,
		DeletedEntries:    deletedEntries,
		DeletedSize:       deletedSize,
		DryRun:            dryRun,
	}, nil
}

func checkAndDeleteKey(ctx context.Context,
	db *badger.DB, indexTime time.Time,
	key string, blob storage.Store,
	logger *zap.Logger, dryRun bool,
	indexedEntries, moreRecentEntries, deletedEntries, deletedSize *uint64,
) error {
	var croak func(string, ...zap.Field)
	logger = logger.With(zap.String("key", key))
	if dryRun {
		croak = logger.Info
	} else {
		croak = logger.Debug
	}

	err := db.View(func(txn *badger.Txn) error {
		_, e := txn.Get([]byte(key))

		return e
	})
	if err == nil {
		// key found in the index
		croak("key found in index: keeping blob")
		_ = atomic.AddUint64(indexedEntries, 1)

		return nil
	}

	if !errors.Is(err, badger.ErrKeyNotFound) {
		// some technical error occurred: interrupt
		logger.Error("searching index", zap.Error(err))

		return err
	}

	// key not found
	attrs, err := blob.GetAttr(ctx, key)
	if err != nil {
		logger.Error("retrieving blob attributes", zap.Error(err))

		return err
	}

	// the blob has been created after the index: skip
	if indexTime.Before(attrs.Updated) {
		croak("key more recent than index. Keeping blob", zap.Time("key_updated_at", attrs.Updated))
		_ = atomic.AddUint64(moreRecentEntries, 1)

		return nil
	}

	_ = atomic.AddUint64(deletedSize, uint64(attrs.Size))
	_ = atomic.AddUint64(deletedEntries, 1)

	if dryRun {
		croak("key to be deleted (dry-run)", zap.Int64("size", attrs.Size))

		return nil
	}

	// proceed with deletion from the blob store
	if err := blob.Delete(ctx, key); err != nil {
		logger.Error("deleting blob", zap.Error(err))

		return err
	}

	return nil
}

func copyIndex(db *badger.DB, r io.Reader) (indexTime *time.Time, numKeys uint64, err error) {
	scanner := bufio.NewScanner(r)
	isFirst := true

	for scanner.Scan() {
		key := scanner.Bytes()
		if isFirst {
			ts, erp := time.Parse(layout, string(key))
			if erp != nil {
				return nil, 0, erp
			}
			indexTime = &ts

			isFirst = false

			continue
		}

		// write key to local KV store. Payload is empty.
		eru := db.Update(func(txn *badger.Txn) error {
			return txn.Set(key, []byte{})
		})
		if eru != nil {
			return nil, numKeys, eru
		}
		numKeys++
	}

	if indexTime == nil {
		return nil, numKeys, errors.New("invalid index file: expect a RFC3339Nano time has header")
	}

	return indexTime, numKeys, nil
}

// PurgeUnlock removes the purge job lock from the metadata store.
func PurgeUnlock(stores context2.Stores, opts ...PurgeOption) error {
	store := getMetaStore(stores)
	path := model.PurgeLock()
	options := defaultPurgeOptions(opts)
	logger := options.l.With(
		zap.Stringer("index_metadata_store", store),
	)
	logger.Info("removing purge lock", zap.String("lock_file", path))

	return store.Delete(context.Background(), path)
}

// PurgeLock sets a purge job lock on the metadata store.
func PurgeLock(stores context2.Stores, opts ...PurgeOption) error {
	r := &bytes.Buffer{}
	fmt.Fprintf(r, "locked_at: %q\n", time.Now().UTC())
	store := getMetaStore(stores)
	options := defaultPurgeOptions(opts)
	logger := options.l.With(
		zap.Stringer("index_metadata_store", store),
	)

	var overwrite bool
	if options.force {
		overwrite = storage.OverWrite
	} else {
		overwrite = storage.NoOverWrite
	}

	path := model.PurgeLock()
	logger.Info("enabling purge lock", zap.String("lock_file", path))

	if err := store.Put(context.Background(), path, r, overwrite); err != nil {
		if strings.Contains(err.Error(), "googleapi: Error 412: Precondition Failed, conditionNotMet") {
			return fmt.Errorf("a lock exist [%s]: %v", path, stores.Metadata())
		}

		return err
	}

	return nil
}

func PurgeDropReverseIndex(stores context2.Stores, opts ...PurgeOption) error {
	store := getMetaStore(stores)
	path := model.ReverseIndex()
	options := defaultPurgeOptions(opts)
	logger := options.l.With(
		zap.Stringer("index_metadata_store", store),
	)

	logger.Info("deleting index file", zap.String("index_file", path))

	return store.Delete(context.Background(), path)
}

type dbReader struct {
	mx        sync.Mutex
	readOnce  bool
	indexTime time.Time
	group     *errgroup.Group
	out       chan []byte
	count     uint64
	logger    *zap.Logger
	partial   []byte
}

func newDBReader(ctx context.Context, db *badger.DB, indexTime time.Time, logger *zap.Logger) *dbReader {
	r := &dbReader{
		indexTime: indexTime,
		out:       make(chan []byte, 1024),
		logger:    logger,
	}

	g, gctx := errgroup.WithContext(ctx)
	r.group = g

	r.group.Go(r.iterateKV(gctx, db))

	return r
}

// iterateKV iterates over all keys and send them over the internal channel of this reader.
//
// NOTE(fred): should have used badger stream there. Maybe next time...
func (r *dbReader) iterateKV(ctx context.Context, db *badger.DB) func() error {
	return func() error {
		return db.View(func(txn *badger.Txn) error {
			defer func() {
				close(r.out)
			}()

			// iterate over all keys
			iterator := txn.NewIterator(badger.IteratorOptions{
				PrefetchSize:   1024,
				PrefetchValues: false,
			})
			defer iterator.Close()

			for iterator.Rewind(); iterator.Valid(); iterator.Next() {
				key := iterator.Item().KeyCopy(nil)

				select {
				case r.out <- key:
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			return nil
		})
	}
}

func (r *dbReader) Count() uint64 {
	r.mx.Lock()
	defer r.mx.Unlock()

	return r.count
}

func (r *dbReader) Read(p []byte) (int, error) {
	// TODO(fredbi): could probably do something more elegant here.
	var b []byte

	r.mx.Lock()
	defer r.mx.Unlock()

	if reminder := len(r.partial); reminder > 0 {
		// handle edge case of partially returned key
		if reminder > len(p) {
			b = r.partial[:len(p)]
			r.partial = r.partial[len(p):]
		} else {
			b = r.partial
			r.partial = nil
		}
	} else {
		if !r.readOnce {
			// first line read is the formatted index timestamp
			var buf bytes.Buffer
			fmt.Fprintln(&buf, r.indexTime.Format(layout))
			b = buf.Bytes()
			r.readOnce = true
		} else {
			key, isOpen := <-r.out
			if !isOpen {
				return 0, io.EOF
			}
			b = key
			b = append(b, '\n') // add newline to separate keys
			r.count++
		}

		if len(p) < len(b) {
			r.partial = b[len(p):]
			b = b[:len(p)]
		}
	}

	copy(p, b)

	return len(b), nil
}

func (r *dbReader) Close() error {
	return r.group.Wait()
}
