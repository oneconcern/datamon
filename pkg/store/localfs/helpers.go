package localfs

import (
	"fmt"
	"log"
	"os"

	"github.com/dgraph-io/badger"
	jsoniter "github.com/json-iterator/go"
	"github.com/oneconcern/trumpet/pkg/store"
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
	if err == nil {
		return nil
	}
	switch err.Error() {
	case badger.ErrKeyNotFound.Error():
		return store.RepoNotFound
	case badger.ErrEmptyKey.Error():
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

func badgerRewriteBundleError(err error) error {
	if err == nil {
		return nil
	}
	switch err.Error() {
	case badger.ErrKeyNotFound.Error():
		return store.BundleNotFound
	case badger.ErrEmptyKey.Error():
		return store.NameIsRequired
	default:
		return err
	}
}

func badgerRewriteBundleItemError(value *badger.Item, err error) (store.Bundle, error) {
	if err != nil {
		return store.Bundle{}, badgerRewriteObjectError(err)
	}

	data, err := value.Value()
	if err != nil {
		return store.Bundle{}, badgerRewriteObjectError(err)
	}

	var result store.Bundle
	if e := jsoniter.Unmarshal(data, &result); e != nil {
		return store.Bundle{}, fmt.Errorf("json unmarshal failed: %v", e)
	}
	return result, nil
}

func badgerRewriteObjectError(err error) error {
	if err == nil {
		return nil
	}
	switch err.Error() {
	case badger.ErrKeyNotFound.Error():
		return store.ObjectNotFound
	case badger.ErrEmptyKey.Error():
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

func badgerRewriteSnapshotError(err error) error {
	if err == nil {
		return nil
	}
	switch err.Error() {
	case badger.ErrKeyNotFound.Error():
		return store.SnapshotNotFound
	case badger.ErrEmptyKey.Error():
		return store.IDIsRequired
	default:
		return err
	}
}

func badgerRewriteSnapshotItemError(value *badger.Item, err error) (store.Snapshot, error) {
	if err != nil {
		return store.Snapshot{}, badgerRewriteSnapshotError(err)
	}

	data, err := value.Value()
	if err != nil {
		return store.Snapshot{}, badgerRewriteSnapshotError(err)
	}

	var result store.Snapshot
	if e := jsoniter.Unmarshal(data, &result); e != nil {
		return store.Snapshot{}, fmt.Errorf("json unmarshal failed: %v", e)
	}
	return result, nil
}
