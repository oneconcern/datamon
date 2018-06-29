// +build !appengine

package store

import (
	"reflect"
	"unsafe"
)

// UnsafeStringToBytes converts strings to []byte without memcopy
func UnsafeStringToBytes(s string) []byte {
	ln := len(s)
	/* #nosec */
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Len:  ln,
		Cap:  ln,
		Data: (*(*reflect.StringHeader)(unsafe.Pointer(&s))).Data,
	}))
}

// UnsafeBytesToString converts []byte to string without a memcopy
func UnsafeBytesToString(b []byte) string {
	/* #nosec */
	return *(*string)(unsafe.Pointer(&reflect.StringHeader{Data: uintptr(unsafe.Pointer(&b[0])), Len: len(b)}))
}
