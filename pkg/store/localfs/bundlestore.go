package localfs

import (
	"context"
	"time"

	"path/filepath"
	"sync"

	"github.com/dgraph-io/badger"
	"github.com/json-iterator/go"
	"github.com/oneconcern/trumpet/pkg/store"
)

// NewBundleStore creates a localfs backed bundle store.
func NewBundleStore(baseDir string) store.BundleStore {
	b := &localBundleStore{
		baseDir: baseDir,
	}
	return b
}

type localBundleStore struct {
	baseDir string
	db      *badger.DB
	init    sync.Once
	close   sync.Once
}

func (l *localBundleStore) Initialize() error {
	var err error

	l.init.Do(func() {
		var db *badger.DB
		db, err = makeBadgerDb(filepath.Join(l.baseDir, indexDb))
		if err != nil {
			return
		}
		l.db = db
	})

	return err
}
func (l *localBundleStore) Close() error {
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

func (l *localBundleStore) ListTopLevel(ctx context.Context) ([]store.Bundle, error) {
	return l.findCommitsByPrefix("", false)
}

func (l *localBundleStore) ListTopLevelIDs(ctx context.Context) ([]string, error) {
	res, err := l.findCommitsByPrefix("", true)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(res))
	for i, v := range res {
		result[i] = v.ID
	}
	return result, nil
}

func (l *localBundleStore) Get(ctx context.Context, hash string) (*store.Bundle, error) {
	var bundle *store.Bundle
	berr := l.db.View(func(tx *badger.Txn) error {
		b, err := mapBundleItemError(tx.Get(commitKey(hash)))
		if err != nil {
			return err
		}
		bundle = &b
		return nil
	})
	if berr != nil {
		return nil, berr
	}
	return bundle, nil
}

func (l *localBundleStore) GetObject(ctx context.Context, hash string) (store.Entry, error) {
	var entry store.Entry
	verr := l.db.View(func(tx *badger.Txn) error {
		var err error
		entry, err = mapEntryError(tx.Get(objectKey(hash)))
		return err
	})
	if verr != nil {
		return store.Entry{}, verr
	}

	return entry, nil
}

func (l *localBundleStore) GetObjectForPath(ctx context.Context, path string) (store.Entry, error) {
	var entry store.Entry
	verr := l.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(pathKey(path))
		if err != nil {
			return mapObjectError(err)
		}
		vb, err := item.Value()
		if err != nil {
			return mapObjectError(err)
		}
		entry, err = mapEntryError(tx.Get(objectKeyBytes(vb)))
		return err
	})
	if verr != nil {
		return store.Entry{}, verr
	}

	return entry, nil
}

func (l *localBundleStore) Create(ctx context.Context, message, branch, snapshot string, parents []string, changes store.ChangeSet) (string, bool, error) {
	key, err := changes.Hash()
	if err != nil {
		return "", true, err
	}

	b := store.Bundle{
		ID:         key,
		Message:    message,
		Changes:    changes,
		Parents:    parents,
		IsSnapshot: snapshot != "",
		Timestamp:  time.Now(),
		Committers: []store.Contributor{
			{Name: "Ivan Porto Carrero", Email: "ivan@oneconcern.com"},
		},
	}
	serr := l.db.Update(func(tx *badger.Txn) error {
		data, err1 := jsoniter.Marshal(b)
		if err1 != nil {
			return err1
		}
		for _, a := range changes.Added {
			bb, err2 := jsoniter.Marshal(a)
			if err2 != nil {
				return err2
			}
			hb := store.UnsafeStringToBytes(a.Hash)
			if err2 = tx.Set(objectKeyBytes(hb), bb); err2 != nil {
				return err2
			}
			if err2 = tx.Set(pathKey(a.Path), hb); err2 != nil {
				return err2
			}
		}

		kb := store.UnsafeStringToBytes(key)
		if err = tx.Set(commitKeyBytes(kb), data); err != nil {
			return err
		}
		return tx.Set(branchKey(branch), kb)
	})
	if serr != nil {
		return "", true, serr
	}
	return key, false, nil
}

func (l *localBundleStore) ListTags(ctx context.Context) ([]string, error) {
	return l.listKeys(tagKey(""))
}

