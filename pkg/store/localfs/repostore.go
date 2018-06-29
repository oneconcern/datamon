package localfs

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/json-iterator/go"

	"github.com/dgraph-io/badger"
	"github.com/oneconcern/trumpet/pkg/store"
)

const (
	repoDb   = "repos"
	modelsDb = "models"
	runsDb   = "runs"
)

func makeBadgerDb(dir string) (*badger.DB, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		log.Println("mkdir -p", dir, err)
	}
	bopts := badger.DefaultOptions
	bopts.Dir = dir
	bopts.ValueDir = dir

	return badger.Open(bopts)
}

func badgerRewriteRepoError(err error) error {
	switch err {
	case badger.ErrKeyNotFound:
		return store.RepoNotFound
	case badger.ErrEmptyKey:
		return store.NameIsRequired
	default:
		return err
	}
}

func badgerRewriteRepoItemError(value *badger.Item, err error) (store.Repo, error) {
	if err != nil {
		return store.Repo{}, badgerRewriteRepoError(err)
	}
	data, err := value.Value()
	if err != nil {
		return store.Repo{}, badgerRewriteRepoError(err)
	}

	var result store.Repo
	if e := jsoniter.Unmarshal(data, &result); e != nil {
		return store.Repo{}, fmt.Errorf("json unmarshal failed: %v", e)
	}
	return result, nil
}

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

func (r *repoStore) List() ([]string, error) {
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

func (r *repoStore) Get(name string) (*store.Repo, error) {
	keyb := store.UnsafeStringToBytes(name)
	var value store.Repo
	verr := r.db.View(func(txn *badger.Txn) error {
		item, err := badgerRewriteRepoItemError(txn.Get(keyb))
		if err != nil {
			return err
		}
		value = item
		return nil
	})
	return &value, verr
}

func (r *repoStore) Create(repo *store.Repo) error {
	return r.put(repo, true)
}

func (r *repoStore) Update(repo *store.Repo) error {
	return r.put(repo, false)
}

func (r *repoStore) Delete(name string) error {
	return r.db.Update(func(txn *badger.Txn) error {
		return badgerRewriteRepoError(txn.Delete(store.UnsafeStringToBytes(name)))
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
				return badgerRewriteRepoError(err)
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
	// need this to be 0 when this is a new entry
	keyb := store.UnsafeStringToBytes(repo.Name)
	return r.db.Update(func(txn *badger.Txn) error {
		_, err := badgerRewriteRepoItemError(txn.Get(keyb))
		if err != store.RepoNotFound {
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
