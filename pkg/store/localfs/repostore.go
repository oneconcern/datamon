package localfs

import (
	"context"
	"encoding/json"
	"log"
	"path/filepath"
	"sync"

	"github.com/json-iterator/go"

	"github.com/dgraph-io/badger"
	"github.com/oneconcern/trumpet/pkg/store"
)

// NewRepos creates a new repo store instance
func NewRepos(baseDir string) store.RepoStore {
	if baseDir == "" {
		baseDir = ".trumpet"
	}
	return &repoStore{
		baseDir: baseDir,
	}
}

type repoStore struct {
	baseDir string
	db      *badger.DB
	init    sync.Once
	close   sync.Once
}

func (r *repoStore) Initialize() error {
	var err error
	r.init.Do(func() {
		var db *badger.DB
		db, err = makeBadgerDb(filepath.Join(r.baseDir, repoDb))
		if err != nil {
			return
		}
		r.db = db
	})

	return err
}

func (r *repoStore) Close() error {
	var err error
	r.close.Do(func() {
		if r.db != nil {
			err = r.db.Close()
			if err == nil {
				r.db = nil
			}
		}
	})
	return err
}

func (r *repoStore) List(ctx context.Context) ([]string, error) {
	res, err := r.findByPrefix("", true)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(res))
	for i := range res {
		result[i] = res[i].Key
	}
	return result, nil
}

func (r *repoStore) Get(ctx context.Context, name string) (*store.Repo, error) {
	keyb := store.UnsafeStringToBytes(name)
	var value store.Repo
	verr := r.db.View(func(txn *badger.Txn) error {
		item, err := mapRepoItemError(txn.Get(keyb))
		if err != nil {
			return err
		}
		value = item
		return nil
	})
	return &value, verr
}

func (r *repoStore) Create(ctx context.Context, repo *store.Repo) error {
	return r.put(repo, true)
}

func (r *repoStore) Update(ctx context.Context, repo *store.Repo) error {
	return r.put(repo, false)
}

func (r *repoStore) Delete(ctx context.Context, name string) error {
	return r.db.Update(func(txn *badger.Txn) error {
		return mapRepoError(txn.Delete(store.UnsafeStringToBytes(name)))
	})
}

type keyValue struct {
	Key   string
	Value json.RawMessage
}

func (r *repoStore) findByPrefix(prefix string, keysOnly bool) ([]keyValue, error) {
	var result []keyValue
	verr := r.db.View(func(tx *badger.Txn) error {
		pref := store.UnsafeStringToBytes(prefix)
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = !keysOnly

		it := tx.NewIterator(opts)

		for it.Seek(pref); it.ValidForPrefix(pref); it.Next() {
			item := it.Item()
			k := store.UnsafeBytesToString(item.Key())
			if keysOnly {
				result = append(result, keyValue{Key: k})
				continue
			}

			v, err := item.Value()
			if err != nil {
				it.Close()
				return mapRepoError(err)
			}

			result = append(result, keyValue{Key: k, Value: v})
		}
		it.Close()
		return nil
	})

	if verr != nil {
		return nil, verr
	}
	return result, nil
}

func (r *repoStore) put(repo *store.Repo, create bool) error {
	log.Println("put", repo.Name, "create:", create)
	// need this to be 0 when this is a new entry
	keyb := store.UnsafeStringToBytes(repo.Name)
	return r.db.Update(func(txn *badger.Txn) error {
		_, err := mapRepoItemError(txn.Get(keyb))
		if err != store.RepoNotFound {
			if err == nil && create {
				return store.RepoAlreadyExists
			}
			return err
		}
		if err == store.RepoNotFound && !create {
			return store.RepoNotFound
		}

		data, err := jsoniter.Marshal(repo)
		if err != nil {
			return err
		}

		return txn.Set(keyb, data)
	})
}