func (l *localBundleStore) HashForTag(ctx context.Context, tag string) (string, error) {
	return l.hashFor(tagKey(tag))
}

func (l *localBundleStore) CreateTag(ctx context.Context, branch, tag string) error {
	return l.db.Update(func(tx *badger.Txn) error {
		if branch == "" {
			return store.IDIsRequired
		}
		if tag == "" {
			return store.NameIsRequired
		}

		pkey, err := valueBytesFor(tx, branchKey(branch))
		if err != nil {
			return err
		}

		return tx.Set(tagKey(tag), pkey)
	})
}

func (l *localBundleStore) DeleteTag(ctx context.Context, tag string) error {
	return l.db.Update(func(tx *badger.Txn) error {
		bk := tagKey(tag)
		_, err := tx.Get(bk)
		if err != nil {
			if err.Error() == badger.ErrKeyNotFound.Error() {
				return nil
			}
			return mapBundleError(err)
		}
		return mapBundleError(tx.Delete(bk))
	})
}

func (l *localBundleStore) ListBranches(ctx context.Context) ([]string, error) {
	return l.listKeys(branchKey(""))
}

func (l *localBundleStore) listKeys(pref []byte) ([]string, error) {
	var result []string
	verr := l.db.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true

		it := tx.NewIterator(opts)
		for it.Seek(pref); it.ValidForPrefix(pref); it.Next() {
			item := it.Item()
			k := store.UnsafeBytesToString(item.Key()[len(pref):])
			result = append(result, k)
		}
		it.Close()
		return nil
	})

	if verr != nil {
		return nil, verr
	}
	return result, nil
}

func (l *localBundleStore) CreateBranch(ctx context.Context, parent, name string) error {
	return l.db.Update(func(tx *badger.Txn) error {
		val := []byte("empty")
		if parent != "" {
			pkey, err := valueBytesFor(tx, branchKey(parent))
			if err != nil {
				return err
			}
			val = pkey
		}
		bk := branchKey(name)
		if _, err := tx.Get(bk); err == nil || err.Error() != badger.ErrKeyNotFound.Error() {
			if err == nil {
				return store.BranchAlreadyExists
			}
			return err
		}
		return tx.Set(bk, val)
	})
}

func (l *localBundleStore) HashForBranch(ctx context.Context, branch string) (string, error) {
	return l.hashFor(branchKey(branch))
}

func (l *localBundleStore) DeleteBranch(ctx context.Context, name string) error {
	return l.db.Update(func(tx *badger.Txn) error {
		bk := branchKey(name)
		_, err := tx.Get(bk)
		if err != nil && err.Error() != badger.ErrKeyNotFound.Error() {
			return mapBundleError(err)
		}
		return mapBundleError(tx.Delete(bk))
	})
}

func (l *localBundleStore) HashForPath(ctx context.Context, path string) (string, error) {
	return l.hashFor(pathKey(path))
}

func valueBytesFor(tx *badger.Txn, key []byte) ([]byte, error) {
	item, err := tx.Get(key)
	if err != nil {
		return nil, mapObjectError(err)
	}
	return item.Value()
}

func (l *localBundleStore) hashFor(key []byte) (string, error) {
	var result string
	berr := l.db.View(func(tx *badger.Txn) error {
		b, err := valueBytesFor(tx, key)
		if err != nil {
			return mapObjectError(err)
		}
		result = store.UnsafeBytesToString(b)
		return nil
	})

	if berr != nil {
		return "", berr
	}
	return result, nil
}

func (l *localBundleStore) findCommitsByPrefix(prefix string, keysOnly bool) ([]store.Bundle, error) {
	var result []store.Bundle
	verr := l.db.View(func(tx *badger.Txn) error {
		pref := commitKey(prefix)
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = !keysOnly

		it := tx.NewIterator(opts)

		for it.Seek(pref); it.ValidForPrefix(pref); it.Next() {
			item := it.Item()
			k := store.UnsafeBytesToString(item.Key())
			if keysOnly {
				result = append(result, store.Bundle{
					ID: k[7:],
				})
				continue
			}

			bundle, err := mapBundleItemError(item, nil)
			if err != nil {
				it.Close()
				return err
			}

			result = append(result, bundle)
		}
		it.Close()
		return nil
	})

	if verr != nil {
		return nil, verr
	}
	return result, nil
}
