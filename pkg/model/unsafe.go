// +build !appengine

package model

import (
	"reflect"
	"unsafe"
)

// https://stackoverflow.com/questions/32223562/how-to-convert-uintptr-to-byte-in-golang
const sizeOfUintPtr = unsafe.Sizeof(uintptr(0))

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

func Uint64ToBytes(u *uint64) []byte {
	return (*[sizeOfUintPtr]byte)(unsafe.Pointer(u))[:]
}
