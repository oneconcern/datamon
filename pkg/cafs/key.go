package cafs

import (
	"encoding/hex"
	"fmt"
)

const (
	// KeySize for blake2b algo
	KeySize = 64

	// KeySizeHex for hex representation of a key
	KeySizeHex = 66
)

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

// Key type for CAFS keys
type Key [KeySize]byte

func (k Key) String() string {
	return hex.EncodeToString(k[:])
}

// BadKeySize is an error that's returned when the key to create has an invalid size.
type BadKeySize struct {
	Key []byte
}

func (b *BadKeySize) Error() string {
	return fmt.Sprintf("%x has invalid size of %d, expected %d", b.Key, len(b.Key), KeySize)
}
