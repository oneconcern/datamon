// +build !appengine

package model

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
		Data: (*reflect.StringHeader)(unsafe.Pointer(&s)).Data,
	}))
}
