package core

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/oneconcern/datamon/pkg/cafs"
	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/errors"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	layout      = time.RFC3339Nano
	batchSize   = 1024
	maxParallel = 10
)

// PurgeBuildReverseIndex creates or update a reverse-lookip index
// of all used blob keys.
func PurgeBuildReverseIndex(stores context2.Stores, opts ...PurgeOption) error {
	// 1. scan all repos, all bundles
	// 2. Fetch root key, explode root key
	// 3. Add root key and children keys to index
	// 4. Write index file (with overwrite)

	options := defaultPurgeOptions(opts)
	ctx := context.Background() // no timeout here
	indexTime := time.Now().UTC()

	db, err := makeKV(options.localStorePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()

	indexStore := getMetaStore(stores)
	blob := getBlobStore(stores)
	logger := options.l.With(
		zap.String("path", options.localStorePath),
		zap.Stringer("index_metadata_store", indexStore),
	)
	logger.Info("copying index entries to local KV store",
		zap.Time("index_recording_time", indexTime),
		zap.Stringer("blob_store", blob),
	)

	repos, err := ListRepos(stores)
	if err != nil {
		return err
	}

	// iterate over all objects referred to by the metadata
	// * for all repos, all bundles, all files, all keys in the root key
	for _, repo := range repos {
		erb := ListBundlesApply(repo.Name, stores, func(bundle model.BundleDescriptor) error {
			b := NewBundle(
				BundleID(bundle.ID),
				Repo(repo.Name),
				BundleDescriptor(&bundle),
				ContextStores(stores),
			)

			keys, erk := bundleKeys(ctx, b, bundle.LeafSize)
			if erk != nil {
				return erk
			}

			for _, key := range keys {
				eru := db.Update(func(txn *badger.Txn) error {
					return txn.Set([]byte(key), []byte{})
				})
				if eru != nil {
					return eru
				}
			}

			return nil
		})
		if erb != nil {
			return erb
		}
	}

	indexPath := model.ReverseIndex()
	logger.Info("uploading index file to metadata",
		zap.String("index_file", indexPath),
	)

	dbReader := newDBReader(ctx, db, indexTime)
	defer func() {
		_ = dbReader.Close()
	}()

	// iterate over all deduplicated keys from KV and upload the index file
	// NOTE: we don't compute CRC here.
	err = indexStore.Put(ctx, indexPath, dbReader, storage.OverWrite)
	if err != nil {
		return err
	}

	logger.Info("done uploading index file to metadata",
		zap.String("index_file", indexPath),
	)

	return nil
}

func bundleKeys(ctx context.Context, b *Bundle, size uint32) ([]string, error) {
	if err := unpackBundleFileList(ctx, b, false, defaultBundleEntriesPerFile); err != nil {
		return nil, err
	}

	keys := make([]string, 0, 1024)

	for _, entry := range b.BundleEntries {
		root, err := cafs.KeyFromString(entry.Hash)
		if err != nil {
			return nil, err
		}
		keys = append(keys, root.String())

		leaves, err := cafs.LeavesForHash(b.BlobStore(), root, size, "")
		if err != nil {
			return nil, err
		}
		for _, leaf := range leaves {
			keys = append(keys, leaf.String())
		}
	}

	return keys, nil
}

// PurgeDeleteUnused deletes blob entries that are not referenced by the reserve-lookup index.
func PurgeDeleteUnused(stores context2.Stores, opts ...PurgeOption) error {
	options := defaultPurgeOptions(opts)
	indexStore := getMetaStore(stores)
	indexPath := model.ReverseIndex()
	logger := options.l.With(zap.String("path", options.localStorePath))
	ctx := context.Background() // no timeout here

	db, err := makeKV(options.localStorePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()

	// 1. Download index and store it on a local badgerdb KV store
	r, err := indexStore.Get(ctx, indexPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = r.Close()
	}()

	logger.Info("copying index entries to local KV store",
		zap.Stringer("index_metadata_store", indexStore),
	)

	indexTime, err := copyIndex(db, r)
	if err != nil {
		return err
	}

	// 2. Scan all keys in blob store
	blob := getBlobStore(stores)
	logger = logger.With(
		zap.Timep("index_creation_time", indexTime),
		zap.Stringer("blob_store", blob),
	)

	logger.Info("done with dumping index entries to local KV store")

	iterator := func(next string) ([]string, string, error) {
		return blob.KeysPrefix(ctx, next, "", "/", batchSize)
	}

	logger.Info("scanning blob entries against index")

	// 3. Remove blob keys that are not indexed
	if err = scanBlob(ctx, blob, iterator, db, *indexTime); err != nil {
		return err
	}

	logger.Info("done with purging unused blobs",
		zap.Timep("index_creation_time", indexTime),
		zap.String("path", options.localStorePath),
		zap.Stringer("blob", blob),
	)

	return nil
}

func makeKV(pth string) (*badger.DB, error) {
	db, err := badger.Open(badger.DefaultOptions(pth))
	if err != nil {
		return nil, err
	}

	//  scratch any pre-existing local index
	if err = db.DropAll(); err != nil {
		return nil, err
	}

	return db, nil
}

func scanBlob(ctx context.Context, blob storage.Store, iterator func(string) ([]string, string, error), db *badger.DB, indexTime time.Time) error {
	var (
		wg  sync.WaitGroup
		erc error
	)
	doneWithKeysChan := make(chan struct{}, 1)
	keysChan := make(chan keyBatchEvent, 1)
	cancellable, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
	}()

	lookupGroup, gctx := errgroup.WithContext(cancellable)
	lookupGroup.SetLimit(maxParallel)

	// fetch a blob keys asynchronously, in batches
	wg.Add(1)
	go fetchKeys(iterator, keysChan, doneWithKeysChan, &wg) // scan for key batches

	// check against KV store for the existing of the key in the index: if not found, delete the blob key
	wg.Add(1)
	go func(batchChan <-chan keyBatchEvent, doneChan <-chan struct{}, wg *sync.WaitGroup) {
		defer func() {
			wg.Done()
		}()

		for {
			select {
			case <-gctx.Done():
				erc = gctx.Err()

				return

			case <-doneChan:
				return

			case batch, isOpen := <-batchChan:
				if !isOpen {
					return
				}

				if batch.err != nil {
					erc = batch.err
					cancel()

					return
				}

				// run up to maxParallel lookup & delete routines
				lookupGroup.TryGo(func() error {
					for _, key := range batch.keys {
						err := db.View(func(txn *badger.Txn) error {
							_, e := txn.Get([]byte(key))

							return e
						})
						if err == nil {
							// key found in the index
							continue
						}

						if !errors.Is(err, badger.ErrKeyNotFound) {
							// some technical error occurred: interrupt
							return err
						}

						// key not found
						attrs, err := blob.GetAttr(gctx, key)
						if err != nil {
							return err
						}

						// the blob has been created after the index: skip
						if !indexTime.Before(attrs.Updated) {
							continue
						}

						// proceed with deletion from the blob store
						if err := blob.Delete(gctx, key); err != nil {
							return err
						}
					}

					return nil
				})
			}
		}
	}(keysChan, doneWithKeysChan, &wg)

	erg := lookupGroup.Wait()
	if erg != nil {
		// interrupt key scanning
		close(doneWithKeysChan)
	}

	wg.Wait()

	if erc == nil {
		erc = erg
	}

	return erc
}

func copyIndex(db *badger.DB, r io.Reader) (indexTime *time.Time, err error) {
	scanner := bufio.NewScanner(r)
	isFirst := true

	for scanner.Scan() {
		key := scanner.Bytes()
		if isFirst {
			ts, erp := time.Parse(layout, string(key))
			if erp != nil {
				return nil, erp
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
			return nil, eru
		}
	}

	if indexTime == nil {
		return nil, errors.New("invalid index file: expect a RFC3339Nano time has header")
	}

	return indexTime, nil
}

// PurgeUnlock removes the purge job lock from the metadata store.
func PurgeUnlock(stores context2.Stores) error {
	store := getMetaStore(stores)
	path := model.PurgeLock()

	return store.Delete(context.Background(), path)
}

// PurgeLock sets a purge job lock on the metadata store.
func PurgeLock(stores context2.Stores, opts ...PurgeOption) error {
	r := &bytes.Buffer{}
	fmt.Fprintf(r, "locked_at: %q\n", time.Now().UTC())
	store := getMetaStore(stores)
	options := defaultPurgeOptions(opts)

	var overwrite bool
	if options.force {
		overwrite = storage.OverWrite
	} else {
		overwrite = storage.NoOverWrite
	}

	path := model.PurgeLock()
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

	return store.Delete(context.Background(), path)
}

type dbReader struct {
	firstRead bool
	indexTime time.Time
	group     *errgroup.Group
	out       chan []byte
}

func newDBReader(ctx context.Context, db *badger.DB, indexTime time.Time) *dbReader {
	r := &dbReader{
		indexTime: indexTime,
		out:       make(chan []byte, 1024),
	}

	g, gctx := errgroup.WithContext(ctx)
	r.group = g

	r.group.Go(r.iterateKV(gctx, db))

	return r
}

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

			for {
				iterator.Next()
				if !iterator.Valid() {
					return nil
				}

				key := iterator.Item().Key()
				select {
				case r.out <- key:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		})
	}
}

func (r *dbReader) Read(p []byte) (int, error) {
	var b []byte
	if r.firstRead {
		// first line read is the formatted index timestamp
		var buf bytes.Buffer
		fmt.Fprintln(&buf, r.indexTime.Format(layout))
		b = buf.Bytes()
	} else {
		key, isOpen := <-r.out
		if !isOpen {
			return 0, io.EOF
		}
		b = key
		b = append(b, '\n') // add newline to separate keys
	}

	if len(p) < len(b) {
		copy(p, b[:len(p)])

		return len(p), nil
	}

	copy(p, b)

	return len(b), nil
}

func (r *dbReader) Close() error {
	return r.group.Wait()
}
