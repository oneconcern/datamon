package filetracker

import (
	"encoding/binary"
	"io"
	"sync"
	"unsafe"

	"github.com/hashicorp/go-immutable-radix"

	"github.com/oneconcern/datamon/pkg/storage"
)

// Tracks writes that occur on top of a base file.
type tFile struct {
	io.ReaderAt
	io.WriterAt
	base    storage.Store
	mutable storage.Store
	file    string
	tracker *iradix.Tree
	lock    sync.Mutex
}

func newTFile(baseStore storage.Store, mutableStore storage.Store, file string) *tFile {

	return &tFile{
		base:    baseStore,
		mutable: mutableStore,
		file:    file,
		tracker: iradix.New(),
	}
}

func (t *tFile) ReadAt(p []byte, off uint64) (n int, err error) {
	return 0, nil
}

func (t *tFile) WriteAt(p []byte, off uint64) (n int, err error) {
	return 0, nil
}

func getFileRange(offset uint64, len uint64) (uint64, uint64) {
	start := offset
	end := offset + len
	return start, end
}

const (
	startFlag = true
	endFlag   = false
	terminate = true
)

func getKey(key uint64) []byte {
	k := make([]byte, unsafe.Sizeof(uint64(0)))
	binary.BigEndian.PutUint64(k, key)
	return k
}

// Deletes keys that are no longer required and inserts new keys to
// allow reads to be performed correctly.
func (t *tFile) trackWrite(offset uint64, length uint64) {

	start, end := getFileRange(offset, length)

	// Lock to protect radix tree, reads can continue.
	t.lock.Lock()
	defer t.lock.Unlock()

	txn := t.tracker.Txn()
	insertStart := true
	insertEnd := true

	if t.tracker.Len() == 0 {

		txn.Insert(getKey(start), startFlag)
		txn.Insert(getKey(end), endFlag)
		t.tracker = txn.Commit()

		return
	}

	fn := func(k []byte, v interface{}) bool {
		isStart := v.(bool)
		isEnd := !isStart
		key := binary.BigEndian.Uint64(k)

		deleteKey := func() {
			if key <= end {
				txn.Delete(k)
			}
		}
		if isStart && (key == start) {
			insertStart = false
			return !terminate
		} else if isStart && (key < start) {
			// Only interim keys need deleting
			return !terminate
		} else if isStart && (key > start) {
			deleteKey()
			return !terminate
		} else if isEnd && (key == start) {
			//Contiguous region grew
			deleteKey()
			return !terminate
		} else if isEnd && (key < start) {
			// Previous end hit and can be ignored, process next key
			return !terminate
		} else if isEnd && (key > start) {
			// There is an end that is after start and no other key in the range.
			// Skip inserting start, previous start will cover the range.
			insertStart = false
			// This key might need deleting and process other other keys
			if key >= end {
				insertEnd = false
				return terminate
			} else {
				deleteKey()
			}
			return !terminate
		}
		return !terminate
	}

	t.tracker.Root().Walk(fn)
	if insertStart {
		txn.Insert(getKey(start), startFlag)
	}
	if insertEnd {
		txn.Insert(getKey(end), endFlag)
	}
	t.tracker = txn.Commit()
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// Given an offset and len , return the next contiguous read possible and the backend store for it.
func (t *tFile) getRangeToRead(offset uint64, len uint64) (uint64, storage.Store) {
	contiguous := len
	storage := t.base
	terminate := true
	fn := func(k []byte, v interface{}) bool {

		isStart := v.(bool)
		isEnd := !isStart
		key := binary.BigEndian.Uint64(k)

		if isStart && (key <= offset) {
			storage = t.mutable
			return !terminate
		} else if isStart && (key > offset) {
			contiguous = min(key-offset, len)
			storage = t.base
			return terminate
		} else if isEnd && (key <= offset) {
			storage = t.base
			return !terminate
		} else if isEnd && (key > offset) {
			storage = t.mutable
			contiguous = min(key-offset, len)
			return terminate
		}
		return !terminate
	}
	t.tracker.Root().Walk(fn)
	return contiguous, storage
}
