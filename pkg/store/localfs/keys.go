package localfs

import "github.com/oneconcern/trumpet/pkg/store"

const (
	indexDb = "index"
	repoDb  = "repos"
)

var (
	pathPref    = [5]byte{'p', 'a', 't', 'h', ':'}
	commitPref  = [7]byte{'c', 'o', 'm', 'm', 'i', 't', ':'}
	objectPref  = [7]byte{'o', 'b', 'j', 'e', 'c', 't', ':'}
	treePref    = [5]byte{'t', 'r', 'e', 'e', ':'}
	deletedPref = [8]byte{'d', 'e', 'l', 'e', 't', 'e', 'd', ':'}
)

func objectKey(key string) []byte {
	return append(objectPref[:], store.UnsafeStringToBytes(key)...)
}

func objectKeyBytes(key []byte) []byte {
	return append(objectPref[:], key...)
}

func pathKey(key string) []byte {
	return append(pathPref[:], store.UnsafeStringToBytes(key)...)
}

func deletedKey(key string) []byte {
	return append(deletedPref[:], store.UnsafeStringToBytes(key)...)
}

func commitKey(key string) []byte {
	return append(commitPref[:], store.UnsafeStringToBytes(key)...)
}
