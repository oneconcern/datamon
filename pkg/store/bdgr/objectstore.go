package bdgr

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/json-iterator/go"

	"github.com/dgraph-io/badger"
	"github.com/oneconcern/trumpet/pkg/store"
)

const (
	blobsDb = "blobs"
)

var (
	hashPref = [5]byte{'h', 'a', 's', 'h', ':'}
	pathPref = [5]byte{'p', 'a', 't', 'h', ':'}
)

func badgerRewriteObjectError(err error) error {
	switch err {
	case badger.ErrKeyNotFound:
		return store.ObjectNotFound
	case badger.ErrEmptyKey:
		return store.NameIsRequired
	default:
		return err
	}
}

func badgerRewriteEntryError(value *badger.Item, err error) (store.Entry, error) {
	if err != nil {
		return store.Entry{}, badgerRewriteObjectError(err)
	}

	data, err := value.Value()
	if err != nil {
		return store.Entry{}, badgerRewriteObjectError(err)
	}

	var result store.Entry
	if e := jsoniter.Unmarshal(data, &result); e != nil {
		return store.Entry{}, fmt.Errorf("json unmarshal failed: %v", e)
	}
	return result, nil
}

// NewObjectMeta creates a badger based object metadata store
func NewObjectMeta(baseDir string) store.ObjectMeta {
	ms := &objectMetaStore{
		baseDir: baseDir,
	}
	return ms
}

type objectMetaStore struct {
	baseDir string
	db      *badger.DB
	init    sync.Once
	close   sync.Once
}

func (o *objectMetaStore) Initialize() error {
	var err error

	o.init.Do(func() {
		var db *badger.DB
		db, err = makeBadgerDb(filepath.Join(o.baseDir, blobsDb))
		if err != nil {
			return
		}
		o.db = db
	})

	return err
}

func (o *objectMetaStore) Close() error {
	var err error

	o.close.Do(func() {
		if o.db != nil {
			err = o.db.Close()
			if err == nil {
				o.db = nil
			}
		}
	})

	return err
}

func (o *objectMetaStore) hashKey(key string) []byte {
	return append(hashPref[:], store.UnsafeStringToBytes(key)...)
}

func (o *objectMetaStore) pathKey(key string) []byte {
	return append(pathPref[:], store.UnsafeStringToBytes(key)...)
}

func (o *objectMetaStore) Add(entry store.Entry) error {
	return o.db.Update(func(txn *badger.Txn) error {
		hv := store.UnsafeStringToBytes(entry.Hash)
		hk := append(hashPref[:], hv...)
		_, err := badgerRewriteEntryError(txn.Get(hk))
		if err != store.ObjectNotFound {
			return err
		}
		data, err := jsoniter.Marshal(entry)
		if err != nil {
			return err
		}

		if err := txn.Set(o.pathKey(entry.Path), hv); err != nil {
			return err
		}
		return txn.Set(hk, data)
	})
}

func (o *objectMetaStore) Remove(key string) error {
	return o.db.Update(func(tx *badger.Txn) error {
		hk := o.hashKey(key)
		entry, err := badgerRewriteEntryError(tx.Get(hk))
		if err != nil {
			if err == store.ObjectNotFound {
				return nil
			}
			return err
		}
		if err := badgerRewriteObjectError(tx.Delete(hk)); err != nil {
			if err == store.ObjectNotFound {
				err2 := badgerRewriteObjectError(tx.Delete(o.pathKey(entry.Path)))
				if err2 == store.ObjectNotFound {
					return nil
				}
				return err2
			}
			return err
		}
		return nil
	})
}

func (o *objectMetaStore) Get(key string) (store.Entry, error) {
	var entry store.Entry
	berr := o.db.View(func(tx *badger.Txn) error {
		item, err := badgerRewriteEntryError(tx.Get(o.hashKey(key)))
		if err != nil {
			return err
		}
		entry = item
		return nil
	})

	if berr != nil {
		return store.Entry{}, berr
	}
	return entry, nil
}

func (o *objectMetaStore) Clear() error {
	berr := o.db.Update(func(tx *badger.Txn) error {
		opts := badger.IteratorOptions{
			PrefetchValues: false,
			PrefetchSize:   1000000,
			Reverse:        false,
			AllVersions:    false,
		}
		iter := tx.NewIterator(opts)
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			if err := tx.Delete(item.Key()); err != nil {
				return err
			}
		}
		return nil
	})
	return berr
}

func (o *objectMetaStore) HashFor(path string) (string, error) {
	var result string
	berr := o.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(o.pathKey(path))
		if err != nil {
			return badgerRewriteObjectError(err)
		}
		b, err := item.Value()
		if err != nil {
			return badgerRewriteObjectError(err)
		}
		result = store.UnsafeBytesToString(b)
		return nil
	})

	if berr != nil {
		return "", berr
	}
	return result, nil
}
