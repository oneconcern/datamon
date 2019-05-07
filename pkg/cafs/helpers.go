package cafs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	blake2b "github.com/minio/blake2b-simd"
	"github.com/oneconcern/datamon/pkg/storage"
)

func CopyPaddedJSON(w io.Writer, buf *bytes.Buffer) {
	paddingLength := KeySize - (buf.Len() % KeySize)

	fmt.Fprint(w, string(buf.Bytes()[:buf.Len()-3]))
	fmt.Fprint(w, strings.Repeat("0", paddingLength))
	fmt.Fprint(w, string(buf.Bytes()[buf.Len()-3:]))
}

func RootHash(leaves []Key, leafSize uint32) (Key, error) {
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
		return Key{}, err
	}

	// Iterate over hashes of all underlying nodes
	for _, leave := range leaves {
		_, err = hasher.Write(leave[:])
		if err != nil {
			return Key{}, err
		}
	}

	k, err := NewKey(hasher.Sum(nil))
	if err != nil {
		return Key{}, err
	}
	return k, nil
}

func LeafsForHash(blobs storage.Store, hash Key, leafSize uint32, prefix string) ([]Key, error) {
	b, err := leafBytesForHash(blobs , hash , leafSize , prefix )
	if err != nil {
		return nil, err
	}

	return LeafKeys(hash, b, leafSize)
}

func leafsForHashInternVerify(blobs storage.Store, hash Key, leafSize uint32, prefix string) ([]Key, error) {
	b, err := leafBytesForHash(blobs , hash , leafSize , prefix )
	if err != nil {
		return nil, err
	}

	return leafKeysInternVerify(b, leafSize)
}

func leafBytesForHash(blobs storage.Store, hash Key, leafSize uint32, prefix string) ([]byte, error) {
	rdr, err := blobs.Get(context.Background(), hash.StringWithPrefix(prefix))
	if err != nil {
		return nil, err
	}
	defer rdr.Close()

	b, err := ioutil.ReadAll(rdr)
	if err != nil {
		return nil, err
	}
	if err = rdr.Close(); err != nil {
		return nil, err
	}

	return b, nil
}


func LeafKeys(verify Key, data []byte, leafSize uint32) ([]Key, error) {
	if len(data) < KeySize {
		return nil, errors.New("the last hash in the file is not the checksum")
	}
	if !bytes.Equal(data[len(data)-KeySize:], verify[:]) {
		return nil, errors.New("the last hash in the file is not the checksum")
	}

	return leafKeysInternVerify(data, leafSize)
}


func leafKeysInternVerify(data []byte, leafSize uint32) ([]Key, error) {
	if len(data) < KeySize {
		return nil, errors.New("the last hash in the file is not the checksum")
	}
	verify, err := NewKey(data[len(data)-KeySize:])
	if err != nil {
		return nil, err
	}

	keys := make([]Key, 0, len(data)/KeySize-1)
	for i := 0; i < len(data)-KeySize; i += KeySize {
		key, kerr := NewKey(data[i : i+KeySize])
		if kerr != nil {
			return nil, kerr
		}
		keys = append(keys, key)
	}

	checksum, err := RootHash(keys, leafSize)
	if err != nil {
		return nil, err
	}
	if verify != checksum {
		return nil, fmt.Errorf("leaves (count: %d) checksum doesn't match hash value\n\t%s\n\t%s", len(keys), verify, checksum)
	}
	return keys, nil
}
