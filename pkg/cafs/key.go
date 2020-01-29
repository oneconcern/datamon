package cafs

import (
	"encoding/hex"
	"fmt"

	blake2b "github.com/minio/blake2b-simd"
)

const (
	// KeySize for blake2b algo
	KeySize = blake2b.Size
)

// KeyFromString constructs a key from an hex-encoded string
func KeyFromString(kv string) (Key, error) {
	vb, err := hex.DecodeString(kv)
	if err != nil {
		return Key{}, err
	}
	return NewKey(vb)
}

// NewKey creates a new key from data
func NewKey(data []byte) (Key, error) {
	var k Key
	n := copy(k[:], data)
	if n != KeySize {
		return Key{}, &BadKeySize{Key: data}
	}
	return k, nil
}

// MustNewKey creates a new key from data but panics if there is an error
func MustNewKey(data []byte) Key {
	k, e := NewKey(data)
	if e != nil {
		panic(e.Error())
	}
	return k
}

// Key represents the address of an object stored in the CAFS
type Key [KeySize]byte

// String representation of a key, hex-encoded
func (k Key) String() string {
	return hex.EncodeToString(k[:])
}

// StringWithPrefix prepends a prefix to the string representation of a key
func (k Key) StringWithPrefix(prefix string) string {
	return prefix + k.String()
}

//TODO(fred): nice - refactor key collections as []byte with this new type

// BadKeySize is an error that's returned when the key to create has an invalid size.
type BadKeySize struct {
	Key []byte
}

func (b *BadKeySize) Error() string {
	return fmt.Sprintf("%x has invalid size of %d, expected %d", b.Key, len(b.Key), KeySize)
}
