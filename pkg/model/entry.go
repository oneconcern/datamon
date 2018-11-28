package model

import (
	"encoding/hex"
	"time"

	units "github.com/docker/go-units"
	blake2b "github.com/minio/blake2b-simd"
)

// Entry for the stage or a bundle
type Entry struct {
	Path  string    `json:"path" yaml:"path"`
	Hash  string    `json:"hash" yaml:"hash"`
	Mtime time.Time `json:"mtime" yaml:"mtime"`
	Mode  FileMode  `json:"mode" yaml:"mode"`
	_     struct{}
}

// Entries represent a collectin of entries
type Entries []Entry

// Hash the entry hashes into a single hash
func (entries Entries) Hash() (string, error) {
	// Compute hash of level 1 root key
	hasher, err := blake2b.New(&blake2b.Config{
		Size: 64,
		Tree: &blake2b.Tree{
			Fanout:        0,
			MaxDepth:      2,
			LeafSize:      5 * units.MiB,
			NodeOffset:    0,
			NodeDepth:     1,
			InnerHashSize: 64,
			IsLastNode:    true,
		},
	})
	if err != nil {
		return "", err
	}

	// Iterate over hashes of all underlying nodes
	for _, leave := range entries {
		//#nosec
		_, _ = hasher.Write(UnsafeStringToBytes(leave.Hash))
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// Without returns a new collection of entries without the ones specified
func (entries Entries) Without(toRemove []Entry) Entries {
	var result []Entry

ENTRIES:
	for _, entry := range entries {
		for _, del := range toRemove {
			if entry.Path == del.Path {
				continue ENTRIES
			}
		}
		result = append(result, entry)
	}

	return result
}

// Without returns a new collection of entries merged with the ones specified
func (entries Entries) With(toAdd []Entry) (current Entries) {
	target := make(map[string]Entries, len(entries))
	var order []string
	for _, e := range append(entries, toAdd...) {
		_, known := target[e.Path]
		if !known {
			order = append(order, e.Path)
		}
		target[e.Path] = append(target[e.Path], e)
	}
	for _, pth := range order {
		ke := target[pth]
		// the last item in the list is the the most recent item, so the actual one
		current = append(current, ke[len(ke)-1])
	}
	return
}

// FlattenToRecent flattens a collection of entries to unique path names
// and retains the most recent entry
func (entries Entries) FlattenToRecent() (current Entries) {
	target := make(map[string]Entries, len(entries))
	var order []string
	for _, e := range entries {
		_, known := target[e.Path]
		if !known {
			order = append(order, e.Path)
		}
		target[e.Path] = append(target[e.Path], e)
	}
	for _, pth := range order {
		ke := target[pth]
		// the last item in the list is the the most recent item, so the actual one
		current = append(current, ke[len(ke)-1])
	}
	return
}
