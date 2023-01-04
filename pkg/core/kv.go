package core

type (
	// kvStore provides an abstraction of what the purge process expects
	// from some underlying KV store implementation.
	kvStore interface {
		// Drop removes all keys in the store
		Drop() error
		// Size reports about the size in bytes of the DB
		Size() uint64
		// Close the DB
		Close() error
		// Exists returns true if a key exists
		Exists([]byte) (bool, error)
		// Get the value for a key
		Get([]byte) ([]byte, error)
		// Set a key with some value
		Set([]byte, []byte) error
		// Set a key if it does not exists (upsert)
		SetIfNotExists([]byte, []byte) error
		// AllKeys returns an iterator over all keys in the DB
		AllKeys() kvIterator
	}

	// kvIterator provides a simplified abstraction to a KV iterator
	kvIterator interface {
		Next() bool
		Item() ([]byte, []byte, error)
		Close() error
	}
)

// openKV opens a kvStore. Select the appropriate KV implementation with the provided options.
func openKV(pth string, options *purgeOptions) (kvStore, error) {
	switch options.kvType {
	case KVTypeBadger:
		return makeKVBadger(pth, options)
	default:
		return makeKVPebble(pth, options)
	}
}
