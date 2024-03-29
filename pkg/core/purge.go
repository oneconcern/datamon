package core

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
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

	db, err := openKV(options.localStorePath, options)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = db.Close()
	}()

	indexStore := getMetaStore(stores)
	indexPath := model.ReverseIndex()
	blob := getBlobStore(stores)

	// about to write the index in the current context
	logger := options.l.With(
		zap.String("index_path", indexStore.String()+"/"+indexPath),
		zap.String("local_index_path", options.localStorePath),
	)

	logger.Info("copying index entries to local KV store",
		zap.Time("index_recording_time", indexTime),
		zap.Stringer("blob_store", blob),
	)

	if options.resume {
		// reload existing index files into a fresh local KV store
		lastIndex, numKeys, ts, erp := preloadIndexFiles(ctx, stores, db, logger, options)
		if erp != nil {
			return nil, erp
		}
		if lastIndex > options.indexStart {
			options.indexStart = lastIndex // will resume index construction at the next chunk
		}
		indexTime = *ts // keep the creation date of the original index

		logger.Info("pre-existing index entries loaded into local KV store",
			zap.Time("index_recording_time", indexTime),
			zap.Uint64("num_keys", numKeys),
			zap.Uint64("resuming new index entries with index", options.indexStart),
			zap.Stringer("blob_store", blob),
		)
	}

	contextStores := []context2.Stores{stores}
	// include extra context stores to scan additional metadata
	contextStores = append(contextStores, options.extraStores...)

	// announce all contexts to be scanned
	for _, contextStore := range contextStores {
		logger.Info("about to scan metadata for context",
			zap.Stringer("context metadata", contextStore.Metadata()),
		)
	}

	cancellable, cancel := context.WithCancel(ctx)
	defer cancel()

	uploadGroup, uctx := errgroup.WithContext(cancellable) // goroutine to regularly upload chunks of indexes
	uploadGroup.SetLimit(1)                                // upload one index chunk at a time

	var allKeysCount, uploadedKeys, uniqueKeys uint64
	doneScanning := make(chan struct{})

	// background upload of index keys into chunks
	uploadGroup.Go(
		uploader(uctx, indexStore, indexTime, &uniqueKeys, &uploadedKeys, db, logger, doneScanning, options),
	)

	// iterate over all contexts that share the same blob store
	for _, toPin := range contextStores {
		contextStore := toPin

		// scan metadata for an entire context store
		if err = scanContext(ctx, contextStore, indexStore, &allKeysCount, &uniqueKeys, db, logger, options); err != nil {
			logger.Error("failed to scan context",
				zap.Stringer("context metadata", contextStore.Metadata()),
				zap.Error(err),
			)
			cancel() // signals the uploader to interrupt

			break
		}
	}

	if err == nil {
		close(doneScanning)

		logger.Info("all contexts successfully scanned. Waiting for all index keys to be uploaded",
			zap.Uint64("unique_keys", uniqueKeys),
			zap.Uint64("uploaded_keys_so_far", atomic.LoadUint64(&uploadedKeys)),
		)
	}

	if err = uploadGroup.Wait(); err != nil {
		logger.Error("failed to upload index", zap.Error(err))

		return nil, err
	}

	logger.Info("uploaded index file to metadata",
		zap.String("index_file", indexPath),
		zap.Uint64("total_keys", allKeysCount),
		zap.Uint64("unique_keys", uniqueKeys),
		zap.Uint64("uploaded_keys", uploadedKeys),
	)

	return &PurgeIndex{
		IndexTime:  indexTime,
		NumEntries: uploadedKeys,
	}, nil
}

