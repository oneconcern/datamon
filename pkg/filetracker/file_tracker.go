package filetracker

import (
	"encoding/binary"
	"io"
	"sync"
	"unsafe"

	"github.com/spf13/afero"

	iradix "github.com/hashicorp/go-immutable-radix"

	"github.com/oneconcern/datamon/pkg/storage"
)

// Tracks writes that occur on top of a base file.
type TFile struct {
	io.ReaderAt
	io.WriterAt
	base    storage.Store
	file    *afero.File
	tracker *iradix.Tree
	lock    sync.Mutex
	name    string
}

func newTFile(baseStore storage.Store, file *afero.File, name string) *TFile {

	return &TFile{
		base:    baseStore,
		file:    file,
		name:    name,
		tracker: iradix.New(),
	}
}

func (t *TFile) ReadAt(p []byte, off int64) (n int, err error) {
	//TODO:
	_, _ = getFileRange(off, int64(len(p)))
	return 0, nil
}

func (t *TFile) WriteAt(p []byte, off int64) (n int, err error) {
	return 0, nil
}

func getFileRange(offset int64, len int64) (int64, int64) {
	start := offset
	end := offset + len
	return start, end
}

const (
	startFlag = true
	endFlag   = false
	terminate = true
	mutable   = true
	base      = !mutable
)

func getKey(key int64) []byte {
	if key < 0 {
		return nil
	}
	k := make([]byte, unsafe.Sizeof(int64(0)))
	binary.BigEndian.PutUint64(k, uint64(key))
	return k
}

func getOffset(k []byte) int64 {
	return int64(binary.BigEndian.Uint64(k))
}

// Deletes keys that are no longer required and inserts new keys to
// allow reads to be performed correctly.
func (t *TFile) trackWrite(offset int64, length int64) {

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
		key := getOffset(k)

		deleteKey := func() {
			if key <= end {
				txn.Delete(k)
			}
		}
		switch {
		case isStart && (key == start):
			insertStart = false
			return !terminate
		case isStart && (key < start):
			// Only interim keys need deleting
			return !terminate
		case isStart && (key > start):
			deleteKey()
			return !terminate
		case isEnd && (key < start):
			// Previous end hit and can be ignored, process next key
			return !terminate
		case isEnd && (key > start):
			// There is an end that is after start and no other key in the range.
			// Skip inserting start, previous start will cover the range.
			insertStart = false
			// This key might need deleting and process other keys
			if key >= end {
				insertEnd = false
				return terminate
			}
			deleteKey()
			return !terminate
		default:
			return !terminate
		}
	}

	// TODO: To reduce the walk use prefix but needs to be walked twice offset and offset + length
	t.tracker.Root().Walk(fn)
	if insertStart {
		txn.Insert(getKey(start), startFlag)
	}
	if insertEnd {
		txn.Insert(getKey(end), endFlag)
	}
	t.tracker = txn.Commit()
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// Given an offset and len , return the next contiguous read possible and the backend store for it.
func (t *TFile) getRangeToRead(offset int64, len int64) (int64, bool) {
	contiguous := len
	storage := base
	fn := func(k []byte, v interface{}) bool {

		isStart := v.(bool)
		isEnd := !isStart
		key := getOffset(k)

		switch {
		case isStart && (key <= offset):
			storage = mutable
			return !terminate
		case isStart && (key > offset):
			contiguous = min(key-offset, len)
			storage = base
			return terminate
		case isEnd && (key <= offset):
			storage = base
			return !terminate
		case isEnd && (key > offset):
			storage = mutable
			contiguous = min(key-offset, len)
			return terminate
		}
		return !terminate
	}
	// TODO: To reduce the walk use prefix but needs to be walked twice offset and offset + length
	t.tracker.Root().Walk(fn)
	return contiguous, storage
}
