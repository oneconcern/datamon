// Copyright © 2018 One Concern

package storage

import (
	"bytes"
	"context"
	"hash/crc32"
	"io/ioutil"
	"sync"
)

// ReadTee reads from a source and duplicates the output to another destination store
func ReadTee(ctx context.Context, sStore Store, source string, dStore Store, destination string) ([]byte, error) {
	reader, err := sStore.Get(ctx, source)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	object, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	err = dStore.Put(ctx, destination, bytes.NewReader(object), NoOverWrite)
	if err != nil {
		return nil, err
	}
	return object, err
}

// MultiStoreUnit is used to specify multiple operations, some of which are tolerated to fail
type MultiStoreUnit struct {
	// Store is the backend to be accessed
	Store Store

	// TolerateFailure to false breaks multi-store operations whenever an error is encountered.
	TolerateFailure bool
}

// MultiPut duplicates write operations to an array of stores, under the same name
func MultiPut(ctx context.Context, stores []MultiStoreUnit, name string, buffer []byte, doesNotExist bool) error {
	errC := make(chan error, len(stores))
	var wg sync.WaitGroup

	for _, w := range stores {
		wg.Add(1)
		go func(w MultiStoreUnit, buffer []byte) {
			defer wg.Done()

			var err error
			switch crcStore := w.Store.(type) {
			case StoreCRC:
				crc := crc32.Checksum(buffer, crc32.MakeTable(crc32.Castagnoli))
				err = crcStore.PutCRC(context.TODO(), name, bytes.NewReader(buffer), doesNotExist, crc)
			default:
				err = w.Store.Put(ctx, name, bytes.NewReader(buffer), doesNotExist)
			}
			if w.TolerateFailure {
				return
			}
			if err != nil {
				errC <- err
			}
		}(w, buffer)
	}
	wg.Wait()
	select {
	case err := <-errC:
		return err
	default:
		return nil
	}
}
