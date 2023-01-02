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
		// AllKeys returns a iterator over all keys in the DB
		AllKeys() kvIterator
	}

	// kvIterator provides a simplified abstraction for some KV iterator
	kvIterator interface {
		Next() bool
		Item() ([]byte, []byte, error)
		Close() error
	}
)

// openKV opens a kvStore
func openKV(pth string, options *purgeOptions) (kvStore, error) {
	return makeKVBadger(pth, options)
}