func scanContext(ctx context.Context, contextStore context2.Stores, indexStore storage.Store, allKeysPtr, uniqueKeysPtr *uint64, db kvStore, logger *zap.Logger, options *purgeOptions) error {
	lg := logger.With(
		zap.Stringer("context metadata", contextStore.Metadata()),
	)

	lg.Info("scanning metadata for context",
		zap.Int("max_parallel", options.maxParallel),
	)

	parallelRepos := max(1, options.maxParallel/4) // reduce undue pressure on gcs
	repos, err := ListRepos(contextStore, ConcurrentList(parallelRepos))
	if err != nil {
		lg.Error("could not list repos for this context", zap.Error(err))

		return err
	}

	lg.Info("scanning repos",
		zap.Int("num_repos_in_context", len(repos)),
	)

	reposGroup, gctx := errgroup.WithContext(ctx) // goroutines scanning for keys in metadata
	reposGroup.SetLimit(options.maxParallel)
	monitorGroup, mctx := errgroup.WithContext(gctx) // goroutine for reporting activity

	var count uint64

	// progress status reporting: report about progress and KV store size every 5 minutes
	monitorGroup.Go(
		monitor(mctx, &count, uniqueKeysPtr, db, lg, options),
	)

	// iterate over all objects referred to by metadata in this context.
	// * for all repos, all bundles, all files, all keys in the root key
LOOP:
	for i, repoToPin := range repos {
		repo := repoToPin
		lg.Info("in-progress percent of current context",
			zap.Int("number of repos in this context", len(repos)),
			zap.Int("percent_repos_in_context", int(float64(i)/float64(len(repos))*100.00)),
		)
		select {
		case <-gctx.Done():
			// early failure
			break LOOP
		default:
		}

		// scan repos in parallel
		reposGroup.Go(
			repoKeysScanner(gctx, contextStore, repo, &count, uniqueKeysPtr, db, lg, options),
		)
	}

	erw := reposGroup.Wait()
	_ = monitorGroup.Wait()
	if erw != nil {
		return erw
	}

	lg.Info("keys scanned over all repos for this context",
		zap.Uint64("num_keys", count),
	)
	atomic.AddUint64(allKeysPtr, count)

	return nil
}

