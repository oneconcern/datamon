package localfs

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/dgraph-io/badger"
	jsoniter "github.com/json-iterator/go"
	"github.com/oneconcern/trumpet/pkg/store"
)

var dbs sync.Map

func makeBadgerDb(dir string) (*badger.DB, error) {
	if v, ok := dbs.Load(dir); ok {
		return v.(*badger.DB), nil
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		log.Println("mkdir -p", dir, err)
	}
	bopts := badger.DefaultOptions
	bopts.Dir = dir
	bopts.ValueDir = dir

	v, err := badger.Open(bopts)
	if err != nil {
		return nil, err
	}
	dbs.Store(dir, v)
	return v, nil
}

func mapRepoError(err error) error {
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

func mapRepoItemError(value *badger.Item, err error) (store.Repo, error) {
	if err != nil {
		return store.Repo{}, mapRepoError(err)
	}
	data, err := value.Value()
	if err != nil {
		return store.Repo{}, mapRepoError(err)
	}

	var result store.Repo
	if e := jsoniter.Unmarshal(data, &result); e != nil {
		return store.Repo{}, fmt.Errorf("json unmarshal failed: %v", e)
	}
	return result, nil
}

func mapBundleError(err error) error {
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

func mapBundleItemError(value *badger.Item, err error) (store.Bundle, error) {
	if err != nil {
		return store.Bundle{}, mapObjectError(err)
	}

	data, err := value.Value()
	if err != nil {
		return store.Bundle{}, mapObjectError(err)
	}

	var result store.Bundle
	if e := jsoniter.Unmarshal(data, &result); e != nil {
		return store.Bundle{}, fmt.Errorf("json unmarshal failed: %v", e)
	}
	return result, nil
}

func mapObjectError(err error) error {
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

func mapEntryError(value *badger.Item, err error) (store.Entry, error) {
	if err != nil {
		return store.Entry{}, mapObjectError(err)
	}

	data, err := value.Value()
	if err != nil {
		return store.Entry{}, mapObjectError(err)
	}

	var result store.Entry
	if e := jsoniter.Unmarshal(data, &result); e != nil {
		return store.Entry{}, fmt.Errorf("json unmarshal failed: %v", e)
	}
	return result, nil
}

func mapSnapshotError(err error) error {
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

func mapSnapshotItemError(value *badger.Item, err error) (store.Snapshot, error) {
	if err != nil {
		return store.Snapshot{}, mapSnapshotError(err)
	}

	data, err := value.Value()
	if err != nil {
		return store.Snapshot{}, mapSnapshotError(err)
	}

	var result store.Snapshot
	if e := jsoniter.Unmarshal(data, &result); e != nil {
		return store.Snapshot{}, fmt.Errorf("json unmarshal failed: %v", e)
	}
	return result, nil
}
