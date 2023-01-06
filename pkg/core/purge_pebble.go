package core

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/cockroachdb/pebble"
)

type (
	// kvPebble provides a KV store implementation based on cockroachdb/pebble
	kvPebble struct {
		*pebble.DB
	}

	kvPebbleIterator struct {
		isFirst  bool
		iterator *pebble.Iterator
	}

	ignoreNewMerger struct {
		buf []byte
	}
)

func (kv *kvPebble) Drop() error {
	iterator := kv.DB.NewIter(nil)
	defer func() {
		_ = iterator.Close()
	}()

	start, end := iterator.RangeBounds()
	if pebble.DefaultComparer.Compare(start, end) >= 0 {
		return nil
	}

	if err := kv.DB.DeleteRange(start, end, &pebble.WriteOptions{Sync: false}); err != nil {
		return err
	}

	// as DeleteRange excludes the upper bound
	if err := kv.DB.Delete(end, &pebble.WriteOptions{Sync: false}); err != nil && !errors.Is(err, pebble.ErrNotFound) {
		return err
	}

	return nil
}

func (kv *kvPebble) Size() uint64 {
	m := kv.DB.Metrics()

	return m.DiskSpaceUsage()
}

func (kv *kvPebble) AllKeys() kvIterator {
	return &kvPebbleIterator{
		isFirst:  true,
		iterator: kv.DB.NewIter(nil),
	}
}

func (kv *kvPebble) Get(key []byte) ([]byte, error) {
	val, closer, err := kv.DB.Get(key)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = closer.Close()
	}()

	dest := make([]byte, len(val))
	copy(dest, val)

	return dest, nil
}

func (kv *kvPebble) Exists(key []byte) (bool, error) {
	_, closer, err := kv.DB.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return false, nil
		}

		return false, err
	}

	_ = closer.Close()

	return true, nil
}

func (kv *kvPebble) Set(key, value []byte) error {
	return kv.DB.Set(key, value, &pebble.WriteOptions{Sync: false})
}

func (kv *kvPebble) SetIfNotExists(key, value []byte) error {
	found, err := kv.Exists(key)
	if err != nil {
		return err
	}

	if found {
		return nil
	}

	return kv.Merge(key, value, &pebble.WriteOptions{Sync: false}) // skip new value
}

func (kv *kvPebble) Compact() error {
	iterator := kv.DB.NewIter(nil)
	start, end := iterator.RangeBounds()
	defer func() {
		_ = iterator.Close()
	}()

	if pebble.DefaultComparer.Compare(start, end) >= 0 {
		return nil
	}

	return kv.DB.Compact(start, end, true)
}

func (i *kvPebbleIterator) Next() bool {
	if i.isFirst {
		_ = i.iterator.First()
		i.isFirst = false

		return i.iterator.Valid()
	}

	return i.iterator.Next()
}

func (i *kvPebbleIterator) Item() ([]byte, []byte, error) {
	k, v := i.iterator.Key(), i.iterator.Value()

	key := make([]byte, len(k))
	copy(key, k)
	value := make([]byte, len(v))
	copy(value, v)

	return key, value, nil
}

func (i *kvPebbleIterator) Close() error {
	return i.iterator.Close()
}

func makeKVPebble(pth string, _ *purgeOptions) (*kvPebble, error) {
	err := os.MkdirAll(pth, 0700)
	if err != nil {
		return nil, fmt.Errorf("makeKV: mkdir: %w", err)
	}

	options := new(pebble.Options)
	options.EnsureDefaults()
	options.DisableWAL = true
	options.Merger = &pebble.Merger{
		Name: "ignore new",
		Merge: func(_, value []byte) (pebble.ValueMerger, error) {
			return &ignoreNewMerger{
				buf: value,
			}, nil
		},
	}

	db, err := pebble.Open(pth, options)
	if err != nil {
		return nil, fmt.Errorf("open KV: %w", err)
	}

	//  scratch any pre-existing local index
	pb := &kvPebble{DB: db}
	if err = pb.Drop(); err != nil {
		return nil, fmt.Errorf("scrach KV: %w", err)
	}

	return pb, nil
}

func (m *ignoreNewMerger) MergeNewer(val []byte) error {
	if m.buf == nil {
		m.buf = val
	}

	return nil
}

func (m *ignoreNewMerger) MergeOlder(val []byte) error {
	if val != nil {
		m.buf = val
	}

	return nil
}

func (m *ignoreNewMerger) Finish(bool) ([]byte, io.Closer, error) {
	return m.buf, nil, nil
}