func monitor(ctx context.Context, countPtr, countUniquePtr *uint64, db kvStore, logger *zap.Logger, options *purgeOptions) func() error {
	return func() error {
		interval := options.monitorInterval
		ticker := time.NewTicker(interval)
		defer func() {
			ticker.Stop()
		}()

		var lastSeen uint64
		for {
			select {
			case <-ticker.C:
				keys := atomic.LoadUint64(countPtr)
				dbSize := db.Size()
				throughput := int64(float64(keys-lastSeen) / interval.Seconds())

				logger.Info("keys scanned so far for this context",
					zap.Uint64("keys", keys),
					zap.Uint64("unique_keys", atomic.LoadUint64(countUniquePtr)),
					zap.Uint64("keys_since_last_report", keys-lastSeen),
					zap.Int64("throughput keys scanned/s", throughput),
					zap.String("db_size", units.HumanSize(float64(dbSize))),
				)
				lastSeen = keys
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func repoKeysScanner(ctx context.Context, contextStore context2.Stores, repo model.RepoDescriptor, countPtr, countUniquePtr *uint64, db kvStore, logger *zap.Logger, options *purgeOptions) func() error {
	return func() error {
		lg := logger.With(
			zap.String("repo", repo.Name),
		)
		lg.Info("start scanning repo entries")
		var repoCount uint64

		// early failure
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		parallelBundles := max(10, options.maxParallel/4)
		err := ListBundlesApply(repo.Name, contextStore, func(bundle model.BundleDescriptor) error {
			b := NewBundle(
				BundleID(bundle.ID),
				Repo(repo.Name),
				BundleDescriptor(&bundle),
				ContextStores(contextStore),
				Logger(zap.NewNop()), // mute verbosity on retrieving bundle details
			)

			keys, erk := bundleKeys(ctx, b, bundle.LeafSize, db, lg)
			if erk != nil {
				return erk
			}
			atomic.AddUint64(countPtr, uint64(len(keys)))
			atomic.AddUint64(&repoCount, uint64(len(keys)))

			for _, key := range keys {
				select {
				// interrupted
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				if eru := db.SetIfNotExists([]byte(key), []byte{}); eru != nil {
					return eru
				}

				atomic.AddUint64(countUniquePtr, uint64(1))
			}

			return nil
		},
			ConcurrentList(parallelBundles),
			WithIgnoreCorruptedMetadata(true), // ignore when bundle.yaml is corrupted (e.g. empty file)
		)

		if err != nil {
			lg.Error("could not retrieve keys for repo",
				zap.Uint64("repo_keys_so_far", repoCount),
				zap.Error(err),
			)

			return err
		}

		lg.Info("finished scanning repo entries",
			zap.Uint64("repo_keys", repoCount),
		)

		return nil
	}
}

// uploader scans over newly written keys every 5 minutes and uploads a new chunk of keys.
func uploader(ctx context.Context, indexStore storage.Store, indexTime time.Time, uniqueKeysPtr, uploadKeysPtr *uint64, db kvStore, logger *zap.Logger, doneScanning <-chan struct{}, options *purgeOptions) func() error {
	return func() error {
		chunkSize := options.indexChunkSize
		interval := options.uploaderInterval
		ticker := time.NewTicker(interval)
		defer func() {
			ticker.Stop()
		}()

		var (
			lastSeen uint64
		)

		chunkIndex := options.indexStart

		chunkGroup, cctx := errgroup.WithContext(ctx)
		chunkGroup.SetLimit(1)

	LOOP:
		for {
			select {
			case <-ctx.Done():
				// the caller cancelled the upload
				_ = chunkGroup.Wait()

				return ctx.Err()

			case <-cctx.Done():
				// some chunk loader failed
				_ = chunkGroup.Wait()

				return cctx.Err()

			case <-ticker.C:
				logger.Info("uploaded keys so far",
					zap.Uint64("num_chunks", chunkIndex),
					zap.Uint64("uploaded_keys", atomic.LoadUint64(uploadKeysPtr)),
				)
				// iterate over newly inserted keys
				uniqueKeys := atomic.LoadUint64(uniqueKeysPtr)
				if uniqueKeys < lastSeen+chunkSize {
					// skip: not enough keys have been produced yet to start a chunk
					logger.Info("skipping keys upload for now. Not enough new keys")

					break
				}
				lastSeen = uniqueKeys

				chunkIndex++
				started := chunkGroup.TryGo(
					chunkUploader(cctx,
						chunkIndex, chunkSize,
						indexStore, indexTime, uploadKeysPtr, db, logger,
						options,
					),
				)
				if started {
					logger.Info("started index chunk upload",
						zap.Uint64("chunk", chunkIndex),
					)
				}
			case <-doneScanning:
				// no more keys are going to be produced
				logger.Info("uploader received done scanning signal")

				break LOOP
			}
		}

		if err := chunkGroup.Wait(); err != nil {
			return err
		}

		// write last chunks
		logger.Info("uploading last chunks: no more keys are being received")
		for {
			chunkIndex++
			uploadedBefore := atomic.LoadUint64(uploadKeysPtr)

			if err := chunkUploader(ctx,
				chunkIndex, chunkSize,
				indexStore, indexTime, uploadKeysPtr, db, logger,
				options,
			)(); err != nil {
				logger.Error("failed to flush index chunk",
					zap.Uint64("chunk", chunkIndex),
					zap.Error(err),
				)

				return err
			}

			uploaded := atomic.LoadUint64(uploadKeysPtr)
			if uploaded <= uploadedBefore {
				break
			}

			logger.Info("uploaded keys so far",
				zap.Uint64("num_chunks", chunkIndex-1),
				zap.Uint64("uploaded_keys", uploaded),
			)
		}

		logger.Info("upload index completed",
			zap.Uint64("num_chunks", chunkIndex),
			zap.Uint64("uploaded_keys", atomic.LoadUint64(uploadKeysPtr)),
		)

		return nil
	}
}

// uploads an index chunk file
func chunkUploader(ctx context.Context,
	chunkIndex, chunkSize uint64,
	indexStore storage.Store, indexTime time.Time, uploadKeysPtr *uint64, db kvStore, logger *zap.Logger,
	options *purgeOptions,
) func() error {
	return func() error {
		return backoff.Retry(func() error {
			indexFile := model.ReverseIndexFile(chunkIndex)
			dbReader := newDBReader(ctx, db, indexTime, logger, chunkSize)
			defer func() {
				_ = dbReader.Close()
			}()

			// make sure the overwrite does not leaves some trailing stuff
			if err := indexStore.Delete(ctx, indexFile); err != nil {
				if !errors.Is(err, status.ErrNotExists) {
					return err
				}
			}

			// iterate over all deduplicated keys from KV and upload the index file
			// NOTE: we don't compute CRC here.
			// Keys are marked when scanned over and next instance of the reader will skip those.
			if err := indexStore.Put(ctx, indexFile, dbReader, storage.NoOverWrite); err != nil {
				return err
			}

			uploaded := dbReader.Count()
			atomic.AddUint64(uploadKeysPtr, uploaded)

			logger.Info("done uploading index chunk file to metadata",
				zap.String("index_file", indexFile),
				zap.Uint64("keys_uploaded_in_chunk", uploaded),
			)

			return nil
		},
			backoff.WithContext(defaultBackoff(), ctx),
		)
	}
}

func defaultBackoff() backoff.BackOff {
	withRetry := backoff.NewExponentialBackOff()
	withRetry.MaxElapsedTime = 30 * time.Second
	withRetry.Reset()

	return withRetry
}

// insistentBackoff will try with up to 5 min delay between retries
func insistantBackoff() backoff.BackOff {
	withRetry := backoff.NewExponentialBackOff()
	withRetry.MaxElapsedTime = 600 * time.Second
	withRetry.Reset()

	return withRetry
}

func bundleKeys(ctx context.Context, b *Bundle, size uint32, db kvStore, logger *zap.Logger) ([]string, error) {
	if err := backoff.Retry(func() error {
		return unpackBundleFileList(ctx, b, false, defaultBundleEntriesPerFile)
	},
		backoff.WithContext(defaultBackoff(), ctx),
	); err != nil {
		logger.Warn("the metadata for this bundle cannot be read", zap.String("bundle_id", b.BundleID), zap.Error(err))

		// some metadata is incomplete and/or invalid: ignore errors
		return nil, nil
	}

	keys := make([]string, 0, 1024)

	logger.Debug("unpacked file entries for bundle", zap.Int("num_entries", len(b.BundleEntries)))
	for _, entry := range b.BundleEntries {
		root, err := cafs.KeyFromString(entry.Hash)
		if err != nil {
			// the root key contains invalid bytes. We've never seen this issue so far: block the process.
			logger.Error("the root key is invalid", zap.String("key", entry.Hash), zap.Error(err))

			return nil, err
		}
		key := root.String()
		found, err := db.Exists([]byte(key))
		if err != nil {
			return nil, err
		}

		if found {
			// the root key is found in store, no need to unpack it: we necessarily have all its leaves in store
			continue
		}

		keys = append(keys, key)

		// NOTE: this section issues a GET on remote store for this key and has been seen as the
		// limiting factor on the throughput of the index building job.
		// By skipping it on already existing root keys, we shall call this about 2.5x less often.
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
		zap.String("index_path", indexStore.String()+"/"+indexPath),
		zap.String("local_index_path", options.localStorePath),
	)
	ctx := context.Background() // no timeout here

	db, err := openKV(options.localStorePath, options)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = db.Close()
	}()

	// 1. Download index and save it on a local KV store
	logger.Info("copying index entries to local KV store")

	indexTime, numKeys, lastIndex, err := copyIndexChunks(ctx, db, indexStore, logger, options) // iterate over multiple index files
	if err != nil {
		return nil, fmt.Errorf("copy index: %w", err)
	}

	// 2. Scan all keys in blob store
	blob := getBlobStore(stores)
	logger = logger.With(
		zap.Stringer("blob_store", blob),
	)

	logger.Info("done with dumping index entries to local KV store",
		zap.Timep("index_creation_time", indexTime),
		zap.Uint64("num_keys", numKeys),
		zap.Uint64("last_index_file", lastIndex),
	)

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
	)

	return descriptor, nil
}

func scanBlob(ctx context.Context, blob storage.Store, iterator func(string) ([]string, string, error), db kvStore, indexTime time.Time, logger *zap.Logger, dryRun bool, maxParallel int) (*PurgeBlobs, error) {
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
				deletedKeys := atomic.LoadUint64(&deletedEntries)
				throughput := int64(float64(keys-lastSeen) / interval.Seconds())
				logger.Info("keys scanned so far from blob store",
					zap.Uint64("keys", keys),
					zap.Uint64("keys_since_last_report", keys-lastSeen),
					zap.Uint64("deleted_keys_so_far", deletedKeys),
					zap.Int64("throughput keys/s", throughput),
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
	db kvStore, indexTime time.Time,
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

	found, err := db.Exists([]byte(key))
	if found {
		// key found in the index
		croak("key found in index: keeping blob")
		_ = atomic.AddUint64(indexedEntries, 1)

		return nil
	}

	if err != nil {
		// some technical error occurred: interrupt
		logger.Error("searching index", zap.Error(err))

		return err
	}

	// key not found in index
	var attrs storage.Attributes
	if err = backoff.Retry(func() error {
		var e error
		attrs, e = blob.GetAttr(ctx, key)
		if !errors.Is(e, status.ErrNotExists) {
			return err
		}

		return nil
	},
		backoff.WithContext(insistantBackoff(), ctx),
	); err != nil {
		logger.Error("retrieving blob attributes", zap.Error(err))
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
	if err = backoff.Retry(func() error {
		e := blob.Delete(ctx, key)
		if !errors.Is(e, status.ErrNotExists) {
			return err
		}
		// under high pressure, google API often fails with: "googleapi: Error 503: We encountered an internal error. Please try again., backendError"

		return nil
	},
		backoff.WithContext(insistantBackoff(), ctx),
	); err != nil {
		logger.Error("deleting blob", zap.Error(err))
	}

	return nil
}

// copyIndexChunks iterates over all index chunks and loads the keys in the local KV store.
func copyIndexChunks(ctx context.Context, db kvStore, indexStore storage.Store, logger *zap.Logger, options *purgeOptions) (indexTime *time.Time, numKeys uint64, lastIndex uint64, err error) {
	iterator := func(next string) ([]string, string, error) {
		return indexStore.KeysPrefix(ctx, next, model.ReverseIndexPrefix(), "", 1024)
	}

	var (
		ks   []string
		next string
		mx   sync.Mutex
	)
	copyGroup, gctx := errgroup.WithContext(ctx)
	copyGroup.SetLimit(options.maxParallel)

	for {
		ks, next, err = iterator(next)
		if err != nil {
			return nil, numKeys, lastIndex, fmt.Errorf("iterating index chunks in metadata [%s]: %w", next, err)
		}

		if len(ks) == 0 {
			break
		}

		for _, toPin := range ks {
			chunk := toPin
			logger.Debug("loading index chunk", zap.String("chunk", chunk))

			index, erp := model.ReverseIndexChunk(chunk)
			if erp != nil {
				return nil, numKeys, lastIndex, fmt.Errorf("invalid index chunk file [%s]: %w", chunk, erp)

			}
			if index > lastIndex {
				lastIndex = index
			}

			copyGroup.Go(func() error {
				r, e := indexStore.Get(gctx, chunk)
				if e != nil {
					return fmt.Errorf("open index chunk in metadata [%s]: %w", chunk, e)
				}
				defer func() {
					_ = r.Close()
				}()

				ts, loadedKeys, e := loadChunk(gctx, db, r)
				if e != nil {
					return fmt.Errorf("loading index chunk in metadata [%s]: %w", chunk, e)
				}
				logger.Debug("index chunk keys",
					zap.String("chunk", chunk),
					zap.Uint64("num_keys_in_chunk", loadedKeys),
				)

				if ts != nil {
					mx.Lock()
					indexTime = ts
					mx.Unlock()
				}

				atomic.AddUint64(&numKeys, loadedKeys)
				logger.Info("loaded index chunk", zap.String("chunk", chunk))

				return nil
			})
		}

		if next == "" {
			break
		}
	}

	if err := copyGroup.Wait(); err != nil {
		return indexTime, numKeys, lastIndex, fmt.Errorf("copy index chunks: %w", err)
	}
	logger.Debug("loaded all index chunks", zap.Uint64("num_keys", numKeys), zap.Uint64("last_chunk", lastIndex))
	if err := db.Compact(); err != nil {
		return indexTime, numKeys, lastIndex, err
	}
	logger.Debug("index compacted")

	return indexTime, numKeys, lastIndex, nil
}

func loadChunk(ctx context.Context, db kvStore, r io.Reader) (*time.Time, uint64, error) {
	scanner := bufio.NewScanner(r)
	isFirst := true
	var (
		indexTime *time.Time
		numKeys   uint64
	)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			// interrupt scanning whenever the group context is cancelled
			return indexTime, numKeys, ctx.Err()
		default:
		}

		key := scanner.Bytes()
		if isFirst {
			ts, err := time.Parse(layout, string(key))
			if err != nil {
				return nil, 0, err
			}
			indexTime = &ts

			isFirst = false

			continue
		}

		// write key to local KV store. Payload is marked as "uploaded", to support the resume use-case.
		if err := db.Set(key, []byte("X")); err != nil {
			return nil, numKeys, err
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

// PurgeDropReverseIndex drop all index chunks in the metadata store
//
// TODO(fred): nice to have - should factorize index chunk iterations.
func PurgeDropReverseIndex(stores context2.Stores, opts ...PurgeOption) error {
	indexStore := getMetaStore(stores)
	indexPath := model.ReverseIndex()
	options := defaultPurgeOptions(opts)
	logger := options.l.With(
		zap.Stringer("index_metadata_store", indexStore),
	)
	ctx := context.Background()

	logger.Info("deleting index files", zap.String("index_prefix", indexPath))
	iterator := func(next string) ([]string, string, error) {
		return indexStore.KeysPrefix(ctx, next, model.ReverseIndexPrefix(), "", 1024)
	}

	var (
		ks   []string
		next string
		err  error
	)

	for {
		ks, next, err = iterator(next)
		if err != nil {
			return fmt.Errorf("iterating index chunks in metadata [%s]: %w", next, err)
		}

		if len(ks) == 0 {
			break
		}

		for _, chunk := range ks {
			if err = indexStore.Delete(ctx, chunk); err != nil {
				return fmt.Errorf("delete index chunk in metadata [%s]: %w", chunk, err)
			}
		}

		if next == "" {
			break
		}
	}

	return nil
}

type dbReader struct {
	mx        sync.Mutex
	db        kvStore
	readOnce  bool
	indexTime time.Time
	group     *errgroup.Group
	out       chan []byte
	count     uint64
	logger    *zap.Logger
	partial   []byte
	maxKeys   uint64
}

func newDBReader(ctx context.Context, db kvStore, indexTime time.Time, logger *zap.Logger, maxKeys uint64) *dbReader {
	r := &dbReader{
		db:        db,
		indexTime: indexTime,
		out:       make(chan []byte, 1024),
		logger:    logger,
		maxKeys:   maxKeys,
	}

	g, gctx := errgroup.WithContext(ctx)
	r.group = g

	r.group.Go(r.iterateKV(gctx, r.db))

	return r
}

// iterateKV iterates over all keys and send them over the internal channel of this reader.
func (r *dbReader) iterateKV(ctx context.Context, db kvStore) func() error {
	return func() error {
		defer func() {
			close(r.out)
		}()

		iterator := db.AllKeys()
		defer func() {
			_ = iterator.Close()
		}()

		var iterated, skipped uint64
		defer func() {
			r.logger.Debug("skipped keys for this upload iteration", zap.Uint64("skipped", skipped))
		}()

		for iterator.Next() {
			key, val, err := iterator.Item()
			if err != nil {
				return fmt.Errorf("failed to fetch KV value [%s]: %w", key, err)
			}

			if len(val) > 0 {
				// key has been marked as already uploaded: skip
				skipped++

				continue
			}

			iterated++
			if iterated > r.maxKeys {
				return nil
			}

			select {
			case r.out <- key:
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		return nil
	}
}

func (r *dbReader) Count() uint64 {
	r.mx.Lock()
	defer r.mx.Unlock()

	return r.count
}

func (r *dbReader) Read(p []byte) (int, error) {
	// TODO(fredbi): nice to have - could probably do something more elegant here.
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

			// mark key as read in the DB
			if err := r.db.Set(key, []byte("X")); err != nil {
				return 0, fmt.Errorf("failed to mark KV key as read: %w", err)
			}

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

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func preloadIndexFiles(ctx context.Context, stores context2.Stores, db kvStore, logger *zap.Logger, options *purgeOptions) (uint64, uint64, *time.Time, error) {
	indexStore := getMetaStore(stores)
	indexTime, numKeys, lastIndex, err := copyIndexChunks(ctx, db, indexStore, logger, options) // iterate over multiple index files
	if err != nil {
		return lastIndex, numKeys, indexTime, fmt.Errorf("copy index: %w", err)
	}

	return lastIndex, numKeys, indexTime, nil
}
