package cafs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"

	blake2b "github.com/minio/blake2b-simd"
	"github.com/oneconcern/datamon/pkg/storage"
)

// TODO(fred): nice
// * introduce pather func rather than prefix
// * externalize as hasher interface

// IsRootKey determines if a given key in this store is a root key
func IsRootKey(fs storage.Store, key Key, leafSize uint32) bool {
	keys, err := leavesForHash(fs, key, leafSize, "")
	if err != nil {
		return false
	}
	return len(keys) > 0
}

// LeavesForHash returns the keys from the blob referred to by a root hash, with checksum validation
func LeavesForHash(blobs storage.Store, root Key, leafSize uint32, prefix string) ([]Key, error) {
	return leavesForHash(blobs, root, leafSize, prefix)
}

// LeafKeys returns the child keys contained in a data bufer, which we assume come from a root entry blob.
//
// Input data is expected to be the concatenation of at least one 64 bytes keys, followed by the root key itself.
func LeafKeys(root Key, data []byte, leafSize uint32) ([]Key, error) {
	verify, err := verificationKey(data, leafSize)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(root[:], verify[:]) {
		return nil, errors.New("the last hash in the file is not the root key")
	}
	return verifiedKeys(data, leafSize)
}

// UnverifiedLeafKeys is the same as LeafKeys, but assumes the []byte buffer is valid and shunts verifications
func UnverifiedLeafKeys(data []byte, leafSize uint32) []Key {
	ks, err := leaves(data, leafSize)
	if err != nil {
		panic(err)
	}
	return ks
}

// RootHash computes the hash of a level 1 root key, given a set of lower level keys
func RootHash(leaves []Key, leafSize uint32) (Key, error) {
	return rootHash(leaves, leafSize)
}

// KeyFromBytes computes the n-th key based on a data buffer. This is used for integrity checksums on unitary leaves
func KeyFromBytes(data []byte, leafSize uint32, n uint64, isLastNode bool) (Key, error) {
	return keyFromBytes(data, leafSize, n, isLastNode)
}

// rootHash compute the root hash from an ordered sequence of leaf keys
func rootHash(leaves []Key, leafSize uint32) (Key, error) {
	// Compute hash of level 1 root key
	hasher, err := blake2b.New(&blake2b.Config{
		Size: blake2b.Size,
		Tree: &blake2b.Tree{
			Fanout:        0,
			MaxDepth:      2,
			LeafSize:      leafSize,
			NodeOffset:    0,
			NodeDepth:     1,
			InnerHashSize: blake2b.Size,
			IsLastNode:    true,
		},
	})
	if err != nil {
		// New only fails when configuration is wrong
		return Key{}, err
	}

	// Iterate over hashes of all underlying nodes
	for _, leave := range leaves {
		_, err = hasher.Write(leave[:])
		if err != nil {
			// hasher is actually always successful and is using padded keys
			return Key{}, err
		}
	}

	k, err := NewKey(hasher.Sum(nil))
	if err != nil {
		// hasher actually always returns the expected size
		return Key{}, err
	}
	return k, nil
}

// leavesForHash reads the blob referred to by a root hash key and extracts the leaf keys
func leavesForHash(blobs storage.Store, hash Key, leafSize uint32, prefix string) ([]Key, error) {
	b, err := bytesFromRoot(blobs, hash, prefix)
	if err != nil {
		return nil, err
	}
	return verifiedKeys(b, leafSize)
}

// bytesFromRoot reads the blob referred to by a root hash key
func bytesFromRoot(blobs storage.Store, hash Key, prefix string) ([]byte, error) {
	rdr, err := blobs.Get(context.Background(), hash.StringWithPrefix(prefix))
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(rdr)
	if err != nil {
		rdr.Close()
		return nil, err
	}

	if err = rdr.Close(); err != nil {
		return nil, err
	}
	return b, nil
}

// leaves extracts all fixed-size keys concatenated in a buffer
func leaves(data []byte, leafSize uint32) ([]Key, error) {
	keys := make([]Key, 0, len(data)/KeySize)
	for i := 0; i < len(data); i += KeySize {
		if i+KeySize > len(data) {
			return nil, &BadKeySize{Key: data[i:]}
		}
		keys = append(keys, MustNewKey(data[i:i+KeySize]))
	}
	return keys, nil
}

// verifiedKeys verifies that a buffer contains a sequence of leaf keys followed by the verification root key,
// then returns the leaf keys
func verifiedKeys(data []byte, leafSize uint32) ([]Key, error) {
	verify, err := verificationKey(data, leafSize)
	if err != nil {
		return nil, err
	}

	keys, err := leaves(data[:len(data)-KeySize], leafSize)
	if err != nil {
		return nil, err
	}

	checksum, err := rootHash(keys, leafSize)
	if err != nil {
		// rootHash actually never fails
		return nil, err
	}
	if verify != checksum {
		return nil, fmt.Errorf("leaves (count: %d) checksum doesn't match hash value. Verification hash: %s, computed checksum: %s", len(keys), verify, checksum)
	}
	return keys, nil
}

// verificationKey verifies the buffer size and extracts the verification key trailing the buffer (i.e the root key)
func verificationKey(data []byte, leafSize uint32) (Key, error) {
	if len(data) < KeySize {
		return Key{}, errors.New("provided data is too short to contain a key")
	}
	// the verification key is the last one
	return MustNewKey(data[len(data)-KeySize:]), nil
}

// keyFromBytes compute the n-th leaf key from a raw data buffer
func keyFromBytes(data []byte, leafSize uint32, n uint64, isLastNode bool) (Key, error) {
	// Calculate hash value
	hasher, err := blake2b.New(&blake2b.Config{
		Size: blake2b.Size,
		Tree: &blake2b.Tree{
			Fanout:        0,
			MaxDepth:      2,
			LeafSize:      leafSize,
			NodeOffset:    n,
			NodeDepth:     0,
			InnerHashSize: blake2b.Size,
			IsLastNode:    isLastNode,
		},
	})
	if err != nil {
		// New only fails when configuration is wrong
		return Key{}, err
	}
	_, err = hasher.Write(data)
	if err != nil {
		// hasher is actually always successful and is using padded keys
		return Key{}, fmt.Errorf("cannot compute data segment hash: %v", err)
	}

	leafKey, err := NewKey(hasher.Sum(nil))
	if err != nil {
		// hasher actually always returns the expected size
		return Key{}, fmt.Errorf("cannot compute key: %v", err)
	}
	return leafKey, nil
}
