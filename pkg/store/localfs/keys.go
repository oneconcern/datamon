package localfs

import "github.com/oneconcern/trumpet/pkg/store"

const (
	indexDb    = "index"
	snapshotDb = "snapshots"
	repoDb     = "repos"
)

var (
	pathPref      = [5]byte{'p', 'a', 't', 'h', ':'}
	commitPref    = [7]byte{'c', 'o', 'm', 'm', 'i', 't', ':'}
	objectPref    = [7]byte{'o', 'b', 'j', 'e', 'c', 't', ':'}
	treePref      = [5]byte{'t', 'r', 'e', 'e', ':'}
	deletedPref   = [8]byte{'d', 'e', 'l', 'e', 't', 'e', 'd', ':'}
	branchPref    = [7]byte{'b', 'r', 'a', 'n', 'c', 'h', ':'}
	snapshotPref  = [9]byte{'s', 'n', 'a', 'p', 's', 'h', 'o', 't', ':'}
	bsnapshotPref = [5]byte{'s', 'n', 'c', 'o', ':'}
	tagPref       = [4]byte{'t', 'a', 'g', ':'}
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

func commitKeyBytes(key []byte) []byte {
	return append(commitPref[:], key...)
}

func branchKey(key string) []byte {
	return append(branchPref[:], store.UnsafeStringToBytes(key)...)
}

func tagKey(key string) []byte {
	return append(tagPref[:], store.UnsafeStringToBytes(key)...)
}

func snapshotKey(key string) []byte {
	return snapshotKeyBytes(store.UnsafeStringToBytes(key))
}

func snapshotKeyBytes(key []byte) []byte {
	return append(snapshotPref[:], key...)
}

func bundleSnapshotKey(key string) []byte {
	return append(bsnapshotPref[:], store.UnsafeStringToBytes(key)...)
}
