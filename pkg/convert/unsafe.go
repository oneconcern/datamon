//go:build !appengine
// +build !appengine

package convert

import (
	"reflect"
	"unsafe"
)

// https://stackoverflow.com/questions/32223562/how-to-convert-uintptr-to-byte-in-golang
const sizeOfUintPtr = unsafe.Sizeof(uintptr(0))

// UnsafeStringToBytes converts strings to []byte without memcopy
func UnsafeStringToBytes(s string) []byte {
	ln := len(s)
	// nolint:  govet
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Len:  ln,
		Cap:  ln,
		Data: (*reflect.StringHeader)(unsafe.Pointer(&s)).Data,
	}))
}

// Uint64ToBytes converts a uint64 to []byte without memcopy
func Uint64ToBytes(u *uint64) []byte {
	return (*[sizeOfUintPtr]byte)(unsafe.Pointer(u))[:]
}

// Int64ToBytes converts a int64 to []byte without memcopy
func Int64ToBytes(u *int64) []byte {
	return (*[sizeOfUintPtr]byte)(unsafe.Pointer(u))[:]
}

// BytesToUint64 converts []byte to uint64 without memcopy
func BytesToUint64(b []byte) uint64 {
	return *(*uint64)(unsafe.Pointer(&b[0]))
}

// BytesToInt64 converts []byte to int64 without memcopy
func BytesToInt64(b []byte) int64 {
	return *(*int64)(unsafe.Pointer(&b[0]))
}

// UnsafeBytesToString converts []byte to string without a memcopy
func UnsafeBytesToString(b []byte) string {
	// nolint:  govet
	return *(*string)(unsafe.Pointer(&reflect.StringHeader{Data: uintptr(unsafe.Pointer(&b[0])), Len: len(b)}))
}
