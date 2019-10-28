package storage

import (
	"bytes"
	"context"
	"hash/crc32"
	"io/ioutil"
	"sync"
)

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

type MultiStoreUnit struct {
	Store           Store
	TolerateFailure bool
}

func MultiPut(ctx context.Context, stores []MultiStoreUnit, name string, buffer []byte, doesNotExist NewKey) error {
	errC := make(chan error, len(stores))
	var wg sync.WaitGroup

	for _, w := range stores {
		wg.Add(1)
		go func(w MultiStoreUnit, buffer []byte) {
			defer wg.Done()
			crcStore, ok := w.Store.(StoreCRC)
			var err error
			if ok {
				crc := crc32.Checksum(buffer, crc32.MakeTable(crc32.Castagnoli))
				err = crcStore.PutCRC(context.TODO(), name, bytes.NewReader(buffer), doesNotExist, crc)
			} else {
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
