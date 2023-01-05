package core

import (
	"fmt"
	"os"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/dgraph-io/badger/v3"
	badgeroptions "github.com/dgraph-io/badger/v3/options"
	"github.com/oneconcern/datamon/pkg/errors"
)

type (
	// kvBAdger provides a KV store implementation based on dgraph-io/badger/v3
	kvBadger struct {
		*badger.DB
	}

	kvBadgerIterator struct {
		isFirst  bool
		txn      *badger.Txn
		iterator *badger.Iterator
	}
)

func (kv *kvBadger) Drop() error {
	return kv.DB.DropAll()
}

func (kv *kvBadger) Size() uint64 {
	lsmSize, logSize := kv.DB.Size()
	dbSize := lsmSize + logSize

	return uint64(dbSize)
}

func (kv *kvBadger) AllKeys() kvIterator {
	txn := kv.DB.NewTransaction(false)
	iterator := txn.NewIterator(badger.IteratorOptions{
		PrefetchSize:   1024,
		PrefetchValues: true,
	})

	return &kvBadgerIterator{
		isFirst:  true,
		txn:      txn,
		iterator: iterator,
	}
}

func (kv *kvBadger) Get(key []byte) ([]byte, error) {
	var value []byte
	err := kv.DB.View(func(txn *badger.Txn) error {
		item, e := txn.Get(key)
		if e != nil {
			return e
		}
		value, e = item.ValueCopy(nil)

		return e
	})

	return value, err
}

func (kv *kvBadger) Exists(key []byte) (bool, error) {
	err := kv.DB.View(func(txn *badger.Txn) error {
		_, e := txn.Get(key)

		return e
	})
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return false, nil
		}

		// some technical error occurred: interrupt
		return false, err
	}

	return true, nil
}

func (kv *kvBadger) Set(key, value []byte) error {
	return backoff.Retry(func() error {
		err := kv.DB.Update(func(txn *badger.Txn) error {
			e := txn.Set(key, value)
			if e != nil {
				if errors.Is(e, badger.ErrConflict) {
					return e // retry
				}

				return backoff.Permanent(e)
			}

			return nil
		})

		return err
	},
		backoff.NewConstantBackOff(10*time.Millisecond),
	)
}

func (kv *kvBadger) SetIfNotExists(key, value []byte) error {
	return backoff.Retry(func() error {
		return kv.DB.Update(func(txn *badger.Txn) error {
			_, err := txn.Get(key)
			if err == nil {
				return nil
			}

			if !errors.Is(err, badger.ErrKeyNotFound) {
				return backoff.Permanent(err)
			}

			err = txn.Set(key, value)
			if err != nil {
				if errors.Is(err, badger.ErrConflict) {
					return err // retry
				}

				return backoff.Permanent(err)
			}

			return nil
		})
	},
		backoff.NewConstantBackOff(10*time.Millisecond),
	)

}

func (kv *kvBadger) Compact() error {
	return kv.DB.Flatten(100)
}

func (i *kvBadgerIterator) Next() bool {
	if i.isFirst {
		i.iterator.Rewind()
		i.isFirst = false

		return i.iterator.Valid()
	}

	i.iterator.Next()

	return i.iterator.Valid()
}

func (i *kvBadgerIterator) Item() ([]byte, []byte, error) {
	key := i.iterator.Item().KeyCopy(nil)
	val, err := i.iterator.Item().ValueCopy(nil)

	return key, val, err
}

func (i *kvBadgerIterator) Close() error {
	i.iterator.Close()
	i.txn.Discard()

	return nil
}

func makeKVBadger(pth string, options *purgeOptions) (*kvBadger, error) {
	err := os.MkdirAll(pth, 0700)
	if err != nil {
		return nil, fmt.Errorf("makeKV: mkdir: %w", err)
	}

	// we queue up one compactor to back each key inserting goroutine.
	// Let's hope this is enough to keep up with keys insertion.
	compactors := max(4, int(float64(options.maxParallel)*1.5))

	db, err := badger.Open(
		badger.LSMOnlyOptions(pth).
			WithLoggingLevel(badger.WARNING).
			WithMetricsEnabled(true).            // need to enable this in order to collect a reporting of the DB size
			WithCompression(badgeroptions.None). // a set of keys that are random hashes is unlikely to compress well
			WithNumCompactors(compactors).       // need quite a few compactors, or the DB grows exceedingly fast
			//
			// Badger tunables...
			// Badger DB defaults seem not suitable for a large operation like the purge job.
			// After ~ 50 millions keys are inserted, the DB performance grinds to a halt. Processing
			// keys becomes 10x slower than at the start of the process. Later on, the insertion speed degrades even more,
			// so we end up with a 20x slower pace... At this point, progress is super slow, almost halted.
			// The DB size remains rather stable at around 14 GB and grows only very slowly.
			//
			// I suspect the compaction process to become prominent at this stage.
			//
			// The specifics of our workload are:
			// * we store mostly keys. Values are either empty or a single character (to flag uploaded keys)
			// * compression is futile (we store random-like hashes)
			// * no specific ordering in how keys appear
			// * # get/set to try insertion ~ 1 billion
			// * # unique keys  ~ 100-500 millions (stopped so far at ~ 85 millions unique keys)
			WithIndexCacheSize(options.kvIndexCacheSize).                   // 0 -> 200MB
			WithBaseLevelSize(options.kvBaseLevelSize).                     // 10MB (default)
			WithBaseTableSize(options.kvBaseTableSize).                     // 2MB (default)
			WithLevelSizeMultiplier(options.kvLevelSizeMultiplier).         // 10 (default)
			WithMaxLevels(options.kvMaxLevels).                             // 7 (default)
			WithMemTableSize(options.kvMemTableSize).                       // 64MB (default) - ~ 500k keys per table
			WithNumLevelZeroTables(options.kvNumLevelZeroTables).           // 5 (default)
			WithNumLevelZeroTablesStall(options.kvNumLevelZeroTablesStall). // 10 -> 500 - stall will occur later
			WithNumMemtables(options.kvNumMemTables).                       // 5 (default)
			WithBlockCacheSize(options.kvBlockCacheSize),                   // disabled by default (badger default: 256MB)
	)
	if err != nil {
		return nil, fmt.Errorf("open KV: %w", err)
	}

	//  scratch any pre-existing local index
	if err = db.DropAll(); err != nil {
		return nil, fmt.Errorf("scrach KV: %w", err)
	}

	return &kvBadger{DB: db}, nil
}
