package localfs

import (
	"container/heap"
	"context"
	"log"
	"path/filepath"
	"sync"

	"github.com/json-iterator/go"

	"github.com/dgraph-io/badger"
	"github.com/oneconcern/datamon/pkg/store"
)

// NewSnapshotStore creates a localfs backed bundle store.
func NewSnapshotStore(baseDir string) store.SnapshotStore {
	b := &localSnapshotStore{
		baseDir: baseDir,
	}
	return b
}

type localSnapshotStore struct {
	baseDir string
	db      *badger.DB
	init    sync.Once
	close   sync.Once
}

func (l *localSnapshotStore) Initialize() error {
	var err error

	l.init.Do(func() {
		var db *badger.DB
		db, err = makeBadgerDb(filepath.Join(l.baseDir, snapshotDb))
		if err != nil {
			return
		}
		l.db = db
	})

	return err
}
func (l *localSnapshotStore) Close() error {
	var err error

	l.close.Do(func() {
		if l.db != nil {
			err = l.db.Close()
			if err == nil {
				l.db = nil
			}
		}
	})

	return err
}

func (l *localSnapshotStore) Get(ctx context.Context, hash string) (*store.Snapshot, error) {
	var result *store.Snapshot
	berr := l.db.View(func(tx *badger.Txn) error {
		sn, err := mapSnapshotItemError(tx.Get(snapshotKey(hash)))
		if err != nil {
			return err
		}
		result = &sn
		return nil
	})
	if berr != nil {
		return nil, berr
	}

	return result, nil
}

func (l *localSnapshotStore) GetForBundle(ctx context.Context, hash string) (*store.Snapshot, error) {
	var result *store.Snapshot
	berr := l.db.View(func(tx *badger.Txn) error {
		bn, err := tx.Get(bundleSnapshotKey(hash))
		if err != nil {
			return err
		}
		bk, err := bn.Value()
		if err != nil {
			return err
		}
		sn, err := mapSnapshotItemError(tx.Get(snapshotKeyBytes(bk)))
		if err != nil {
			return err
		}
		result = &sn
		return nil
	})
	if berr != nil {
		return nil, berr
	}

	return result, nil
}

func (l *localSnapshotStore) Create(ctx context.Context, bundle *store.Bundle) (*store.Snapshot, error) {
	var result *store.Snapshot
	berr := l.db.Update(func(txn *badger.Txn) error {
		var snapshots store.Snapshots
		heap.Init(&snapshots)

		var ids []string
		for _, v := range bundle.Parents {
			item, err := txn.Get(commitKey(v))
			if err != nil {
				return err
			}

			vb, err := item.Value()
			if err != nil {
				return err
			}

			snapshot, err := mapSnapshotItemError(txn.Get(snapshotKeyBytes(vb)))
			if err != nil {
				if err == store.SnapshotNotFound {
					log.Printf("skipping %s because: %v", vb, err)
					continue
				}
				return err
			}
			ids = append(ids, store.UnsafeBytesToString(vb))
			heap.Push(&snapshots, snapshot)
		}

		// start from the previous snapshot
		var previous store.Entries
		mrs := heap.Pop(&snapshots)
		if mrs != nil {
			previous = mrs.(*store.Snapshot).Entries.FlattenToRecent()
		}
		current := previous.With(bundle.Changes.Added).Without(bundle.Changes.Deleted).FlattenToRecent()

		id, err := current.Hash()
		if err != nil {
			return err
		}

		var snapshot store.Snapshot
		snapshot.NewCommit = bundle.ID
		snapshot.Parents = ids
		snapshot.PreviousCommits = bundle.Parents
		snapshot.Timestamp = bundle.Timestamp
		snapshot.Entries = current
		snapshot.ID = id

		b, err := jsoniter.Marshal(snapshot)
		if err != nil {
			return err
		}
		if err = txn.Set(snapshotKey(id), b); err != nil {
			return err
		}
		bsk := bundleSnapshotKey(bundle.ID)
		kb := store.UnsafeStringToBytes(id)
		if err = txn.Set(bsk, kb); err != nil {
			return err
		}
		result = &snapshot
		return nil
	})

	if berr != nil {
		return nil, berr
	}
	return result, nil
}
